package crawler

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// ExtractMetadata parses metadata from the provided HTML document and returns a
// JSON string describing the download URL and page details.
func ExtractMetadata(doc *html.Node, pageURL, downloadURL string) string {
	md := downloadMetadata{URL: downloadURL}
	var xmlLinks []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if md.Title == "" && n.FirstChild != nil {
					md.Title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				var name, property, content string
				for _, a := range n.Attr {
					switch strings.ToLower(a.Key) {
					case "name":
						name = strings.ToLower(a.Val)
					case "property":
						property = strings.ToLower(a.Val)
					case "content":
						content = a.Val
					}
				}
				key := name
				if key == "" {
					key = property
				}
				switch key {
				case "description", "og:description":
					if md.Description == "" {
						md.Description = content
					}
				case "keywords":
					if len(md.Keywords) == 0 && content != "" {
						parts := strings.Split(content, ",")
						for i, p := range parts {
							parts[i] = strings.TrimSpace(p)
						}
						md.Keywords = parts
					}
				case "og:title":
					if md.Title == "" {
						md.Title = content
					}
				}
			case "script":
				var typ string
				for _, a := range n.Attr {
					if strings.ToLower(a.Key) == "type" {
						typ = strings.ToLower(a.Val)
					}
				}
				if strings.Contains(typ, "ld+json") && n.FirstChild != nil {
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(n.FirstChild.Data), &data); err == nil {
						if d, ok := data["description"].(string); ok && md.Description == "" {
							md.Description = d
						}
						if t, ok := data["name"].(string); ok && md.Title == "" {
							md.Title = t
						}
					}
				}
			case "link":
				var href, typ string
				for _, a := range n.Attr {
					if a.Key == "href" {
						href = a.Val
					} else if a.Key == "type" {
						typ = strings.ToLower(a.Val)
					}
				}
				if strings.Contains(typ, "xml") {
					xmlLinks = append(xmlLinks, href)
				}
			case "b", "h3", "p":
				if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
					md.Description += " " + n.FirstChild.Data
				}
			case "h1", "h2":
				if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
					md.Title += " " + n.FirstChild.Data
				}

			}
		} else if n.Type == html.TextNode {
			md.Description += " " + n.Data
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	base, _ := url.Parse(pageURL)
	for _, l := range xmlLinks {
		u, err := base.Parse(l)
		if err != nil {
			continue
		}
		resp, err := http.Get(u.String())
		if err != nil {
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		var x struct {
			Title       string `xml:"title"`
			Description string `xml:"description"`
		}
		if err := xml.Unmarshal(data, &x); err == nil {
			if md.Title == "" {
				md.Title = strings.TrimSpace(x.Title)
			}
			if md.Description == "" {
				md.Description = strings.TrimSpace(x.Description)
			}
		}
	}

	// Clean any newlines or excess whitespace that may have been
	// captured from the HTML so they don't appear as escaped "\n" when
	// printed or marshalled to JSON.
	md.Description = strings.Join(strings.Fields(md.Description), " ")

	b, _ := json.Marshal(md)
	return string(b)
}
