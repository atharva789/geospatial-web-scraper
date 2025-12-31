package crawler

import (
	"net/http"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// VisitNode recursively walks the HTML node tree collecting child links. Links
// to geospatial files are recorded with metadata while regular links are queued
// for further crawling up to a maximum depth.
func (m *Manager) VisitNode(n *html.Node, links *[]WebNode, resp *http.Response, parent *WebNode, root *html.Node, searchQuery string) {
	const maxDepth = 4
	checkedPageType := false
	if n.Type == html.ElementNode {
		if DetectFTP(n, resp) && !checkedPageType {
			// launch an async job
			IndexFTP(n, resp)
			return
		}
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
				if GeoFileExtensions[ext] || GeoMetaFileExtensions[ext] || ContainsAnySubstring(link.Path, []string{"open", "open-data", "data-access"}) {
					// Extract comprehensive metadata
					metadata := ExtractMetadata(root, resp.Request.URL.String(), link.String())

					// Add to batch buffer
					m.dbBatch[m.dbBatchCount] = metadata
					m.dbBatchCount++

					// Flush to database when batch is full
					if m.dbBatchCount >= 50 {
						m.flushDatasetBatch()
					}

					// Add to crawl worklist if within depth limit
					if parent.Depth+1 < maxDepth {
						*links = append(*links, WebNode{URL: link.String(), Parent: parent, Depth: parent.Depth + 1})
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
					*links = append(*links, WebNode{URL: link.URL, Parent: parent, Depth: parent.Depth + 1})
				}
			}

		}
	}

	// Recurse into children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && !HasUnwantedClassOrID(c) {
			m.VisitNode(c, links, resp, parent, root, searchQuery)
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
