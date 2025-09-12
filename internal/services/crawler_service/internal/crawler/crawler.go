package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"golang.org/x/net/html"
)

//kafka topic: Website{URL: string, metadata: (XML/json)}

type DatasetEvent struct {
	PartitionID string
	DatasetURL  string
	MetaURL     []string
	PageContent string
}

// VisitNode recursively walks the HTML node tree collecting child links. Links
// to geospatial files are recorded with metadata while regular links are queued
// for further crawling up to a maximum depth.
func VisitNode(n *html.Node, links *[]WebNode, resp *http.Response, parent *WebNode, root *html.Node, searchQuery string, kw *kafka.Writer) {
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
					md, xmlLinks := GetPageMetadata(root)
					// stream page URL/html content --> Kafka service
					de := DatasetEvent{ // make this a batched request in the future
						PartitionID: "dataset-event",
						DatasetURL:  link.String(),
						MetaURL:     xmlLinks,
						PageContent: md.ToString(),
					}
					payload, err := json.Marshal(de)
					if err != nil {
						fmt.Println("An error occured: %s", err)
					}

					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					err = kw.WriteMessages(ctx, kafka.Message{
						Key:   []byte(de.PartitionID),
						Value: payload,
					})
					cancel()
					if err != nil {
						fmt.Println("Failed to write to kafka: %s", err)
					}
					WriteToLog(kw, "Kafka stream successful!")

					// check if filename or immediately surrounding metadata about file matches search query
					queryWords := strings.Split(searchQuery, " ")
					if ContainsAnySubstring(link.Path, queryWords) {
						// send to check list?
						queryString := "query: " + searchQuery + ".\n" + "filename: " + link.Path + "\n" + "metadata: \n" + metadata
						response, llmErr := DataQuery("you are a geospatial data expert.", "Matches user-query (filename, metadata)? Answer 'yes'/'no' only.", queryString)
						if llmErr != nil {
							fmt.Println("An error occured: ", llmErr)
						}
						fmt.Println("		(LLM response): ", response)
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
			if !contains {
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
		if c.Type == html.ElementNode && !HasUnwantedClassOrID(c) {
			VisitNode(c, links, resp, parent, root, searchQuery, kw)
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
