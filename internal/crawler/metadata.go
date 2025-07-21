package crawler

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Substrings that mark a subtree as boilerplate (tag OR class/id/role).
var unwanted = []string{
	"nav", "menu", "header", "footer", "sidebar", "aside", "ads", "cookie",
	"usa-banner",
}

// writes to stringbuilder if the string being appended isn't
// already in the stringbuilder
func AddToStringbuilder(strBuf strings.Builder, newStr string) {
	if strings.Contains(strBuf.String(), newStr) != true {
		strBuf.WriteString(" " + newStr)
	}
}

// ExtractMetadata parses metadata from the provided HTML document
// and returns a JSON string describing the download URL and page details.
func ExtractMetadata(doc *html.Node, pageURL, downloadURL string) string {
	md := downloadMetadata{URL: downloadURL}
	var xmlLinks []string

	var titleBuf, descBuf strings.Builder // cheap, no extra allocs

	// Helper: shouldSkip returns true if node is undesirable.
	shouldSkip := func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}
		// Tag check.
		for _, bad := range unwanted {
			if n.Data == bad {
				return true
			}
		}
		// Attribute token check.
		for _, a := range n.Attr {
			if a.Key == "class" || a.Key == "id" || a.Key == "role" {
				for _, bad := range unwanted {
					if strings.Contains(strings.ToLower(a.Val), bad) {
						return true
					}
				}
			}
		}
		return false
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if shouldSkip(n) {
			return // do not look at, or inside, boilerplate
		}

		switch n.Type {
		case html.ElementNode:
			switch n.Data {
			case "title":
				if md.Title == "" && n.FirstChild != nil {
					titleBuf.WriteString(strings.TrimSpace(n.FirstChild.Data))
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
						content = strings.TrimSpace(a.Val)
					}
				}
				key := name
				if key == "" {
					key = property
				}
				switch key {
				case "description", "og:description":
					if md.Description == "" {
						descBuf.WriteString(" " + content)
					}
				case "keywords":
					if len(md.Keywords) == 0 && content != "" {
						parts := strings.Split(content, ",")
						for i, p := range parts {
							parts[i] = strings.TrimSpace(p)
						}
						md.Keywords = parts
					}
				case "og:title", "headline":
					if md.Title == "" {
						titleBuf.WriteString(" " + content)
					}
				}

			case "script":
				var typ string
				for _, a := range n.Attr {
					if strings.EqualFold(a.Key, "type") {
						typ = strings.ToLower(a.Val)
						break
					}
				}
				// Accept only JSON-LD; skip every other <script>.
				if !strings.Contains(typ, "ld+json") {
					return
				}
				if n.FirstChild == nil {
					return
				}
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(n.FirstChild.Data), &data); err == nil {
					if d, ok := data["description"].(string); ok && md.Description == "" {
						descBuf.WriteString(" " + strings.TrimSpace(d))
					}
					if t, ok := data["name"].(string); ok && md.Title == "" {
						titleBuf.WriteString(" " + strings.TrimSpace(t))
					}
					if h, ok := data["headline"].(string); ok && md.Title == "" {
						titleBuf.WriteString(" " + strings.TrimSpace(h))
					}
					if kw, ok := data["keywords"].(string); ok && len(md.Keywords) == 0 {
						for _, p := range strings.Split(kw, ",") {
							md.Keywords = append(md.Keywords, strings.TrimSpace(p))
						}
					}
				}

			case "link":
				var href, typ string
				for _, a := range n.Attr {
					switch strings.ToLower(a.Key) {
					case "href":
						href = a.Val
					case "type":
						typ = strings.ToLower(a.Val)
					}
				}
				if strings.Contains(typ, "xml") {
					xmlLinks = append(xmlLinks, href)
				}
			}
		case html.TextNode:
			// Collect visible text only if parent is a paragraph-like tag.
			switch n.Parent.Data {
			case "p", "h1", "h2", "h3", "h4", "li":
				descBuf.WriteString(" " + strings.TrimSpace(n.Data))
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Secondary XML harvest (RSS/Atom) – single client with timeout.
	client := &http.Client{Timeout: 5 * time.Second}
	base, _ := url.Parse(pageURL)
	for _, l := range xmlLinks {
		u, err := base.Parse(l)
		if err != nil {
			continue
		}
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.Get(u.String())
		if err != nil {
			cancel()
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if err != nil {
			continue
		}
		var x struct {
			Title       string `xml:"title"`
			Description string `xml:"description"`
		}
		if err := xml.Unmarshal(data, &x); err == nil {
			if md.Title == "" {
				titleBuf.WriteString(" " + strings.TrimSpace(x.Title))
			}
			if md.Description == "" {
				descBuf.WriteString(" " + strings.TrimSpace(x.Description))
			}
		}
	}

	// Final clean-up & assign.
	md.Title = strings.TrimSpace(strings.Join(strings.Fields(titleBuf.String()), " "))
	md.Description = strings.TrimSpace(strings.Join(strings.Fields(descBuf.String()), " "))

	out, _ := json.Marshal(md)
	return string(out)
}
