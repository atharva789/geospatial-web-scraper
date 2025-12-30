package crawler

import (
	"testing"
)

func TestGeoFileExtensions(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".tif", true},
		{".tiff", true},
		{".geojson", true},
		{".nc", true},
		{".txt", false},
		{".html", false},
		{".zip", true},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := GeoFileExtensions[tt.ext]
			if got != tt.expected {
				t.Errorf("GeoFileExtensions[%s] = %v, want %v", tt.ext, got, tt.expected)
			}
		})
	}
}

func TestGeoMetaFileExtensions(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".xml", true},
		{".json", true},
		{".jsonld", true},
		{".txt", false},
		{".html", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := GeoMetaFileExtensions[tt.ext]
			if got != tt.expected {
				t.Errorf("GeoMetaFileExtensions[%s] = %v, want %v", tt.ext, got, tt.expected)
			}
		})
	}
}

func TestGeoMIMETypes(t *testing.T) {
	tests := []struct {
		mime     string
		expected bool
	}{
		{"application/geo+json", true},
		{"application/x-geotiff", true},
		{"application/x-netcdf", true},
		{"text/html", false},
		{"text/plain", false},
		{"application/json", true},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			got := GeoMIMETypes[tt.mime]
			if got != tt.expected {
				t.Errorf("GeoMIMETypes[%s] = %v, want %v", tt.mime, got, tt.expected)
			}
		})
	}
}
