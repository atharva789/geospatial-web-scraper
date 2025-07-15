package crawler

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func setupManager() *Manager {
	return &Manager{
		downloadPath: new(string),
		linkChan:     make(chan struct{}, 1),
		smTokens:     make(chan struct{}, 1),
		dlTokens:     make(chan struct{}, 1),
		worklist:     make(chan []WebNode),
		done:         make(chan bool),
		seen:         make(map[string]bool),
	}
}

func TestExtract2_Downloadable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Write([]byte("zipdata"))
	}))
	defer ts.Close()

	mg := setupManager()
	node := &WebNode{Url: ts.URL}
	links, err := mg.Extract2(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if links != nil {
		t.Fatalf("expected nil links, got %v", links)
	}
	if len(mg.downloadURLs) != 0 {
		t.Fatalf("expected downloadURLs empty, got %v", mg.downloadURLs)
	}
}

func TestExtract2_HTMLAddsDownloadURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><a href='/file.zip'>f</a></body></html>"))
	}))
	defer ts.Close()

	mg := setupManager()
	node := &WebNode{Url: ts.URL}
	links, err := mg.Extract2(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("expected no crawl links, got %v", links)
	}
	if len(mg.downloadURLs) != 1 {
		t.Fatalf("expected 1 download URL, got %d", len(mg.downloadURLs))
	}
	if mg.downloadURLs[0].Url != ts.URL+"/file.zip" {
		t.Fatalf("unexpected URL %s", mg.downloadURLs[0].Url)
	}
}

func TestToLinks(t *testing.T) {
	mg := Manager{downloadURLs: []WebNode{{Url: "a"}, {Url: "b"}}}
	got := mg.ToLinks()
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGenerateEmbeddings(t *testing.T) {
	// override seeds with two entries
	orig := PublicGeospatialDataSeeds
	PublicGeospatialDataSeeds = map[string]DataContext{
		"u1": {description: "d1"},
		"u2": {description: "d2"},
	}
	defer func() { PublicGeospatialDataSeeds = orig }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p TextPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		resp := EmbeddingResponse{Embeddings: [][]float64{{1, 2}, {3, 4}}}
		json.NewEncoder(w).Encode(resp)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Skipf("can't listen on 8080: %v", err)
	}
	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)
	defer srv.Close()

	embeddings, err := GenerateEmbeddings()
	if err != nil {
		t.Fatalf("GenerateEmbeddings error: %v", err)
	}
	want := [][]float64{{1, 2}, {3, 4}}
	if !reflect.DeepEqual(embeddings, want) {
		t.Fatalf("got %v want %v", embeddings, want)
	}
}
