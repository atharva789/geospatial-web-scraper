package crawler

type WebNode struct {
	Url              string
	Parent           *WebNode // node is a parent if parentURL == "root"
	Depth            int
	context          DataContext
	CosineSimilarity float64
}

type Manager struct {
	secure              bool
	downloadPath        *string
	searchQuery         *string
	downloadURLs        []WebNode
	CachedURLEmbeddings map[string][]float64
	searchFrom          map[string]DataContext
	linkChan            chan struct{}
	smTokens            chan struct{}
	dlTokens            chan struct{}
	worklist            chan []WebNode
	done                chan bool
	seen                map[string]bool
}

// DataContext holds metadata about a public data source.
type DataContext struct {
	description string    // human-readable description of the endpoint
	embedding   []float64 // placeholder for a future embedding value
}

type TextPayload struct {
	Texts []string `json:"texts"`
}

type EmbeddingResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

//how .gob files will be stored
// link string : DataContext{description string, embedding float64}

// .gob file is map[string] float64 for now. In the future, it should be
// map[string] Cache

// Cache will have Cache{embedding float64, description string, filepath string}
func SlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	freq := make(map[string]int)
	for _, x := range a {
		freq[x]++
	}
	for _, y := range b {
		freq[y]--
	}

	for _, count := range freq {
		if count != 0 {
			return false
		}
	}
	return true
}
