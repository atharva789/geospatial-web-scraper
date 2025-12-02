# Simple image to run the PySpark-based dataset consumer
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY dataset_consumer.py .

ENV PYTHONUNBUFFERED=1

ENTRYPOINT ["python", "dataset_consumer.py"]
