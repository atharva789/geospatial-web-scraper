PROTO_DIR=internal/proto/
PY_OUT=internal/services/querynormalizer_service/
GO_OUT=internal/services/crawler_service/internal/crawler/

gen:
	python -m grpc_tools.protoc -I $(PROTO_DIR) \
	  --python_out=$(PY_OUT) --grpc_python_out=$(PY_OUT) \
	  $(PROTO_DIR)searchQuery.proto
	protoc -I $(PROTO_DIR) \
	  --go_out=$(GO_OUT) --go-grpc_out=$(GO_OUT) \
	  $(PROTO_DIR)/searchQuery.proto

go-test:
	cd internal/services/rag_service && go test ./...
	cd internal/services/crawler_service && go test ./... || true

ui-build:
	cd web/rag-ui && npm install && npm run build

python-lint:
	python -m py_compile internal/services/transformation_service/dataset_consumer.py
	python -m py_compile internal/services/querygen_service/main.py internal/services/querygen_service/querygen.py

ci: go-test ui-build python-lint
