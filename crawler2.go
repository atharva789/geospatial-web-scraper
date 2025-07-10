package concurrent_scraper

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"golang.org/x/net/html"
)

func (m *Manager) findLinks() []string {
	log.Println("------------------------------------------------------------------------------")
	log.Println("							STARTED NEW CRAWL SESSION")
	log.Println("------------------------------------------------------------------------------")

	queryWords := strings.Split(*m.searchQuery, " ")
	var sources []string
	var searchTerms []string

	for _, src := range m.searchFrom {
		sources = append(sources, src)
	}
	for _, word := range queryWords {
		idx := Contains(word, sources)
		if idx != -1 {
			searchTerms = append(searchTerms, word)
		}
	}

	var JobQueue []WebNode
	for key, _ := range m.searchFrom {
		JobQueue = append(JobQueue, WebNode{
			Url:    key,
			Parent: nil,
			Depth:  0,
		})
	}

	go func() {
		m.worklist <- JobQueue
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
	return m.toLinks()

}

func (m *Manager) toLinks() []string {
	var links []string
	for _, node := range m.downloadURLs {
		links = append(links, node.Url)
	}
	return links
}

func (m *Manager) Crawl2(node *WebNode) []WebNode {
	m.smTokens <- struct{}{}
	m.Extract2(node)
	return []WebNode{}
}

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
	downloadable := ValidateDownloadable(resp, node.Url)
	if downloadable {
		if *m.downloadPath != "" {
			bodyBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close() // safe to close now
			if err != nil {
				log.Printf("Failed to buffer body for download: %v", err)
				return nil, nil
			}
			go DownloadBuffered(bodyBytes, node.Url, m.downloadPath)
		} else {
			m.linkChan <- node.Url
			links = append(links, WebNode{
				Url: node.Url,
			})
			<-m.linkChan
		}
		return nil, nil
	}

	doc, err := html.Parse(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %v", node.Url, err)
	}

	VisitNode(doc, &m.downloadURLs, resp, node)

	return links, nil
}

func (m *Manager) DownloadBuffered(data []byte, rawURL string) {
	m.dlTokens <- struct{}{}
	if m.secure {
		cmd := exec.Command(
			"firejail",
			"--private="+*m.downloadPath,
			"--net=none",
			"--caps.drop=all",
			"--seccomp",
			"--shell=none",
			"--quiet",
			fmt.Sprintf("downloader -o=%s -u=%s", *m.downloadPath),
		)
		cmd.Run()
	} else {

	}
	<-m.dlTokens
}
