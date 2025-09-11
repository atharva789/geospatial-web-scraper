package crawler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// FindLinks embeds the search query, compares it against cached seed
// descriptions and returns the most relevant URLs which are then crawled. The
// resulting downloadable links are accumulated in m.downloadURLs.
func (m *Manager) FindLinks() ([]WebNode, error) {
	var jobs []WebNode
	if len(m.searchQuery.Sources) > 0 {
		for _, url := range m.searchQuery.Sources {
			jobs = append(jobs, WebNode{Url: url, Depth: 0})
		}
	}
	// get google-search results
	nodes, err := GoogleSearch(&m.searchQuery)
	if err != nil {
		return jobs, fmt.Errorf("google-search failed: %s", err)
	}
	jobs = append(jobs, nodes...)
	return jobs, nil

}

func (m *Manager) ScheduleCrawl() []WebNode {
	log.Println("------------------------------------------------------------------------------")
	log.Println("							STARTED NEW CRAWL SESSION")
	log.Println("------------------------------------------------------------------------------")
	newJobs, err := m.FindLinks()
	if err != nil {
		panic("Error initializing seed jobs")
	}
	//Crawling begins
	go func() {
		m.worklist <- newJobs
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

// ToLinks returns the URLs from the download queue as a plain slice of strings.
func (m *Manager) ToLinks() []string {
	var links []string
	for _, node := range m.downloadURLs {
		links = append(links, node.Url)
	}
	return links
}

// Crawl2 is a concurrency limited wrapper around Extract2 used during the main
// crawl loop. It returns any new links discovered for further processing.
func (m *Manager) Crawl2(node *WebNode) []WebNode {
	m.smTokens <- struct{}{}
	links, err := m.Extract2(node)
	<-m.smTokens
	if err != nil {
		log.Printf("Error occured while crawling %v: %v", node.Url, err)
	}

	return links
}

func isDir(link string) bool {
	tokens := strings.Split(link, "")
	if tokens[0] == "/" || tokens[len(tokens)-1] == "/" {
		return true
	}
	return false
}

func DetectFTP(n *html.Node, URL string, resp *http.Response) bool {
	if n.Data == "pre" {
		var containsHR, containsDir int
		for w := range n.ChildNodes() {
			switch w.Data {
			case "hr":
				containsHR++
			case "a":
				// clean and get text
				for _, a := range w.Attr {
					if a.Key != "href" {
						continue
					}
					if strings.HasPrefix(a.Val, "mailto:") || strings.HasPrefix(a.Val, "tel:") {
						continue
					}
					link, err := resp.Request.URL.Parse(a.Val)
					URL = link.String()
					if err != nil {
						continue // ignore bad URLs
					}
					ext := strings.ToLower(path.Ext(link.Path))
					if GeoFileExtensions[ext] || isDir(link.String()) {
						containsDir++
					}
				}
			default:
				continue
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		DetectFTP(c, URL, resp)
	}
	return false
}

// Indexes every file in a FTP silesystem with a geofile
func IndexFTP(n *html.Node, resp *http.Response) (FTPDir, error) {
	// if pre, skip everything before the hr tag, then make every link a directory
	startParsing := false
	for w := range n.ChildNodes() {
		switch w.Data {
		//parse to FTP dir struct
		case "hr":
			if !startParsing {
				startParsing = true
			}
			startParsing = false
		case "a":
			// do something
			// if GEOFIles:
			// 		1. search for metadata:
			//		2. if no metadata on current page, create 'File' object File{filename: something.tiff, url: something.com, description: abc, meta: something}
			//		3. send to DB/stream
			//		4. Clean later
			// if directory, go to next directory
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

				}
			}
		}
	}
	return FTPDir{}, fmt.Errorf("Error: Couldn't Index FTP page to FTP object(s)")
}

// Extract2 performs the actual HTTP GET for a node during the main crawl. It
// appends downloadable URLs to m.downloadURLs and returns any follow-on links
// for further crawling.
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
	// classify page: FTP portal, data catalog, general page
	downloadable := ValidateDownloadable(resp, node.Url)
	if downloadable {
		m.linkChan <- struct{}{} //replace with mu.Lock()
		links = append(links, WebNode{Url: node.Url})
		<-m.linkChan //replace with mu.UnLock()
		if *m.downloadPath != "" {
			//	go m.DownloadBuffered(resp, node.Url, m.downloadPath)
			fmt.Printf("\n Dummy downloading file %v", node.Url)
		}
		return links, nil
	}

	doc, err := html.Parse(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %v", node.Url, err)
	}

	VisitNode(doc, &m.downloadURLs, resp, node, doc, "*m.searchQuery", m.kWriter)

	return links, nil
}

// DownloadBuffered reads the HTTP response body and writes it to disk when
// running in secure mode. Downloads are serialized using dlTokens.
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
