package crawler

import (
	"encoding/json"
	"strings"
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
	htmlStr := `<html><head><title>Dataset</title><meta name="description" content="some data"><meta name="keywords" content="geo,data"></head><body></body></html>`
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	res := ExtractMetadata(doc, "http://example.com/page", "http://example.com/file.zip")
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
