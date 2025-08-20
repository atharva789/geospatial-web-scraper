package crawler

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// VisitNode recursively walks the HTML node tree collecting child links. Links
// to geospatial files are recorded with metadata while regular links are queued
// for further crawling up to a maximum depth.
func VisitNode(n *html.Node, links *[]WebNode, resp *http.Response, parent *WebNode, root *html.Node, searchQuery string) {
	const maxDepth = 4

	if n.Type == html.ElementNode {
		switch n.Data {
		case "a":
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
				if GeoFileExtensions[ext] || ContainsAnySubstring(link.Path, []string{"open", "open-data", "data-access"}) {
					metadata := ExtractMetadata(root, resp.Request.URL.String(), link.String())

					// check if filename or immediately surrounding metadata about file matches search query
					queryWords := strings.Split("search query", " ")
					if ContainsAnySubstring(link.Path, queryWords) {
						// send to check list?
						queryString := "query: " + searchQuery + ".\n" + "filename: " + link.Path + "\n" + "metadata: \n" + metadata
						response, llmErr := DataQuery("you are a geospatial data expert.", "is the file what the user is looking for (based on filename, metadata)? answer 'yes' or 'no' only.", queryString, nil)
						if llmErr != nil {
							fmt.Println("An error occured: ", llmErr)
						}
						if response == "yes" {
							fmt.Println("Is this the file you're looking for? ", queryString)
						}
					}
					if parent.Depth+1 < maxDepth {
						*links = append(*links, WebNode{Url: link.String(), Parent: parent, Depth: parent.Depth + 1, context: DataContext{Description: metadata}})
					}
				}
			}
		case "input":
			// check if placeholder tag, id contains words like "search", "data": if yes, search
			// check if ancesstors contain keywords ("search", "data")
			contains := false
			var newLinks []responseStruct
			for _, a := range n.Attr {
				if strings.Contains(a.Val, "data") && strings.Contains(a.Val, "search") {
					contains = true
					// normalise search query
					newLinks = searchCatalog(searchQuery)
				}
			}
			if contains == false {
				ancesstors := Ancestors(n)
				depth := 0
				for _, p := range ancesstors {
					depth++
					if depth > 5 || p.Data == "body" {
						break
					}
					if p.Type == html.TextNode && ContainsAnySubstring(p.Data, []string{"data", "search"}) {
						// do something
						searchCatalog(searchQuery)
					}
				}
			}

			if parent.Depth+1 < maxDepth {
				for _, link := range newLinks {
					*links = append(*links, WebNode{Url: link.URL, Parent: parent, Depth: parent.Depth + 1, context: DataContext{Description: link.Metadata}})
				}
			}

		}
	}

	// Recurse into children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && HasUnwantedClassOrID(c) == false {
			VisitNode(c, links, resp, parent, root, searchQuery)
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

type responseStruct struct {
	URL      string
	Metadata string
}

// sends a string to search in an <input> form, returns a
// list: each element is a search result and any text/information about it
func searchCatalog(query string) []responseStruct {
	relevantLinks := []responseStruct{}
	return relevantLinks
}
