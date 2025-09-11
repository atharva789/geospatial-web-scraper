from kafka import KafkaConsumer

c = KafkaConsumer("dataset-events", 
  bootstrap_servers="localhost:9092", auto_offset_reset="earliest", group_id="check")

for i, m in enumerate(c):
  dataEvent = m.value.decode()
  print(f"  {dataEvent.PageContent}")