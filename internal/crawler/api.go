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

func WriteToLog(filepath string) (*os.File, error) {
	logFile, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening log file: %w", err)
	}
	log.SetOutput(logFile)
	return logFile, nil
}

func WriteToGob(filepath string, data interface{}) error {
	file, err := os.Open(filepath)
	if err != nil {
		log.Println(".gob file specified doesn't exist, creating. Error: ", err)
		file, err = os.Create(filepath)
		if err != nil {
			log.Println("	An error occured while creating .gob file: ", err)
			return err
		}

	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		log.Println("An error occured while writing data to .gob file: ", err)
		return err
	}
	return nil

}

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

func (m *Manager) Close(newURLs []WebNode) {
	//	2. Write newly-scraped URL(s) to .gob file?
	var wg sync.WaitGroup

	//iterate with a channel, and add links to chan
	unSeen := make(chan WebNode, 500)
	for _, node := range newURLs {
		wg.Add(1)
		go func(newNode WebNode) {
			for cachedURL, _ := range m.CachedURLEmbeddings {
				if cachedURL == newNode.Url {
					return
				}
			}
			unSeen <- newNode
			wg.Done()
		}(node)
	}
	close(unSeen)
	for node := range unSeen {
		m.CachedURLEmbeddings[node.Url] = node.context
	}
	wg.Wait()
	//write to .gob file:
	WriteToGob(dataPath, m.CachedURLEmbeddings)
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
