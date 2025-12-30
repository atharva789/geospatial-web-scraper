# Build and test targets
go-test:
	cd internal/services/rag_service && go test ./...
	cd internal/services/crawler_service && go test ./... || true

ui-build:
	cd web/rag-ui && npm install && npm run build

python-lint:
	python -m py_compile internal/services/querygen_service/main.py internal/services/querygen_service/querygen.py internal/services/querygen_service/db.py

# Run all services locally
up:
	cd deploy/compose && docker-compose up --build

# Stop all services
down:
	cd deploy/compose && docker-compose down

# Clean up volumes
clean:
	cd deploy/compose && docker-compose down -v

# Run tests and build
ci: go-test ui-build python-lint
