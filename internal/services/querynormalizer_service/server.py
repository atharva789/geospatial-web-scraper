import grpc, time
from concurrent import futures
import searchQuery_pb2_grpc as pb
from sentenceParsing import QueryNormalizer 

def serve(addr="0.0.0.0:50051"):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    pb.add_NormalizerServiceServicer_to_server(QueryNormalizer(), server)
    server.add_insecure_port(addr)
    server.start()
    print(f"normalizer gRPC on {addr}")
    try:
        while True: time.sleep(9000)
    except KeyboardInterrupt:
        server.stop(0)
if __name__ == "__main__":
    serve()

