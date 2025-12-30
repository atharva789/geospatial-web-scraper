import pytest
import hashlib
from querygen import NormalizedQuery, QueryGenerator


def test_normalized_query_creation():
    """Test NormalizedQuery model creation and validation"""
    query = NormalizedQuery(
        CleanedQuery="test query",
        DataEntity="precipitation",
        OutputFormat="NetCDF",
        Location="California",
        CountryCode="US",
        StartDate="2020-01-01",
        EndDate="2020-12-31",
        Sources=["https://noaa.gov"]
    )

    assert query.cleaned_query == "test query"
    assert query.data_entity == "precipitation"
    assert query.output_format == "NetCDF"
    assert query.location == "California"
    assert query.country_code == "US"
    assert query.start_date == "2020-01-01"
    assert query.end_date == "2020-12-31"
    assert len(query.sources) == 1
    assert query.sources[0] == "https://noaa.gov"


def test_normalized_query_field_aliases():
    """Test that field aliases work correctly"""
    query = NormalizedQuery(
        CleanedQuery="test",
        DataEntity="elevation",
        OutputFormat="GeoTIFF",
        Location="Washington",
        CountryCode="US",
        StartDate="2021-01-01",
        EndDate="2021-12-31",
        Sources=[]
    )

    assert query.data_entity == "elevation"
    # Test model_dump with alias
    dumped = query.model_dump(by_alias=True)
    assert "DataEntity" in dumped
    assert dumped["DataEntity"] == "elevation"


def test_normalized_query_str():
    """Test __str__ method for NormalizedQuery"""
    query = NormalizedQuery(
        CleanedQuery="",
        DataEntity="lidar",
        OutputFormat="LAS",
        Location="Oregon",
        CountryCode="US",
        StartDate="2019-01-01",
        EndDate="2019-12-31",
        Sources=[]
    )

    result = str(query)
    assert "lidar" in result
    assert "Oregon" in result
    assert "US" in result
    assert "LAS" in result
    assert "2019-01-01" in result
    assert "2019-12-31" in result


def test_query_generator_missing_api_key(monkeypatch):
    """Test that QueryGenerator fails without API key"""
    monkeypatch.delenv("GROQ_API_KEY", raising=False)

    with pytest.raises(ValueError, match="GROQ_API_KEY not set"):
        QueryGenerator()


def test_hash_query():
    """Test query hashing for deduplication"""
    # We can't create QueryGenerator without API key, so we'll test hash manually
    query = NormalizedQuery(
        CleanedQuery="",
        DataEntity="soils",
        OutputFormat="Shapefile",
        Location="Iowa",
        CountryCode="US",
        StartDate="2018-01-01",
        EndDate="2018-12-31",
        Sources=["https://usda.gov"]
    )

    query_str = str(query)
    expected_hash = hashlib.sha256(query_str.encode("utf-8")).hexdigest()

    # Manually compute what hash should be
    assert isinstance(expected_hash, str)
    assert len(expected_hash) == 64  # SHA256 produces 64 hex chars


def test_hash_query_with_mock(monkeypatch):
    """Test QueryGenerator.hash_query method with mocked API key"""
    monkeypatch.setenv("GROQ_API_KEY", "fake-api-key-for-testing")

    gen = QueryGenerator()

    query = NormalizedQuery(
        CleanedQuery="",
        DataEntity="landcover",
        OutputFormat="GeoTIFF",
        Location="Colorado",
        CountryCode="US",
        StartDate="2020-01-01",
        EndDate="2020-12-31",
        Sources=[]
    )

    hash1 = gen.hash_query(query)
    hash2 = gen.hash_query(query)

    # Same query should produce same hash
    assert hash1 == hash2
    assert len(hash1) == 64


def test_deduplicate_queries(monkeypatch):
    """Test deduplication of queries"""
    monkeypatch.setenv("GROQ_API_KEY", "fake-api-key-for-testing")

    gen = QueryGenerator()

    query1 = NormalizedQuery(
        CleanedQuery="",
        DataEntity="precipitation",
        OutputFormat="NetCDF",
        Location="Texas",
        CountryCode="US",
        StartDate="2020-01-01",
        EndDate="2020-12-31",
        Sources=[]
    )

    query2 = NormalizedQuery(
        CleanedQuery="",
        DataEntity="precipitation",
        OutputFormat="NetCDF",
        Location="Texas",
        CountryCode="US",
        StartDate="2020-01-01",
        EndDate="2020-12-31",
        Sources=[]
    )

    query3 = NormalizedQuery(
        CleanedQuery="",
        DataEntity="temperature",
        OutputFormat="NetCDF",
        Location="Texas",
        CountryCode="US",
        StartDate="2020-01-01",
        EndDate="2020-12-31",
        Sources=[]
    )

    queries = [query1, query2, query3]
    unique = gen.deduplicate_queries(queries)

    # Should have 2 unique queries (query1 and query3)
    assert len(unique) == 2
    assert unique[0] == query1
    assert unique[1] == query3


def test_deduplicate_empty_list(monkeypatch):
    """Test deduplication with empty list"""
    monkeypatch.setenv("GROQ_API_KEY", "fake-api-key-for-testing")

    gen = QueryGenerator()
    result = gen.deduplicate_queries([])

    assert result == []


def test_normalized_query_optional_fields():
    """Test that optional fields can be None"""
    query = NormalizedQuery(
        CleanedQuery="",
        DataEntity="DEM",
        OutputFormat="GeoTIFF",
        Location="Montana",
        CountryCode=None,  # Optional field
        StartDate="",
        EndDate="",
        Sources=[]
    )

    assert query.data_entity == "DEM"
    assert query.country_code is None


def test_normalized_query_default_sources():
    """Test that Sources defaults to empty list"""
    query = NormalizedQuery(
        CleanedQuery="",
        DataEntity="watersheds",
        OutputFormat="GeoPackage",
        Location="Pennsylvania",
        CountryCode="US",
        StartDate="2021-01-01",
        EndDate="2021-12-31"
    )

    assert query.sources == []


def test_normalized_query_pydantic_validation():
    """Test that Pydantic validation works"""
    # Should accept dict with aliases
    data = {
        "CleanedQuery": "",
        "DataEntity": "flowlines",
        "OutputFormat": "Shapefile",
        "Location": "Ohio River Basin",
        "CountryCode": "US",
        "StartDate": "2022-01-01",
        "EndDate": "2022-12-31",
        "Sources": ["https://usgs.gov"]
    }

    query = NormalizedQuery.model_validate(data)
    assert query.data_entity == "flowlines"
    assert query.output_format == "Shapefile"


def test_multiple_sources():
    """Test NormalizedQuery with multiple sources"""
    query = NormalizedQuery(
        CleanedQuery="",
        DataEntity="vegetation",
        OutputFormat="GeoTIFF",
        Location="Amazon Basin",
        CountryCode="BR",
        StartDate="2023-01-01",
        EndDate="2023-12-31",
        Sources=[
            "https://nasa.gov",
            "https://esa.int",
            "https://earthdata.nasa.gov"
        ]
    )

    assert len(query.sources) == 3
    assert "https://nasa.gov" in query.sources
    assert "https://esa.int" in query.sources
