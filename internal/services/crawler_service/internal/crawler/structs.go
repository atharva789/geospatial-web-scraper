// Package crawler provides functionality for discovering and crawling geospatial dataset URLs
// from public data sources. It supports structured query-based crawling with metadata extraction.
package crawler

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// ============================================================================
// Google Search API Types
// ============================================================================

// GoogleSearchResult represents the response from Google Custom Search API.
type GoogleSearchResult struct {
	Items []GoogleResultItem `json:"items"`
}

// GoogleResultItem holds metadata for a single Google search result entry.
type GoogleResultItem struct {
	Title   string `json:"title"`   // Page title
	Link    string `json:"link"`    // Target URL
	Snippet string `json:"snippet"` // Brief description excerpt
}

// GoogleAPIErrorResponse wraps errors returned by the Google API.
type GoogleAPIErrorResponse struct {
	Error GoogleAPIError `json:"error"`
}

// GoogleAPIError contains structured error information from Google API.
type GoogleAPIError struct {
	Code    int                 `json:"code"`    // HTTP status code
	Message string              `json:"message"` // Human-readable error message
	Errors  []GoogleErrorDetail `json:"errors"`  // Detailed error breakdown
	Status  string              `json:"status"`  // Error status identifier
}

// GoogleErrorDetail provides granular error information for debugging.
type GoogleErrorDetail struct {
	Message string `json:"message"` // Specific error message
	Domain  string `json:"domain"`  // Error domain (e.g., "usageLimits")
	Reason  string `json:"reason"`  // Machine-readable error reason
}

// Deprecated types for backwards compatibility
type SearchResult = GoogleSearchResult
type ResultItem = GoogleResultItem
type APIError = GoogleAPIError
type ErrorDetail = GoogleErrorDetail

// ============================================================================
// Query Types
// ============================================================================

// NormalizedQuery represents a structured geospatial search query with
// parsed components for location, data type, format, and temporal extent.
type NormalizedQuery struct {
	CleanedQuery string   `json:"cleanedQuery"`           // Original search string
	DataEntity   string   `json:"dataEntity,omitempty"`   // Type of data (e.g., "precipitation", "DEM")
	OutputFormat string   `json:"outputFormat,omitempty"` // Desired format (e.g., "GeoTIFF", "Shapefile")
	Location     string   `json:"location,omitempty"`     // Geographic location
	CountryCode  string   `json:"countryCode,omitempty"`  // ISO country code
	StartDate    string   `json:"startDate,omitempty"`    // Temporal extent start (YYYY-MM-DD)
	EndDate      string   `json:"endDate,omitempty"`      // Temporal extent end (YYYY-MM-DD)
	Sources      []string `json:"sources"`                // Preferred data source URLs
}

// GRPCNormalizedQuery is a deprecated alias maintained for backwards compatibility.
// Use NormalizedQuery instead.
type GRPCNormalizedQuery = NormalizedQuery

// ============================================================================
// Crawler Core Types
// ============================================================================

// CrawlNode represents a discovered URL in the crawl tree with its context
// and relevance scoring.
type CrawlNode struct {
	URL              string      // The discovered URL
	Parent           *CrawlNode  // Parent node in the crawl tree (nil for root)
	Depth            int         // Distance from seed URL (0 = seed)
	context          DataContext // Extracted metadata
	CosineSimilarity float64     // Relevance score (0.0 to 1.0)
}

// WebNode is a deprecated alias for CrawlNode maintained for backwards compatibility.
// Use CrawlNode instead in new code.
type WebNode = CrawlNode

// CrawlManager coordinates a single crawl session with concurrency control
// and state management.
type CrawlManager struct {
	// Configuration
	searchQuery NormalizedQuery // The structured query driving this crawl

	// State
	downloadURLs        []CrawlNode            // Accumulated dataset URLs
	CachedURLEmbeddings map[string]DataContext // Embeddings cache (future use)
	searchFrom          map[string]DataContext // Seed URLs with context
	seen                map[string]bool        // Deduplication tracker

	// Concurrency control
	linkChan chan struct{}    // Link discovery signaling
	smTokens chan struct{}    // Semaphore for crawl concurrency (max 40)
	dlTokens chan struct{}    // Semaphore for download concurrency (max 40)
	worklist chan []CrawlNode // Queue of nodes to process
	done     chan bool        // Shutdown signal

	// Optional features
	LlmApiKey string // API key for LLM-based enhancements
}

// Manager is a deprecated alias for CrawlManager maintained for backwards compatibility.
type Manager = CrawlManager

// ============================================================================
// Metadata Types
// ============================================================================

// DataContext holds metadata about a discovered data source including
// optional embedding vectors for similarity search.
type DataContext struct {
	Title       string    // Human-readable title
	Description string    // Detailed description of the dataset
	Embedding   []float64 // Optional embedding vector for semantic search
}

// BoundingBox represents geographic bounds for a dataset using decimal degrees.
type BoundingBox struct {
	EastBC  float64 `json:"eastbc,omitempty"`  // Eastern boundary (longitude)
	WestBC  float64 `json:"westbc,omitempty"`  // Western boundary (longitude)
	NorthBC float64 `json:"northbc,omitempty"` // Northern boundary (latitude)
	SouthBC float64 `json:"southbc,omitempty"` // Southern boundary (latitude)
}

// VerticalMeta represents vertical/elevation metadata for a dataset.
type VerticalMeta struct {
	VerticalCRS string  `json:"verticalcrs,omitempty"` // Vertical coordinate reference system
	AltRes      float64 `json:"altres,omitempty"`      // Altitude resolution
	AltUnits    string  `json:"altunits,omitempty"`    // Altitude units (e.g., "meters", "feet")
}

// HorizontalMeta represents lat/long metadata
// (CRS, resolution)
type HorizontalMeta struct {
	LatRes        float64 `json:"latres,omitempty"`        // Latitude resolution
	LongRes       float64 `json:"longres,omitempty"`       // Longitude resolution
	GeoUnit       string  `json:"geounit,omitempty"`       // Geographic unit (e.g., "degrees", "meters")
	HorizontalCRS string  `json:"horizontalcrs,omitempty"` // Horizontal coordinate reference system
}

// DatasetMetadata represents comprehensive metadata extracted from HTML pages
// about downloadable geospatial files.
type DatasetMetadata struct {
	Title          string         `json:"title,omitempty"`          // Dataset title
	Source         string         `json:"source,omitempty"`         // Source/provider name
	Description    string         `json:"description,omitempty"`    // Long-form description
	Keywords       []string       `json:"keywords,omitempty"`       // Searchable keywords
	URL            string         `json:"url"`                      // Download URL
	Bounds         BoundingBox    `json:"bounds,omitempty"`         // Geographic bounding box
	HorizontalMeta HorizontalMeta `json:"horizontalmeta,omitempty"` //horizontal/lat-long metadata
	VerticalMeta   VerticalMeta   `json:"verticalmeta,omitempty"`   // Vertical/elevation metadata
	StartDate      string         `json:"start_date,omitempty"`     // Temporal coverage start (ISO 8601: YYYY-MM-DD)
	EndDate        string         `json:"end_date,omitempty"`       // Temporal coverage end (ISO 8601: YYYY-MM-DD)
}

// ToString formats the metadata as a human-readable string with newline-separated fields.
func (dm *DatasetMetadata) ToString() string {
	var sb strings.Builder
	sb.WriteString("TITLE: " + dm.Title + "\n")
	if dm.Source != "" {
		sb.WriteString("SOURCE: " + dm.Source + "\n")
	}
	sb.WriteString("DESCRIPTION: " + dm.Description + "\n")
	for _, word := range dm.Keywords {
		sb.WriteString("  KEYWORD: " + word + "\n")
	}
	sb.WriteString("URL: " + dm.URL + "\n")
	if dm.Bounds.EastBC != 0 || dm.Bounds.WestBC != 0 || dm.Bounds.NorthBC != 0 || dm.Bounds.SouthBC != 0 {
		sb.WriteString("BOUNDS: ")
		sb.WriteString(fmt.Sprintf("N:%.4f S:%.4f E:%.4f W:%.4f\n",
			dm.Bounds.NorthBC, dm.Bounds.SouthBC, dm.Bounds.EastBC, dm.Bounds.WestBC))
	}
	if dm.HorizontalMeta.LatRes != 0 || dm.HorizontalMeta.LongRes != 0 || dm.HorizontalMeta.GeoUnit != "" || dm.HorizontalMeta.HorizontalCRS != "" {
		sb.WriteString("HORIZONTAL METADATA:\n")
		if dm.HorizontalMeta.LatRes != 0 || dm.HorizontalMeta.LongRes != 0 {
			sb.WriteString("  RESOLUTION: ")
			sb.WriteString(fmt.Sprintf("Lat:%.6f Long:%.6f", dm.HorizontalMeta.LatRes, dm.HorizontalMeta.LongRes))
			if dm.HorizontalMeta.GeoUnit != "" {
				sb.WriteString(" (" + dm.HorizontalMeta.GeoUnit + ")")
			}
			sb.WriteString("\n")
		}
		if dm.HorizontalMeta.HorizontalCRS != "" {
			sb.WriteString("  CRS: " + dm.HorizontalMeta.HorizontalCRS + "\n")
		}
	}
	if dm.VerticalMeta.VerticalCRS != "" || dm.VerticalMeta.AltRes != 0 || dm.VerticalMeta.AltUnits != "" {
		sb.WriteString("VERTICAL METADATA:\n")
		if dm.VerticalMeta.VerticalCRS != "" {
			sb.WriteString("  CRS: " + dm.VerticalMeta.VerticalCRS + "\n")
		}
		if dm.VerticalMeta.AltRes != 0 {
			sb.WriteString(fmt.Sprintf("  ALTITUDE RESOLUTION: %.6f", dm.VerticalMeta.AltRes))
			if dm.VerticalMeta.AltUnits != "" {
				sb.WriteString(" " + dm.VerticalMeta.AltUnits)
			}
			sb.WriteString("\n")
		}
	}
	if dm.StartDate != "" || dm.EndDate != "" {
		sb.WriteString("TEMPORAL: ")
		if dm.StartDate != "" {
			sb.WriteString(dm.StartDate)
		} else {
			sb.WriteString("N/A")
		}
		sb.WriteString(" to ")
		if dm.EndDate != "" {
			sb.WriteString(dm.EndDate)
		} else {
			sb.WriteString("N/A")
		}
	}
	return sb.String()
}

// downloadMetadata is a deprecated alias for DatasetMetadata.
type downloadMetadata = DatasetMetadata

// ============================================================================
// FTP Directory Types
// ============================================================================

// GeoFile represents a geospatial file discovered during FTP indexing
// with its download URL and optional metadata file.
// DEPRECATED: Use DatasetMetadata instead.
type GeoFile struct {
	URL      string // Download URL for the geospatial file
	Metadata string // Optional URL to metadata/sidecar file
}

// FTPDirectory represents a hierarchical FTP directory structure with
// downloadable geospatial files and their extracted metadata.
type FTPDirectory struct {
	Parent         *FTPDirectory      // Parent directory (nil for root)
	SubDirectories []*FTPDirectory    // Child directories
	Datasets       []DatasetMetadata  // Geospatial datasets with extracted metadata
}

// FTPDir is a deprecated alias for FTPDirectory.
type FTPDir = FTPDirectory

// ============================================================================
// HTML Parsing Types
// ============================================================================

// TableData represents an HTML table extracted from a page with headers
// and data cells for structured metadata extraction.
type TableData struct {
	Root    *html.Node // The <table> element node
	Headers []string   // Column headers from <th> elements
	Data    []string   // Cell values from <td> elements
}

// ============================================================================
// LLM Integration Types
// ============================================================================

// TextPayload is a request payload for batch text processing via embedding APIs.
type TextPayload struct {
	Texts []string `json:"texts"`
}

// EmbeddingResponse contains embedding vectors returned from an embedding service.
type EmbeddingResponse struct {
	Embeddings [][]float64 `json:"embeddings"` // One vector per input text
}

// LLMQuery represents a request to a chat-based language model API.
type LLMQuery struct {
	Model    string       `json:"model"`    // Model identifier (e.g., "llama-3-70b")
	Messages []LLMMessage `json:"messages"` // Conversation history
}

// LLMMessage represents a single message in a chat conversation.
type LLMMessage struct {
	Role    string `json:"role"`    // "user", "assistant", or "system"
	Content string `json:"content"` // Message text
}

// GroqAPIError represents an error returned by the Groq API (OpenAI-compatible format).
type GroqAPIError struct {
	Message string `json:"message"` // Error description
	Type    string `json:"type"`    // Error type (e.g., "invalid_request_error")
	Param   string `json:"param"`   // Parameter that caused the error
	Code    string `json:"code"`    // Machine-readable error code
}

// Error implements the error interface for GroqAPIError.
func (g *GroqAPIError) Error() string {
	return fmt.Sprintf("Groq API Error: %s (type=%s, param=%s, code=%s)",
		g.Message, g.Type, g.Param, g.Code)
}

// GroqAPIResponse represents a complete response from the Groq API.
type GroqAPIResponse struct {
	ID      string `json:"id"`      // Unique response ID
	Object  string `json:"object"`  // Object type (e.g., "chat.completion")
	Created int64  `json:"created"` // Unix timestamp
	Model   string `json:"model"`   // Model used for generation

	Choices []struct {
		Index   int `json:"index"` // Choice index (for n>1)
		Message struct {
			Role    string `json:"role"`    // "assistant"
			Content string `json:"content"` // Generated text
		} `json:"message"`
		FinishReason string `json:"finish_reason"` // "stop", "length", etc.
	} `json:"choices"`

	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`     // Input tokens
		CompletionTokens int `json:"completion_tokens"` // Output tokens
		TotalTokens      int `json:"total_tokens"`      // Sum
	} `json:"usage,omitempty"`

	APIError *GroqAPIError `json:"error,omitempty"` // Present on error responses
}

// GROQAPIError is a deprecated alias for GroqAPIError.
type GROQAPIError = GroqAPIError

// GroqApiResp is a deprecated alias for GroqAPIResponse.
type GroqApiResp = GroqAPIResponse

// ============================================================================
// Utility Functions
// ============================================================================

// SlicesEqualUnordered compares two string slices regardless of element order.
// Returns true when both slices contain the same elements with the same multiplicities.
//
// Example:
//
//	SlicesEqualUnordered([]string{"a", "b", "c"}, []string{"c", "a", "b"}) // true
//	SlicesEqualUnordered([]string{"a", "a", "b"}, []string{"a", "b", "b"}) // false
func SlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	freq := make(map[string]int, len(a))
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
