package crawler

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var dataPath = "/Users/thorbthorb/Downloads/geospatial-web-scraper/data.gob"
var findLinksLogPath = "/Users/thorbthorb/Downloads/geospatial-web-scraper/logs/findLinks.log"

// GetBatchedEmbeddings sends a slice of strings to the local embedding service
// and returns the resulting embeddings. The function performs a single HTTP
// POST request with the provided texts and decodes the JSON response.
func GetBatchedEmbeddings(texts []string) (EmbeddingResponse, error) {
	var buf bytes.Buffer
	newPayload := TextPayload{Texts: texts}
	if err := json.NewEncoder(&buf).Encode(newPayload); err != nil {
		log.Printf("	Error occured while encoding data JSON payload: %v", err)
		return EmbeddingResponse{}, err
	}

	resp, err := http.Post(
		"http://localhost:8000/embed",
		"application/json",
		&buf,
	)
	if err != nil {
		log.Printf("	error while sending embedding request for data: %v", err)
		return EmbeddingResponse{}, err
	}
	defer resp.Body.Close()
	var res EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Fatalf("	error returned from vectorization endpoint while embedding search-query: %v", err)
	}
	return res, nil
}

// WriteToLog opens or creates the specified log file and sets the logger output
// to this file. It returns the opened *os.File so the caller can close it when
// finished.
func WriteToLog(filepath string) (*os.File, error) {
	logFile, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening log file: %w", err)
	}
	log.SetOutput(logFile)
	return logFile, nil
}

// WriteToGob serializes the provided data value to a gob file at the given
// path. Existing data is appended to, allowing the cache to persist between
// runs.
func WriteToGob(filepath string, data interface{}) error {
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		log.Println("An error occured while writing data to .gob file: ", err)
		return err
	}
	return nil

}

// GenerateEmbeddings reads all seed descriptions, sends them for embedding, and
// returns the embeddings in the same order as the seeds.
func GenerateEmbeddings() ([][]float64, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var texts []string

	for link, dataContext := range PublicGeospatialDataSeeds {
		wg.Add(1)
		go func(link string, ctx DataContext) {
			defer wg.Done()
			mu.Lock()
			texts = append(texts, ctx.Description)
			mu.Unlock()
		}(link, dataContext)
	}

	wg.Wait()
	// texts is safe to use here
	var buf bytes.Buffer
	payload := TextPayload{Texts: texts}
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return nil, err
	}

	resp, err := http.Post(
		"http://localhost:8000/embed",
		"application/json",
		&buf,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var res EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Embeddings, nil
}

// Init prepares the Manager by loading or creating the embedding cache stored
// on disk. If no cache exists, all seed URLs are embedded and written to the
// gob file. Loaded or generated embeddings are stored in
// m.CachedURLEmbeddings.
func (m *Manager) Init() {
	data := make(map[string]DataContext)
	//data is a map of URL : embedding
	// m.CachedURLEmbeddings = data

	//data SHOULD BE a map of URL: DataContext{Description, Embedding}

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		//embed every link in PublicGeospatialDataSeeds,
		//then write to .gob file
		embeddings, err := GenerateEmbeddings()
		if err != nil {
			log.Println("Error occured while embedding PublicGeospatialDataSeeds data:", err)
			return
		}
		var counter = 0
		for url, ctx := range PublicGeospatialDataSeeds {
			data[url] = DataContext{
				Description: ctx.Description,
				Embedding:   embeddings[counter],
			}
			counter++
		}
		WriteToGob(dataPath, data)
		m.CachedURLEmbeddings = data
		return
	}
	//read searchFrom .gob file
	file, err := os.Open(dataPath)
	if err != nil {
		log.Fatalf("An error occured while reading the .gob file at %s: %v", dataPath, err)
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		log.Fatalf("An error occured while decoding .gob file to cache: %v", err)
	}
	m.CachedURLEmbeddings = data
	log.Println("Cached URL-embeddings loaded")
}

// Close stores any newly discovered URLs and persists the cache.
//
//  1. Each producer goroutine decides whether a URL is new.
//  2. All brand-new URLs go down a channel to a single consumer.
//  3. The consumer batches descriptions (≤ batchSize) and calls
//     GetBatchedEmbeddings once per batch.
//  4. It writes the finished embeddings into m.CachedURLEmbeddings
//     under a mutex so there are no data races.
//  5. When all producers are done the channel is closed, any
//     leftover batch is flushed, and the whole cache is written to
//     data.gob.
func (m *Manager) Close(newURLs []WebNode) {
	const batchSize = 50

	embedCh := make(chan WebNode, batchSize)
	var (
		wgProducers sync.WaitGroup
		mu          sync.Mutex // protects m.CachedURLEmbeddings
	)

	//------------------------------------------------------------------
	// 1. CONSUMER – runs once, processes batches from embedCh
	//------------------------------------------------------------------
	go func() {
		var (
			nodes []WebNode // URLs waiting for an embedding
			descs []string  // matching descriptions
		)

		flush := func() {
			if len(nodes) == 0 {
				return
			}
			log.Printf("embedding %d new items...", len(nodes))

			emb, err := GetBatchedEmbeddings(descs)
			if err != nil {
				log.Printf("embedding batch failed: %v", err)
				return
			}

			mu.Lock()
			for i, n := range nodes {
				m.CachedURLEmbeddings[n.Url] = DataContext{
					Description: n.context.Description,
					Embedding:   emb.Embeddings[i],
				}
			}
			mu.Unlock()

			// reuse underlying arrays – zero-cost reset
			nodes = nodes[:0]
			descs = descs[:0]
		}

		for n := range embedCh {
			nodes = append(nodes, n)
			descs = append(descs, n.context.Description)
			if len(nodes) == batchSize {
				flush()
			}
		}
		flush() // flush any tail batch when channel closes
	}()

	//------------------------------------------------------------------
	// 2. PRODUCERS – one lightweight goroutine per candidate URL
	//------------------------------------------------------------------
	for _, node := range newURLs {
		wgProducers.Add(1)
		go func(n WebNode) {
			defer wgProducers.Done()

			mu.Lock()
			_, seen := m.CachedURLEmbeddings[n.Url]
			mu.Unlock()
			if seen {
				return
			}

			embedCh <- n // send only if it’s truly new
		}(node)
	}

	//------------------------------------------------------------------
	// 3. SHUTDOWN & PERSIST
	//------------------------------------------------------------------
	wgProducers.Wait() // wait until all sends are finished
	close(embedCh)     // tells consumer to finish

	// By now the consumer must have flushed everything and exited.
	// Persist the whole cache.
	if err := WriteToGob(dataPath, m.CachedURLEmbeddings); err != nil {
		log.Printf("failed to write cache to %s: %v", dataPath, err)
	}
	for _, context := range m.CachedURLEmbeddings {
		fmt.Println(" extract description: ", context.Description)
	}
}

// Run executes the CLI application.
func Run() {
	logFile, logErr := WriteToLog(findLinksLogPath)
	if logErr != nil {
		log.Fatalf("An error occured while making log-file: %v", logErr)
	}
	defer logFile.Close()
	// Flags
	searchPtr := flag.String("s", "", "Search query for dataset. Required.")
	downloadDir := flag.String("download", "", "Directory to download datasets to. If empty, only prints URLs.")
	noSec := flag.Bool("nosec", false, "Disable security sandboxing (enabled by default).")

	flag.Parse()

	// Validate search query
	if strings.TrimSpace(*searchPtr) == "" {
		fmt.Println("ERROR: Search query (-s) is required.")
		flag.Usage()
		os.Exit(1)
	}

	mg := Manager{
		secure:       *noSec,
		downloadPath: downloadDir,
		searchQuery:  searchPtr,
		downloadURLs: []WebNode{},
		searchFrom:   PublicGeospatialDataSeeds,
		linkChan:     make(chan struct{}, 1),
		smTokens:     make(chan struct{}, 40),
		dlTokens:     make(chan struct{}, 40),
		worklist:     make(chan []WebNode),
		done:         make(chan bool),
		seen:         make(map[string]bool),
	}
	mg.Init()
	// Begin search
	var downloadableLinks []WebNode
	fmt.Printf("Searching for: \"%s\"\n", *searchPtr)

	if *downloadDir != "" {
		if _, err := os.Stat(*downloadDir); os.IsNotExist(err) {
			if err := os.MkdirAll(*downloadDir, 0755); err != nil {
				log.Fatalf("Failed to create directory %s: %v", *downloadDir, err)
			}
		}
	}

	downloadableLinks = mg.FindLinks()
	log.Printf("For searchQuery '%v'", *searchPtr)
	log.Printf("	found %v URLs:", len(downloadableLinks))

	for _, node := range downloadableLinks {
		log.Println("		URL: ", node.Url)
	}
	mg.Close(downloadableLinks)
	return

}
