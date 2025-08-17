package crawler

import (
	"encoding/json"
	"net/http"
	"testing"

	"golang.org/x/net/html"
)

type testMeta struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	URL         string   `json:"url"`
}

func TestExtractMetadata(t *testing.T) {
	url := "https://catalog.data.gov/dataset/electric-vehicle-population-data"
	downloadURL := "https://data.wa.gov/api/views/f6w7-q2d2/rows.csv?accessType=DOWNLOAD"
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("Error while requesting url: %v, %v", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Request returned invalid response: %v", err)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	res := ExtractMetadata(doc, url, downloadURL)
	var md testMeta
	if err := json.Unmarshal([]byte(res), &md); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if md.Title != "Dataset" {
		t.Errorf("title mismatch: %s", md.Title)
	}
	if md.Description != "some data" {
		t.Errorf("description mismatch: %s", md.Description)
	}
	if len(md.Keywords) != 2 || md.Keywords[0] != "geo" || md.Keywords[1] != "data" {
		t.Errorf("keywords mismatch: %v", md.Keywords)
	}
	if md.URL != "http://example.com/file.zip" {
		t.Errorf("url mismatch: %s", md.URL)
	}
}
