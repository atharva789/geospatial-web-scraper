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

var dataPath = "/crawler/data.gob"

func GenerateEmbeddings() ([][]float64, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var texts []string

	for link, dataContext := range PublicGeospatialDataSeeds {
		wg.Add(1)
		go func(link string, ctx DataContext) {
			defer wg.Done()
			mu.Lock()
			texts = append(texts, ctx.description)
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
		"http://localhost:8080/embed",
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

func (m *Manager) Init() {
	data := make(map[string][]float64)

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		//embed every link in PublicGeospatialDataSeeds,
		//then write to .gob file
		embeddings, err := GenerateEmbeddings()
		if err != nil {
			log.Println("Error occured while embedding PublicGeospatialDataSeeds data:", err)
			return
		}
		var counter = 0
		for key, _ := range PublicGeospatialDataSeeds {
			data[key] = embeddings[counter]
			counter++
		}
		file, err := os.Create(dataPath)
		if err != nil {
			log.Fatalf("Failed to create directory %s: %v", dataPath, err)
		}
		defer file.Close()
		//write .gob file
		encoder := gob.NewEncoder(file)
		if err := encoder.Encode(data); err != nil {
			log.Fatalf("Failed to write data to .gob file: %v", err)
		}
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

func (m *Manager) Close(newURLs []WebNode) {
	//	2. Write newly-scraped URL(s) to .gob file?
	var wg sync.WaitGroup
	var mu sync.Mutex
	var unCachedURLs []string
	for _, checkURL := range newURLs {
		wg.Add(1)
		go func(unCachedURL string) {
			seen := false
			for cachedURL, _ := range m.CachedURLEmbeddings {
				if cachedURL == unCachedURL {
					seen = true
				}
			}
			if seen == false {
				mu.Lock()
				unCachedURLs = append(unCachedURLs, unCachedURL)
				mu.Unlock()
			}
			wg.Done()
		}(checkURL.Url)
	}
	wg.Wait()
	//write to .gob file:
	// 1. Find new URLs --> 2. find metadata/description --> get embedding
	// 4. write to file

	//read searchFrom .gob file
	file, err := os.Open(dataPath)
	if err != nil {
		log.Printf("error occured while reading the .gob file, re-writing: %v: %v", dataPath, err)

	}
	defer file.Close()

}

// Run executes the CLI application.
func Run() {
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
	if *downloadDir == "" {
		fmt.Println("Found URLs:")
		for _, node := range downloadableLinks {
			fmt.Println(node.Url)
		}
		return
	}

}
