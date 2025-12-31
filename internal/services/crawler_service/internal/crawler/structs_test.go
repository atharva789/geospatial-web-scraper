package crawler

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDatasetMetadata_NewStructure(t *testing.T) {
	// Create a DatasetMetadata with all new fields populated
	metadata := DatasetMetadata{
		Title:       "Test Elevation Dataset",
		Source:      "USGS",
		Description: "High-resolution elevation data",
		Keywords:    []string{"elevation", "DEM", "USGS"},
		URL:         "https://example.com/dem.tif",
		Bounds: BoundingBox{
			NorthBC: 45.0,
			SouthBC: 44.0,
			EastBC:  -120.0,
			WestBC:  -121.0,
		},
		HorizontalMeta: HorizontalMeta{
			LatRes:        0.0001,
			LongRes:       0.0001,
			GeoUnit:       "degrees",
			HorizontalCRS: "EPSG:4326",
		},
		VerticalMeta: VerticalMeta{
			VerticalCRS: "NAVD88",
			AltRes:      0.5,
			AltUnits:    "meters",
		},
		StartDate: "2020-01-01",
		EndDate:   "2020-12-31",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledMetadata DatasetMetadata
	if err := json.Unmarshal(jsonData, &unmarshaledMetadata); err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	// Verify HorizontalMeta fields
	if unmarshaledMetadata.HorizontalMeta.LatRes != 0.0001 {
		t.Errorf("HorizontalMeta.LatRes = %v, want 0.0001", unmarshaledMetadata.HorizontalMeta.LatRes)
	}
	if unmarshaledMetadata.HorizontalMeta.HorizontalCRS != "EPSG:4326" {
		t.Errorf("HorizontalMeta.HorizontalCRS = %v, want EPSG:4326", unmarshaledMetadata.HorizontalMeta.HorizontalCRS)
	}

	// Verify VerticalMeta fields
	if unmarshaledMetadata.VerticalMeta.VerticalCRS != "NAVD88" {
		t.Errorf("VerticalMeta.VerticalCRS = %v, want NAVD88", unmarshaledMetadata.VerticalMeta.VerticalCRS)
	}
	if unmarshaledMetadata.VerticalMeta.AltRes != 0.5 {
		t.Errorf("VerticalMeta.AltRes = %v, want 0.5", unmarshaledMetadata.VerticalMeta.AltRes)
	}
	if unmarshaledMetadata.VerticalMeta.AltUnits != "meters" {
		t.Errorf("VerticalMeta.AltUnits = %v, want meters", unmarshaledMetadata.VerticalMeta.AltUnits)
	}
}

func TestDatasetMetadata_ToString_WithNewFields(t *testing.T) {
	metadata := DatasetMetadata{
		Title: "Test Dataset",
		URL:   "https://example.com/data.tif",
		Bounds: BoundingBox{
			NorthBC: 40.0,
			SouthBC: 39.0,
			EastBC:  -105.0,
			WestBC:  -106.0,
		},
		HorizontalMeta: HorizontalMeta{
			LatRes:        0.0001,
			LongRes:       0.0002,
			GeoUnit:       "degrees",
			HorizontalCRS: "WGS84",
		},
		VerticalMeta: VerticalMeta{
			VerticalCRS: "EGM96",
			AltRes:      1.0,
			AltUnits:    "meters",
		},
	}

	result := metadata.ToString()

	// Verify BOUNDS section
	if !strings.Contains(result, "BOUNDS:") {
		t.Error("ToString() missing BOUNDS section")
	}

	// Verify HORIZONTAL METADATA section
	if !strings.Contains(result, "HORIZONTAL METADATA:") {
		t.Error("ToString() missing HORIZONTAL METADATA section")
	}
	if !strings.Contains(result, "RESOLUTION:") {
		t.Error("ToString() missing RESOLUTION in horizontal metadata")
	}
	if !strings.Contains(result, "CRS: WGS84") {
		t.Error("ToString() missing horizontal CRS")
	}

	// Verify VERTICAL METADATA section
	if !strings.Contains(result, "VERTICAL METADATA:") {
		t.Error("ToString() missing VERTICAL METADATA section")
	}
	if !strings.Contains(result, "CRS: EGM96") {
		t.Error("ToString() missing vertical CRS")
	}
	if !strings.Contains(result, "ALTITUDE RESOLUTION:") {
		t.Error("ToString() missing altitude resolution")
	}
}

func TestDatasetMetadata_ToString_EmptyMetadata(t *testing.T) {
	// Test with minimal metadata to ensure no empty sections are printed
	metadata := DatasetMetadata{
		Title: "Minimal Dataset",
		URL:   "https://example.com/data.nc",
	}

	result := metadata.ToString()

	// Should not contain metadata sections if fields are empty
	if strings.Contains(result, "HORIZONTAL METADATA:") {
		t.Error("ToString() should not include HORIZONTAL METADATA when fields are empty")
	}
	if strings.Contains(result, "VERTICAL METADATA:") {
		t.Error("ToString() should not include VERTICAL METADATA when fields are empty")
	}
}
