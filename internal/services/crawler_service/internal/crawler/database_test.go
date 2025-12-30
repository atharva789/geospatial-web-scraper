package crawler

import (
	"os"
	"testing"
)

func TestInitDB_MissingConfig(t *testing.T) {
	// Clear all database env vars
	originalURL := os.Getenv("DATABASE_URL")
	originalHost := os.Getenv("POSTGRES_HOST")
	originalPort := os.Getenv("POSTGRES_PORT")
	originalDB := os.Getenv("POSTGRES_DB")

	defer func() {
		os.Setenv("DATABASE_URL", originalURL)
		os.Setenv("POSTGRES_HOST", originalHost)
		os.Setenv("POSTGRES_PORT", originalPort)
		os.Setenv("POSTGRES_DB", originalDB)
	}()

	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("POSTGRES_HOST")
	os.Unsetenv("POSTGRES_PORT")
	os.Unsetenv("POSTGRES_DB")

	err := InitDB()
	if err == nil {
		t.Error("InitDB() should fail with missing configuration")
	}
	if err.Error() != "database connection info not provided" {
		t.Errorf("Expected 'database connection info not provided', got %v", err)
	}
}

func TestDatasetStruct(t *testing.T) {
	dataset := Dataset{
		ID:           1,
		DatasetURL:   "https://example.com/data.tif",
		Title:        "Test Dataset",
		Description:  "A test geospatial dataset",
		SourceQuery:  "elevation data California",
		DataEntity:   "elevation",
		OutputFormat: "GeoTIFF",
		Location:     "California",
		CountryCode:  "US",
		StartDate:    "2020-01-01",
		EndDate:      "2020-12-31",
	}

	if dataset.DatasetURL != "https://example.com/data.tif" {
		t.Errorf("DatasetURL = %v, want https://example.com/data.tif", dataset.DatasetURL)
	}
	if dataset.DataEntity != "elevation" {
		t.Errorf("DataEntity = %v, want elevation", dataset.DataEntity)
	}
}

func TestSaveDatasets_NoConnection(t *testing.T) {
	// Ensure no DB connection exists
	db = nil

	// Clear all database env vars to force failure
	originalURL := os.Getenv("DATABASE_URL")
	originalHost := os.Getenv("POSTGRES_HOST")

	defer func() {
		os.Setenv("DATABASE_URL", originalURL)
		os.Setenv("POSTGRES_HOST", originalHost)
	}()

	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("POSTGRES_HOST")

	query := NormalizedQuery{
		CleanedQuery: "test query",
		DataEntity:   "test",
	}

	nodes := []WebNode{
		{URL: "https://example.com/test.tif"},
	}

	err := SaveDatasets(nodes, query)
	if err == nil {
		t.Error("SaveDatasets() should fail without database connection")
	}
}
