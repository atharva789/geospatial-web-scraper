from groq import Groq
import os

#checks against already present data in datalake, then sends 
def generateQueries():
  return []

while True:
  client = Groq(api_key=os.environ.get("GROQ_API_KEY"))  # Using env variable is safer

  response = client.chat.completions.create(
      messages=[{"role": "user", "content": "Explain the importance of fast language models"}],
      model="openai/gpt-oss-20b",
  )