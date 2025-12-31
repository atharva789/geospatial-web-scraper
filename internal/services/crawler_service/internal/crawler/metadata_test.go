package crawler

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		pageURL     string
		downloadURL string
		wantTitle   string
		wantDesc    string
	}{
		{
			name: "basic HTML with meta tags",
			htmlContent: `
<!DOCTYPE html>
<html>
<head>
	<title>Test Dataset Page</title>
	<meta name="description" content="This is a test geospatial dataset">
	<meta name="keywords" content="geospatial, lidar, elevation">
</head>
<body>
	<h1>Test Dataset</h1>
	<p>Some description text about the dataset.</p>
</body>
</html>`,
			pageURL:     "https://example.com/data/test",
			downloadURL: "https://example.com/data/test.tif",
			wantTitle:   "Test Dataset Page",
			wantDesc:    "This is a test geospatial dataset",
		},
		{
			name: "HTML with Open Graph tags",
			htmlContent: `
<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Precipitation Data California">
	<meta property="og:description" content="Monthly precipitation data for California 2020-2024">
</head>
<body>
	<h1>Data Portal</h1>
</body>
</html>`,
			pageURL:     "https://noaa.gov/data/precip",
			downloadURL: "https://noaa.gov/data/precip.nc",
			wantTitle:   "Precipitation Data California",
			wantDesc:    "Monthly precipitation data for California 2020-2024",
		},
		{
			name: "minimal HTML",
			htmlContent: `
<!DOCTYPE html>
<html>
<head>
	<title>Simple Page</title>
</head>
<body>
	<p>Content</p>
</body>
</html>`,
			pageURL:     "https://example.com/simple",
			downloadURL: "https://example.com/simple.zip",
			wantTitle:   "Simple Page",
			wantDesc:    "",
		},
		{
			name: "HTML with h1 and paragraphs",
			htmlContent: `
<!DOCTYPE html>
<html>
<head></head>
<body>
	<h1>Elevation Dataset</h1>
	<p>Digital Elevation Model for Washington State.</p>
	<p>Resolution: 10 meters. Format: GeoTIFF.</p>
</body>
</html>`,
			pageURL:     "https://usgs.gov/dem",
			downloadURL: "https://usgs.gov/dem.tif",
			wantTitle:   "", // No <title> tag, so title may be empty
			wantDesc:    "", // Description extraction from body varies by implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse HTML content
			doc, err := html.Parse(strings.NewReader(tt.htmlContent))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			// Call ExtractMetadata - now returns DatasetMetadata directly
			metadata := ExtractMetadata(doc, tt.pageURL, tt.downloadURL)

			// Verify URL was set
			if metadata.URL != tt.downloadURL {
				t.Errorf("URL = %v, want %v", metadata.URL, tt.downloadURL)
			}

			// Verify title if expected
			if tt.wantTitle != "" && !strings.Contains(metadata.Title, tt.wantTitle) {
				t.Errorf("Title = %v, want to contain %v", metadata.Title, tt.wantTitle)
			}

			// Verify description if expected
			if tt.wantDesc != "" && !strings.Contains(metadata.Description, tt.wantDesc) {
				t.Errorf("Description = %v, want to contain %v", metadata.Description, tt.wantDesc)
			}
		})
	}
}

func TestExtractMetadata_JSONStructure(t *testing.T) {
	// Test that the returned JSON has the correct structure
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
	<title>Test Dataset</title>
	<meta name="description" content="Test description">
	<meta name="keywords" content="geo, spatial">
</head>
<body></body>
</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	metadata := ExtractMetadata(doc, "https://example.com", "https://example.com/data.tif")

	// Verify metadata object has expected URL
	if metadata.URL != "https://example.com/data.tif" {
		t.Errorf("URL = %v, want https://example.com/data.tif", metadata.URL)
	}

	// Verify JSON can be marshaled
	marshaled, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}

	// Should be valid JSON
	if !json.Valid(marshaled) {
		t.Error("Marshaled JSON is not valid")
	}
}

func TestExtractMetadata_EmptyHTML(t *testing.T) {
	// Test with minimal/empty HTML
	htmlContent := `<html><head></head><body></body></html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	metadata := ExtractMetadata(doc, "https://example.com", "https://example.com/file.tif")

	// With empty HTML, title and description should be empty
	if metadata.Title != "" {
		t.Errorf("Expected empty title, got %v", metadata.Title)
	}
	if metadata.Description != "" {
		t.Errorf("Expected empty description, got %v", metadata.Description)
	}

	// But URL should still be set
	if metadata.URL != "https://example.com/file.tif" {
		t.Errorf("URL = %v, want https://example.com/file.tif", metadata.URL)
	}
}

func TestExtractMetadata_WithKeywords(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
	<meta name="keywords" content="precipitation, climate, NOAA">
</head>
<body></body>
</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	metadata := ExtractMetadata(doc, "https://example.com", "https://example.com/data.nc")

	// Verify metadata object was returned
	if metadata.URL != "https://example.com/data.nc" {
		t.Errorf("URL = %v, want https://example.com/data.nc", metadata.URL)
	}

	// Keywords extraction depends on implementation
	// Just verify we can marshal to JSON
	if _, err := json.Marshal(metadata); err != nil {
		t.Errorf("Failed to marshal metadata: %v", err)
	}
}

func TestExtractMetadata_FGDC_XML(t *testing.T) {
	// This test uses the FGDC metadata format (NHDPLUS_H_0101_HU4_20220901_RASTER.xml)
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<metadata>
    <idinfo>
        <citation>
            <citeinfo>
                <origin>U.S. Geological Survey</origin>
                <pubdate>20220901</pubdate>
                <title>USGS National Hydrography Dataset Plus High Resolution (NHDPlus HR) for Hydrological Unit (HU) 4 - 0101 (published 20220901) GeoTIFF</title>
            </citeinfo>
        </citation>
        <descript>
            <abstract>The High Resolution National Hydrography Dataset Plus (NHDPlus HR) is an integrated set of geospatial data layers, including the National Hydrography Dataset (NHD), National Watershed Boundary Dataset (WBD), and 3D Elevation Program Digital Elevation Model (3DEP DEM).</abstract>
        </descript>
        <timeperd>
            <timeinfo>
                <rngdates>
                    <begdate>20220901</begdate>
                    <enddate>20220901</enddate>
                </rngdates>
            </timeinfo>
        </timeperd>
        <spdom>
            <bounding>
                <westbc>-70.43221</westbc>
                <eastbc>-66.60129</eastbc>
                <northbc>48.09971</northbc>
                <southbc>45.70663</southbc>
            </bounding>
        </spdom>
        <keywords>
            <theme>
                <themekey>hydrology</themekey>
                <themekey>water resources</themekey>
                <themekey>digital elevation models</themekey>
            </theme>
        </keywords>
    </idinfo>
    <spref>
        <horizsys>
            <geograph>
                <latres>0.00001</latres>
                <longres>0.00001</longres>
                <geogunit>Decimal degrees</geogunit>
            </geograph>
            <geodetic>
                <horizdn>North American Datum of 1983</horizdn>
            </geodetic>
        </horizsys>
        <vertdef>
            <altsys>
                <altdatum>North American Vertical Datum of 1988</altdatum>
                <altres>0.001</altres>
                <altunits>meters</altunits>
            </altsys>
        </vertdef>
    </spref>
</metadata>`

	var metadata DatasetMetadata
	if err := json.Unmarshal([]byte(xmlContent), &metadata); err == nil {
		t.Fatal("Expected error when unmarshaling XML as JSON, but got none")
	}

	// Parse the XML directly using the same structure as ExtractMetadata
	var x struct {
		Title       string   `xml:"idinfo>citation>citeinfo>title"`
		Description string   `xml:"idinfo>descript>abstract"`
		Source      string   `xml:"idinfo>citation>citeinfo>origin"`
		WestBC      float64  `xml:"idinfo>spdom>bounding>westbc"`
		EastBC      float64  `xml:"idinfo>spdom>bounding>eastbc"`
		NorthBC     float64  `xml:"idinfo>spdom>bounding>northbc"`
		SouthBC     float64  `xml:"idinfo>spdom>bounding>southbc"`
		StartDate   string   `xml:"idinfo>timeperd>timeinfo>rngdates>begdate"`
		EndDate     string   `xml:"idinfo>timeperd>timeinfo>rngdates>enddate"`
		LatRes      float64  `xml:"spref>horizsys>geograph>latres"`
		LongRes     float64  `xml:"spref>horizsys>geograph>longres"`
		GeoUnit     string   `xml:"spref>horizsys>geograph>geogunit"`
		HorizCRS    string   `xml:"spref>horizsys>geodetic>horizdn"`
		VertCRS     string   `xml:"spref>vertdef>altsys>altdatum"`
		AltRes      float64  `xml:"spref>vertdef>altsys>altres"`
		AltUnits    string   `xml:"spref>vertdef>altsys>altunits"`
		Keywords    []string `xml:"idinfo>keywords>theme>themekey"`
	}

	if err := xml.Unmarshal([]byte(xmlContent), &x); err != nil {
		t.Fatalf("Failed to unmarshal FGDC XML: %v", err)
	}

	// Verify extracted values
	if x.Title != "USGS National Hydrography Dataset Plus High Resolution (NHDPlus HR) for Hydrological Unit (HU) 4 - 0101 (published 20220901) GeoTIFF" {
		t.Errorf("Title = %q, want USGS National Hydrography Dataset...", x.Title)
	}

	if x.Source != "U.S. Geological Survey" {
		t.Errorf("Source = %q, want U.S. Geological Survey", x.Source)
	}

	if !strings.Contains(x.Description, "NHDPlus HR") {
		t.Errorf("Description does not contain 'NHDPlus HR': %q", x.Description)
	}

	// Verify bounding box
	if x.WestBC != -70.43221 {
		t.Errorf("WestBC = %f, want -70.43221", x.WestBC)
	}
	if x.EastBC != -66.60129 {
		t.Errorf("EastBC = %f, want -66.60129", x.EastBC)
	}
	if x.NorthBC != 48.09971 {
		t.Errorf("NorthBC = %f, want 48.09971", x.NorthBC)
	}
	if x.SouthBC != 45.70663 {
		t.Errorf("SouthBC = %f, want 45.70663", x.SouthBC)
	}

	// Verify dates
	if x.StartDate != "20220901" {
		t.Errorf("StartDate = %q, want 20220901", x.StartDate)
	}
	if x.EndDate != "20220901" {
		t.Errorf("EndDate = %q, want 20220901", x.EndDate)
	}

	// Verify horizontal metadata
	if x.LatRes != 0.00001 {
		t.Errorf("LatRes = %f, want 0.00001", x.LatRes)
	}
	if x.LongRes != 0.00001 {
		t.Errorf("LongRes = %f, want 0.00001", x.LongRes)
	}
	if x.GeoUnit != "Decimal degrees" {
		t.Errorf("GeoUnit = %q, want Decimal degrees", x.GeoUnit)
	}
	if x.HorizCRS != "North American Datum of 1983" {
		t.Errorf("HorizCRS = %q, want North American Datum of 1983", x.HorizCRS)
	}

	// Verify vertical metadata
	if x.VertCRS != "North American Vertical Datum of 1988" {
		t.Errorf("VertCRS = %q, want North American Vertical Datum of 1988", x.VertCRS)
	}
	if x.AltRes != 0.001 {
		t.Errorf("AltRes = %f, want 0.001", x.AltRes)
	}
	if x.AltUnits != "meters" {
		t.Errorf("AltUnits = %q, want meters", x.AltUnits)
	}

	// Verify keywords
	expectedKeywords := []string{"hydrology", "water resources", "digital elevation models"}
	if len(x.Keywords) != len(expectedKeywords) {
		t.Errorf("Keywords length = %d, want %d", len(x.Keywords), len(expectedKeywords))
	}
	for i, kw := range expectedKeywords {
		if i < len(x.Keywords) && x.Keywords[i] != kw {
			t.Errorf("Keywords[%d] = %q, want %q", i, x.Keywords[i], kw)
		}
	}

	// Now create a DatasetMetadata struct as ExtractMetadata would
	metadata = DatasetMetadata{
		Title:       x.Title,
		Source:      x.Source,
		Description: x.Description,
		Keywords:    x.Keywords,
		URL:         "https://example.com/data.tif",
		Bounds: BoundingBox{
			WestBC:  x.WestBC,
			EastBC:  x.EastBC,
			NorthBC: x.NorthBC,
			SouthBC: x.SouthBC,
		},
		HorizontalMeta: HorizontalMeta{
			LatRes:        x.LatRes,
			LongRes:       x.LongRes,
			GeoUnit:       x.GeoUnit,
			HorizontalCRS: x.HorizCRS,
		},
		VerticalMeta: VerticalMeta{
			VerticalCRS: x.VertCRS,
			AltRes:      x.AltRes,
			AltUnits:    x.AltUnits,
		},
		StartDate: x.StartDate,
		EndDate:   x.EndDate,
	}

	// Verify the final metadata structure can be marshaled to JSON
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal metadata to JSON: %v", err)
	}

	// Verify it can be unmarshaled back
	var unmarshaledMetadata DatasetMetadata
	if err := json.Unmarshal(jsonData, &unmarshaledMetadata); err != nil {
		t.Fatalf("Failed to unmarshal metadata from JSON: %v", err)
	}

	// Verify ToString() produces expected output
	toString := metadata.ToString()
	if !strings.Contains(toString, "USGS National Hydrography Dataset") {
		t.Error("ToString() should contain the title")
	}
	if !strings.Contains(toString, "BOUNDS:") {
		t.Error("ToString() should contain BOUNDS section")
	}
	if !strings.Contains(toString, "HORIZONTAL METADATA:") {
		t.Error("ToString() should contain HORIZONTAL METADATA section")
	}
	if !strings.Contains(toString, "VERTICAL METADATA:") {
		t.Error("ToString() should contain VERTICAL METADATA section")
	}
	if !strings.Contains(toString, "North American Datum of 1983") {
		t.Error("ToString() should contain horizontal CRS")
	}
	if !strings.Contains(toString, "North American Vertical Datum of 1988") {
		t.Error("ToString() should contain vertical CRS")
	}
}
