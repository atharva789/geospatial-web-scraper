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

# {
#   "date": "2025-07-29",
#   "title": "Title",
#   "author": null,
#   "url": "https://example.org/article",
#   "hostname": "example.org",
#   "sitename": "Example Site",
#   "description": "Meta description of the page if available",
#   "text": "Hello World",
#   "language": "en",
#   "categories": null,
#   "tags": null,
#   "id": "6c92285fa6d3e827d6994220659413d6"
# }
