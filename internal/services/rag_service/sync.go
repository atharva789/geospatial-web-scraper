package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// syncDatasets synchronizes discovered_datasets to dataset_vectors table
// This function reads from discovered_datasets and creates embeddings for new entries
func syncDatasets(ctx context.Context, pool *pgxpool.Pool, table string, embedClient embedder, vectorDim int) error {
	// Query for datasets that haven't been vectorized yet
	query := fmt.Sprintf(`
		SELECT DISTINCT d.dataset_url, d.title, d.description, d.data_entity, d.location
		FROM discovered_datasets d
		LEFT JOIN %s v ON d.dataset_url = v.dataset_url
		WHERE v.dataset_url IS NULL
		LIMIT 100
	`, table)

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query discovered datasets: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var url, title, description, dataEntity, location string
		if err := rows.Scan(&url, &title, &description, &dataEntity, &location); err != nil {
			log.Printf("Warning: failed to scan row: %v", err)
			continue
		}

		// Create a label from the available metadata
		label := fmt.Sprintf("%s - %s", dataEntity, location)
		if title != "" {
			label = title
		}

		// Create rationale from description and metadata
		rationale := description
		if rationale == "" {
			rationale = fmt.Sprintf("Dataset for %s in %s", dataEntity, location)
		}

		// Generate embedding for the dataset
		textToEmbed := fmt.Sprintf("%s %s %s %s", title, description, dataEntity, location)
		vecVals, err := embedClient.Embed(ctx, textToEmbed)
		if err != nil {
			log.Printf("Warning: failed to generate embedding for %s: %v", url, err)
			continue
		}

		// Ensure correct dimensions
		if len(vecVals) != vectorDim {
			if len(vecVals) < vectorDim {
				padded := make([]float32, vectorDim)
				copy(padded, vecVals)
				vecVals = padded
			} else {
				vecVals = vecVals[:vectorDim]
			}
		}

		vec := pgvector.NewVector(vecVals)

		// Insert into dataset_vectors table
		insertQuery := fmt.Sprintf(`
			INSERT INTO %s (dataset_url, label, rationale, embedding)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (dataset_url) DO NOTHING
		`, table)

		_, err = pool.Exec(ctx, insertQuery, url, label, rationale, vec)
		if err != nil {
			log.Printf("Warning: failed to insert vector for %s: %v", url, err)
			continue
		}

		count++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	if count > 0 {
		log.Printf("Successfully synchronized %d datasets to vector table", count)
	}

	return nil
}

// startSyncWorker runs periodic synchronization in the background
func startSyncWorker(pool *pgxpool.Pool, table string, embedClient embedder, vectorDim int) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Starting dataset sync worker...")

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		if err := syncDatasets(ctx, pool, table, embedClient, vectorDim); err != nil {
			log.Printf("Sync error: %v", err)
		}
		cancel()
	}
}
