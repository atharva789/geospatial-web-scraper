package main

import (
	"os"
	"testing"
)

func TestHashEmbedderLength(t *testing.T) {
	emb := hashEmbedder{dim: 32}
	v, err := emb.Embed(nil, "test text")
	if err != nil {
		t.Fatalf("embed error: %v", err)
	}
	if len(v) != 32 {
		t.Fatalf("expected length 32, got %d", len(v))
	}
}

func TestParseIntEnv(t *testing.T) {
	os.Setenv("TMP_INT", "42")
	defer os.Unsetenv("TMP_INT")
	if got := parseIntEnv("TMP_INT", 1); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
	if got := parseIntEnv("MISSING_INT", 7); got != 7 {
		t.Fatalf("expected default 7, got %d", got)
	}
}

func TestParseOrigins(t *testing.T) {
	os.Setenv("CORS_ORIGINS", "http://a.com, http://b.com")
	defer os.Unsetenv("CORS_ORIGINS")
	origins := parseOrigins()
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(origins))
	}
}
