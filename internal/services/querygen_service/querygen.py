import hashlib
import json
import os
from typing import List, Optional

from groq import Groq
from pydantic import BaseModel, Field


class NormalizedQuery(BaseModel):
  cleaned_query: str = Field("", alias="CleanedQuery")
  data_entity: str = Field("", alias="DataEntity")
  output_format: str = Field("", alias="OutputFormat")
  location: str = Field("", alias="Location")
  country_code: Optional[str] = Field(None, alias="CountryCode")
  start_date: str = Field("", alias="StartDate")
  end_date: str = Field("", alias="EndDate")
  sources: List[str] = Field(default_factory=list, alias="Sources")
  
  def __str__(self):
    return f"{self.data_entity}+{self.location}+{self.country_code}+{self.output_format}+{self.start_date}:{self.end_date}"

  class Config:
    allow_population_by_field_name = True
    extra = "ignore"


class QueryGenerator:
  def __init__(self):
    api_key = os.environ.get("GROQ_API_KEY")
    if not api_key:
      raise ValueError("GROQ_API_KEY not set")
    self.client = Groq(api_key=api_key)
  
  def hash_query(self, query: NormalizedQuery) -> str:
    return hashlib.sha256(str(query).encode("utf-8")).hexdigest()

  def deduplicate_queries(self, queries: List[NormalizedQuery]) -> List[NormalizedQuery]:
    seen = set()
    unique = []
    for q in queries:
      h = self.hash_query(q)
      if h in seen:
        continue
      seen.add(h)
      unique.append(q)
    return unique
  
  def generateQueries(self) -> List[NormalizedQuery]:
    raw_output = self.client.chat.completions.create(
      messages=[{
        "role": "system", 
        "content": 
"""
Generate structured output of EXACTLY the following format:
type NormalizedQuery struct {
	CleanedQuery string
	DataEntity   string   `json:"dataEntity,omitempty"`
	OutputFormat string   `json:"outputFormat,omitempty"`
	Location     string   `json:"location,omitempty"`
	CountryCode  string   `json:"cc, omitempty"`
	StartDate    string   `json:"startDate,omitempty"`
	EndDate      string   `json:"endDate,omitempty"`
	Sources      []string `json:"sources"` // normalized URLs
}  
DataEntity is a geospatial entity (precipitation, manuresheds, HUCs, etc. Refer to USGS/EPA/NASA geospatial data entities)
Sources are links to FTP/AWS S3-like URLs from US Government Agencies in Scientific Research or IBM earth, Google Earth, etc. They MUST be easily scrapable.
OutputFormat must be a geospatial format: geotiff, JSON, geoJSON, geoDB, shapefile, csv, hdf, zip, etc.
StartDate, EndDate: %dd-%md-%yyyy. These should be time ranges when such data is likely to be found.
CleanedQueryString: leave blank.
"""}, {"role": "user","content": "Generate 500 UNQIUE 'NormalizedQuery' JSONs at a time"}],
      model="llama-3.3-70b-versatile",
      response_format={"type": "json_object"}
    )
    
    content = raw_output.choices[0].message.content
    data = json.loads(content)

    # Normalize the payload into a list of dicts
    if isinstance(data, dict):
      if "queries" in data:
        data = data["queries"]
      elif "normalizedQueries" in data:
        data = data["normalizedQueries"]
      elif len(data) == 1:
        data = next(iter(data.values()))
      else:
        data = []

    if not isinstance(data, list):
      data = []

    structured_output = [NormalizedQuery.model_validate(d) for d in data]
    return structured_output
