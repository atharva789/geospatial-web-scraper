import json
import os
import uuid
from datetime import datetime
import time

import boto3
import requests
from pyspark.sql import SparkSession, functions as F, types as T
import psycopg2
from psycopg2.extras import execute_values

# Kafka / streaming config
KAFKA_BOOTSTRAP = os.getenv("KAFKA_BOOTSTRAP_SERVERS", "kafka:9092")
KAFKA_TOPIC = os.getenv("KAFKA_TOPIC", "dataset-event")

# LLM config
LLM_API_URL = os.getenv("LLM_API_URL", "")
LLM_API_KEY = os.getenv("LLM_API_KEY", "")
LLM_MODEL = os.getenv("LLM_MODEL", "llama-3.3-70b-versatile")

# S3 config
S3_BUCKET = os.getenv("S3_BUCKET", "")
S3_PREFIX = os.getenv("S3_PREFIX", "llm-classifications/")
S3_ENDPOINT_URL = os.getenv("S3_ENDPOINT_URL")  # optional for custom/minio

# Vector DB (pgvector on AWS RDS/Postgres)
VECTOR_HOST = os.getenv("VECTORDB_HOST", "")
VECTOR_PORT = int(os.getenv("VECTORDB_PORT", "5432"))
VECTOR_DB = os.getenv("VECTORDB_NAME", "")
VECTOR_USER = os.getenv("VECTORDB_USER", "")
VECTOR_PASSWORD = os.getenv("VECTORDB_PASSWORD", "")
VECTOR_TABLE = os.getenv("VECTORDB_TABLE", "dataset_vectors")
VECTOR_DIM = int(os.getenv("VECTOR_DIM", "1536"))
BEDROCK_EMBED_MODEL = os.getenv("BEDROCK_EMBED_MODEL", "amazon.titan-embed-text-v1")
AWS_REGION = os.getenv("AWS_REGION", "us-east-1")


def _build_spark():
    return (
        SparkSession.builder.appName("dataset-event-consumer")
        .getOrCreate()
    )


def _post_with_retries(url, retries=3, backoff=2, **kwargs):
    attempt = 0
    last_exc = None
    while attempt < retries:
        attempt += 1
        try:
            resp = requests.post(url, **kwargs)
            resp.raise_for_status()
            return resp
        except Exception as exc:
            last_exc = exc
            if attempt >= retries:
                break
            time.sleep(backoff * attempt)
    raise last_exc


def _llm_batch_classify(items):
    """
    Send a batch of dataset descriptions to the LLM.
    Expected input: items = [{"id": str, "url": str, "meta": str, "sources": [...]}, ...]
    Returns list of {"id": ..., "url": ..., "label": str, "rationale": str}
    """
    if not LLM_API_URL or not LLM_API_KEY:
        # Stub response when not configured
        return [
            {
                "id": i["id"],
                "url": i["url"],
                "label": "unknown",
                "rationale": "LLM_API_URL/LLM_API_KEY not configured; skipped classification",
            }
            for i in items
        ]

    system_prompt = (
        "You are a geospatial data expert. Given a dataset link and page content, "
        "identify with high specificity what the dataset contains (variables, spatial/temporal scope, format, source agency). "
        "Return short bullet text, not prose paragraphs."
    )
    user_payload_lines = []
    for i in items:
        user_payload_lines.append(
            f"ID: {i['id']}\nURL: {i['url']}\nMetadata URLs: {', '.join(i.get('meta_urls', []))}\nPageContent:\n{i['meta']}\n---"
        )
    user_content = "\n\n".join(user_payload_lines)

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {LLM_API_KEY}",
    }
    body = {
        "model": LLM_MODEL,
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_content},
        ],
        "response_format": {"type": "json_object"},
    }

    try:
        resp = _post_with_retries(LLM_API_URL, headers=headers, json=body, timeout=30)
        data = resp.json()
        # Try to unwrap JSON content; be lenient about shapes
        raw_content = None
        if isinstance(data, dict):
            if "choices" in data:
                raw_content = data["choices"][0]["message"]["content"]
            elif "content" in data:
                raw_content = data["content"]
        if isinstance(raw_content, str):
            parsed = json.loads(raw_content)
        else:
            parsed = data
        # Expected parsed shape: {"results":[{"id":..,"label":..,"rationale":..},...]}
        results = []
        if isinstance(parsed, dict) and "results" in parsed:
            results = parsed["results"]
        elif isinstance(parsed, list):
            results = parsed
        else:
            results = []
        normalized = []
        for r in results:
            normalized.append(
                {
                    "id": r.get("id"),
                    "url": r.get("url"),
                    "label": r.get("label", ""),
                    "rationale": r.get("rationale", ""),
                }
            )
        if normalized:
            return normalized
    except Exception as exc:  # broad so streaming keeps going
        print(f"LLM call failed, falling back to stub: {exc}")

    return [
        {
            "id": i["id"],
            "url": i["url"],
            "label": "llm_error",
            "rationale": "LLM call failed",
        }
        for i in items
    ]


def _write_results_to_s3(results):
    if not S3_BUCKET:
        print("S3_BUCKET not set; skipping upload")
        return

    s3 = boto3.client(
        "s3",
        endpoint_url=S3_ENDPOINT_URL,
        aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID"),
        aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY"),
        aws_session_token=os.getenv("AWS_SESSION_TOKEN"),
        region_name=os.getenv("AWS_REGION", "us-east-1"),
    )
    key = f"{S3_PREFIX.rstrip('/')}/{datetime.utcnow().strftime('%Y%m%dT%H%M%S')}-{uuid.uuid4().hex}.json"
    body = json.dumps({"results": results}, indent=2)
    # light retry for transient network/S3 blips
    attempts = 0
    while attempts < 3:
        attempts += 1
        try:
            s3.put_object(Bucket=S3_BUCKET, Key=key, Body=body.encode("utf-8"))
            print(f"Wrote classifications to s3://{S3_BUCKET}/{key}")
            return
        except Exception as exc:
            if attempts >= 3:
                print(f"S3 put_object failed after retries: {exc}")
            else:
                time.sleep(2 * attempts)
    print(f"Wrote classifications to s3://{S3_BUCKET}/{key}")


def _vector_db_enabled():
    return all([VECTOR_HOST, VECTOR_DB, VECTOR_USER, VECTOR_PASSWORD])


def _init_vector_db(conn):
    with conn.cursor() as cur:
        cur.execute("CREATE EXTENSION IF NOT EXISTS vector;")
        cur.execute(
            f"""
            CREATE TABLE IF NOT EXISTS {VECTOR_TABLE} (
                id BIGSERIAL PRIMARY KEY,
                dataset_url TEXT,
                label TEXT,
                rationale TEXT,
                embedding VECTOR({VECTOR_DIM}),
                created_at TIMESTAMPTZ DEFAULT now()
            );
            """
        )
    conn.commit()


def _bedrock_embed(text: str):
    if not BEDROCK_EMBED_MODEL:
        return None
    try:
        client = boto3.client("bedrock-runtime", region_name=AWS_REGION)
        body = json.dumps({"inputText": text})
        resp = client.invoke_model(modelId=BEDROCK_EMBED_MODEL, body=body)
        payload = json.loads(resp["body"].read())
        embedding = payload.get("embedding") or payload.get("vector") or payload.get("embedding_vector")
        if embedding:
            return embedding
    except Exception as exc:
        print(f"bedrock embedding failed: {exc}")
    return None


def _stub_embed(text: str):
    # Fallback deterministic hash-based embedding to keep pipeline running without Bedrock
    import hashlib
    h = hashlib.sha256(text.encode("utf-8")).digest()
    repeat = (VECTOR_DIM + len(h) - 1) // len(h)
    raw = (h * repeat)[:VECTOR_DIM]
    return [b / 255.0 for b in raw]


def _generate_embedding(text: str):
    emb = _bedrock_embed(text)
    if emb:
        return emb
    return _stub_embed(text)


def _insert_vectors(classifications):
    if not _vector_db_enabled():
        print("Vector DB env not set; skipping vector insert")
        return

    conn = psycopg2.connect(
        host=VECTOR_HOST,
        port=VECTOR_PORT,
        dbname=VECTOR_DB,
        user=VECTOR_USER,
        password=VECTOR_PASSWORD,
        connect_timeout=5,
    )
    try:
        _init_vector_db(conn)
        rows = []
        for c in classifications:
            desc = f"{c.get('label','')}\n{c.get('rationale','')}"
            embedding = _generate_embedding(desc)
            rows.append(
                (
                    c.get("url", ""),
                    c.get("label", ""),
                    c.get("rationale", ""),
                    embedding,
                )
            )
        if rows:
            with conn.cursor() as cur:
                execute_values(
                    cur,
                    f"INSERT INTO {VECTOR_TABLE} (dataset_url, label, rationale, embedding) VALUES %s",
                    rows,
                    template="(%s, %s, %s, %s)",
                )
            conn.commit()
            print(f"Inserted {len(rows)} vectors into {VECTOR_TABLE}")
    finally:
        conn.close()


def _process_batch(batch_df, batch_id):
    rows = [row.asDict(recursive=True) for row in batch_df.collect()]
    if not rows:
        return

    items = []
    for r in rows:
        items.append(
            {
                "id": r.get("DatasetURL") or r.get("id") or str(uuid.uuid4()),
                "url": r.get("DatasetURL") or "",
                "meta_urls": r.get("MetaURL") or [],
                "meta": r.get("PageContent") or "",
            }
        )

    classifications = _llm_batch_classify(items)
    _write_results_to_s3(classifications)
    _insert_vectors(classifications)


def main():
    spark = _build_spark()

    schema = T.StructType(
        [
            T.StructField("PartitionID", T.StringType()),
            T.StructField("DatasetURL", T.StringType()),
            T.StructField("MetaURL", T.ArrayType(T.StringType())),
            T.StructField("PageContent", T.StringType()),
        ]
    )

    raw = (
        spark.readStream.format("kafka")
        .option("kafka.bootstrap.servers", KAFKA_BOOTSTRAP)
        .option("subscribe", KAFKA_TOPIC)
        .option("startingOffsets", "earliest")
        .load()
    )

    parsed = (
        raw.select(F.from_json(F.col("value").cast("string"), schema).alias("data"))
        .select("data.*")
    )

    (
        parsed.writeStream.foreachBatch(_process_batch)
        .option("checkpointLocation", os.getenv("CHECKPOINT_DIR", "/tmp/dataset-events-checkpoint"))
        .start()
        .awaitTermination()
    )


if __name__ == "__main__":
    main()
