package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type seedRecord struct {
	URL       string
	Label     string
	Rationale string
}

func SeedDemo(ctx context.Context, pool *pgxpool.Pool, table string, dim int) error {
	samples := []seedRecord{
		{
			URL:       "https://example.com/precipitation-ohio-2010-2020.tif",
			Label:     "NOAA precipitation rasters for Ohio (2010-2020)",
			Rationale: "GeoTIFF rasters covering Ohio precipitation over a decade window from NOAA archives",
		},
		{
			URL:       "https://example.com/usgs-huc8-shapefiles.zip",
			Label:     "USGS HUC8 watershed shapefiles (CONUS)",
			Rationale: "Shapefile bundle for hydrologic unit codes level 8, continental US",
		},
		{
			URL:       "https://example.com/epa-airquality-2022.csv",
			Label:     "EPA air quality monitoring CSVs (2022)",
			Rationale: "Station readings, PM2.5/O3 values, nationwide for 2022",
		},
	}

	if err := ensureSchema(ctx, pool, table, dim); err != nil {
		return err
	}

	// check if data exists
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(1) FROM "+table).Scan(&count); err != nil {
		log.Printf("seed check failed: %v", err)
	}
	if count > 0 {
		return nil
	}

	embedder := hashEmbedder{dim: dim}
	type row struct {
		url       string
		label     string
		rationale string
		embedding []float32
	}
	var rows []row
	for _, s := range samples {
		emb, _ := embedder.Embed(ctx, s.Label+" "+s.Rationale)
		rows = append(rows, row{
			url:       s.URL,
			label:     s.Label,
			rationale: s.Rationale,
			embedding: emb,
		})
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	insert := "INSERT INTO " + table + " (dataset_url, label, rationale, embedding, created_at) VALUES ($1,$2,$3,$4,$5)"
	now := time.Now().UTC()
	for _, r := range rows {
		if _, err := tx.Exec(ctx, insert, r.url, r.label, r.rationale, pgvector.NewVector(r.embedding), now); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	log.Printf("Seeded %d demo records into %s", len(rows), table)
	return nil
}
