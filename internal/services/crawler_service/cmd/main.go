package main

import (
	"context"
	"crawler_service/internal/crawler"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type OkayResponse struct {
	Message string `json:"message"`
	Time    string `json:"time"`
}

type normalizedQueriesRequest struct {
	NormalizedQueries []normalizedQuery `json:"normalizedQueries"`
}

type normalizedQuery struct {
	CleanedQuery string   `json:"CleanedQuery"`
	DataEntity   string   `json:"DataEntity"`
	OutputFormat string   `json:"OutputFormat"`
	Location     string   `json:"Location"`
	CountryCode  string   `json:"CountryCode"`
	StartDate    string   `json:"StartDate"`
	EndDate      string   `json:"EndDate"`
	Sources      []string `json:"Sources"`
}

func TestActive(w http.ResponseWriter, r *http.Request) {
	resp := OkayResponse{
		Message: "go_crawler endpoint active!",
		Time:    time.Now().String(),
	}
	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

//override: Assume python service returns object of type QueryResponse
// type QueryResponse struct {
// 	NormalizedQueries []*struct {
// 		CleanedQuery string
// 		DataEntity   string `json:"dataEntity,omitempty"`
// 		OutputFormat string `json:"outputFormat,omitempty"`
// 		Location     string `json:"location,omitempty"`
// 		StartDate    string `json:"startDate,omitempty"`
// 		EndDate      string `json:"endDate,omitempty"`
// 	}
// 	Sources []string // normalized URLs
// }

func getAttr(r *http.Request, attrName string) (string, error) {
	resp := r.URL.Query().Get(attrName)
	if resp == "" {
		return "", errors.New("param was empty")
	}
	return resp, nil
}

func StartCrawl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")

	var payload normalizedQueriesRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	if len(payload.NormalizedQueries) == 0 {
		http.Error(w, "normalizedQueries is required", http.StatusBadRequest)
		return
	}

	type crawlResult struct {
		Query  string `json:"query"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}
	var results []crawlResult

	for _, q := range payload.NormalizedQueries {
		normedQuery := crawler.NormalizedQuery{
			CleanedQuery: q.CleanedQuery,
			DataEntity:   q.DataEntity,
			OutputFormat: q.OutputFormat,
			Location:     q.Location,
			CountryCode:  q.CountryCode,
			StartDate:    q.StartDate,
			EndDate:      q.EndDate,
			Sources:      q.Sources,
		}

		if err := crawler.Run(normedQuery); err != nil {
			results = append(results, crawlResult{
				Query:  q.CleanedQuery,
				Status: "error",
				Error:  err.Error(),
			})
			continue
		}
		results = append(results, crawlResult{
			Query:  q.CleanedQuery,
			Status: "ok",
		})
	}

	json.NewEncoder(w).Encode(map[string]any{
		"count":   len(results),
		"results": results,
	})

}

func main() {
	// Initialize database connection
	if err := crawler.InitDB(); err != nil {
		log.Printf("Warning: Failed to initialize database: %v", err)
		log.Println("Crawler will continue but won't persist datasets")
	}
	defer crawler.CloseDB()

	addr := ":8080"
	mux := http.NewServeMux()
	mux.HandleFunc("/test", TestActive)
	mux.HandleFunc("/crawl", StartCrawl)
	mux.HandleFunc("/healthz", Health)
	mux.HandleFunc("/ready", Health)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("Starting crawler server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("crawler server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down crawler server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("crawler server shutdown error: %v", err)
	}
	log.Println("Crawler server stopped.")

}
