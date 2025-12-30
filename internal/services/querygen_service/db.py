import os
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine, async_sessionmaker
from sqlalchemy.orm import declarative_base

# Default to PostgreSQL; fallback to SQLite for local development only
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql+asyncpg://postgres:postgres@localhost:5432/geospatial_datasets")

engine = create_async_engine(DATABASE_URL, future=True, echo=False)
SessionLocal = async_sessionmaker(bind=engine, expire_on_commit=False, class_=AsyncSession)
Base = declarative_base()


async def get_db():
    async with SessionLocal() as session:
        yield session
