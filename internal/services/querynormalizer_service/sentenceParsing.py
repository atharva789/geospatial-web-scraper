# ---- spacy-based rewrite (no NLTK) ----
from concurrent.futures import ThreadPoolExecutor
import math
import re
import time
import spacy
from spacy.matcher import Matcher
from datetime import datetime, timedelta
# from dateutil.relativedelta import relativedelta

from geopy.geocoders import Nominatim  # optional (not used below)
import searchQuery_pb2 as pb
import searchQuery_pb2_grpc as pb_grpc

# Load spaCy (small English model is fine)
# If you only need tokenization + NER (no parser), disable parser/tagger for speed
nlp = spacy.load("en_core_web_sm")  # or: spacy.load("en_core_web_sm", disable=["parser","tagger"])

def get_output_format(tokens):
    """
    Extract common geospatial output formats from token list.
    Works with hyphen-split tokens (e.g., 'shape', '-', 'file').
    Returns a list of canonical format names in first-seen order.
    """
    toks = [t.lower() for t in tokens]

    single_token_aliases = {
        "csv": "CSV",
        "gpkg": "GeoPackage",
        "geopackage": "GeoPackage",
        "kml": "KML",
        "kmz": "KMZ",
        "geojson": "GeoJSON",
        "shp": "Shapefile",
        "shapefile": "Shapefile",
        "tif": "GeoTIFF",
        "tiff": "GeoTIFF",
        "geotiff": "GeoTIFF",
        "nc": "NetCDF",
        "netcdf": "NetCDF",
        "h5": "HDF5",
        "hdf5": "HDF5",
        "parquet": "Parquet",
        "las": "LAS",
        "laz": "LAZ",
        "sqlite": "SQLite/SpatiaLite",
        "spatialite": "SQLite/SpatiaLite",
        "gdb": "Esri FileGDB",
        "fgdb": "Esri FileGDB",
        "filegdb": "Esri FileGDB",
    }

    multi_patterns = [
        (["shape", "-", "file"], "Shapefile"),
        (["geo", "json"], "GeoJSON"),
        (["file", "gdb"], "Esri FileGDB"),
        (["comma", "-", "separated", "-", "values"], "CSV"),
        (["comma", "separated", "values"], "CSV"),
    ]

    out, seen = [], set()
    def add(name):
        if name not in seen:
            seen.add(name)
            out.append(name)

    i = 0
    n = len(toks)
    while i < n:
        matched = False
        for pattern, label in multi_patterns:
            L = len(pattern)
            if i + L <= n and toks[i:i+L] == pattern:
                add(label)
                i += L
                matched = True
                break
        if matched:
            continue

        tok = toks[i]
        if tok in single_token_aliases:
            add(single_token_aliases[tok])

        if tok.startswith("."):
            ext = tok[1:]
            if ext in single_token_aliases:
                add(single_token_aliases[ext])

        i += 1

    return out

# Keep your regex tokenization idea for hyphen/colon splitting specifically for output-format detection
KEEPER_RE = re.compile(r"\w+|[-:]")
def split_hyphens_and_colons(text: str):
    return KEEPER_RE.findall(text)


# DATA_ENTITY: <JJ>* <NN>+   (adjectives + one or more nouns/proper nouns)
data_entity_matcher = Matcher(nlp.vocab)
data_entity_matcher.add(
    "DATA_ENTITY",
    [
        [
            {"POS": "ADJ", "OP": "*"},
            {"POS": {"IN": ["NOUN", "PROPN"]}, "OP": "+"},
        ]
    ],
)

# TIME_RANGE helper: we rely on dateparser.search.search_dates on matched text,
# but we also define a light matcher to grab numeric spans around linkers.
time_linkers = {"to", "and", "-", "–", "—", "through", "thru", "till", "until"}
time_range_matcher = Matcher(nlp.vocab)
time_range_matcher.add(
    "TIME_RANGE",
    [
        # NUM (linker) NUM  e.g., 2010 - 2020
        [{"LIKE_NUM": True}, {"LOWER": {"IN": list(time_linkers)}}, {"LIKE_NUM": True}],
        # "last/next <NUM> <unit>"
        [{"LOWER": {"IN": ["last", "past", "next"]}}, {"LIKE_NUM": True}, {"POS": "NOUN"}],
        # "between <NUM> and <NUM>"
        [{"LOWER": "between"}, {"LIKE_NUM": True}, {"LOWER": {"IN": ["and", "to"]}}, {"LIKE_NUM": True}],
    ],
)

# LOCATION: prefer NER (GPE/LOC)
def extract_locations(doc):
    locs = [ent.text for ent in doc.ents if ent.label_ in ("GPE", "LOC")]
    # Deduplicate preserving order
    seen, out = set(), []
    for s in locs:
        if s not in seen:
            seen.add(s)
            out.append(s)
    return out

# DATA_ENTITY from matcher (longest-first unique)
def extract_data_entities(doc):
    matches = data_entity_matcher(doc)
    spans = [doc[start:end] for _, start, end in matches]
    # Heuristic: prefer spans that are lowercase/common nouns (vs. proper names) and keep longer ones
    spans = sorted(spans, key=lambda s: (-len(s), s.start))
    out, covered = [], set()
    for sp in spans:
        key = (sp.start, sp.end)
        if key in covered:
            continue
        out.append(sp.text)
        covered.add(key)
    # Light post-filter: drop pure stopword spans
    cleaned = []
    for t in out:
        toks = [w for w in nlp.make_doc(t)]
        if any(not w.is_stop for w in toks):
            cleaned.append(t)
    return cleaned

# TIME_RANGE using matcher + dateparser
def extract_time_range(doc):
    """
    Returns [start_date_str, end_date_str] if two ends found, else [].
    Falls back to relative parsing ('last 5 years', etc.).
    """
    text = doc.text

    # Try: dateparser search across whole text (robust)
    try:
        hits = list(dateparser.search.search_dates(text))
    except Exception:
        hits = []

    # If we have >= 2 parsed date-like things, take min/max by datetime
    if hits:
        datetimes = [dt for _, dt in hits]
        datetimes.sort()
        start, end = datetimes[0], datetimes[-1]
        return [start.strftime("%Y-%m-%d"), end.strftime("%Y-%m-%d")]

    # If nothing found, try our matcher to isolate a smaller phrase and parse again
    matches = time_range_matcher(doc)
    for _, s, e in matches:
        phrase = doc[s:e].text
        try:
            hits2 = list(dateparser.search.search_dates(phrase))
        except Exception:
            hits2 = []
        if hits2:
            datetimes = [dt for _, dt in hits2]
            datetimes.sort()
            start, end = datetimes[0], datetimes[-1]
            return [start.strftime("%Y-%m-%d"), end.strftime("%Y-%m-%d")]

    # Handle phrases like "last 5 years" explicitly if dateparser didn't resolve
    m = re.search(r"(last|past)\s+(\d+)\s+(years?|months?|weeks?|days?)", text, flags=re.I)
    if m:
        qty = int(m.group(2))
        unit = m.group(3).lower()
        now = datetime.utcnow()
        if unit.startswith("year"):
            start = now - relativedelta(years=qty)
        elif unit.startswith("month"):
            start = now - relativedelta(months=qty)
        elif unit.startswith("week"):
            start = now - relativedelta(weeks=qty)
        else:
            start = now - relativedelta(days=qty)
        return [start.strftime("%Y-%m-%d"), now.strftime("%Y-%m-%d")]

    return []

def construct_string_from_tokens(tokens):
    return " ".join(tokens)

def clean_user_prompt(prompt: str):
    # spaCy doc
    doc = nlp(prompt)

    # Tokens for output-format detector: we want words AND '-' ':' as standalone tokens
    raw_tokens = [t.text for t in doc]
    output_format = get_output_format(raw_tokens if any(x in raw_tokens for x in ["-", ":"]) else split_hyphens_and_colons(prompt))

    locations = extract_locations(doc)

    data_entities = extract_data_entities(doc)

    time_range = extract_time_range(doc)

    lbl_to_tokens = {
        "LOCATION": locations,
        "DATA_ENTITY": data_entities,
        "TIME_RANGE": time_range,  # [] or [start, end]
        "OUTPUT_FORMAT": output_format,
    }
    return lbl_to_tokens

def get_spatial_entity(addr: dict, loc: str) -> tuple[str, str, str]:
    """
    Returns (canonical_name, level, country_code).
    Level is one of: city, county, state, country.
    """
    # Order matters: most specific first
    if "city" in addr or "town" in addr or "village" in addr or "hamlet" in addr:
        return addr.get("city") or addr.get("town") or addr.get("village") or addr.get("hamlet"), "city", addr.get("country_code")
    if "county" in addr:
        return addr["county"], "county", addr.get("country_code")
    if "state" in addr:
        return addr["state"], "state", addr.get("country_code")
    if "country" in addr:
        return addr["country"], "country", addr.get("country_code")
    # Fallback: return the raw string
    return loc, "unknown", addr.get("country_code")


spatial_levels = ["city village town", "county", "state", "country"]
level_index = {lvl: i for i, lvl in enumerate(spatial_levels)}

def isDescendant(lvl_a: str, lvl_b: str) -> bool:
    """True if lvl_a is more fine-grained than lvl_b (city < county < state < country)."""
    return level_index.get(lvl_a, 99) < level_index.get(lvl_b, 99)

def isAncestor(lvl_a: str, lvl_b: str) -> bool:
    return level_index.get(lvl_a, -1) > level_index.get(lvl_b, -1)


# ASSUMPTION
# 1. service will NEVER recieve a query with 2 distinct locations 
#       Ex. "find data for Cleveland, Columbus, Parma"
# 2. queries with 2 locations are related

# RULES:
# 1. minimum to maximum resolution: city, county, state, country (country-code)
# 2. Query is for a continent or global: country-code is None, query is raw
# RETURNS: list of 3 tuples: (target-1, country-code), ..., (target3, country-code)
#  where target 1 - 3 is moving from lowest-level (city) -> lowest + 2 (country, state)
def get_parent_locations(locations: list[str]) -> list[str]: 
    seen = set() # contains (loc, level) where level is (city/county/state/country)
    # check that current loc is not a parent of previous
    geolocator = Nominatim(user_agent="geo_parser")
    
    # get name, lvl, code for each loc
    resolved = []
    for loc in locations:
        try:
            geo = geolocator.geocode(loc)
            if not geo:
                continue
            addr = geo.raw.get("address", {})
            name, lvl, code = get_spatial_entity(addr, loc)
            resolved.append((name,lvl,code))
        except Exception as e:
            print(f"geocode failed for loc: {loc}")
            continue
    
    if not resolved:
        return []
    
    # sort 'resolved' by level
    resolved.sort(key= lambda x: level_index.get(x[1], 99))
    # canon-entry is the first
    canon = resolved[0]
    canon_name, canon_lvl, canon_ccode = canon
    out = [(canon_name, canon_ccode)]
    if canon[1] == 99:
        return [(canon_name, "")]
    
    geo = geolocator.geocode(canon_name)
    addr = geo.raw.get("addr", {}) if geo else {}
    
    idx = level_index.get(canon_lvl, 0)
    for higher in spatial_levels[idx+1:]:
        val = addr.get(higher)
        if val:
            out.append((val, canon_ccode))
        if len(out) >= 3:
            out
            break
    
    return out
# return value: ("cleveland", "us"), ("Ohio", "us"), ("United States", "us")


temporal_scale = ['y', 'decade', 'quartercentinel', 'semicentinal']
temporal_lvls = {ent: i for i, ent in enumerate(temporal_scale)} # where i denotes 'level': ex. 'y' = 0, 'decade' = 1, … , 'semicentinal = 3

def get_temporal_entity(days: int) -> int:
    if days <= 365:
        return temporal_lvls['y']
    elif 365 <= days < 3650:
        return temporal_lvls['decade']
    elif 3650 <= days < 365*25:
        return temporal_lvls['quartercentinel']
    return temporal_lvls['semicentinal']


def get_wider_time_windows(start_date, end_date: str) -> list[(str,str)]:
    # subtract dates, get month, year, bicentinel threshold. If time_entity ≤ month, return year, 10 year
    start, end = datetime.strptime(start_date, "%d-%m-%y"), datetime.strptime(end_date, "%d-%m-%y")
    start_floor, end_ceil = (math.floor(start/10)*10), (math.ceil(end/10)*10)
    out = [(start, end), (start_floor, end_ceil)]
    now_year = datetime.now().year
    while now_year - end_ceil > 0:
        end_ceil += 10
        start_floor -= 10
        out.append(end_ceil, start_floor)
    out.append((start_floor, now_year))
    return out


def construct_optimal_query(data_entity, output_format, spatial_tuple, temporal_tuple):
    val = pb.QueryStructure(
            dataEntity=data_entity,
            outputFromat=output_format,
            location=spatial_tuple[0],
            country_code=spatial_tuple[1],
            startDate=temporal_tuple[0],
            endDate=temporal_tuple[-1]
        )
    return val

class QueryNormalizer(pb_grpc.NormalizerServiceServicer):
    def GetNormalizedQuery(self, request, context):
        query = request.searchQuery
        lbl_to_token = clean_user_prompt(str(query))

        # Safe getters
        def first_or_empty(k):
            v = lbl_to_token.get(k, [])
            return v[0] if v else ""

        start_date = ""
        end_date = ""
        if lbl_to_token.get("TIME_RANGE"):
            start_date, end_date = lbl_to_token["TIME_RANGE"][0], lbl_to_token["TIME_RANGE"][-1]
        
        location_tuples = get_parent_locations(first_or_empty("LOCATION"))
        time_tuples = []
        if start_date and end_date != "":
            time_tuples = get_wider_time_windows(start_date, end_date)
            
        # for simplicity and compute, let's NOT take cartesian product of time x loc (that's 9 search combos PER query)
        spatio_temporal_spans = [(location_tuples[i], time_tuples[i]) for i in range(min(len(time_tuples), len(location_tuples)))]
        data_entity, out_format = first_or_empty("DATA_ENTITY"), first_or_empty("OUTPUT_FORMAT")

        normalized_queries = [
            construct_optimal_query(data_entity, out_format, s, t)
            for s, t in spatio_temporal_spans
        ]
            
        return pb.QueryResponse(normalizedQuery=normalized_queries)

