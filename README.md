# Geospatial Data Ingestion + RAG Search

End-to-end pipeline that ingests geospatial datasets, labels them with an LLM, stores embeddings in pgvector, and exposes a RAG-style search API with a Next.js UI. Built to showcase data engineering, containerization, and applied AI search for cloud environments.

## Core Functionality
- Crawl geospatial sites, detect downloadable datasets, and stream events to Kafka.
- Normalize free-text queries (gRPC service), generate structured queries (LLM), and deduplicate.
- Classify dataset pages with an LLM batch call, store labels to S3, and insert embeddings into pgvector.
- Serve semantic search over vectors via a Go backend, with a simple Next.js frontend.

## Technologies
- **Data pipeline**: Go crawler, Kafka, PySpark streaming consumer, S3, pgvector (Postgres).
- **AI/LLM**: LLM-based labeling (REST), optional Bedrock for embeddings; Groq for query generation.
- **Services**: Go (net/http) RAG backend, FastAPI query generator, Python gRPC normalizer, Next.js 14 UI.
- **Infra/ops**: Docker Compose for local stack (Kafka, Postgres, services, UI), GitHub Actions CI, Makefile helpers.

## Key Differentiators
- **Data engineering**: Kafka-based ingestion, PySpark streaming, pgvector storage, S3 writes, schema bootstrapping.
- **Containerization**: Multi-service Compose (crawler, normalizer, querygen, consumer, RAG backend, UI, Kafka, Postgres).
- **AI-powered search**: LLM labeling + vector search, Bedrock-ready embeddings, RAG API + UI.

## Architecture (high level)
1) User query → gRPC normalizer → structured query.
2) Go crawler searches public data portals → emits `dataset-event` to Kafka.
3) PySpark consumer batches events → LLM labels → S3 archive + pgvector embeddings.
4) Go RAG backend queries pgvector → returns ranked datasets; Next.js UI calls `/search`.

## Running Locally
Prereqs: Docker/Compose, Node 20, Go 1.22, Python 3.11.

1) Copy `.env.example` to `.env` and fill secrets (Kafka, Postgres/pgvector, GROQ_API_KEY, LLM_API_URL/KEY, S3 creds, optional Bedrock).
2) `docker compose -f deploy/compose/docker-compose.yaml up --build`
   - RAG backend: `http://localhost:8082`
   - UI: `http://localhost:3000`
   - Crawler: `http://localhost:8080/test`
   - Normalizer (gRPC): `localhost:50051`
   - Query generator: `http://localhost:50052`
3) Optional: seed demo vectors for instant search results: set `SEED_DEMO_DATA=true` and ensure pgvector envs are set.

## Make Targets
- `make gen` — regenerate gRPC stubs.
- `make go-test` — Go unit tests.
- `make ui-build` — Next.js build.
- `make python-lint` — Python syntax checks.
- `make ci` — run all of the above.

## Testing & CI
- Go unit tests for RAG backend (hash embedder/parsers); extend with pgvector integration via Testcontainers for higher confidence.
- Next.js build to catch type/compile issues.
- Python bytecode checks for consumer/querygen.
- GitHub Actions workflow (`.github/workflows/ci.yml`) runs go test, npm build, and Python lint.

## Deployment Notes
- Use RDS with pgvector or Aurora Postgres for embeddings.
- Kafka: MSK or MSK Serverless; set `KAFKA_BOOTSTRAP_SERVERS` accordingly.
- Bedrock embeddings: set `BEDROCK_EMBED_MODEL` and AWS creds; falls back to hash embeddings if unset.
- Secrets: store in AWS Secrets Manager/SSM; avoid baking into images.
- Networking: enable TLS for DB/Kafka in production; lock down security groups/IAM policies (least privilege for S3/RDS/MSK/Bedrock).

## Project Status
- Local Compose stack is runnable; services expose health/readiness endpoints.
- Observability/logging: structured logs; add metrics/tracing as next step.
- LLM evaluation harness not included yet; add a labeled sample set to measure label quality, latency, and cost.

## Suggested Enhancements
- Add Testcontainers-based integration tests (pgvector, Kafka) and Playwright UI smoke tests.
- Add Prometheus/OTel metrics, dashboards, and alert thresholds.
- Add IaC (CDK/Terraform) for RDS+pgvector, MSK, ECS/Fargate services, S3 buckets, and Bedrock permissions.
