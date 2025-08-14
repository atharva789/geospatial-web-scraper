import grpc
from concurrent import futures
import asyncio
import json
import html_text
import trafilatura
import extract_pb2
import extract_pb2_grpc
from urllib.parse import urlparse
import hashlib

from pydantic import BaseModel
from typing import Optional, List

class ExtractedDocument(BaseModel):
    date: Optional[str]
    title: Optional[str]
    author: Optional[str] # the source/govt agency
    url: Optional[str]
    hostname: Optional[str]
    sitename: Optional[str]
    description: Optional[str]
    text: str
    language: Optional[str]
    categories: Optional[List[str]]
    tags: Optional[List[str]]
    id: str

class ExtractorServicer(extract_pb2_grpc.ExtractorServicer):
    async def Extract(self, request, context):
        html = request.html
        url = request.url if request.url else None
        
        ### decide html-text or trafilatura here?
        extracted_data_one = await extract_trafilatura(html, url)
        extracted_data_two = await extract_html_text(html, url)
        
        #compare 
        if abs(len(extracted_data_one.description) - len(extracted_data_two.description)) <=  (0.20 * len(extracted_data_one.description)):
            if len(extracted_data_one.text) >= len(extracted_data_two.text):
                json_result = extracted_data_one
            else:
                json_result = extracted_data_two
        json_result = json_result.model_dump_json(indent=2, ensure_ascii=False)

        if not json_result:
            json_result = '{"error": "extraction failed"}'

        return extract_pb2.ExtractResponse(json_result=json_result)

async def extract_trafilatura(html: str, url: str) -> ExtractedDocument:
    json_result = trafilatura.extract(html, output_format="json")
    data = json.loads(json_result)
    doc = ExtractedDocument(**data)
    return doc

async def parse_with_html_text(html: str, url: Optional[str] = None) -> ExtractedDocument:
    title_text = None
    title_sel = sel.xpath('//title')
    if title_sel:
        title_text = html_text.selector_to_text(title_sel)
    desc_text = None
    meta_sel = sel.xpath('//meta[@name="description"]/@content')
    if meta_sel:
        desc_text = meta_sel.get().strip()
    # Full body text
    full_text = html_text.selector_to_text(sel)
    hostname = urlparse(url).hostname if url else None
    uid = hashlib.md5((full_text + (url or "")).encode('utf-8')).hexdigest()
    doc = ExtractedDocument(
        date=None,
        title=title_text,
        author=None,
        url=url,
        hostname=hostname,
        sitename=None,
        description=desc_text,
        text=full_text,
        language=None,
        categories=None,
        tags=None,
        id=uid,
    )
    return doc

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    extract_pb2_grpc.add_ExtractorServicer_to_server(ExtractorServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("Server running on port 50051")
    server.wait_for_termination()

if __name__ == "__main__":
    serve()