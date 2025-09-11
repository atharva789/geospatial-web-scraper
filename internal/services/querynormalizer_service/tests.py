from sentenceParsing import NormalizerService

query = "find precipitation data for the state of Ohio between 2010 and 2020"
service = NormalizerService()
out = service.GetNormalizedQuery(query)
print(f"result: {out}")
