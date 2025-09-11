package crawler

import (
	"errors"
	"math"
	"runtime"
	"strings"
	"sync"
)

// Contains returns the index of value in slice, ignoring case. If the value is
// not present it returns -1.
func Contains(value string, slice []string) int {
	for idx, wrd := range slice {
		if strings.Compare(strings.ToLower(wrd), strings.ToLower(value)) == 0 {
			return idx
		}
	}
	return -1
}

// MergeSort sorts the slice of WebNodes in place by CosineSimilarity using a
// classic recursive merge sort algorithm.
func MergeSort(list *[]WebNode, start int, end int) []WebNode {
	if end-start <= 1 {
		return (*list)[start:end]
	}

	midpoint := start + (end-start)/2

	left := MergeSort(list, start, midpoint)
	right := MergeSort(list, midpoint, end)

	return Merge(&left, &right)
}

// Merge merges two already sorted WebNode slices by CosineSimilarity and
// returns the combined sorted slice.
func Merge(a *[]WebNode, b *[]WebNode) []WebNode {
	result := make([]WebNode, 0, len(*a)+len(*b))
	i, j := 0, 0
	aRef, bRef := *a, *b

	for i < len(aRef) && j < len(bRef) {
		if aRef[i].CosineSimilarity <= bRef[j].CosineSimilarity {
			result = append(result, aRef[i])
			i++
		} else {
			result = append(result, bRef[j])
			j++
		}
	}

	if i < len(aRef) {
		result = append(result, aRef[i:]...)
	}
	if j < len(bRef) {
		result = append(result, bRef[j:]...)
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
