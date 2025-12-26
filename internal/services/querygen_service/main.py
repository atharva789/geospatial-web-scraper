import os
import asyncio
import httpx
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from fastapi import Depends, FastAPI, HTTPException
from sqlalchemy import Column, Integer, String, select
from sqlalchemy.ext.asyncio import AsyncSession
from db import Base, engine, get_db
from querygen import QueryGenerator

app = FastAPI()
scheduler = AsyncIOScheduler()
db: AsyncSession = None

class GeneratedQueries(Base):
  __tablename__ = "generated_queries"
  qid = Column(Integer, primary_key=True, index=True)
  hash = Column(String, unique=True, index=True)

@app.on_event("startup")
async def startup():
  async with engine.begin() as conn:
    await conn.run_sync(Base.metadata.create_all)
  global db
  db = await get_db()
  
  # make function call: generate new QUERIES
  scheduler.add_job(job, 'cron', hour=12, minute=0)
  scheduler.start()

@app.get("/")
async def home():
  return [{"message": "Endpoint active"}]


@app.get("/begin")
async def generate_queries():
  queries, new_queries = await job()

  return {
    "inserted": len(new_queries),
    "skipped": len(queries) - len(new_queries),
    "sample": [str(q) for q in new_queries[:3]],
  }


async def job():
  if not os.getenv("GROQ_API_KEY"):
    raise HTTPException(status_code=500, detail="GROQ_API_KEY not configured")

  generator = QueryGenerator()
  queries = generator.generateQueries()  
  new_queries = []

  for q in queries:
    q_hash = generator.hash_query(q)
    existing = await db.execute(select(GeneratedQueries).where(GeneratedQueries.hash == q_hash))
    if existing.scalar_one_or_none():
      continue
    db.add(GeneratedQueries(hash=q_hash))
    new_queries.append(q)

  await db.commit()
  
  # start fetching new_queries: start a background/async job
  asyncio.create_task(start_crawl(new_queries))
  return queries, new_queries

async def start_crawl(new_queries):
  payload = {
    "normalizedQueries": [q.model_dump(by_alias=True) for q in new_queries],
  }
  async with httpx.AsyncClient(timeout=30) as client:
    try:
      await client.post("http://localhost:8080/crawl", json=payload)
    except httpx.HTTPError as exc:
      # For now just log; do not raise to avoid breaking the background task
      print(f"crawl trigger failed: {exc}")
  return
