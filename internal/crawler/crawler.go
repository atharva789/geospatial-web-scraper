package crawler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"golang.org/x/net/html"
)

var tokens = make(chan struct{}, 40)
var downloadTokens = make(chan struct{}, 40)

var worklist = make(chan []WebNode)
var seen = make(map[string]bool)
var done = make(chan bool)

func BreadthFirst(scrapeQueue []string, downloadDir string) ([]string, error) {
	log.Println("------------------------------------------------------------------------------")
	log.Println("							STARTED NEW CRAWL SESSION")
	log.Printf("							SEED URL: '%v'", scrapeQueue)
	log.Println("------------------------------------------------------------------------------")
	var n int
	n++
	const maxCrawl = 400
	var count int
	var JobQueue []WebNode
	var results []string
	for _, url := range scrapeQueue {
		JobQueue = append(JobQueue, WebNode{
			Url:    url,
			Parent: nil,
			Depth:  0,
		})
	}

	go func() {
		worklist <- JobQueue
	}()

	for ; n > 0; n-- {
		list := <-worklist
		for _, node := range list {
			if count > maxCrawl {
				go func() { done <- true }()
				// log.Println("HIT MAX CRWL LIMIT!")
			} else {
				go func() { done <- false }()
				if !seen[node.Url] {
					count++
					n++
					log.Printf("Currently Crawled: %d/%d URLs", count, maxCrawl)
					seen[node.Url] = true
					results = append(results, node.Url)
					stop := <-done
					if !stop {
						go func(node WebNode) {
							res := Crawl(&node, &downloadDir)
							worklist <- res
						}(node)
					}

				}
			}
		}

	}
	log.Println("------------------------------------------------------------------------------")
	log.Printf("					Done! scraped %d URLs ", len(results))
	log.Println("------------------------------------------------------------------------------")
	return results, nil
}

func Crawl(node *WebNode, downloadDir *string) []WebNode {
	tokens <- struct{}{}
	list, err := Extract(node, downloadDir)
	<-tokens
	if err != nil {
		log.Printf("Error occured while crawling %v", err)
	}
	return list
}

func VisitNode(n *html.Node, links *[]WebNode, resp *http.Response, parent *WebNode, root *html.Node) {
	const maxDepth = 4

	if n.Type == html.ElementNode && n.Data == "a" {

		for _, a := range n.Attr {
			if a.Key != "href" {
				continue
			}
			if strings.HasPrefix(a.Val, "mailto:") || strings.HasPrefix(a.Val, "tel:") {
				continue
			}
			link, err := resp.Request.URL.Parse(a.Val)
			if err != nil {
				continue // ignore bad URLs
			}
			ext := strings.ToLower(path.Ext(link.Path))
			if GeoFileExtensions[ext] {
				meta := ExtractMetadata(root, resp.Request.URL.String(), link.String())
				if parent.Depth+1 < maxDepth {
					*links = append(*links, WebNode{Url: link.String(), Parent: parent, Depth: parent.Depth + 1, context: DataContext{Description: meta}})
				}
			} else if parent.Depth+1 < maxDepth {
				*links = append(*links, WebNode{Url: link.String(), Parent: parent, Depth: parent.Depth + 1})
			}
		}
	}

	// Recurse into children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && HasUnwantedClassOrID(c) == false {
			VisitNode(c, links, resp, parent, root)
		}
	}
}

func HasUnwantedClassOrID(n *html.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" || attr.Key == "id" {
			val := strings.ToLower(attr.Val)
			for substr := range UnwantedClassOrIDSubstrings {
				if strings.Contains(val, substr) {
					return true
				}
			}
		}
	}
	return false
}

func Extract(node *WebNode, downloadDir *string) ([]WebNode, error) {
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
		go DownloadBuffered(resp, node.Url, downloadDir)
		return nil, nil
	}

	doc, err := html.Parse(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %v", node.Url, err)
	}
	var links []WebNode

	VisitNode(doc, &links, resp, node, doc)

	return links, nil
}

func ValidateDownloadable(resp *http.Response, url string) bool {
	contentType := resp.Header.Get("Content-Type")
	if GeoMIMETypes[contentType] {
		//initiate download
		return true
	}
	return false
}

func DownloadBuffered(resp *http.Response, rawURL string, downloadDir *string) {
	downloadTokens <- struct{}{}

	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		log.Printf("Failed to buffer body for download: %v", err)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("Error parsing URL: %v", err)
		return
	}
	filename := path.Base(parsedURL.Path)
	filepath := path.Join(*downloadDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Printf("Error writing data: %v", err)
	}
	<-downloadTokens
}

func Download(rawURL string, data []byte, downloadDir *string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("Error parsing URL: %v", err)
		return err
	}
	filename := path.Base(parsedURL.Path)
	filepath := path.Join(*downloadDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Printf("Error writing data: %v", err)
		return err
	}
	return nil

}
