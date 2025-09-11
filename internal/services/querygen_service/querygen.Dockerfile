FROM python:3.11-slim AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
  build-essential gcc g++ libffi-dev libblas-dev liblapack-dev \
  && rm -rf /var/lib/apt/lists/*


WORKDIR /app
# consider referencing independent 'querygenRequirements.txt' file, this has LOT fewer dependencies
COPY ../requirements.txt . 
RUN pip wheel --no-cache-dir --wheel-dir=/wheels -r requirements.txt


# ---------- Final Stage ----------
FROM python:3.11-slim

# Runtime dependencies (no compilers here)
RUN apt-get update && apt-get install -y --no-install-recommends \
  libffi8 libblas3 liblapack3 \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# copy built wheels
COPY --from=builder /wheels /wheels
RUN pip install --no-cache /wheels/*


COPY . .

# expose fastapi port
EXPOSE 50052

CMD ["uvicorn", "main:app", "--host", "0.0.0.0"]
