package crawler

import (
	"errors"
	"math"
	"runtime"
	"strings"
	"sync"
)

func Contains(value string, slice []string) int {
	for idx, wrd := range slice {
		if strings.Compare(strings.ToLower(wrd), strings.ToLower(value)) == 0 {
			return idx
		}
	}
	return -1
}

func MergeSort(list *[]WebNode, start int, end int) []EmbeddedNode {
	midpoint := end - start
	middle := float64(midpoint / 2)
	midpoint = int(math.Round(middle))

	//sort left
	a := MergeSort(list, start, midpoint)
	//sort right
	b := MergeSort(list, midpoint, end)
	//join
	final := Merge(&a, &b)
	return final

}

func Merge(a *[]EmbeddedNode, b *[]EmbeddedNode) []EmbeddedNode {
	result := []EmbeddedNode{}
	i, j := 0, 0
	a_dref, b_dref := *a, *b
	for {
		if i <= len(a_dref)-1 || j <= len(b_dref)-1 {
			if a_dref[i].cosine_similarity <= b_dref[j].cosine_similarity {
				result = append(result, a_dref[i])
				i++
			}
			if a_dref[i].cosine_similarity > b_dref[j].cosine_similarity {
				result = append(result, b_dref[j])
				j++
			}
		} else {
			break
		}
	}

	if i == len(a_dref)-1 {
		for _, embNode := range b_dref[j:] {
			result = append(result, embNode)
		}
	} else {
		for _, embNode := range a_dref[i:] {
			result = append(result, embNode)
		}
	}
	return result

}

// Cosine returns the cosine similarity of a and b.
//
//   - a and b must have identical length.
//   - If either vector is all-zero, an error is returned.
//   - Internally the work is split across GOMAXPROCS workers.
func Cosine(a, b []float64) (float64, error) {
	n := len(a)
	if n == 0 || n != len(b) {
		return 0, errors.New("vectors must be same non-zero length")
	}

	// tiny vectors: sequential is faster than goroutines
	if n < 1_024 {
		var dot, na2, nb2 float64
		for i := 0; i < n; i++ {
			dot += a[i] * b[i]
			na2 += a[i] * a[i]
			nb2 += b[i] * b[i]
		}
		return finalize(dot, na2, nb2)
	}

	workers := runtime.GOMAXPROCS(0)
	chunk := (n + workers - 1) / workers // ceil(n/workers)

	var wg sync.WaitGroup
	wg.Add(workers)

	type partial struct{ dot, na2, nb2 float64 }
	part := make([]partial, workers)

	for w := 0; w < workers; w++ {
		start := w * chunk
		end := start + chunk
		if end > n {
			end = n
		}

		go func(id, lo, hi int) {
			defer wg.Done()
			var p partial
			for i := lo; i < hi; i++ {
				p.dot += a[i] * b[i]
				p.na2 += a[i] * a[i]
				p.nb2 += b[i] * b[i]
			}
			part[id] = p
		}(w, start, end)
	}

	wg.Wait()

	var dot, na2, nb2 float64
	for _, p := range part {
		dot += p.dot
		na2 += p.na2
		nb2 += p.nb2
	}

	return finalize(dot, na2, nb2)
}

// helper: handle zero-vector cases & compute final ratio
func finalize(dot, na2, nb2 float64) (float64, error) {
	den := math.Sqrt(na2) * math.Sqrt(nb2)
	if den == 0 {
		return 0, errors.New("one of the vectors is zero (undefined similarity)")
	}
	return dot / den, nil
}
