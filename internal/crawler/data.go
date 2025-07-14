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
		description: "USGS The National Map – browsable S3 bucket for staged elevation, hydrography, imagery, and land-cover files.",
	},
	"https://rockyweb.usgs.gov/vdelivery/Datasets/Staged/Elevation/": {
		description: "USGS Elevation delivery directory (1 m & 10 m DEM GeoTIFFs).",
	},
	"https://www.mrlc.gov/viewer/": {
		description: "MRLC NLCD tile viewer & bulk download tool.",
	},
	"https://coast.noaa.gov/htdata/": {
		description: "NOAA Digital Coast ‘htdata’ directory with LiDAR, shoreline, and coastal imagery datasets.",
	},
	"https://www.ncei.noaa.gov/data/": {
		description: "NOAA NCEI bulk data access root for climate, ocean, and geophysical archives.",
	},
	"https://nassgeodata.gmu.edu/CropScape/": {
		description: "USDA CropScape interface for Cropland Data Layer raster downloads.",
	},
	"https://www.nrcs.usda.gov/resources/data-and-reports/ssurgo": {
		description: "NRCS SSURGO detailed soil survey database downloads.",
	},
	"https://www2.census.gov/geo/tiger/": {
		description: "Census TIGER/Line FTP site for boundaries, roads, and address ranges (shapefiles).",
	},
	"https://portal.opentopography.org/datasets": {
		description: "OpenTopography catalogue of LiDAR point clouds and derived DEMs.",
	},

	// --- Original national seeds ---
	"https://catalog.data.gov/dataset/?metadata_type=geospatial": {
		description: "Data.gov geospatial catalog – U.S. government open datasets.",
	},
	"https://www.usgs.gov/core-science-systems/ngp/tnm-services": {
		description: "USGS ‘The National Map’ services and downloads.",
	},
	"https://earthexplorer.usgs.gov/": {
		description: "USGS EarthExplorer – satellite, aerial, cartographic data (login for large downloads).",
	},
	"https://lpdaac.usgs.gov/": {
		description: "NASA LP DAAC – land-processes satellite products.",
	},
	"https://www.ncei.noaa.gov/products": {
		description: "NOAA NCEI product landing page (climate, ocean, environmental).",
	},
	"https://nowcoast.noaa.gov/": {
		description: "NOAA nowCOAST real-time coastal observations & forecasts (web services).",
	},
	"https://www.weather.gov/gis/": {
		description: "National Weather Service GIS shapefiles & web services.",
	},
	"https://geodata.epa.gov/": {
		description: "EPA GeoData portal – environmental layers.",
	},
	"https://datagateway.nrcs.usda.gov/": {
		description: "USDA Geospatial Data Gateway – soils, agriculture, conservation.",
	},
	"https://www.fs.usda.gov/geodata/": {
		description: "U.S. Forest Service Geodata Clearinghouse.",
	},
	"https://www.mrlc.gov/": {
		description: "MRLC – NLCD U.S. land-cover products.",
	},
	"https://www.fws.gov/program/national-wetlands-inventory/data": {
		description: "U.S. Fish & Wildlife Service – National Wetlands Inventory.",
	},
	"https://www.census.gov/geographies/mapping-files.html": {
		description: "Census mapping files page – TIGER/Line, cartographic boundaries.",
	},
	"https://www.bts.gov/mapping": {
		description: "Bureau of Transportation Statistics open mapping data.",
	},
	"https://nationalmap.gov/elevation.html": {
		description: "USGS National Elevation Dataset info & links.",
	},
	"https://hydro.nationalmap.gov/arcgis/rest/services/nhd/MapServer": {
		description: "USGS National Hydrography Dataset (ArcGIS REST).",
	},
	"https://catalog.data.gov/dataset?q=lidar": {
		description: "Data.gov LiDAR search results.",
	},
	"https://coast.noaa.gov/dataviewer/#/lidar/search/": {
		description: "NOAA Digital Coast interactive LiDAR search & download.",
	},
	"https://www.usgs.gov/programs/earthquake-hazards/data": {
		description: "USGS Earthquake Hazards Program datasets.",
	},
	"https://geonarrative.usgs.gov/datagateway/data/shapefiles/": {
		description: "USGS historical topo quadrangles & other shapefiles.",
	},
	"https://www.dot.gov/data": {
		description: "U.S. Department of Transportation open-data portal.",
	},

	// ------------------------------------------------------------------
	// GLOBAL DATA SOURCES
	// ------------------------------------------------------------------
	"https://www.openstreetmap.org/data": {
		description: "OpenStreetMap planet dumps and extracts.",
	},
	"https://download.geofabrik.de/": {
		description: "Geofabrik regional OSM PBF/shape extracts.",
	},
	"https://www.naturalearthdata.com/downloads/": {
		description: "Natural Earth vector & raster small-scale datasets.",
	},
	"https://scihub.copernicus.eu/": {
		description: "Copernicus Open Access Hub – Sentinel imagery (registration).",
	},
	"https://www.gebco.net/data_and_products/gridded_bathymetry_data/": {
		description: "GEBCO global ocean bathymetry grids.",
	},
	"http://www.diva-gis.org/gdata": {
		description: "DIVA-GIS free GIS country layers (boundaries, roads, rivers, elevation).",
	},
	"https://sedac.ciesin.columbia.edu/data/sets/browse": {
		description: "NASA SEDAC socioeconomic & environmental raster/vector data.",
	},
	"https://search.earthdata.nasa.gov/": {
		description: "NASA Earthdata Search – multi-mission satellite catalog (login for download).",
	},
	"https://lta.cr.usgs.gov/GLCC": {
		description: "USGS Global Land Cover Characterization (GLCC).",
	},
	"https://www.pancgaea.org/": {
		description: "PANGAEA repository for earth & environmental science datasets.",
	},
	"https://earthengine.google.com/datasets/": {
		description: "Google Earth Engine public data catalog.",
	},
	"https://registry.opendata.aws/": {
		description: "AWS Open Data Registry – S3-hosted geospatial datasets.",
	},
	"https://open-data.europa.eu/en/data?q=geospatial": {
		description: "EU Open Data Portal – search geospatial datasets.",
	},
	"https://www.un-spider.org/links/data-providers": {
		description: "UN-SPIDER list of satellite and disaster-response data providers.",
	},
	"https://gpm.nasa.gov/data/directory": {
		description: "NASA GPM precipitation products.",
	},
	"https://firms.modaps.eosdis.nasa.gov/": {
		description: "NASA FIRMS – active fire / hotspot data.",
	},
	"https://ghrc.nsstc.nasa.gov/home/data": {
		description: "NASA GHRC – precipitation & severe-weather satellite data.",
	},
	"https://nsidc.org/data": {
		description: "NSIDC cryosphere data (snow, ice, glaciers).",
	},
	"https://www.unep.org/explore-topics/environmental-data-and-assessment": {
		description: "UN Environment Programme open environmental datasets.",
	},
	"https://data.mendeley.com/datasets/tag/geospatial": {
		description: "Mendeley Data repository – datasets tagged ‘geospatial’.",
	},
	"https://ourworldindata.org/grapher/data-downloads": {
		description: "Our World in Data bulk CSV/ZIP downloads (many with geospatial attributes).",
	},
	"https://www.worldpop.org/geodata": {
		description: "WorldPop high-resolution gridded population layers.",
	},
	"https://www.ncei.noaa.gov/maps/historical_weather/": {
		description: "NOAA NCEI historical weather map images (GeoTIFF/PNG).",
	},

	// ------------------------------------------------------------------
	// THEMATIC / SPECIALTY DATA SOURCES
	// ------------------------------------------------------------------
	"https://hydrosheds.org/products": {
		description: "HydroSHEDS global hydrological basins & river networks.",
	},
	"https://worldclim.org/data/index.html": {
		description: "WorldClim high-resolution climate normals & scenarios.",
	},
	"https://www.isric.org/explore/soilgrids": {
		description: "ISRIC SoilGrids – global gridded soil properties.",
	},
	"https://www.onegeology.org/data.html": {
		description: "OneGeology worldwide geologic map services & downloads.",
	},
	"https://www.obis.org/": {
		description: "OBIS – global marine species occurrence records.",
	},
	"https://www.movebank.org/cms/movebank-main": {
		description: "Movebank animal movement (GPS tracking) data.",
	},
	"https://www.gbif.org/data": {
		description: "GBIF – global biodiversity occurrence datasets.",
	},
	"https://gadm.org/data.html": {
		description: "GADM detailed global administrative boundaries.",
	},
	"https://data.humdata.org/": {
		description: "Humanitarian Data Exchange (HDX) – crisis & development datasets.",
	},
	"https://geoportal.kogis.or.kr/eng/index.do": {
		description: "Korea NSDI GeoPortal – national spatial data (example international).",
	},
	"https://www.jpl.nasa.gov/earth/earth-science-data/": {
		description: "NASA JPL Earth-science specialty datasets.",
	},
	"https://data.openstreetmap.la/": {
		description: "OpenStreetMap Latin America extracts.",
	},
	"https://land.copernicus.eu/global/products/gdd": {
		description: "Copernicus Global Land Service datasets.",
	},
	"https://climatedata.wri.org/": {
		description: "World Resources Institute climate indicators.",
	},
	"https://datacatalog.worldbank.org/search/type/dataset?sort_by=field_geo_coverage&sort_order=ASC": {
		description: "World Bank Data Catalog – geospatial filter.",
	},
	"https://www.fao.org/geospatial/resources/data/en/": {
		description: "FAO GeoNetwork global agriculture & land-use layers.",
	},
	"https://maps.ngdc.noaa.gov/viewers/bathymetry/": {
		description: "NOAA NGDC global bathymetry viewer & downloads.",
	},
	"https://marinecadastre.gov/data/": {
		description: "U.S. Marine Cadastre – ocean planning GIS layers.",
	},
	"https://www.nrcan.gc.ca/maps-tools-publications/geoscientific-data/17799": {
		description: "Natural Resources Canada national geoscience datasets.",
	},
	"https://www.statistikportal.de/de/daten/geodaten": {
		description: "Destatis (Germany) geospatial statistics downloads.",
	},
	"https://www.ign.es/web/ign/portal/ide-ign": {
		description: "Spanish National Geographic Institute open data.",
	},
	"https://www.data.gouv.fr/fr/datasets/?q=geospatial": {
		description: "French government open-data portal – geospatial search.",
	},
	"https://www.data.wa.gov/data-categories/geospatial": {
		description: "Washington State open geospatial data portal.",
	},
	"https://opendata.arcgis.com/": {
		description: "ArcGIS Hub global open-data endpoint (search thousands of orgs).",
	},
	"https://www.geopunt.be/en/data/open-data": {
		description: "Flanders (Belgium) GeoPunt open geospatial data.",
	},
	"https://geodata.lib.berkeley.edu/": {
		description: "UC Berkeley Library geospatial data repository.",
	},
	"https://www.caris.com/data/": {
		description: "CARIS sample hydrographic datasets.",
	},
	"https://digitalglobe.com/open-data/": {
		description: "Maxar DigitalGlobe Open Data (disaster imagery).",
	},
	"https://www.planet.com/open-data/": {
		description: "Planet Labs Open Data Program (disaster/event imagery).",
	},
	"https://www.esa.int/ESA_Multimedia/Images": {
		description: "ESA Earth Online imagery & data links.",
	},
	"https://www.copernicus.eu/en/access-data/copernicus-data-access": {
		description: "Copernicus consolidated data-access page.",
	},
	"https://www.ncei.noaa.gov/metadata/geoportal/rest/rpc/search/": {
		description: "NOAA NCEI GeoPortal REST search (machine-friendly).",
	},
	"https://earthquake.usgs.gov/earthquakes/feed/v1.0/geojson.php": {
		description: "USGS real-time earthquake GeoJSON feed.",
	},
	"https://www.un-sp.org/gis-open-data": {
		description: "United Nations spatial open-data hub.",
	},
	"https://ghgdata.epa.gov/ghgp/main.do": {
		description: "EPA Greenhouse Gas Reporting downloadable datasets.",
	},
	"https://www.data.gouv.fr/fr/datasets/r/d060c5a1-77e4-4d80-bc4f-4d43615b67d5": {
		description: "Direct French urban-area shapefile download example.",
	},
	"https://www.ngdc.noaa.gov/thredds/catalog.html": {
		description: "NOAA NCEI THREDDS scientific data server.",
	},
	"https://registry.opendata.aws/tag/geospatial/": {
		description: "AWS Open Data sets tagged ‘geospatial’.",
	},
	"https://s3-us-west-2.amazonaws.com/elevation-tiles-prod/tiles/9/152/207.terrain.mapbox": {
		description: "Sample Mapbox Terrain-RGB tile (pattern for all DEM tiles).",
	},
	"https://assets.publishing.service.gov.uk/government/uploads/system/uploads/attachment_data/file/762512/Local_Authority_Districts__December_2018__Boundaries_UK_BFC.zip": {
		description: "UK Local Authority boundary shapefile (direct ZIP).",
	},
	"https://geofabric.s3.amazonaws.com/updates/2024-07-07/europe/germany-latest.osm.pbf": {
		description: "Geofabrik daily OSM PBF extract (Germany) – direct S3 link.",
	},
	"https://opendata.arcgis.com/datasets/counties.zip": {
		description: "ArcGIS Hub example direct ZIP – U.S. counties shapefile.",
	},
	"https://www.hydro.washington.edu/data/grdc/hydro_data/GRDC_Monthly_Summary_Files.zip": {
		description: "GRDC monthly river-discharge summaries (ZIP).",
	},
	"https://www.epa.gov/sites/default/files/2015-12/documents/us_epa_facilities_20151218.zip": {
		description: "EPA facilities geodata (direct ZIP download).",
	},
	"https://data.europa.eu/euodp/data/dataset/eu-nuts-regions-2021": {
		description: "EU NUTS administrative regions shapefile & GeoPackage.",
	},
	"https://www.istat.it/it/archivio/104317": {
		description: "ISTAT (Italy) official boundary datasets.",
	},
	"https://geodata.statoil.com/": {
		description: "Equinor (Statoil) energy-sector open geodata.",
	},
	"https://data.linz.govt.nz/": {
		description: "Land Information New Zealand national datasets.",
	},
	"https://www.bgs.ac.uk/data-and-resources/": {
		description: "British Geological Survey downloadable data & maps.",
	},
	"https://data.nasa.gov/": {
		description: "NASA open-data portal with geospatial datasets.",
	},
	"https://www.eumetsat.int/data": {
		description: "EUMETSAT meteorological satellite data centre.",
	},
	"https://www.dwd.de/DE/leistungen/opendata/opendata_node.html": {
		description: "German Weather Service (DWD) open data catalogue.",
	},
	"https://www.jrc.ec.europa.eu/en/data": {
		description: "European Commission Joint Research Centre datasets.",
	},
	"https://www.eea.europa.eu/data-and-maps": {
		description: "European Environment Agency data & maps portal.",
	},
	"https://data.un.org/": {
		description: "United Nations open statistics portal (some spatial).",
	},
	"https://www.istat.it/it/archivio/267860": {
		description: "ISTAT (Italy) socio-economic geodata.",
	},
	"https://www.data.go.jp/data/jp_go_opendata_catalogue_dataset/?q=geospatial": {
		description: "Japan Government open-data (search ‘geospatial’).",
	},
	"https://geodata.bund.de/web/guest/start": {
		description: "Germany’s National Spatial Data Infrastructure (GDI-DE) portal.",
	},
	"https://www.esri.com/en-us/arcgis/products/data/open-data": {
		description: "Esri curated list of open data portals worldwide.",
	},
	"https://data.cityofnewyork.us/browse?q=geospatial&sortBy=relevance": {
		description: "NYC Open Data – geospatial filter.",
	},
	"https://www.chgis.org/data/geodatabase/": {
		description: "China Historical GIS geodatabases.",
	},
	"https://www.cegis.dk/geodata-downloads/": {
		description: "Danish Centre for Environment & Geoscience geodata.",
	},
	"https://geoportal.cuzk.cz/geoportal/eng/default.aspx": {
		description: "Czech national mapping authority GeoPortal.",
	},
}
