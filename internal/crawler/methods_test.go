package crawler

import (
	"math"
	"testing"
)

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if idx := Contains("B", slice); idx != 1 {
		t.Errorf("expected 1, got %d", idx)
	}
	if idx := Contains("d", slice); idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}

func TestMergeSort(t *testing.T) {
	nodes := []WebNode{
		{Url: "u1", CosineSimilarity: 0.9},
		{Url: "u2", CosineSimilarity: 0.1},
		{Url: "u3", CosineSimilarity: 0.5},
		{Url: "u4", CosineSimilarity: 0.3},
	}
	sorted := MergeSort(&nodes, 0, len(nodes))
	expected := []string{"u2", "u4", "u3", "u1"}
	for i, node := range sorted {
		if node.Url != expected[i] {
			t.Fatalf("at %d want %s got %s", i, expected[i], node.Url)
		}
	}
}

func TestMerge(t *testing.T) {
	a := []WebNode{{Url: "a1", CosineSimilarity: 0.2}, {Url: "a2", CosineSimilarity: 0.4}}
	b := []WebNode{{Url: "b1", CosineSimilarity: 0.1}, {Url: "b2", CosineSimilarity: 0.3}}
	merged := Merge(&a, &b)
	expected := []string{"b1", "a1", "b2", "a2"}
	for i, node := range merged {
		if node.Url != expected[i] {
			t.Fatalf("at %d want %s got %s", i, expected[i], node.Url)
		}
	}
}

func TestCosine(t *testing.T) {
	a := []float64{1, 0, -1}
	b := []float64{1, 0, -1}
	sim, err := Cosine(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(sim-1) > 1e-9 {
		t.Fatalf("expected similarity 1, got %v", sim)
	}
	if _, err := Cosine([]float64{0, 0}, []float64{0, 0}); err == nil {
		t.Fatalf("expected error for zero vector")
	}
}
