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
	var normedQuery crawler.GRPCNormalizedQuery
	var respString string
	cleanedQuery, err := getAttr(r, "cleandquery")
	dataEntity, err := getAttr(r, "de")
	if err != nil {
		respString = "incomplete request!"
	}
	location, err := getAttr(r, "loc")
	startDate, err := getAttr(r, "start")
	endDate, err := getAttr(r, "end")
	countryCode, err := getAttr(r, "cc")
	if err != nil {
		respString = "empty request!"
	}
	respString = "ALL GOOD!"
	normedQuery.CleanedQuery = cleanedQuery
	normedQuery.DataEntity = dataEntity
	normedQuery.Location = location
	normedQuery.StartDate, normedQuery.EndDate = startDate, endDate
	normedQuery.CountryCode = countryCode
	//run crawler, return error if failed to write to S3
	w.Header().Set("Content-type", "application/json")

	if err := crawler.Run(normedQuery); err != nil {
		respString = err.Error()
	}

	json.NewEncoder(w).Encode(respString)

}

func main() {
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
