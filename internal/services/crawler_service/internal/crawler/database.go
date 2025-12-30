package crawler

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// InitDB initializes the database connection and creates tables if needed
func InitDB() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// Fallback to individual env vars
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		dbname := os.Getenv("POSTGRES_DB")
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")

		if host == "" || port == "" || dbname == "" {
			return fmt.Errorf("database connection info not provided")
		}

		databaseURL = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)
	}

	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables if they don't exist
	if err = createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	fmt.Println("Database connection established successfully")
	return nil
}

// createTables creates the necessary database tables
func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS discovered_datasets (
		id SERIAL PRIMARY KEY,
		dataset_url TEXT NOT NULL UNIQUE,
		title TEXT,
		description TEXT,
		source_query TEXT,
		data_entity TEXT,
		output_format TEXT,
		location TEXT,
		country_code TEXT,
		start_date TEXT,
		end_date TEXT,
		discovered_at TIMESTAMPTZ DEFAULT NOW(),
		metadata JSONB
	);

	CREATE INDEX IF NOT EXISTS idx_discovered_datasets_url ON discovered_datasets(dataset_url);
	CREATE INDEX IF NOT EXISTS idx_discovered_datasets_data_entity ON discovered_datasets(data_entity);
	CREATE INDEX IF NOT EXISTS idx_discovered_datasets_location ON discovered_datasets(location);
	CREATE INDEX IF NOT EXISTS idx_discovered_datasets_discovered_at ON discovered_datasets(discovered_at);
	`

	_, err := db.Exec(schema)
	return err
}

// SaveDatasets saves discovered dataset URLs to the database
func SaveDatasets(nodes []WebNode, query NormalizedQuery) error {
	if db == nil {
		if err := InitDB(); err != nil {
			return err
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO discovered_datasets
			(dataset_url, title, description, source_query, data_entity, output_format, location, country_code, start_date, end_date)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (dataset_url) DO UPDATE SET
			discovered_at = NOW()
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	insertedCount := 0
	for _, node := range nodes {
		_, err := stmt.Exec(
			node.URL,
			node.context.Title,
			node.context.Description,
			query.CleanedQuery,
			query.DataEntity,
			query.OutputFormat,
			query.Location,
			query.CountryCode,
			query.StartDate,
			query.EndDate,
		)
		if err != nil {
			fmt.Printf("Warning: failed to insert dataset %s: %v\n", node.URL, err)
			continue
		}
		insertedCount++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully saved %d datasets to database\n", insertedCount)
	return nil
}

// GetDB returns the database connection
func GetDB() *sql.DB {
	return db
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// Dataset represents a discovered dataset record
type Dataset struct {
	ID           int       `json:"id"`
	DatasetURL   string    `json:"dataset_url"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	SourceQuery  string    `json:"source_query"`
	DataEntity   string    `json:"data_entity"`
	OutputFormat string    `json:"output_format"`
	Location     string    `json:"location"`
	CountryCode  string    `json:"country_code"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

// GetRecentDatasets retrieves recently discovered datasets
func GetRecentDatasets(limit int) ([]Dataset, error) {
	if db == nil {
		if err := InitDB(); err != nil {
			return nil, err
		}
	}

	query := `
		SELECT id, dataset_url, title, description, source_query, data_entity,
		       output_format, location, country_code, start_date, end_date, discovered_at
		FROM discovered_datasets
		ORDER BY discovered_at DESC
		LIMIT $1
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query datasets: %w", err)
	}
	defer rows.Close()

	var datasets []Dataset
	for rows.Next() {
		var d Dataset
		err := rows.Scan(
			&d.ID, &d.DatasetURL, &d.Title, &d.Description, &d.SourceQuery,
			&d.DataEntity, &d.OutputFormat, &d.Location, &d.CountryCode,
			&d.StartDate, &d.EndDate, &d.DiscoveredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dataset: %w", err)
		}
		datasets = append(datasets, d)
	}

	return datasets, nil
}
