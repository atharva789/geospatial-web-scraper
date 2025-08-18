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

func ContainsAnySubstring(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// Ancestors returns the chain of ancestors from parent up to root.
func Ancestors(n *html.Node) []*html.Node {
	var ancestors []*html.Node
	for p := n.Parent; p != nil; p = p.Parent {
		ancestors = append(ancestors, p)
	}
	return ancestors
}

// IsDescendantOf reports whether n has 'ancestor' in its parent chain.
func IsDescendantOf(n, ancestor *html.Node) bool {
	if n == nil || ancestor == nil {
		return false
	}
	for p := n.Parent; p != nil; p = p.Parent {
		if p == ancestor {
			return true
		}
	}
	return false
}

// ToString renders a simple markdown-like table.
// If Headers is non-empty, groups Data into rows of len(Headers).
func (t *TableData) ToString() string {
	if len(t.Headers) == 0 && len(t.Data) == 0 {
		return ""
	}

	var b strings.Builder
	writeLine := func(cols []string) {
		b.WriteString(strings.Join(cols, " | "))
		b.WriteByte('\n')
	}

	if len(t.Headers) > 0 {
		// header
		writeLine(t.Headers)
		// separator
		sep := make([]string, len(t.Headers))
		for i := range sep {
			sep[i] = "---"
		}
		writeLine(sep)

		// rows
		nCols := len(t.Headers)
		row := make([]string, 0, nCols)
		for _, d := range t.Data {
			row = append(row, strings.TrimSpace(d))
			if len(row) == nCols {
				writeLine(row)
				row = row[:0]
			}
		}
		// if leftover cells exist, flush them
		if len(row) > 0 {
			// pad to nCols for a consistent row
			for len(row) < nCols {
				row = append(row, "")
			}
			writeLine(row)
		}
	} else {
		// no headers; just one row with all data
		clean := make([]string, 0, len(t.Data))
		for _, d := range t.Data {
			clean = append(clean, strings.TrimSpace(d))
		}
		writeLine(clean)
	}

	return b.String()
}

// AddToStringBuilder de-duplicates by substring and skips empty writes.
func AddToStringBuilder(s *strings.Builder, newStr string) {
	newStr = strings.TrimSpace(newStr)
	if newStr == "" {
		return
	}
	if strings.Contains(s.String(), newStr) {
		return
	}
	if s.Len() > 0 && !strings.HasSuffix(s.String(), "\n") {
		s.WriteByte('\n')
	}
	s.WriteString(newStr)
}

// CheckDescendants - placeholder for future use.
func CheckDescendants(n *html.Node) {}

// CheckAncestors finds the nearest ancestor whose tag name or attribute values
// suggest a table-like container. Returns that node or nil.
func CheckAncestors(n *html.Node) *html.Node {
	tableAttrHints := []string{"list", "table", "product"}
	tableTags := []string{"table", "tr", "li", "ul", "ol", "thead", "tbody", "th", "td"}

	for _, w := range Ancestors(n) {
		// tag check
		for _, tg := range tableTags {
			if w.Type == html.ElementNode && w.Data == tg {
				return w
			}
		}
		// attribute value hint check
		for _, a := range w.Attr {
			if ContainsAnySubstring(a.Val, tableAttrHints) {
				return w
			}
		}
	}
	return nil
}

func GetPageMetadata(doc *html.Node) (downloadMetadata, []string) {
	unwanted := []string{}
	md := downloadMetadata{}
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

	tableData := &TableData{
		Root:    nil,
		Headers: []string{},
		Data:    []string{},
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if shouldSkip(n) {
			return
		}

		switch n.Type {
		case html.ElementNode:
			switch n.Data {
			case "title":
				if md.Title == "" && n.FirstChild != nil {
					AddToStringBuilder(&titleBuf, n.FirstChild.Data)
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
						AddToStringBuilder(&descBuf, content)
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
						AddToStringBuilder(&titleBuf, content)
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
						AddToStringBuilder(&descBuf, d)
					}
					if t, ok := data["name"].(string); ok && md.Title == "" {
						AddToStringBuilder(&titleBuf, t)
					}
					if h, ok := data["headline"].(string); ok && md.Title == "" {
						AddToStringBuilder(&titleBuf, h)
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
			case "table":
				tableData.Root = n
			}

		case html.TextNode:
			txt := strings.TrimSpace(n.Data)
			if txt != "" {
				switch n.Parent.Data {
				case "th":
					if tableData.Root != nil && IsDescendantOf(n, tableData.Root) {
						// add unique header
						dup := false
						for _, h := range tableData.Headers {
							if h == txt {
								dup = true
								break
							}
						}
						if !dup {
							tableData.Headers = append(tableData.Headers, txt)
						}
					}
				case "td":
					tableData.Data = append(tableData.Data, txt)
				case "div":
					_ = CheckAncestors(n) // available for future logic
				case "h1":
					AddToStringBuilder(&descBuf, "# "+txt)
				case "h2":
					AddToStringBuilder(&descBuf, "## "+txt)
				case "h3", "h4":
					AddToStringBuilder(&descBuf, "### "+txt)
				case "pre":
					_ = CheckAncestors(n)
					AddToStringBuilder(&descBuf, "```\n"+txt+"\n```")
				case "li", "b":
					AddToStringBuilder(&descBuf, "- "+txt)
				case "style":

				default:
					// plain text fallthrough (optional)
					AddToStringBuilder(&descBuf, txt)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Append any table we collected.
	if s := tableData.ToString(); strings.TrimSpace(s) != "" {
		AddToStringBuilder(&descBuf, s)
	}
	md.Title = strings.TrimSpace(strings.Join(strings.Fields(titleBuf.String()), " "))
	md.Description = strings.TrimSpace(strings.Join(strings.Fields(descBuf.String()), " "))
	return md, xmlLinks
}

// ExtractMetadata parses metadata from the provided HTML document
// and returns a JSON string describing the download URL and page details.
func ExtractMetadata(doc *html.Node, pageURL, downloadURL string) string {
	md, xmlLinks := GetPageMetadata(doc)

	var titleBuf, descBuf strings.Builder

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
				AddToStringBuilder(&titleBuf, x.Title)
			}
			if md.Description == "" {
				AddToStringBuilder(&descBuf, x.Description)
			}
		}
	}

	// Final clean-up & assign.
	md.Title += strings.TrimSpace(strings.Join(strings.Fields(titleBuf.String()), " "))
	md.Description += strings.TrimSpace(strings.Join(strings.Fields(descBuf.String()), " "))

	out, err := json.Marshal(md)
	if err != nil {
		panic("ExtractMetadata error while coverting data to JSON")
	}
	return string(out)
}

func CheckAncesstors(n *html.Node) {
	// unimplemented
}
