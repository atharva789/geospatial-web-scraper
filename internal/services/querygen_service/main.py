import os
from fastapi import Depends, FastAPI, HTTPException
from sqlalchemy import Column, Integer, String, select
from sqlalchemy.ext.asyncio import AsyncSession
from db import Base, engine, get_db
from querygen import QueryGenerator

app = FastAPI()

class GeneratedQueries(Base):
  __tablename__ = "generated_queries"
  qid = Column(Integer, primary_key=True, index=True)
  hash = Column(String, unique=True, index=True)

@app.on_event("startup")
async def startup():
  async with engine.begin() as conn:
    await conn.run_sync(Base.metadata.create_all)

@app.get("/")
async def home():
  return [{"message": "Endpoint active"}]


# Dedup-schema
# (QID) PK
# Hash Unique

@app.get("/begin")
async def generate_queries(db: AsyncSession = Depends(get_db)):
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

  return {
    "inserted": len(new_queries),
    "skipped": len(queries) - len(new_queries),
    "sample": [str(q) for q in new_queries[:3]],
  }
