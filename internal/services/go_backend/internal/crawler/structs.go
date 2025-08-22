package crawler

import (
        "net/http"

        "golang.org/x/net/html"
        "google.golang.org/grpc"
)

type WebNode struct {
	Url              string
	Parent           *WebNode // node is a parent if parentURL == "root"
	Depth            int
	context          DataContext
	CosineSimilarity float64
}

// Query captures the components of a user provided query after it has been
// normalized by the Python gRPC service.
type Query struct {
        DataEntity   string   `json:"data_entity"`
        Locations    []string `json:"locations"`
        OutputFormat string   `json:"output_format"`
        StartDate    string   `json:"start_date"`
        EndDate      string   `json:"end_date"`
}

// SearchQuery represents the complete query request coming from the client. It
// may include direct URLs, data sources and the normalized user query.
type SearchQuery struct {
        URLs     []string `json:"urls"`
        Sources  []string `json:"sources"`
        UserQuery Query   `json:"user_query"`
}

type Manager struct {
        secure              bool
        downloadPath        *string
        searchQuery         *SearchQuery
        downloadURLs        []WebNode
        CachedURLEmbeddings map[string]DataContext
        searchFrom          map[string]DataContext
        linkChan            chan struct{}
        smTokens            chan struct{}
        dlTokens            chan struct{}
        worklist            chan []WebNode
        done                chan bool
        seen                map[string]bool
        conn                *grpc.ClientConn // python metadata service
        httpClient          *http.Client
        LlmApiKey           string
}

// DataContext holds metadata about a public data source.
type DataContext struct {
	Description string    // human-readable description of the endpoint
	Embedding   []float64 // placeholder for a future embedding value
}

// downloadMetadata represents extracted information about a downloadable file.
type downloadMetadata struct {
	Title       string            `json:"title,omitempty"`
	Source      *downloadMetadata `json:"source,omitempty"`
	Description string            `json:"description,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	URL         string            `json:"url"`
}

type TextPayload struct {
	Texts []string `json:"texts"`
}

type EmbeddingResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

type LLMQuery struct {
	Model    string       `json:"model"`
	Messages []LLMMessage `json:"messages"`
}

type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqApiResp struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`

	// When non-200, Groq (OpenAI-compatible) sends an error object:
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   any    `json:"param"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type TableData struct {
	Root    *html.Node
	Headers []string
	Data    []string
}

//how .gob files will be stored
// link string : DataContext{Description string, Embedding []float64}

// .gob file is map[string] float64 for now. In the future, it should be
// map[string] Cache

// Cache will have Cache{Embedding []float64, Description string, filepath string}
// SlicesEqualUnordered compares two string slices regardless of element order.
// It returns true when both slices contain the same elements with the same
// multiplicities.
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
