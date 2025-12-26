# Geospatial Ingestion + RAG Search

An end-to-end pipeline that crawls geospatial data sources, streams events through Kafka, batch-labels them with an LLM, stores embeddings in pgvector, and exposes a search API with a simple UI. Built to show strength in data engineering, containers, and applied search/LLM integration.

## What works today
- **Kafka pipeline**: Go crawler discovers dataset links and pushes `dataset-event` messages to Kafka.
- **PySpark consumer**: Reads the Kafka topic, batches events, calls an LLM for labels, writes labels to S3, and inserts embeddings into pgvector (RDS-ready).
- **RAG backend**: Go service queries pgvector for semantic search; health/ready endpoints and graceful shutdowns are in place. Optional demo seeding for instant results.
- **UI**: Next.js 14 frontend calling `/search` and rendering ranked datasets.
- **Compose stack**: One `docker compose up --build` brings up Kafka, Postgres/pgvector, crawler, normalizer (gRPC), query generator, consumer, RAG backend, and UI.
- **CI**: GitHub Actions runs Go tests, Next.js build, and Python syntax checks; Makefile targets wrap common tasks.

## Core pieces
- **Data engineering**: Kafka ingestion, PySpark streaming, S3 writes, pgvector storage, schema bootstrap on service start.
- **Containerization**: All services are Dockerized; Compose orchestrates the stack for local/dev.
- **Search/LLM**: LLM-based labeling (REST) with optional Bedrock embeddings; hash fallback keeps the pipeline running without cloud creds.

## Run it locally
Prereqs: Docker/Compose, Node 20, Go 1.22, Python 3.11.

1) Copy `.env.example` → `.env` and fill secrets (Kafka, Postgres/pgvector, GROQ_API_KEY, LLM_API_URL/KEY, S3 creds, optional Bedrock).
2) `docker compose -f deploy/compose/docker-compose.yaml up --build`
   - RAG backend: `http://localhost:8082` (`/search`, `/healthz`, `/ready`)
   - UI: `http://localhost:3000`
   - Crawler: `http://localhost:8080/test`
   - Normalizer (gRPC): `localhost:50051`
   - Query generator: `http://localhost:50052`
3) For instant search data, set `SEED_DEMO_DATA=true` with pgvector envs.

## Make targets
- `make gen` — regenerate gRPC stubs.
- `make go-test` — Go unit tests.
- `make ui-build` — Next.js build.
- `make python-lint` — Python syntax checks.
- `make ci` — run all of the above.

## CI
GitHub Actions workflow (`.github/workflows/ci.yml`) runs:
- `go test ./...` in the RAG service
- `npm run build` for the UI
- Python bytecode checks for consumer/querygen

## Production-minded notes
- Use RDS with pgvector or Aurora Postgres; Kafka via MSK; Bedrock for embeddings (falls back to hash if unset).
- Store secrets in AWS Secrets Manager/SSM; use TLS for DB/Kafka; lock down IAM for S3/RDS/MSK/Bedrock.
- Add metrics/tracing (Prometheus/OTel) and alerting; add an LLM eval harness with a labeled sample set to measure label quality/latency/cost.

## Roadmap
- Integration tests (Testcontainers for pgvector/Kafka) and UI smoke tests (Playwright).
- IaC (CDK/Terraform) for RDS+pgvector, MSK, ECS/Fargate services, S3, Bedrock permissions.
- LLM evaluation harness and simple dashboards for latency/error rates.
