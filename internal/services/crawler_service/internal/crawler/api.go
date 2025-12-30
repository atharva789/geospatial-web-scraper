package crawler

import (
	"fmt"
	"strings"
)

// Run executes a complete crawl session for the given normalized query.
//
// The function orchestrates the following workflow:
//  1. Validates the search query parameters
//  2. Initializes a CrawlManager with concurrency controls
//  3. Discovers seed URLs and performs Google Search
//  4. Recursively crawls discovered pages up to maximum depth
//  5. Extracts geospatial dataset URLs with metadata
//  6. Persists discovered datasets to PostgreSQL database
//
// Parameters:
//   - query: A NormalizedQuery containing search terms, location, data type, etc.
//
// Returns:
//   - error: nil on success, or an error describing what went wrong
//
// Example usage:
//
//	query := NormalizedQuery{
//	    CleanedQuery: "precipitation California 2020",
//	    DataEntity:   "precipitation",
//	    Location:     "California",
//	    StartDate:    "2020-01-01",
//	    EndDate:      "2020-12-31",
//	}
//	if err := Run(query); err != nil {
//	    log.Fatalf("Crawl failed: %v", err)
//	}
func Run(query NormalizedQuery) error {
	// Validate that we have sufficient search criteria
	if strings.TrimSpace(query.CleanedQuery) == "" && query.DataEntity == "" {
		return fmt.Errorf("search query validation failed: either CleanedQuery or DataEntity must be provided")
	}

	// Initialize the crawl manager with default configuration
	manager := CrawlManager{
		searchQuery:  query,
		downloadURLs: []CrawlNode{},
		searchFrom:   PublicGeospatialDataSeeds, // Seed URLs from data.go
		linkChan:     make(chan struct{}, 1),
		smTokens:     make(chan struct{}, 40), // Limit to 40 concurrent crawls
		dlTokens:     make(chan struct{}, 40), // Limit to 40 concurrent downloads
		worklist:     make(chan []CrawlNode),
		done:         make(chan bool),
		seen:         make(map[string]bool),
	}

	fmt.Printf("Starting crawl session for query: \"%s\"\n", query.CleanedQuery)

	// Execute the breadth-first crawl
	discoveredDatasets := manager.ScheduleCrawl()

	fmt.Printf("Crawl completed for query: \"%s\"\n", query.CleanedQuery)
	fmt.Printf("  Found %d dataset URLs\n", len(discoveredDatasets))

	// Persist all discovered datasets to the database
	if err := SaveDatasets(discoveredDatasets, query); err != nil {
		fmt.Printf("ERROR: Failed to save datasets to database: %v\n", err)
		return fmt.Errorf("database persistence failed: %w", err)
	}

	// Log discovered URLs for debugging
	if len(discoveredDatasets) > 0 {
		fmt.Println("  Discovered URLs:")
		for i, node := range discoveredDatasets {
			fmt.Printf("    [%d] %s\n", i+1, node.URL)
			if i >= 9 && len(discoveredDatasets) > 10 {
				fmt.Printf("    ... and %d more\n", len(discoveredDatasets)-10)
				break
			}
		}
	}

	return nil
}
