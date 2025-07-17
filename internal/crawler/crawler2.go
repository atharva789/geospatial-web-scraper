package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/html"
)

func (m *Manager) FindLinks() []WebNode {
	log.Println("------------------------------------------------------------------------------")
	log.Println("							STARTED NEW CRAWL SESSION")
	log.Println("------------------------------------------------------------------------------")
	//finding relevant seeds
	//1. embed search query
	var buf bytes.Buffer
	newPayload := TextPayload{Texts: []string{*m.searchQuery}}
	if err := json.NewEncoder(&buf).Encode(newPayload); err != nil {
		log.Fatalf("Error occured while encoing search-query JSON payload: %v", err)
	}

	resp, err := http.Post(
		"http://localhost:8080/embed",
		"application/json",
		&buf,
	)
	if err != nil {
		log.Fatalf("error while sending embedding request for search-query: %v", err)
	}
	defer resp.Body.Close()
	var res EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Fatalf("error returned from vectorization endpoint while embedding search-query: %v", err)
	}

	queryEmbedding := res.Embeddings[0]
	var relevantURLs []WebNode
	//2. compare with cached URL-embeddings

	//create cosine-similarity and sort for top 10

	var wg sync.WaitGroup
	var mu sync.Mutex
	for url, ctx := range m.CachedURLEmbeddings {
		wg.Add(1)
		go func(context DataContext, url string) {
			score, err := Cosine(queryEmbedding, context.Embedding)
			if err != nil {
				log.Fatalf("Error while computing cosine similarity: %v", err)
			}
			mu.Lock()
			relevantURLs = append(relevantURLs, WebNode{Url: url, Parent: nil, Depth: 0, context: context, CosineSimilarity: score})
			mu.Unlock()
			wg.Done()
		}(ctx, url)
	}
	wg.Wait()

	length := len(relevantURLs) - 1
	minusTen := length - 10
	//sort, top-10
	relevantURLs = MergeSort(&relevantURLs, 0, length)
	//3. chose top 5 seeds using cosine similarity
	JobQueue := relevantURLs[minusTen:length]
	//relevant seeds have been found
	fmt.Println("Number of relevant URLs: ", len(relevantURLs))
	for _, node := range relevantURLs {
		fmt.Println("	closest-match URL: ", node.Url, node.context.Description)
	}

	//Crawling begins
	go func() {
		m.worklist <- JobQueue
	}()

	n := 1
	count := 0
	maxCrawl := 600
	for ; n > 0; n-- {
		list := <-m.worklist
		for _, node := range list {
			if count > maxCrawl {
				go func() { m.done <- true }()
			} else {
				go func() { m.done <- false }()
				if !m.seen[node.Url] {
					count++
					n++
					m.seen[node.Url] = true
					stop := <-m.done
					if !stop {
						go func(node WebNode) {
							res := m.Crawl2(&node)
							m.worklist <- res
						}(node)
					}
				}
			}

		}
	}
	log.Println("------------------------------------------------------------------------------")
	log.Printf("					Done! scraped %d URLs ", len(m.downloadURLs))
	log.Println("------------------------------------------------------------------------------")
	return m.downloadURLs

}

func (m *Manager) ToLinks() []string {
	var links []string
	for _, node := range m.downloadURLs {
		links = append(links, node.Url)
	}
	return links
}

func (m *Manager) Crawl2(node *WebNode) []WebNode {
	m.smTokens <- struct{}{}
	links, err := m.Extract2(node)
	<-m.smTokens
	if err != nil {
		log.Printf("Error occured while crawling %v: %v", node.Url, err)
	}

	return links
}

func (m *Manager) Extract2(node *WebNode) ([]WebNode, error) {
	var links []WebNode

	resp, err := http.Get(node.Url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("getting %s: %s", node.Url, resp.Status)
	}
	downloadable := ValidateDownloadable(resp, node.Url)
	if downloadable {
		m.linkChan <- struct{}{} //replace with mu.Lock()
		links = append(links, WebNode{Url: node.Url})
		<-m.linkChan //replace with mu.UnLock()
		if *m.downloadPath != "" {
			go DownloadBuffered(resp, node.Url, m.downloadPath)
		}
		return nil, nil
	}

	doc, err := html.Parse(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %v", node.Url, err)
	}

	VisitNode(doc, &m.downloadURLs, resp, node, doc)

	return links, nil
}

func (m *Manager) DownloadBuffered(resp *http.Response, rawURL string) {
	if m.secure {
		m.dlTokens <- struct{}{}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close() // safe to close now
		if err != nil {
			log.Printf("Failed to buffer body for download: %v", err)
		}
		// cmd := exec.Command(
		// 	"firejail",
		// 	"--private="+*m.downloadPath,
		// 	"--net=none",
		// 	"--caps.drop=all",
		// 	"--seccomp",
		// 	"--shell=none",
		// 	"--quiet",
		// 	fmt.Sprintf("downloader -u=%s -b=%s -d=%s", rawURL, data, *m.downloadPath),
		// )
		// cmd.Run()

		Download(rawURL, data, m.downloadPath)
		<-m.dlTokens
	}
}
