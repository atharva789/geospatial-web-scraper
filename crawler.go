package concurrent_scraper

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

var tokens = make(chan struct{}, 40)
var downloadTokens = make(chan struct{}, 40)

func BreadthFirst(scrapeQueue []string) ([]string, error) {
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

	seen := make(map[string]bool)
	worklist := make(chan []WebNode)
	done := make(chan bool)
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
							res := Crawl(&node)
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

func Crawl(node *WebNode) []WebNode {
	tokens <- struct{}{}
	list, err := Extract(node)
	<-tokens
	if err != nil {
		log.Printf("Error occured while crawling %v", err)
	}
	return list
}

func VisitNode(n *html.Node, links *[]WebNode, resp *http.Response, parent *WebNode) {
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
			if parent.Depth+1 < maxDepth {
				*links = append(*links, WebNode{Url: link.String(), Parent: parent, Depth: parent.Depth + 1})
			}
		}
	}

	// Recurse into children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && HasUnwantedClassOrID(c) == false {
			VisitNode(c, links, resp, parent)
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

func Extract(node *WebNode) ([]WebNode, error) {
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
		go Download(resp, node.Url)
		resp.Body.Close()
		return nil, nil
	}
	// else {
	// 	log.Println("scraping URL", node.Url)
	// }
	doc, err := html.Parse(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %v", node.Url, err)
	}
	var links []WebNode

	VisitNode(doc, &links, resp, node)

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

func Download(resp *http.Response, url string) {
	downloadTokens <- struct{}{}
	log.Println("	Downloading file:", url)
	<-downloadTokens
}

var GeoMIMETypes = map[string]bool{
	"application/csv":                      true,
	"application/zip":                      true,
	"application/json":                     true,
	"application/geo+json":                 true,
	"application/x-geotiff":                true,
	"application/x-shapefile":              true,
	"application/x-esri-shape":             true,
	"application/x-filegdb":                true,
	"application/x-esri-geodatabase":       true,
	"application/x-netcdf":                 true,
	"application/x-hdf":                    true,
	"application/x-hdf5":                   true,
	"application/x-hdf4":                   true,
	"application/x-grib":                   true,
	"application/grib":                     true,
	"application/x-bil":                    true,
	"application/x-bip":                    true,
	"application/x-bsq":                    true,
	"application/vnd.las":                  true,
	"application/vnd.laz":                  true,
	"application/vnd.google-earth.kml+xml": true,
	"application/vnd.google-earth.kmz":     true,
	"application/x-sqlite3":                true,
	"application/geopackage+sqlite3":       true,
	"application/vnd.ogc.wms_xml":          true,
	"application/vnd.ogc.wfs_xml":          true,
	"application/topo+json":                true,
}

var UnwantedClassOrIDSubstrings = map[string]bool{
	// Navigation, headers, menus
	"nav":        true,
	"menu":       true,
	"header":     true,
	"breadcrumb": true,
	"skip":       true,

	// Sidebars and secondary panels
	"sidebar": true,
	"aside":   true,
	"related": true,

	// Footers and banners
	"footer": true,
	"banner": true,

	// Cookie/legal/accessibility notices
	"cookie":        true,
	"consent":       true,
	"disclaimer":    true,
	"notice":        true,
	"privacy":       true,
	"alert":         true,
	"accessibility": true,

	// Social, sharing, subscribing
	"social":     true,
	"share":      true,
	"subscribe":  true,
	"newsletter": true,

	// Feedback, modals, popups
	"feedback": true,
	"modal":    true,
	"popup":    true,

	// USGS S3 directory-specific
	"search":   true,
	"contact":  true,
	"foia":     true,
	"policies": true,

	// Generic
	"identifier": true,
}
