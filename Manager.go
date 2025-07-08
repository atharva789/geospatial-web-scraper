package concurrent_scraper

// PublicGeospatialDataSeeds provides a slice of URLs for publicly accessible geospatial data.
// PublicGeospatialDataSeeds provides a slice of URLs for publicly accessible geospatial data.
var PublicGeospatialDataSeedsMap = map[string]string{
	// ------------------------------------------------------------------
	// NATIONAL DATA SOURCES (UNITED STATES)
	// ------------------------------------------------------------------

	// --- Newly added, highly structured endpoints ---
	"https://prd-tnm.s3.amazonaws.com/index.html?prefix=StagedProducts/": "USGS The National Map – browsable S3 bucket for staged elevation, hydrography, imagery, and land-cover files.",
	"https://rockyweb.usgs.gov/vdelivery/Datasets/Staged/Elevation/":     "USGS Elevation delivery directory (1 m & 10 m DEM GeoTIFFs).",
	"https://www.mrlc.gov/viewer/":                                       "MRLC NLCD tile viewer & bulk download tool.",
	"https://coast.noaa.gov/htdata/":                                     "NOAA Digital Coast ‘htdata’ directory with LiDAR, shoreline, and coastal imagery datasets.",
	"https://www.ncei.noaa.gov/data/":                                    "NOAA NCEI bulk data access root for climate, ocean, and geophysical archives.",
	"https://nassgeodata.gmu.edu/CropScape/":                             "USDA CropScape interface for Cropland Data Layer raster downloads.",
	"https://www.nrcs.usda.gov/resources/data-and-reports/ssurgo":        "NRCS SSURGO detailed soil survey database downloads.",
	"https://www2.census.gov/geo/tiger/":                                 "Census TIGER/Line FTP site for boundaries, roads, and address ranges (shapefiles).",
	"https://portal.opentopography.org/datasets":                         "OpenTopography catalogue of LiDAR point clouds and derived DEMs.",

	// --- Original national seeds ---
	"https://catalog.data.gov/dataset/?metadata_type=geospatial":       "Data.gov geospatial catalog – U.S. government open datasets.",
	"https://www.usgs.gov/core-science-systems/ngp/tnm-services":       "USGS ‘The National Map’ services and downloads.",
	"https://earthexplorer.usgs.gov/":                                  "USGS EarthExplorer – satellite, aerial, cartographic data (login for large downloads).",
	"https://lpdaac.usgs.gov/":                                         "NASA LP DAAC – land-processes satellite products.",
	"https://www.ncei.noaa.gov/products":                               "NOAA NCEI product landing page (climate, ocean, environmental).",
	"https://nowcoast.noaa.gov/":                                       "NOAA nowCOAST real-time coastal observations & forecasts (web services).",
	"https://www.weather.gov/gis/":                                     "National Weather Service GIS shapefiles & web services.",
	"https://geodata.epa.gov/":                                         "EPA GeoData portal – environmental layers.",
	"https://datagateway.nrcs.usda.gov/":                               "USDA Geospatial Data Gateway – soils, agriculture, conservation.",
	"https://www.fs.usda.gov/geodata/":                                 "U.S. Forest Service Geodata Clearinghouse.",
	"https://www.mrlc.gov/":                                            "MRLC – NLCD U.S. land-cover products.",
	"https://www.fws.gov/program/national-wetlands-inventory/data":     "U.S. Fish & Wildlife Service – National Wetlands Inventory.",
	"https://www.census.gov/geographies/mapping-files.html":            "Census mapping files page – TIGER/Line, cartographic boundaries.",
	"https://www.bts.gov/mapping":                                      "Bureau of Transportation Statistics open mapping data.",
	"https://nationalmap.gov/elevation.html":                           "USGS National Elevation Dataset info & links.",
	"https://hydro.nationalmap.gov/arcgis/rest/services/nhd/MapServer": "USGS National Hydrography Dataset (ArcGIS REST).",
	"https://catalog.data.gov/dataset?q=lidar":                         "Data.gov LiDAR search results.",
	"https://coast.noaa.gov/dataviewer/#/lidar/search/":                "NOAA Digital Coast interactive LiDAR search & download.",
	"https://www.usgs.gov/programs/earthquake-hazards/data":            "USGS Earthquake Hazards Program datasets.",
	"https://geonarrative.usgs.gov/datagateway/data/shapefiles/":       "USGS historical topo quadrangles & other shapefiles.",
	"https://www.dot.gov/data":                                         "U.S. Department of Transportation open-data portal.",

	// ------------------------------------------------------------------
	// GLOBAL DATA SOURCES
	// ------------------------------------------------------------------
	"https://www.openstreetmap.org/data":                                    "OpenStreetMap planet dumps and extracts.",
	"https://download.geofabrik.de/":                                        "Geofabrik regional OSM PBF/shape extracts.",
	"https://www.naturalearthdata.com/downloads/":                           "Natural Earth vector & raster small-scale datasets.",
	"https://scihub.copernicus.eu/":                                         "Copernicus Open Access Hub – Sentinel imagery (registration).",
	"https://www.gebco.net/data_and_products/gridded_bathymetry_data/":      "GEBCO global ocean bathymetry grids.",
	"http://www.diva-gis.org/gdata":                                         "DIVA-GIS free GIS country layers (boundaries, roads, rivers, elevation).",
	"https://sedac.ciesin.columbia.edu/data/sets/browse":                    "NASA SEDAC socioeconomic & environmental raster/vector data.",
	"https://search.earthdata.nasa.gov/":                                    "NASA Earthdata Search – multi-mission satellite catalog (login for download).",
	"https://lta.cr.usgs.gov/GLCC":                                          "USGS Global Land Cover Characterization (GLCC).",
	"https://www.pancgaea.org/":                                             "PANGAEA repository for earth & environmental science datasets.",
	"https://earthengine.google.com/datasets/":                              "Google Earth Engine public data catalog.",
	"https://registry.opendata.aws/":                                        "AWS Open Data Registry – S3-hosted geospatial datasets.",
	"https://open-data.europa.eu/en/data?q=geospatial":                      "EU Open Data Portal – search geospatial datasets.",
	"https://www.un-spider.org/links/data-providers":                        "UN-SPIDER list of satellite and disaster-response data providers.",
	"https://gpm.nasa.gov/data/directory":                                   "NASA GPM precipitation products.",
	"https://firms.modaps.eosdis.nasa.gov/":                                 "NASA FIRMS – active fire / hotspot data.",
	"https://ghrc.nsstc.nasa.gov/home/data":                                 "NASA GHRC – precipitation & severe-weather satellite data.",
	"https://nsidc.org/data":                                                "NSIDC cryosphere data (snow, ice, glaciers).",
	"https://www.unep.org/explore-topics/environmental-data-and-assessment": "UN Environment Programme open environmental datasets.",
	"https://data.mendeley.com/datasets/tag/geospatial":                     "Mendeley Data repository – datasets tagged ‘geospatial’.",
	"https://ourworldindata.org/grapher/data-downloads":                     "Our World in Data bulk CSV/ZIP downloads (many with geospatial attributes).",
	"https://www.worldpop.org/geodata":                                      "WorldPop high-resolution gridded population layers.",
	"https://www.ncei.noaa.gov/maps/historical_weather/":                    "NOAA NCEI historical weather map images (GeoTIFF/PNG).",

	// ------------------------------------------------------------------
	// THEMATIC / SPECIALTY DATA SOURCES
	// ------------------------------------------------------------------
	"https://hydrosheds.org/products":                    "HydroSHEDS global hydrological basins & river networks.",
	"https://worldclim.org/data/index.html":              "WorldClim high-resolution climate normals & scenarios.",
	"https://www.isric.org/explore/soilgrids":            "ISRIC SoilGrids – global gridded soil properties.",
	"https://www.onegeology.org/data.html":               "OneGeology worldwide geologic map services & downloads.",
	"https://www.obis.org/":                              "OBIS – global marine species occurrence records.",
	"https://www.movebank.org/cms/movebank-main":         "Movebank animal movement (GPS tracking) data.",
	"https://www.gbif.org/data":                          "GBIF – global biodiversity occurrence datasets.",
	"https://gadm.org/data.html":                         "GADM detailed global administrative boundaries.",
	"https://data.humdata.org/":                          "Humanitarian Data Exchange (HDX) – crisis & development datasets.",
	"https://geoportal.kogis.or.kr/eng/index.do":         "Korea NSDI GeoPortal – national spatial data (example international).",
	"https://www.jpl.nasa.gov/earth/earth-science-data/": "NASA JPL Earth-science specialty datasets.",
	"https://data.openstreetmap.la/":                     "OpenStreetMap Latin America extracts.",
	"https://land.copernicus.eu/global/products/gdd":     "Copernicus Global Land Service datasets.",
	"https://climatedata.wri.org/":                       "World Resources Institute climate indicators.",
	"https://datacatalog.worldbank.org/search/type/dataset?sort_by=field_geo_coverage&sort_order=ASC": "World Bank Data Catalog – geospatial filter.",
	"https://www.fao.org/geospatial/resources/data/en/":                                               "FAO GeoNetwork global agriculture & land-use layers.",
	"https://maps.ngdc.noaa.gov/viewers/bathymetry/":                                                  "NOAA NGDC global bathymetry viewer & downloads.",
	"https://marinecadastre.gov/data/":                                                                "U.S. Marine Cadastre – ocean planning GIS layers.",
	"https://www.nrcan.gc.ca/maps-tools-publications/geoscientific-data/17799":                        "Natural Resources Canada national geoscience datasets.",
	"https://www.statistikportal.de/de/daten/geodaten":                                                "Destatis (Germany) geospatial statistics downloads.",
	"https://www.ign.es/web/ign/portal/ide-ign":                                                       "Spanish National Geographic Institute open data.",
	"https://www.data.gouv.fr/fr/datasets/?q=geospatial":                                              "French government open-data portal – geospatial search.",
	"https://www.data.wa.gov/data-categories/geospatial":                                              "Washington State open geospatial data portal.",
	"https://opendata.arcgis.com/":                                                                    "ArcGIS Hub global open-data endpoint (search thousands of orgs).",
	"https://www.geopunt.be/en/data/open-data":                                                        "Flanders (Belgium) GeoPunt open geospatial data.",
	"https://geodata.lib.berkeley.edu/":                                                               "UC Berkeley Library geospatial data repository.",
	"https://www.caris.com/data/":                                                                     "CARIS sample hydrographic datasets.",
	"https://digitalglobe.com/open-data/":                                                             "Maxar DigitalGlobe Open Data (disaster imagery).",
	"https://www.planet.com/open-data/":                                                               "Planet Labs Open Data Program (disaster/event imagery).",
	"https://www.esa.int/ESA_Multimedia/Images":                                                       "ESA Earth Online imagery & data links.",
	"https://www.copernicus.eu/en/access-data/copernicus-data-access":                                 "Copernicus consolidated data-access page.",
	"https://www.ncei.noaa.gov/metadata/geoportal/rest/rpc/search/":                                   "NOAA NCEI GeoPortal REST search (machine-friendly).",
	"https://earthquake.usgs.gov/earthquakes/feed/v1.0/geojson.php":                                   "USGS real-time earthquake GeoJSON feed.",
	"https://www.un-sp.org/gis-open-data":                                                             "United Nations spatial open-data hub.",
	"https://ghgdata.epa.gov/ghgp/main.do":                                                            "EPA Greenhouse Gas Reporting downloadable datasets.",
	"https://www.data.gouv.fr/fr/datasets/r/d060c5a1-77e4-4d80-bc4f-4d43615b67d5":                     "Direct French urban-area shapefile download example.",
	"https://www.ngdc.noaa.gov/thredds/catalog.html":                                                  "NOAA NCEI THREDDS scientific data server.",
	"https://registry.opendata.aws/tag/geospatial/":                                                   "AWS Open Data sets tagged ‘geospatial’.",
	"https://s3-us-west-2.amazonaws.com/elevation-tiles-prod/tiles/9/152/207.terrain.mapbox":          "Sample Mapbox Terrain-RGB tile (pattern for all DEM tiles).",
	"https://assets.publishing.service.gov.uk/government/uploads/system/uploads/attachment_data/file/762512/Local_Authority_Districts__December_2018__Boundaries_UK_BFC.zip": "UK Local Authority boundary shapefile (direct ZIP).",
	"https://geofabric.s3.amazonaws.com/updates/2024-07-07/europe/germany-latest.osm.pbf":                                                                                    "Geofabrik daily OSM PBF extract (Germany) – direct S3 link.",
	"https://opendata.arcgis.com/datasets/counties.zip":                                                                                                                      "ArcGIS Hub example direct ZIP – U.S. counties shapefile.",
	"https://www.hydro.washington.edu/data/grdc/hydro_data/GRDC_Monthly_Summary_Files.zip":                                                                                   "GRDC monthly river-discharge summaries (ZIP).",
	"https://www.epa.gov/sites/default/files/2015-12/documents/us_epa_facilities_20151218.zip":                                                                               "EPA facilities geodata (direct ZIP download).",
	"https://data.europa.eu/euodp/data/dataset/eu-nuts-regions-2021":                                                                                                         "EU NUTS administrative regions shapefile & GeoPackage.",
	"https://www.istat.it/it/archivio/104317":                                                                                                                                "ISTAT (Italy) official boundary datasets.",
	"https://geodata.statoil.com/":                                               "Equinor (Statoil) energy-sector open geodata.",
	"https://data.linz.govt.nz/":                                                 "Land Information New Zealand national datasets.",
	"https://www.bgs.ac.uk/data-and-resources/":                                  "British Geological Survey downloadable data & maps.",
	"https://data.nasa.gov/":                                                     "NASA open-data portal with geospatial datasets.",
	"https://www.eumetsat.int/data":                                              "EUMETSAT meteorological satellite data centre.",
	"https://www.dwd.de/DE/leistungen/opendata/opendata_node.html":               "German Weather Service (DWD) open data catalogue.",
	"https://www.jrc.ec.europa.eu/en/data":                                       "European Commission Joint Research Centre datasets.",
	"https://www.eea.europa.eu/data-and-maps":                                    "European Environment Agency data & maps portal.",
	"https://data.un.org/":                                                       "United Nations open statistics portal (some spatial).",
	"https://www.istat.it/it/archivio/267860":                                    "ISTAT (Italy) socio-economic geodata.",
	"https://www.data.go.jp/data/jp_go_opendata_catalogue_dataset/?q=geospatial": "Japan Government open-data (search ‘geospatial’).",
	"https://geodata.bund.de/web/guest/start":                                    "Germany’s National Spatial Data Infrastructure (GDI-DE) portal.",
	"https://www.esri.com/en-us/arcgis/products/data/open-data":                  "Esri curated list of open data portals worldwide.",
	"https://data.cityofnewyork.us/browse?q=geospatial&sortBy=relevance":         "NYC Open Data – geospatial filter.",
	"https://www.chgis.org/data/geodatabase/":                                    "China Historical GIS geodatabases.",
	"https://www.cegis.dk/geodata-downloads/":                                    "Danish Centre for Environment & Geoscience geodata.",
	"https://geoportal.cuzk.cz/geoportal/eng/default.aspx":                       "Czech national mapping authority GeoPortal.",
}

func main() {
	find := "30m x 30m resolution image data about CAFOs in the US"
	searchList := []string{}
	manager := Manager{downloadPath: "", searchQuery: find, searchFrom: searchList}
	manager.findLinks()

}
