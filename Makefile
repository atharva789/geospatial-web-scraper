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
