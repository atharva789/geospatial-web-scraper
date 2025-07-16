package crawler

var GeoMIMETypes = map[string]bool{
	"application/csv":                      true,
	"application/zip":                      true,
	"application/json":                     true,
	"application/geo+json":                 true,
	"application/x-geotiff":                true,
	"application/x-shapefile":              true,
	"application/x-esri-shape":             true,
	"application/x-filegdb":                true,
	"application/x-esri-geodatabase":       true,
	"application/x-netcdf":                 true,
	"application/x-hdf":                    true,
	"application/x-hdf5":                   true,
	"application/x-hdf4":                   true,
	"application/x-grib":                   true,
	"application/grib":                     true,
	"application/x-bil":                    true,
	"application/x-bip":                    true,
	"application/x-bsq":                    true,
	"application/vnd.las":                  true,
	"application/vnd.laz":                  true,
	"application/vnd.google-earth.kml+xml": true,
	"application/vnd.google-earth.kmz":     true,
	"application/x-sqlite3":                true,
	"application/geopackage+sqlite3":       true,
	"application/vnd.ogc.wms_xml":          true,
	"application/vnd.ogc.wfs_xml":          true,
	"application/topo+json":                true,
}

var UnwantedClassOrIDSubstrings = map[string]bool{
	// Navigation, headers, menus
	"nav":        true,
	"menu":       true,
	"header":     true,
	"breadcrumb": true,
	"skip":       true,

	// Sidebars and secondary panels
	"sidebar": true,
	"aside":   true,
	"related": true,

	// Footers and banners
	"footer": true,
	"banner": true,

	// Cookie/legal/accessibility notices
	"cookie":        true,
	"consent":       true,
	"disclaimer":    true,
	"notice":        true,
	"privacy":       true,
	"alert":         true,
	"accessibility": true,

	// Social, sharing, subscribing
	"social":     true,
	"share":      true,
	"subscribe":  true,
	"newsletter": true,

	// Feedback, modals, popups
	"feedback": true,
	"modal":    true,
	"popup":    true,

	// USGS S3 directory-specific
	"search":   true,
	"contact":  true,
	"foia":     true,
	"policies": true,

	// Generic
	"identifier": true,
}

// structure:
//
//	URLs, descriptions, and embeddings are 1-1 in 3 slices:
//	new Manager session -> validate URL-description-embedding slices (ernsure they exist and are 1-1) -->
//	validation:
//			1. find last URL index
//			2. check description, embedding
//			if embedding/description exists, no URL:
//				1. Delete row
//			if URL and
//				1. no description but embedding: continue
//				2. no dembedding: add to 'embedding' queue, -> /embed -> add embedding
//

// PublicGeospatialDataSeeds maps each seed URL to its DataContext.
var PublicGeospatialDataSeeds = map[string]DataContext{
	// ------------------------------------------------------------------
	// NATIONAL DATA SOURCES (UNITED STATES)
	// ------------------------------------------------------------------

	// --- Newly added, highly structured endpoints ---
	"https://prd-tnm.s3.amazonaws.com/index.html?prefix=StagedProducts/": {
		Description: "USGS The National Map – browsable S3 bucket for staged elevation, hydrography, imagery, and land-cover files.",
	},
	"https://rockyweb.usgs.gov/vdelivery/Datasets/Staged/Elevation/": {
		Description: "USGS Elevation delivery directory (1 m & 10 m DEM GeoTIFFs).",
	},
	"https://www.mrlc.gov/viewer/": {
		Description: "MRLC NLCD tile viewer & bulk download tool.",
	},
	"https://coast.noaa.gov/htdata/": {
		Description: "NOAA Digital Coast ‘htdata’ directory with LiDAR, shoreline, and coastal imagery datasets.",
	},
	"https://www.ncei.noaa.gov/data/": {
		Description: "NOAA NCEI bulk data access root for climate, ocean, and geophysical archives.",
	},
	"https://nassgeodata.gmu.edu/CropScape/": {
		Description: "USDA CropScape interface for Cropland Data Layer raster downloads.",
	},
	"https://www.nrcs.usda.gov/resources/data-and-reports/ssurgo": {
		Description: "NRCS SSURGO detailed soil survey database downloads.",
	},
	"https://www2.census.gov/geo/tiger/": {
		Description: "Census TIGER/Line FTP site for boundaries, roads, and address ranges (shapefiles).",
	},
	"https://portal.opentopography.org/datasets": {
		Description: "OpenTopography catalogue of LiDAR point clouds and derived DEMs.",
	},

	// --- Original national seeds ---
	"https://catalog.data.gov/dataset/?metadata_type=geospatial": {
		Description: "Data.gov geospatial catalog – U.S. government open datasets.",
	},
	"https://www.usgs.gov/core-science-systems/ngp/tnm-services": {
		Description: "USGS ‘The National Map’ services and downloads.",
	},
	"https://earthexplorer.usgs.gov/": {
		Description: "USGS EarthExplorer – satellite, aerial, cartographic data (login for large downloads).",
	},
	"https://lpdaac.usgs.gov/": {
		Description: "NASA LP DAAC – land-processes satellite products.",
	},
	"https://www.ncei.noaa.gov/products": {
		Description: "NOAA NCEI product landing page (climate, ocean, environmental).",
	},
	"https://nowcoast.noaa.gov/": {
		Description: "NOAA nowCOAST real-time coastal observations & forecasts (web services).",
	},
	"https://www.weather.gov/gis/": {
		Description: "National Weather Service GIS shapefiles & web services.",
	},
	"https://geodata.epa.gov/": {
		Description: "EPA GeoData portal – environmental layers.",
	},
	"https://datagateway.nrcs.usda.gov/": {
		Description: "USDA Geospatial Data Gateway – soils, agriculture, conservation.",
	},
	"https://www.fs.usda.gov/geodata/": {
		Description: "U.S. Forest Service Geodata Clearinghouse.",
	},
	"https://www.mrlc.gov/": {
		Description: "MRLC – NLCD U.S. land-cover products.",
	},
	"https://www.fws.gov/program/national-wetlands-inventory/data": {
		Description: "U.S. Fish & Wildlife Service – National Wetlands Inventory.",
	},
	"https://www.census.gov/geographies/mapping-files.html": {
		Description: "Census mapping files page – TIGER/Line, cartographic boundaries.",
	},
	"https://www.bts.gov/mapping": {
		Description: "Bureau of Transportation Statistics open mapping data.",
	},
	"https://nationalmap.gov/elevation.html": {
		Description: "USGS National Elevation Dataset info & links.",
	},
	"https://hydro.nationalmap.gov/arcgis/rest/services/nhd/MapServer": {
		Description: "USGS National Hydrography Dataset (ArcGIS REST).",
	},
	"https://catalog.data.gov/dataset?q=lidar": {
		Description: "Data.gov LiDAR search results.",
	},
	"https://coast.noaa.gov/dataviewer/#/lidar/search/": {
		Description: "NOAA Digital Coast interactive LiDAR search & download.",
	},
	"https://www.usgs.gov/programs/earthquake-hazards/data": {
		Description: "USGS Earthquake Hazards Program datasets.",
	},
	"https://geonarrative.usgs.gov/datagateway/data/shapefiles/": {
		Description: "USGS historical topo quadrangles & other shapefiles.",
	},
	"https://www.dot.gov/data": {
		Description: "U.S. Department of Transportation open-data portal.",
	},

	// ------------------------------------------------------------------
	// GLOBAL DATA SOURCES
	// ------------------------------------------------------------------
	"https://www.openstreetmap.org/data": {
		Description: "OpenStreetMap planet dumps and extracts.",
	},
	"https://download.geofabrik.de/": {
		Description: "Geofabrik regional OSM PBF/shape extracts.",
	},
	"https://www.naturalearthdata.com/downloads/": {
		Description: "Natural Earth vector & raster small-scale datasets.",
	},
	"https://scihub.copernicus.eu/": {
		Description: "Copernicus Open Access Hub – Sentinel imagery (registration).",
	},
	"https://www.gebco.net/data_and_products/gridded_bathymetry_data/": {
		Description: "GEBCO global ocean bathymetry grids.",
	},
	"http://www.diva-gis.org/gdata": {
		Description: "DIVA-GIS free GIS country layers (boundaries, roads, rivers, elevation).",
	},
	"https://sedac.ciesin.columbia.edu/data/sets/browse": {
		Description: "NASA SEDAC socioeconomic & environmental raster/vector data.",
	},
	"https://search.earthdata.nasa.gov/": {
		Description: "NASA Earthdata Search – multi-mission satellite catalog (login for download).",
	},
	"https://lta.cr.usgs.gov/GLCC": {
		Description: "USGS Global Land Cover Characterization (GLCC).",
	},
	"https://www.pancgaea.org/": {
		Description: "PANGAEA repository for earth & environmental science datasets.",
	},
	"https://earthengine.google.com/datasets/": {
		Description: "Google Earth Engine public data catalog.",
	},
	"https://registry.opendata.aws/": {
		Description: "AWS Open Data Registry – S3-hosted geospatial datasets.",
	},
	"https://open-data.europa.eu/en/data?q=geospatial": {
		Description: "EU Open Data Portal – search geospatial datasets.",
	},
	"https://www.un-spider.org/links/data-providers": {
		Description: "UN-SPIDER list of satellite and disaster-response data providers.",
	},
	"https://gpm.nasa.gov/data/directory": {
		Description: "NASA GPM precipitation products.",
	},
	"https://firms.modaps.eosdis.nasa.gov/": {
		Description: "NASA FIRMS – active fire / hotspot data.",
	},
	"https://ghrc.nsstc.nasa.gov/home/data": {
		Description: "NASA GHRC – precipitation & severe-weather satellite data.",
	},
	"https://nsidc.org/data": {
		Description: "NSIDC cryosphere data (snow, ice, glaciers).",
	},
	"https://www.unep.org/explore-topics/environmental-data-and-assessment": {
		Description: "UN Environment Programme open environmental datasets.",
	},
	"https://data.mendeley.com/datasets/tag/geospatial": {
		Description: "Mendeley Data repository – datasets tagged ‘geospatial’.",
	},
	"https://ourworldindata.org/grapher/data-downloads": {
		Description: "Our World in Data bulk CSV/ZIP downloads (many with geospatial attributes).",
	},
	"https://www.worldpop.org/geodata": {
		Description: "WorldPop high-resolution gridded population layers.",
	},
	"https://www.ncei.noaa.gov/maps/historical_weather/": {
		Description: "NOAA NCEI historical weather map images (GeoTIFF/PNG).",
	},

	// ------------------------------------------------------------------
	// THEMATIC / SPECIALTY DATA SOURCES
	// ------------------------------------------------------------------
	"https://hydrosheds.org/products": {
		Description: "HydroSHEDS global hydrological basins & river networks.",
	},
	"https://worldclim.org/data/index.html": {
		Description: "WorldClim high-resolution climate normals & scenarios.",
	},
	"https://www.isric.org/explore/soilgrids": {
		Description: "ISRIC SoilGrids – global gridded soil properties.",
	},
	"https://www.onegeology.org/data.html": {
		Description: "OneGeology worldwide geologic map services & downloads.",
	},
	"https://www.obis.org/": {
		Description: "OBIS – global marine species occurrence records.",
	},
	"https://www.movebank.org/cms/movebank-main": {
		Description: "Movebank animal movement (GPS tracking) data.",
	},
	"https://www.gbif.org/data": {
		Description: "GBIF – global biodiversity occurrence datasets.",
	},
	"https://gadm.org/data.html": {
		Description: "GADM detailed global administrative boundaries.",
	},
	"https://data.humdata.org/": {
		Description: "Humanitarian Data Exchange (HDX) – crisis & development datasets.",
	},
	"https://geoportal.kogis.or.kr/eng/index.do": {
		Description: "Korea NSDI GeoPortal – national spatial data (example international).",
	},
	"https://www.jpl.nasa.gov/earth/earth-science-data/": {
		Description: "NASA JPL Earth-science specialty datasets.",
	},
	"https://data.openstreetmap.la/": {
		Description: "OpenStreetMap Latin America extracts.",
	},
	"https://land.copernicus.eu/global/products/gdd": {
		Description: "Copernicus Global Land Service datasets.",
	},
	"https://climatedata.wri.org/": {
		Description: "World Resources Institute climate indicators.",
	},
	"https://datacatalog.worldbank.org/search/type/dataset?sort_by=field_geo_coverage&sort_order=ASC": {
		Description: "World Bank Data Catalog – geospatial filter.",
	},
	"https://www.fao.org/geospatial/resources/data/en/": {
		Description: "FAO GeoNetwork global agriculture & land-use layers.",
	},
	"https://maps.ngdc.noaa.gov/viewers/bathymetry/": {
		Description: "NOAA NGDC global bathymetry viewer & downloads.",
	},
	"https://marinecadastre.gov/data/": {
		Description: "U.S. Marine Cadastre – ocean planning GIS layers.",
	},
	"https://www.nrcan.gc.ca/maps-tools-publications/geoscientific-data/17799": {
		Description: "Natural Resources Canada national geoscience datasets.",
	},
	"https://www.statistikportal.de/de/daten/geodaten": {
		Description: "Destatis (Germany) geospatial statistics downloads.",
	},
	"https://www.ign.es/web/ign/portal/ide-ign": {
		Description: "Spanish National Geographic Institute open data.",
	},
	"https://www.data.gouv.fr/fr/datasets/?q=geospatial": {
		Description: "French government open-data portal – geospatial search.",
	},
	"https://www.data.wa.gov/data-categories/geospatial": {
		Description: "Washington State open geospatial data portal.",
	},
	"https://opendata.arcgis.com/": {
		Description: "ArcGIS Hub global open-data endpoint (search thousands of orgs).",
	},
	"https://www.geopunt.be/en/data/open-data": {
		Description: "Flanders (Belgium) GeoPunt open geospatial data.",
	},
	"https://geodata.lib.berkeley.edu/": {
		Description: "UC Berkeley Library geospatial data repository.",
	},
	"https://www.caris.com/data/": {
		Description: "CARIS sample hydrographic datasets.",
	},
	"https://digitalglobe.com/open-data/": {
		Description: "Maxar DigitalGlobe Open Data (disaster imagery).",
	},
	"https://www.planet.com/open-data/": {
		Description: "Planet Labs Open Data Program (disaster/event imagery).",
	},
	"https://www.esa.int/ESA_Multimedia/Images": {
		Description: "ESA Earth Online imagery & data links.",
	},
	"https://www.copernicus.eu/en/access-data/copernicus-data-access": {
		Description: "Copernicus consolidated data-access page.",
	},
	"https://www.ncei.noaa.gov/metadata/geoportal/rest/rpc/search/": {
		Description: "NOAA NCEI GeoPortal REST search (machine-friendly).",
	},
	"https://earthquake.usgs.gov/earthquakes/feed/v1.0/geojson.php": {
		Description: "USGS real-time earthquake GeoJSON feed.",
	},
	"https://www.un-sp.org/gis-open-data": {
		Description: "United Nations spatial open-data hub.",
	},
	"https://ghgdata.epa.gov/ghgp/main.do": {
		Description: "EPA Greenhouse Gas Reporting downloadable datasets.",
	},
	"https://www.data.gouv.fr/fr/datasets/r/d060c5a1-77e4-4d80-bc4f-4d43615b67d5": {
		Description: "Direct French urban-area shapefile download example.",
	},
	"https://www.ngdc.noaa.gov/thredds/catalog.html": {
		Description: "NOAA NCEI THREDDS scientific data server.",
	},
	"https://registry.opendata.aws/tag/geospatial/": {
		Description: "AWS Open Data sets tagged ‘geospatial’.",
	},
	"https://s3-us-west-2.amazonaws.com/elevation-tiles-prod/tiles/9/152/207.terrain.mapbox": {
		Description: "Sample Mapbox Terrain-RGB tile (pattern for all DEM tiles).",
	},
	"https://assets.publishing.service.gov.uk/government/uploads/system/uploads/attachment_data/file/762512/Local_Authority_Districts__December_2018__Boundaries_UK_BFC.zip": {
		Description: "UK Local Authority boundary shapefile (direct ZIP).",
	},
	"https://geofabric.s3.amazonaws.com/updates/2024-07-07/europe/germany-latest.osm.pbf": {
		Description: "Geofabrik daily OSM PBF extract (Germany) – direct S3 link.",
	},
	"https://opendata.arcgis.com/datasets/counties.zip": {
		Description: "ArcGIS Hub example direct ZIP – U.S. counties shapefile.",
	},
	"https://www.hydro.washington.edu/data/grdc/hydro_data/GRDC_Monthly_Summary_Files.zip": {
		Description: "GRDC monthly river-discharge summaries (ZIP).",
	},
	"https://www.epa.gov/sites/default/files/2015-12/documents/us_epa_facilities_20151218.zip": {
		Description: "EPA facilities geodata (direct ZIP download).",
	},
	"https://data.europa.eu/euodp/data/dataset/eu-nuts-regions-2021": {
		Description: "EU NUTS administrative regions shapefile & GeoPackage.",
	},
	"https://www.istat.it/it/archivio/104317": {
		Description: "ISTAT (Italy) official boundary datasets.",
	},
	"https://geodata.statoil.com/": {
		Description: "Equinor (Statoil) energy-sector open geodata.",
	},
	"https://data.linz.govt.nz/": {
		Description: "Land Information New Zealand national datasets.",
	},
	"https://www.bgs.ac.uk/data-and-resources/": {
		Description: "British Geological Survey downloadable data & maps.",
	},
	"https://data.nasa.gov/": {
		Description: "NASA open-data portal with geospatial datasets.",
	},
	"https://www.eumetsat.int/data": {
		Description: "EUMETSAT meteorological satellite data centre.",
	},
	"https://www.dwd.de/DE/leistungen/opendata/opendata_node.html": {
		Description: "German Weather Service (DWD) open data catalogue.",
	},
	"https://www.jrc.ec.europa.eu/en/data": {
		Description: "European Commission Joint Research Centre datasets.",
	},
	"https://www.eea.europa.eu/data-and-maps": {
		Description: "European Environment Agency data & maps portal.",
	},
	"https://data.un.org/": {
		Description: "United Nations open statistics portal (some spatial).",
	},
	"https://www.istat.it/it/archivio/267860": {
		Description: "ISTAT (Italy) socio-economic geodata.",
	},
	"https://www.data.go.jp/data/jp_go_opendata_catalogue_dataset/?q=geospatial": {
		Description: "Japan Government open-data (search ‘geospatial’).",
	},
	"https://geodata.bund.de/web/guest/start": {
		Description: "Germany’s National Spatial Data Infrastructure (GDI-DE) portal.",
	},
	"https://www.esri.com/en-us/arcgis/products/data/open-data": {
		Description: "Esri curated list of open data portals worldwide.",
	},
	"https://data.cityofnewyork.us/browse?q=geospatial&sortBy=relevance": {
		Description: "NYC Open Data – geospatial filter.",
	},
	"https://www.chgis.org/data/geodatabase/": {
		Description: "China Historical GIS geodatabases.",
	},
	"https://www.cegis.dk/geodata-downloads/": {
		Description: "Danish Centre for Environment & Geoscience geodata.",
	},
	"https://geoportal.cuzk.cz/geoportal/eng/default.aspx": {
		Description: "Czech national mapping authority GeoPortal.",
	},
}
