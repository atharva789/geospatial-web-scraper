package crawler

import (
	"encoding/xml"
	"testing"
)

func TestMatchGeoFilesWithMetadata(t *testing.T) {
	tests := []struct {
		name      string
		geoFiles  []string
		metaFiles []string
		expected  []GeoFile
	}{
		{
			name: "perfect match",
			geoFiles: []string{
				"https://example.com/data/elevation.tif",
				"https://example.com/data/temperature.nc",
			},
			metaFiles: []string{
				"https://example.com/data/elevation.xml",
				"https://example.com/data/temperature.json",
			},
			expected: []GeoFile{
				{URL: "https://example.com/data/elevation.tif", Metadata: "https://example.com/data/elevation.xml"},
				{URL: "https://example.com/data/temperature.nc", Metadata: "https://example.com/data/temperature.json"},
			},
		},
		{
			name: "partial match",
			geoFiles: []string{
				"https://example.com/data/elevation.tif",
				"https://example.com/data/temperature.nc",
			},
			metaFiles: []string{
				"https://example.com/data/elevation.xml",
			},
			expected: []GeoFile{
				{URL: "https://example.com/data/elevation.tif", Metadata: "https://example.com/data/elevation.xml"},
				{URL: "https://example.com/data/temperature.nc", Metadata: ""},
			},
		},
		{
			name: "no matches",
			geoFiles: []string{
				"https://example.com/data/elevation.tif",
			},
			metaFiles: []string{
				"https://example.com/data/other.xml",
			},
			expected: []GeoFile{
				{URL: "https://example.com/data/elevation.tif", Metadata: ""},
			},
		},
		{
			name:      "empty inputs",
			geoFiles:  []string{},
			metaFiles: []string{},
			expected:  []GeoFile{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchGeoFilesWithMetadata(tt.geoFiles, tt.metaFiles)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if result[i].URL != expected.URL {
					t.Errorf("result[%d].URL = %v, want %v", i, result[i].URL, expected.URL)
				}
				if result[i].Metadata != expected.Metadata {
					t.Errorf("result[%d].Metadata = %v, want %v", i, result[i].Metadata, expected.Metadata)
				}
			}
		})
	}
}

func TestIsDir(t *testing.T) {
	tests := []struct {
		link     string
		expected bool
	}{
		{"https://example.com/data/", true},
		{"https://example.com/data", false},
		{"/path/to/dir/", true},
		{"/path/to/file.txt", true}, // starts with / so isDir returns true
		{"subdir/", true},
		{"file.tif", false},
	}

	for _, tt := range tests {
		t.Run(tt.link, func(t *testing.T) {
			got := isDir(tt.link)
			if got != tt.expected {
				t.Errorf("isDir(%s) = %v, want %v", tt.link, got, tt.expected)
			}
		})
	}
}

func TestS3ListBucketResult(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
    <Name>test-bucket</Name>
    <Prefix>data/</Prefix>
    <IsTruncated>false</IsTruncated>
    <Contents>
        <Key>data/file1.tif</Key>
        <LastModified>2024-01-01T00:00:00.000Z</LastModified>
        <Size>1024</Size>
    </Contents>
    <Contents>
        <Key>data/file2.nc</Key>
        <LastModified>2024-01-02T00:00:00.000Z</LastModified>
        <Size>2048</Size>
    </Contents>
    <CommonPrefixes>
        <Prefix>data/subdir/</Prefix>
    </CommonPrefixes>
</ListBucketResult>`

	var result S3ListBucketResult
	err := xml.Unmarshal([]byte(xmlData), &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	if result.Name != "test-bucket" {
		t.Errorf("Name = %v, want test-bucket", result.Name)
	}
	if result.Prefix != "data/" {
		t.Errorf("Prefix = %v, want data/", result.Prefix)
	}
	if result.IsTruncated {
		t.Error("IsTruncated should be false")
	}
	if len(result.Contents) != 2 {
		t.Errorf("Contents length = %d, want 2", len(result.Contents))
	}
	if len(result.CommonPrefixes) != 1 {
		t.Errorf("CommonPrefixes length = %d, want 1", len(result.CommonPrefixes))
	}
	if result.Contents[0].Key != "data/file1.tif" {
		t.Errorf("First content key = %v, want data/file1.tif", result.Contents[0].Key)
	}
	if result.Contents[0].Size != 1024 {
		t.Errorf("First content size = %v, want 1024", result.Contents[0].Size)
	}
}

func TestCrawlNodeStructure(t *testing.T) {
	parent := &CrawlNode{
		URL:   "https://example.com/root",
		Depth: 0,
	}

	child := &CrawlNode{
		URL:              "https://example.com/root/child",
		Parent:           parent,
		Depth:            1,
		CosineSimilarity: 0.85,
	}

	if child.Parent.URL != "https://example.com/root" {
		t.Errorf("Child parent URL = %v, want https://example.com/root", child.Parent.URL)
	}
	if child.Depth != 1 {
		t.Errorf("Child depth = %v, want 1", child.Depth)
	}
	if child.CosineSimilarity != 0.85 {
		t.Errorf("CosineSimilarity = %v, want 0.85", child.CosineSimilarity)
	}
}

func TestFTPDirectoryStructure(t *testing.T) {
	// Test hierarchical structure
	root := &FTPDirectory{
		Parent:         nil,
		SubDirectories: nil,
		DownloadFiles: []GeoFile{
			{URL: "https://example.com/root/file1.tif", Metadata: "https://example.com/root/file1.xml"},
		},
	}

	subdir := &FTPDirectory{
		Parent:         root,
		SubDirectories: nil,
		DownloadFiles: []GeoFile{
			{URL: "https://example.com/root/subdir/file2.nc", Metadata: ""},
		},
	}

	root.SubDirectories = []*FTPDirectory{subdir}

	// Verify root
	if root.Parent != nil {
		t.Error("Root should have no parent")
	}
	if len(root.SubDirectories) != 1 {
		t.Errorf("Root should have 1 subdirectory, got %d", len(root.SubDirectories))
	}
	if len(root.DownloadFiles) != 1 {
		t.Errorf("Root should have 1 file, got %d", len(root.DownloadFiles))
	}

	// Verify subdirectory
	if subdir.Parent != root {
		t.Error("Subdirectory parent should be root")
	}
	if len(subdir.DownloadFiles) != 1 {
		t.Errorf("Subdirectory should have 1 file, got %d", len(subdir.DownloadFiles))
	}
}

func TestNormalizedQueryStructure(t *testing.T) {
	query := NormalizedQuery{
		CleanedQuery: "precipitation data California 2020",
		DataEntity:   "precipitation",
		OutputFormat: "NetCDF",
		Location:     "California",
		CountryCode:  "US",
		StartDate:    "2020-01-01",
		EndDate:      "2020-12-31",
		Sources:      []string{"https://noaa.gov", "https://usgs.gov"},
	}

	if query.DataEntity != "precipitation" {
		t.Errorf("DataEntity = %v, want precipitation", query.DataEntity)
	}
	if query.OutputFormat != "NetCDF" {
		t.Errorf("OutputFormat = %v, want NetCDF", query.OutputFormat)
	}
	if len(query.Sources) != 2 {
		t.Errorf("Sources length = %d, want 2", len(query.Sources))
	}
}
