package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	normalizerpb "crawler_service/internal/crawler/querynormalizer"

	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
)

func getenv(envVar, replacement string) string {
	key := os.Getenv(envVar)
	if key == "" {
		return replacement
	}
	return key
}

// only writing links found with kafka to localhost for now
func WriteToLog(k *kafka.Writer, msg string) error {
	// streams to kafka
	var logEvent struct {
		Message string
		Time    string
	}

	logEvent.Message, logEvent.Time = msg, time.Now().String()
	payload, err := json.Marshal(logEvent)
	if err != nil {
		panic(err)
	}

	topic := "log.event"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = k.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(topic),
			Value: payload,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// Run executes the CLI application.
func Run(query GRPCNormalizedQuery) error {

	// Validate search query
	if strings.TrimSpace(query.CleanedQuery) == "" && query.DataEntity == "" {
		fmt.Println("ERROR: Search query (-s) is required.")
		return fmt.Errorf("no search-query provided!")
	}

	var normedQuery GRPCNormalizedQuery

	if query.DataEntity == "" || query.Location == "" {
		// start new gRPC session
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, "localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			fmt.Println("Error starting metadata gRPC service, exiting")
		}
		defer conn.Close()
		request := normalizerpb.QueryRequest{SearchQuery: query.CleanedQuery}
		client := normalizerpb.NewNormalizerServiceClient(conn)
		normalized_queries, err := client.GetNormalizedQuery(ctx, &request)
		if err != nil {
			fmt.Println("Error recieved while normalizing request. %s", err)
			panic(err)
		}
		fmt.Println("normalized_queries: ", normalized_queries)
		query := normalized_queries.NormalizedQuery[0]
		normedQuery.DataEntity, normedQuery.StartDate, normedQuery.EndDate, normedQuery.CountryCode = query.DataEntity, query.StartDate, query.EndDate, query.CountryCode
	}
	normedQuery = query

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

	// gRPC returns 3 NormalizedQuery objects: one with specific location (if specified), the other with the country location

	broker := getenv("BROKERS", "localhost:9092")
	topic := getenv("TOPIC", "dataset-event")

	// Writer is safe for concurrent use; reuse it for performance
	w := &kafka.Writer{
		Addr:                   kafka.TCP(broker),
		Topic:                  topic,
		Balancer:               &kafka.Hash{}, // same Key -> same partition
		BatchTimeout:           10 * time.Millisecond,
		BatchBytes:             64 << 10, // 64KB
		AllowAutoTopicCreation: true,     // fine for local dev
		RequiredAcks:           kafka.RequireAll,
	}
	defer func() {
		if err := w.Close(); err != nil {
			fmt.Printf("writer close: %v", err)
		}
	}()

	mg := Manager{
		searchQuery:  normedQuery,
		downloadURLs: []WebNode{},
		searchFrom:   PublicGeospatialDataSeeds,
		linkChan:     make(chan struct{}, 1),
		smTokens:     make(chan struct{}, 40),
		dlTokens:     make(chan struct{}, 40),
		worklist:     make(chan []WebNode),
		done:         make(chan bool),
		seen:         make(map[string]bool),
	}

	var downloadableLinks []WebNode
	fmt.Printf("Searching for: \"%s\"\n", query)

	downloadableLinks = mg.ScheduleCrawl()
	WriteToLog(mg.kWriter, fmt.Sprintf("For searchQuery '%v'", query))
	WriteToLog(mg.kWriter, fmt.Sprintf("	found %v URLs:", len(downloadableLinks)))

	for _, node := range downloadableLinks {
		WriteToLog(mg.kWriter, fmt.Sprint("		URL: ", node.Url))
	}
	return nil
}
