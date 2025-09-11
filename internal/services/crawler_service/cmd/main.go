package main

import (
	"crawler_service/internal/crawler"
	"encoding/json"
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

func getAttr(r *http.Request, attrName string) string {
	return r.URL.Query().Get(attrName)
}

func StartCrawl(w *http.ResponseWriter, r *http.Request) {
	var normedQuery crawler.GRPCNormalizedQuery
	normedQuery.CleanedQuery = getAttr(r, "cleandquery")
	normedQuery.DataEntity = getAttr(r, "de")
	normedQuery.Location = getAttr(r, "loc")
	normedQuery.StartDate = getAttr(r, "start")
	normedQuery.EndDate = getAttr(r, "end")
	normedQuery.CountryCode = getAttr(r, "cc")
	//run crawler, return error if failed to write to S3
	crawler.Run(normedQuery)

}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", TestActive)
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
