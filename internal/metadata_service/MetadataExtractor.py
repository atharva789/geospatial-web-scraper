import grpc
from concurrent import futures
import asyncio
import trafilatura
import extractor_pb2
import extractor_pb2_grpc

class ExtractorServicer(extractor_pb2_grpc.ExtractorServicer):
    def Extract(self, request, context):
        html = request.html
        url = request.url if request.url else None
        
        ### decide html-text or trafilatura here?
        extracted_data_json_one = await extract_trafilatura(html)
        extracted_data_json_two = await extract_html_text(html)
        # compare 

        

        json_result = extracted_data_json_one

        if not json_result:
            json_result = '{"error": "extraction failed"}'

        return extractor_pb2.ExtractResponse(json_result=json_result)

def extract_trafilatura(html):
    return trafilatura.extract(html, output_format="json", with_metadata=True)


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    extractor_pb2_grpc.add_ExtractorServicer_to_server(ExtractorServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("Server running on port 50051")
    server.wait_for_termination()

if __name__ == "__main__":
    serve()