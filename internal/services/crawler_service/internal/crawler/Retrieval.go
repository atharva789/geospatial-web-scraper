package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// func (m *Manager) GetClosestSeedsFromDB() [][]WebNode {
// 	//finding relevant seeds
// 	//1. embed search query
// 	// 	search query comprises a 'DATA_ENTITY', 'LOCATION', 'OUTPUT_FROMAT', 'TIME_RANGE'. Average
// 	//	each tag.
// 	// 	importance of query structure is as follows:
// 	//	DATA_ENTITY == LOCATION ≥ TIME_RANGE > OUTPUT_FROMAT
// 	// every crawled page needs to be stored like this:
// 	// 			watershedPage = {
// 	//				URL = "usds.gov/opendata/watersheds/",
// 	//				structure = {
// 	//					DATA_ENTITY=["watershed", "HUC6", "HUC8"],
// 	//					TAGS=["HUC8", "HUC6", "Riverbasin", …, "HUC12"],
// 	//					LOCATION="USA",
// 	//					TIME_RANGE=[1990, 1991, … , 2025]
// 	//				}
// 	//		}
// 	var buf bytes.Buffer
// 	// make 3 parallel requests? Data Entity + location + time range
// 	queries := ConstructOptimalQueries(m.searchQuery.NormalizedQuery[0])
// 	newPayload := TextPayload{Texts: queries}
// 	if err := json.NewEncoder(&buf).Encode(newPayload); err != nil {
// 		log.Fatalf("Error occured while encoding search-query JSON payload: %v", err)
// 	}

// 	resp, err := http.Post(
// 		"http://localhost:8000/embed",
// 		"application/json",
// 		&buf,
// 	)
// 	if err != nil {
// 		log.Fatalf("error while sending embedding request for search-query: %v", err)
// 	}
// 	defer resp.Body.Close()
// 	var res EmbeddingResponse
// 	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
// 		log.Fatalf("error returned from vectorization endpoint while embedding search-query: %v", err)
// 	}

// 	// Embeddings are in multiple-query-order
// 	queryEmbeddings := res.Embeddings
// 	var relevantURLs []WebNode
// 	//2. compare with cached URL-embeddings

// 	//create cosine-similarity and sort for top 10

// 	var wg sync.WaitGroup
// 	var mu sync.Mutex
// 	var JobQueues [][]WebNode
// 	for _, queryEmbedding := range queryEmbeddings {

// 		for url, ctx := range m.CachedURLEmbeddings {
// 			wg.Add(1)
// 			go func(context DataContext, url string) {
// 				score, err := Cosine(queryEmbedding, context.Embedding)
// 				if err != nil {
// 					log.Fatalf("Error while computing cosine similarity: %v", err)
// 				}
// 				mu.Lock()
// 				relevantURLs = append(relevantURLs, WebNode{URL: url, Parent: nil, Depth: 0, context: context, CosineSimilarity: score})
// 				mu.Unlock()
// 				wg.Done()
// 			}(ctx, url)
// 		}
// 		wg.Wait()

// 		length := len(relevantURLs) - 1
// 		minusTen := length - 10
// 		//sort, top-10
// 		relevantURLs = MergeSort(&relevantURLs, 0, length)
// 		//3. chose top 5 seeds using cosine similarity
// 		JobQueues = append(JobQueues, relevantURLs[minusTen:length])
// 		//relevant seeds have been found
// 	}

// 	for _, queue := range JobQueues {
// 		fmt.Println("Number of relevant URLs: ", len(queue))
// 		for _, node := range queue {
// 			fmt.Println("	closest-match URL: ", node.Url, node.context.Description)
// 		}
// 	}

// 	return JobQueues
// }

func GetGoogleSearchAPIKey() string {

	key := os.Getenv("GOOGLE_SEARCH_API_KEY")
	if key == "" {
		fmt.Print("Google API Key not found in .env")
	}
	return key
}

func GetCX() string {
	key := os.Getenv("GOOGLE_CX")
	if key == "" {
		fmt.Print("Google CX Key not found in .env")
	}
	return key
}

const googleCustomSearchEndpoint = "https://www.googleapis.com/customsearch/v1"

// s *QueryResponse should be as follows:
type QueryResponse struct {
	NormalizedQueries []*struct {
		CleanedQuery string
		DataEntity   string `json:"dataEntity,omitempty"`
		OutputFormat string `json:"outputFormat,omitempty"`
		Location     string `json:"location,omitempty"`
		StartDate    string `json:"startDate,omitempty"`
		EndDate      string `json:"endDate,omitempty"`
	}
	Sources []string // normalized URLs
}

func GoogleSearch(q *GRPCNormalizedQuery) ([]WebNode, error) {
	var nodes []WebNode

	if q.CleanedQuery == "" {
		return nodes, fmt.Errorf("no query error (GoogleSearch func)")
	}
	baseURl, err := url.Parse(googleCustomSearchEndpoint)
	if err != nil {
		return nodes, fmt.Errorf("error while parsing google-api-endpoint: %v", err)
	}
	params := baseURl.Query()
	params.Add("key", GetGoogleSearchAPIKey())
	params.Add("cx", GetCX())
	params.Add("q", q.CleanedQuery+fmt.Sprintf(", Location: %s", q.Location))
	params.Add("gl", q.CountryCode)
	for _, source := range q.Sources {
		params.Add("siteSearch", source)
		// params.Add("start", fmt.Sprint(start))
		baseURl.RawQuery = params.Encode()
		finalURL := baseURl.String()

		resp, err := http.Get(finalURL)
		if err != nil {
			return nodes, fmt.Errorf("error recieved making GET request to parsed & cleaned URL: %v", err)
		}

		defer resp.Body.Close()

		// read body bytes

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nodes, fmt.Errorf("error copying body bytes into 'bodybytes': %s", err)
		}

		if resp.StatusCode != 200 {
			return nodes, fmt.Errorf("bad Response: %v", resp.StatusCode)
		}
		var errorResponse GoogleAPIErrorResponse
		if err := json.Unmarshal(bodyBytes, &errorResponse); err != nil {
			// If we can't decode the JSON, just return the status code
			return nodes, fmt.Errorf("bad response status: %s", resp.Status)
		}

		fmt.Printf("API Error Response: %+v\n", errorResponse)
		if errorResponse.Error.Code != 0 {
			return nodes, fmt.Errorf("bad response status: %s", errorResponse.Error.Message)

		}

		var searchResult SearchResult
		if err := json.Unmarshal(bodyBytes, &searchResult); err != nil {
			return nodes, fmt.Errorf("error decoding successful response: %w", err)
		}

		// iterate, parse searchResult into node
		for _, item := range searchResult.Items {
			nodes = append(nodes, WebNode{URL: item.Link, context: DataContext{Title: item.Title, Description: item.Snippet}, Depth: 0, Parent: nil})
		}
	}

	return nodes, nil
}

// implements searching from common-crawl
func GetCommoncrawl() {}
