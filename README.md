# Geospatial Web Scraper + RAG Search

An automated pipeline that discovers geospatial datasets through LLM-generated queries, crawls public data sources to extract comprehensive metadata (including FGDC-compliant XML), and provides semantic search via vector embeddings. Built to demonstrate data engineering, geospatial metadata extraction, and applied search/LLM integration.

## What Works Today

- **Query Generation**: Python service uses LLM (Groq) to generate diverse, structured geospatial search queries on a schedule
- **Intelligent Crawling**: Go crawler service discovers datasets from seed URLs and Google Search results, extracting rich metadata
- **Metadata Extraction**: Comprehensive extraction from HTML and FGDC XML, including:
  - Spatial bounds (bounding boxes)
  - Horizontal metadata (CRS, lat/long resolution, units)
  - Vertical metadata (altitude CRS, resolution, units)
  - Temporal extent (start/end dates)
  - Keywords, descriptions, and source information
- **FTP/S3 Indexing**: Automatically indexes geospatial files from FTP directories and S3 buckets, matching data files with metadata
- **Vector Search**: Go RAG service provides semantic search over discovered datasets using pgvector
- **Modern UI**: Next.js 14 frontend with clean search interface
- **Production Ready**: Docker Compose stack, health checks, graceful shutdowns, comprehensive test coverage
- **CI/CD**: GitHub Actions runs Go tests (22 passing), UI builds, and Python linting

## Architecture

```
┌─────────────────┐      ┌──────────────────┐      ┌────────────────┐
│  QueryGen       │─────▶│  Crawler         │─────▶│  PostgreSQL    │
│  Service        │ HTTP │  Service         │      │  (pgvector)    │
│  (Python)       │      │  (Go)            │      └────────────────┘
│                 │      │                  │              │
│ - LLM query gen │      │ - Google Search  │              ▼
│ - Deduplication │      │ - FTP indexing   │      ┌────────────────┐
│ - Daily cron    │      │ - S3 indexing    │      │  RAG Service   │
└─────────────────┘      │ - HTML parsing   │      │  (Go)          │
                         │ - FGDC XML parse │      │                │
                         │ - Metadata save  │      │ - Vector search│
                         └──────────────────┘      │ - Embeddings   │
                                                    └────────────────┘
                                                            │
                                                            ▼
                                                    ┌────────────────┐
                                                    │  UI (Next.js)  │
                                                    │  Port 3000     │
                                                    └────────────────┘
```

## Core Services

### QueryGen Service (Python/FastAPI)
- **Port**: 50052
- **Purpose**: Generates diverse geospatial search queries using LLM
- **Features**:
  - Scheduled daily query generation (cron: 12:00)
  - Deduplication via SHA-256 hashing
  - Generates 500 unique queries per run
  - Automatically triggers crawler for new queries
- **Endpoints**: `/`, `/begin`

### Crawler Service (Go)
- **Port**: 8080
- **Purpose**: Discovers and catalogs geospatial datasets
- **Features**:
  - Google Custom Search API integration
  - Recursive web crawling (max depth 4)
  - FTP directory indexing with metadata matching
  - S3 bucket listing support
  - FGDC XML metadata parsing
  - HTML metadata extraction (meta tags, Open Graph)
  - PostgreSQL persistence with comprehensive schema
- **Endpoints**: `/crawl`, `/test`, `/healthz`, `/ready`
- **Supported Formats**: GeoTIFF, NetCDF, Shapefile, GeoJSON, HDF5, LAS/LAZ, GeoPackage, and more

### RAG Service (Go)
- **Port**: 8082
- **Purpose**: Semantic search over dataset metadata
- **Features**:
  - pgvector integration for similarity search
  - Bedrock embeddings (optional, falls back to hash)
  - Demo data seeding for testing
  - CORS support
- **Endpoints**: `/search`, `/seed`, `/sync`, `/healthz`, `/ready`

### UI (Next.js 14)
- **Port**: 3000
- **Purpose**: Clean search interface for datasets
- **Features**:
  - Real-time search with ranked results
  - Dataset metadata display
  - Responsive design

## Data Model

### DatasetMetadata Structure
```go
type DatasetMetadata struct {
    Title          string         // Dataset title
    Source         string         // Provider (e.g., "USGS")
    Description    string         // Long-form description
    Keywords       []string       // Searchable keywords
    URL            string         // Download URL
    Bounds         BoundingBox    // Geographic extent
    HorizontalMeta HorizontalMeta // Lat/long metadata
    VerticalMeta   VerticalMeta   // Elevation metadata
    StartDate      string         // Temporal start (ISO 8601)
    EndDate        string         // Temporal end (ISO 8601)
}

type HorizontalMeta struct {
    LatRes        float64 // Latitude resolution
    LongRes       float64 // Longitude resolution
    GeoUnit       string  // Units (e.g., "degrees", "meters")
    HorizontalCRS string  // Coordinate system (e.g., "EPSG:4326")
}

type VerticalMeta struct {
    VerticalCRS string  // Vertical datum (e.g., "NAVD88")
    AltRes      float64 // Altitude resolution
    AltUnits    string  // Altitude units (e.g., "meters")
}
```

## Run It Locally

**Prerequisites**: Docker/Compose, Node 20, Go 1.22, Python 3.11

1. **Configure environment**:
   ```bash
   cp deploy/compose/.env.example deploy/compose/.env
   # Edit .env and add required API keys:
   # - GROQ_API_KEY (for query generation)
   # - GOOGLE_API_KEY, GOOGLE_CSE_ID (for crawler)
   # - AWS credentials (optional, for Bedrock embeddings)
   ```

2. **Start the stack**:
   ```bash
   cd deploy/compose
   docker compose up --build
   ```

3. **Access services**:
   - **UI**: http://localhost:3000
   - **RAG API**: http://localhost:8082
   - **Crawler**: http://localhost:8080
   - **QueryGen**: http://localhost:50052

4. **Seed demo data** (optional):
   ```bash
   curl -X POST http://localhost:8082/seed
   ```

5. **Trigger manual crawl**:
   ```bash
   curl http://localhost:50052/begin
   ```

## Development

### Run Tests
```bash
# All tests
make test

# Go tests only (crawler + RAG)
make go-test

# Python tests
make python-test

# Python lint
make python-lint

# UI build
make ui-build

# Full CI suite
make ci
```

### Individual Service Testing
```bash
# Crawler service (22 tests)
cd internal/services/crawler_service
go test -v ./...

# RAG service
cd internal/services/rag_service
go test -v ./...

# QueryGen service
cd internal/services/querygen_service
python -m pytest test_querygen.py -v
```

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`) runs on every push:
- ✅ Go tests for crawler_service (22 tests including FGDC XML parsing)
- ✅ Go tests for rag_service
- ✅ Python linting for querygen_service
- ✅ Python tests for querygen_service
- ✅ Next.js UI build

## Database Schema

### Datasets Table
```sql
CREATE TABLE datasets (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    source TEXT,
    description TEXT,
    keywords TEXT[],
    url TEXT UNIQUE NOT NULL,
    west_bc DOUBLE PRECISION,
    east_bc DOUBLE PRECISION,
    north_bc DOUBLE PRECISION,
    south_bc DOUBLE PRECISION,
    lat_res DOUBLE PRECISION,
    long_res DOUBLE PRECISION,
    geo_unit TEXT,
    horizontal_crs TEXT,
    vertical_crs TEXT,
    alt_res DOUBLE PRECISION,
    alt_units TEXT,
    start_date TEXT,
    end_date TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Vector Embeddings (RAG)
```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE dataset_vectors (
    id SERIAL PRIMARY KEY,
    dataset_id INTEGER REFERENCES datasets(id),
    embedding vector(1536),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Supported Metadata Formats

### HTML Metadata
- Standard meta tags (`description`, `keywords`)
- Open Graph protocol (`og:title`, `og:description`)
- Schema.org structured data
- Header tags (h1, h2) and paragraph content

### FGDC XML (Federal Geographic Data Committee)
Fully supports FGDC CSDGM metadata standard:
- **Citation**: Title, origin, publication date
- **Description**: Abstract, purpose
- **Spatial Domain**: Bounding coordinates
- **Temporal Extent**: Begin/end dates
- **Spatial Reference**:
  - Horizontal: Geographic resolution, units, datum (e.g., NAD83)
  - Vertical: Altitude datum, resolution, units (e.g., NAVD88)
- **Keywords**: Theme keywords from controlled vocabularies

## Production Deployment Notes

### Infrastructure
- **Database**: Use AWS RDS with pgvector extension or Aurora PostgreSQL
- **Secrets**: Store in AWS Secrets Manager or Parameter Store
- **Networking**: Enable TLS for all database connections
- **IAM**: Least-privilege roles for S3, Bedrock, and RDS access

### Scaling Considerations
- Crawler can run multiple instances with shared PostgreSQL for coordination
- QueryGen runs as singleton (cron-based, uses DB for deduplication)
- RAG service is stateless and horizontally scalable
- Consider rate limiting for Google Search API calls

### Monitoring
- Expose Prometheus metrics for crawler throughput, metadata extraction rates
- Track query generation success rates and LLM token costs
- Monitor pgvector query latencies
- Set up alerts for service health endpoints

## Roadmap

- [ ] Integration tests using Testcontainers for PostgreSQL/pgvector
- [ ] IaC templates (Terraform/CDK) for AWS deployment
- [ ] UI improvements: filtering by date range, location, format
- [ ] Additional metadata standards: ISO 19115, STAC (SpatioTemporal Asset Catalog)
- [ ] Webhook support for real-time crawl triggers
- [ ] Admin dashboard for monitoring crawl statistics
- [ ] Export search results as CSV/GeoJSON

## License

MIT

## Contributing

Contributions welcome! Please ensure:
- All tests pass (`make test`)
- Code is formatted (`go fmt`, `black` for Python)
- New features include tests
- Update README for significant changes
