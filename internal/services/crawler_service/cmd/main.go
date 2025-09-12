package main

import (
	"crawler_service/internal/crawler"
	"encoding/json"
	"errors"
	"log"
	"net/http"
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
	cleanedQuery, err := getAttr(r, "cleandquery")
	dataEntity, err := getAttr(r, "de")
	location, err := getAttr(r, "loc")
	startDate, err := getAttr(r, "start")
	endDate, err := getAttr(r, "end")
	countryCode, err := getAttr(r, "cc")
	var respString string
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
	mux := http.NewServeMux()
	mux.HandleFunc("/test", TestActive)
	mux.HandleFunc("/crawl", StartCrawl)
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}

}
