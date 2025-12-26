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
You are an API that ONLY returns a single JSON object with one key: "normalizedQueries".
normalizedQueries is an array of objects with EXACTLY these fields:
{
  "CleanedQuery": "",            // always empty string
  "DataEntity": "",              // geospatial entity (precipitation, manuresheds, HUC8/12 watersheds, NLCD landcover, NHD flowlines, soils, DEM, etc.)
  "OutputFormat": "",            // geospatial file format: geotiff, geojson, geopackage, shapefile, csv, hdf, netcdf, zip, parquet, etc.
  "Location": "",                // human readable region (state/country/basin/etc.)
  "CountryCode": "",             // ISO 2 country code when known, else empty
  "StartDate": "",               // date string dd-mm-yyyy
  "EndDate": "",                 // date string dd-mm-yyyy
  "Sources": []                  // array of scrape-friendly data host URLs (USGS/EPA/NOAA/NASA/USDA/ESA/Google/IBM Earth or similar)
}
Rules:
- Return ONLY that JSON object, nothing else.
- Every entry must be unique and plausible; vary DataEntity, time range, location, and source to avoid duplicates.
- Time ranges must be realistic for the dataset (e.g., satellite eras, survey years).
- Sources must be direct data hosts/endpoints (FTP/HTTPS/S3/OPeNDAP), not marketing pages.
- OutputFormat must match the type of data (rasters -> geotiff/netcdf; vectors -> shapefile/geojson/geopackage; tabular -> csv/parquet).
- Prefer USGS/EPA/NOAA/NASA/USDA/ESA/Google/IBM Earth domains; include diverse agencies.
- Keep CleanedQuery empty.
"""}, {"role": "user","content": "Generate 500 UNQIUE 'NormalizedQuery' JSONs at a time"}],
      model="openai/gpt-oss-20b",
      response_format={
        "type": "json_schema",
        "json_schema": {
          "name": "normalized_queries_response",
          "strict": True,
          "schema": {
            "type": "object",
            "properties": {
              "normalizedQueries": {
                "type": "array",
                "items": {
                  "type": "object",
                  "properties": {
                    "CleanedQuery": {"type": "string"},
                    "DataEntity": {"type": "string"},
                    "OutputFormat": {"type": "string"},
                    "Location": {"type": "string"},
                    "CountryCode": {"type": "string"},
                    "StartDate": {"type": "string"},
                    "EndDate": {"type": "string"},
                    "Sources": {
                      "type": "array",
                      "items": {"type": "string"}
                    }
                  },
                  "required": [
                    "CleanedQuery",
                    "DataEntity",
                    "OutputFormat",
                    "Location",
                    "CountryCode",
                    "StartDate",
                    "EndDate",
                    "Sources"
                  ],
                  "additionalProperties": False
                }
              }
            },
            "required": ["normalizedQueries"],
            "additionalProperties": False
          }
        }
      }
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
