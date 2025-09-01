import nltk
from nltk.tokenize import RegexpTokenizer
import re
import searchQuery_pb2 as pb
import searchQuery_pb2_grpc as pb_grpc
#nltk.download('averaged_perceptron_tagger_eng')
#nltk.download("punkt_tab")


grammar = """
    TASK: {<VB.*><NP|PP|ADVP|NUM>+}
    DATA_ENTITY: {<JJ>*<NN>+} 
    LOCATION: {<NNP>+}
    TIME_RANGE: {<CD><IN|:|CC|,|\$|HYPH|-><CD>} 
"""
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

    # multi-token / hyphenated patterns to recognize
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
        # Try multi-token patterns first (greedy longest match)
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

        # Single-token aliases
        tok = toks[i]
        if tok in single_token_aliases:
            add(single_token_aliases[tok])

        # Common dotted/extension-style tokens if they survived as a single token
        if tok.startswith("."):
            ext = tok[1:]
            if ext in single_token_aliases:
                add(single_token_aliases[ext])

        i += 1

    return out
# ---------- END NEW ----------

splitter = RegexpTokenizer(r'[^-:\s]+')   # keep runs of non-hyphen/colon/space

# Option 2: keep words AND separators as separate tokens
keeper = RegexpTokenizer(r'\w+|[-:]')     # words OR literal - or :

def split_hyphens_and_colons(tokens):
    out = []
    for tok in tokens:
        out.extend(keeper.tokenize(tok))  # or use `splitter` if you want to drop the separators
    return out

def get_tuples_by_label(label, tree):
    tuples = []
    for chunk in tree.subtrees(lambda t: t.label() == label):
        for tup in chunk.leaves():
            tuples.append(tup)
    return tuples

def get_label_by_token(token):
    return nltk.pos_tag([token])[0]

def clean_tree(parser, tree, label):
    labeled_tuples = get_tuples_by_label(label, tree)
    cleaned_tuples = []
    for tup in labeled_tuples:
        tup_lbl = get_label_by_token(tup[0])
        lbl, tup_lbl = str(tup[1]), str(tup_lbl[1])
        if tup_lbl not in lbl and lbl not in tup_lbl:
            continue
        cleaned_tuples.append(tup)
    return cleaned_tuples

labels = ["LOCATION", "DATA_ENTITY", "TIME_RANGE"]

def clean_user_prompt(prompt):
    tokens = nltk.word_tokenize(prompt)
    #tokenize further by removing hyphens
    tokens = split_hyphens_and_colons(tokens)
    tagged = nltk.pos_tag(tokens)
    parser = nltk.RegexpParser(grammar)
    tree = parser.parse(tagged)
    output_format = get_output_format(tokens)
    lbl_to_tokens = {}
    for lbl in labels:
        raw_tuples = get_tuples_by_label(lbl, tree)
        cleaned_tuples = clean_tree(parser, tree, lbl)
        cleaned_tokens = [token for token,_ in cleaned_tuples]
        lbl_to_tokens[lbl] = cleaned_tokens
    lbl_to_tokens['OUTPUT_FORMAT'] = output_format
    return lbl_to_tokens

def construct_string_from_tokens(tokens):
    sentence = ""
    for tkn in tokens:
        sentence += tkn + " "
    return sentence

def get_parent_location(loc):
    #should return a larger area. Ex. parent of city is the state, state is the country
    return ""

def get_methods(object, spacing=20):
  methodList = []
  for method_name in dir(object):
    try:
        if callable(getattr(object, method_name)):
            methodList.append(str(method_name))
    except Exception:
        methodList.append(str(method_name))
  processFunc = (lambda s: ' '.join(s.split())) or (lambda s: s)
  for method in methodList:
    try:
        print(str(method.ljust(spacing)) + ' ' +
              processFunc(str(getattr(object, method).__doc__)[0:90]))
    except Exception:
        print(method.ljust(spacing) + ' ' + ' getattr() failed')

class QueryNormalizer(pb_grpc.NormalizerServiceServicer):
    def GetNormalizedQuery(self, request, context):
        query = request.searchQuery
        lbl_to_token = clean_user_prompt(str(query))
        # {data_entity} + {output_format} + "for" {location} (or superset of location)
        # alternative queries:
        # 1. Add/Remove time
        # 2. Look for broader region
        optimal_query = pb.QueryStructure(dataEntity=lbl_to_token["DATA_ENTITY"][0], outputFromat=lbl_to_token["OUTPUT_FORMAT"][0], location=lbl_to_token["LOCATION"][0], startDate=lbl_to_token["TIME_RANGE"][0], endDate=lbl_to_token["TIME_RANGE"][-1])
        print(f"        (Python gRPC) Optimal query: {optimal_query}")
        return pb.QueryResponse(normalizedQuery=[optimal_query])
