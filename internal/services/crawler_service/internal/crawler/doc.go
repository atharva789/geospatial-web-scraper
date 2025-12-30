/*
Package crawler implements a concurrent web crawler specialized for discovering
geospatial dataset URLs from public data sources.

# Overview

The crawler performs intelligent, query-driven discovery of geospatial datasets by:
  - Starting from curated seed URLs (USGS, NOAA, NASA, etc.)
  - Performing Google Custom Search for additional sources
  - Recursively following links up to a maximum depth
  - Identifying geospatial file formats (GeoTIFF, Shapefile, NetCDF, etc.)
  - Extracting metadata from HTML pages
  - Deduplicating discoveries across crawl sessions
  - Persisting results to PostgreSQL for semantic search

# Architecture

The crawler uses a breadth-first traversal with token-based concurrency control:

	┌─────────────┐
	│ API Handler │ receives NormalizedQuery
	└──────┬──────┘
	       │
	       v
	┌──────────────┐
	│ CrawlManager │ orchestrates the crawl session
	└──────┬───────┘
	       │
	       ├──> Seed URLs (data.go)
	       ├──> Google Search API
	       │
	       v
	┌──────────────┐
	│  Worklist    │ queue of CrawlNodes to process
	└──────┬───────┘
	       │
	       ├──> [Token Pool: 40 concurrent fetches]
	       │
	       v
	┌──────────────┐
	│ HTML Parser  │ extracts links & metadata
	└──────┬───────┘
	       │
	       ├──> Geospatial URLs → Database
	       └──> Regular URLs → Back to worklist

# Core Types

  - NormalizedQuery: Structured search with location, data type, temporal extent
  - CrawlManager: Orchestrates a single crawl session with concurrency control
  - CrawlNode: A discovered URL with parent relationship and relevance score
  - DataContext: Metadata about a dataset (title, description, embeddings)

# Usage Example

Basic crawl execution:

	query := crawler.NormalizedQuery{
	    CleanedQuery: "lidar elevation data Washington State",
	    DataEntity:   "lidar",
	    OutputFormat: "LAS",
	    Location:     "Washington",
	}

	if err := crawler.Run(query); err != nil {
	    log.Fatalf("Crawl failed: %v", err)
	}

# Supported File Formats

The crawler recognizes numerous geospatial formats:

  Raster: GeoTIFF (.tif, .tiff), NetCDF (.nc), GRIB (.grib), HDF (.hdf, .hdf5)
  Vector: Shapefile (.shp), GeoJSON (.geojson), KML (.kml), GML (.gml)
  Point Cloud: LAS (.las), LAZ (.laz)
  Database: GeoPackage (.gpkg), File Geodatabase (.gdb), SpatiaLite (.sqlite)

See GeoFileExtensions and GeoMIMETypes in data.go for the complete list.

# Database Schema

Discovered datasets are stored in the `discovered_datasets` table:

	CREATE TABLE discovered_datasets (
	    id SERIAL PRIMARY KEY,
	    dataset_url TEXT UNIQUE NOT NULL,
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

# Concurrency Model

The crawler uses Go channels for concurrency control:

  - smTokens: Semaphore limiting concurrent HTTP requests (40)
  - dlTokens: Semaphore limiting concurrent downloads (40)
  - worklist: Buffered channel for BFS queue
  - done: Signal channel for graceful shutdown
  - seen: Map for URL deduplication

This design prevents overwhelming target servers while maintaining high throughput.

# Error Handling

Errors are handled gracefully at multiple levels:

  - Network errors: Logged and skipped (crawl continues)
  - Parse errors: Logged and skipped
  - Database errors: Returned to caller (critical failure)
  - API errors: Logged and retried with exponential backoff

# Performance Characteristics

Typical performance metrics:

  - Throughput: 40 concurrent requests with ~100ms average latency = ~400 URLs/sec
  - Discovery rate: 10-50 dataset URLs per 100 pages crawled (depends on seed quality)
  - Memory usage: ~50 MB base + ~1 KB per URL in worklist
  - Database writes: Batched transactions for efficiency

# Future Enhancements

Planned improvements include:

  - Distributed crawling with Redis-based coordination
  - Incremental updates (only re-crawl stale URLs)
  - Content-based deduplication (hash identical files)
  - LLM-powered metadata enrichment
  - Robots.txt compliance
  - Rate limiting per domain
  - S3 direct downloads for discovered datasets

*/
package crawler
