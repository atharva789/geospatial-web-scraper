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

// BreadthFirst crawls starting URLs breadth-first up to a fixed limit. It
// returns all discovered URLs in the order they were seen and optionally
// downloads any directly downloadable resources.
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

// Crawl retrieves links from the given node URL. It applies a token based
// concurrency limit and delegates HTML parsing to Extract.
func Crawl(node *WebNode, downloadDir *string) []WebNode {
	tokens <- struct{}{}
	list, err := Extract(node, downloadDir)
	<-tokens
	if err != nil {
		log.Printf("Error occured while crawling %v", err)
	}
	return list
}

// VisitNode recursively walks the HTML node tree collecting child links. Links
// to geospatial files are recorded with metadata while regular links are queued
// for further crawling up to a maximum depth.
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

// HasUnwantedClassOrID returns true if the element has a class or id attribute
// containing any blacklisted substring defined in UnwantedClassOrIDSubstrings.
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

// Extract fetches the URL in the provided WebNode and returns any child links
// discovered on the page. If the URL points directly to a downloadable
// geospatial file, the file is scheduled for download and no further links are
// returned.
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

// ValidateDownloadable checks the HTTP response headers to determine if the
// resource is a geospatial file that should be downloaded directly.
func ValidateDownloadable(resp *http.Response, url string) bool {
	contentType := resp.Header.Get("Content-Type")
	if GeoMIMETypes[contentType] {
		//initiate download
		return true
	}
	return false
}

// DownloadBuffered saves the body of an HTTP response to disk using a buffered
// read to avoid holding the connection open. A token channel limits concurrent
// downloads.
func DownloadBuffered(resp *http.Response, rawURL string, downloadDir *string) {
	// acquire a download slot so we don't overwhelm the disk or network
	downloadTokens <- struct{}{}
	defer func() { <-downloadTokens }()

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("error parsing URL %s: %v", rawURL, err)
		resp.Body.Close()
		return
	}

	// Determine file name from URL path. If none exists, fall back to a
	// generic name so the download is still saved.
	filename := path.Base(parsedURL.Path)
	if filename == "" || filename == "." || filename == "/" {
		filename = "download"
	}

	filepath := path.Join(*downloadDir, filename)

	// Create or truncate the output file then stream the body directly to
	// disk. Using io.Copy avoids buffering the entire file in memory.
	outFile, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("error creating file %s: %v", filepath, err)
		resp.Body.Close()
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		log.Printf("error writing to file %s: %v", filepath, err)
	}
	resp.Body.Close()
}

// Download writes the provided data to a file in downloadDir using the
// filename derived from the URL. It is used by the secure downloader helper
// and tests.
func Download(rawURL string, data []byte, downloadDir *string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("error parsing URL %s: %v", rawURL, err)
		return err
	}
	filename := path.Base(parsedURL.Path)
	if filename == "" || filename == "." || filename == "/" {
		filename = "download"
	}
	filepath := path.Join(*downloadDir, filename)
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("error creating file %s: %v", filepath, err)
		return err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		log.Printf("error writing data to %s: %v", filepath, err)
		return err
	}
	return nil
}
