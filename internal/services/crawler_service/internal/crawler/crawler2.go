package crawler

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// S3ListBucketResult represents the XML response from S3 ListObjectsV2 API
type S3ListBucketResult struct {
	XMLName               xml.Name   `xml:"ListBucketResult"`
	Name                  string     `xml:"Name"`
	Prefix                string     `xml:"Prefix"`
	IsTruncated           bool       `xml:"IsTruncated"`
	Contents              []S3Object `xml:"Contents"`
	CommonPrefixes        []S3Prefix `xml:"CommonPrefixes"`
	NextContinuationToken string     `xml:"NextContinuationToken"`
}

// S3Object represents a file object in S3
type S3Object struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	Size         int64  `xml:"Size"`
}

// S3Prefix represents a common prefix (subdirectory) in S3
type S3Prefix struct {
	Prefix string `xml:"Prefix"`
}

func (m *Manager) FindLinks() ([]WebNode, error) {
	var jobs []WebNode
	if len(m.searchQuery.Sources) > 0 {
		for _, url := range m.searchQuery.Sources {
			jobs = append(jobs, WebNode{URL: url, Depth: 0})
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
	fmt.Println("STARTED NEW CRAWL SESSION")
	newJobs, err := m.FindLinks()
	if err != nil {
		fmt.Println("Error initializing seed jobs")
		panic("")
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
				if !m.seen[node.URL] {
					count++
					n++
					m.seen[node.URL] = true
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
	fmt.Printf("Done! scraped %d URLs \n", len(m.downloadURLs))
	return m.downloadURLs

}

// ToLinks returns the URLs from the download queue as a plain slice of strings.
func (m *Manager) ToLinks() []string {
	var links []string
	for _, node := range m.downloadURLs {
		links = append(links, node.URL)
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
		fmt.Printf("Error occured while crawling %v: %v\n", node.URL, err)
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

func DetectFTP(n *html.Node, resp *http.Response) bool {
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
					if err != nil {
						continue // ignore bad URLs
					}
					ext := strings.ToLower(path.Ext(link.Path))
					if GeoFileExtensions[ext] || GeoMetaFileExtensions[ext] || isDir(link.String()) {
						containsDir++
					}
				}
			default:
				continue
			}
		}
		if containsDir+containsHR > 0 {
			return true
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if DetectFTP(c, resp) {
			return true
		}
	}
	return false
}

// IndexFTP indexes an FTP directory listing and returns a hierarchical FTPDirectory tree.
// It recursively crawls subdirectories, collecting GeoFiles with their associated metadata URLs.
// The returned tree mirrors the original FTP filesystem structure with proper parent relationships.
func IndexFTP(n *html.Node, resp *http.Response) (*FTPDirectory, error) {
	return indexFTPRecursive(n, resp, nil)
}

// indexFTPRecursive builds the FTPDirectory tree recursively.
func indexFTPRecursive(n *html.Node, resp *http.Response, parent *FTPDirectory) (*FTPDirectory, error) {
	currentDir := &FTPDirectory{
		Parent:         parent,
		SubDirectories: nil,
		DownloadFiles:  nil,
	}

	// Collect all links from the current directory listing
	var geoFiles []string
	var metaFiles []string
	var subDirURLs []string

	// Parse all anchor tags in this directory listing
	var parseAnchors func(*html.Node)
	parseAnchors = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key != "href" {
					continue
				}
				if strings.HasPrefix(attr.Val, "mailto:") || strings.HasPrefix(attr.Val, "tel:") {
					continue
				}
				// Skip parent directory links
				if attr.Val == "../" || attr.Val == ".." || attr.Val == "/" {
					continue
				}

				link, err := resp.Request.URL.Parse(attr.Val)
				if err != nil {
					continue
				}

				ext := strings.ToLower(path.Ext(link.Path))
				linkStr := link.String()

				// Classify the link
				if GeoFileExtensions[ext] {
					geoFiles = append(geoFiles, linkStr)
				} else if GeoMetaFileExtensions[ext] {
					metaFiles = append(metaFiles, linkStr)
				} else if isDir(attr.Val) {
					subDirURLs = append(subDirURLs, linkStr)
				}
			}
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			parseAnchors(c)
		}
	}
	parseAnchors(n)

	// Match geo files with their metadata files
	currentDir.DownloadFiles = matchGeoFilesWithMetadata(geoFiles, metaFiles)

	// Recursively process subdirectories
	for _, subDirURL := range subDirURLs {
		childDoc, childResp, err := getDOMFromURL(subDirURL)
		if err != nil {
			fmt.Printf("[WARN] Failed to fetch subdirectory %s: %v\n", subDirURL, err)
			continue
		}

		childDir, err := indexFTPRecursive(childDoc, childResp, currentDir)
		if err != nil {
			fmt.Printf("[WARN] Failed to index subdirectory %s: %v\n", subDirURL, err)
			continue
		}

		// Only add subdirectory if it contains files or has subdirectories with files
		if len(childDir.DownloadFiles) > 0 || len(childDir.SubDirectories) > 0 {
			currentDir.SubDirectories = append(currentDir.SubDirectories, childDir)
		}
	}

	return currentDir, nil
}

// matchGeoFilesWithMetadata pairs geo files with their corresponding metadata files.
// It matches based on filename similarity (same base name with different extension).
func matchGeoFilesWithMetadata(geoFiles, metaFiles []string) []GeoFile {
	var result []GeoFile

	// Build a map of base names to metadata URLs
	metaByBase := make(map[string]string)
	for _, metaURL := range metaFiles {
		base := strings.TrimSuffix(path.Base(metaURL), path.Ext(metaURL))
		metaByBase[base] = metaURL
	}

	for _, geoURL := range geoFiles {
		geoFile := GeoFile{URL: geoURL}

		// Try to find matching metadata by base name
		base := strings.TrimSuffix(path.Base(geoURL), path.Ext(geoURL))
		if metaURL, ok := metaByBase[base]; ok {
			geoFile.Metadata = metaURL
		}

		result = append(result, geoFile)
	}

	return result
}

// IndexS3 indexes an S3 bucket path and returns a hierarchical FTPDirectory tree.
// It uses the S3 ListObjectsV2 API to enumerate objects and builds a tree structure
// that mirrors the S3 prefix hierarchy. Each GeoFile has URL and optional Metadata.
//
// The s3URL should be in the format:
//   - https://bucket-name.s3.amazonaws.com/prefix/path/
//   - https://bucket-name.s3.region.amazonaws.com/prefix/path/
//   - Or an index.html URL with ?prefix= parameter
func IndexS3(s3URL string) (*FTPDirectory, error) {
	bucketURL, prefix, err := parseS3URL(s3URL)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 URL: %w", err)
	}

	return indexS3Recursive(bucketURL, prefix, nil)
}

// parseS3URL extracts the bucket base URL and prefix from various S3 URL formats.
func parseS3URL(s3URL string) (bucketURL, prefix string, err error) {
	parsed, err := url.Parse(s3URL)
	if err != nil {
		return "", "", err
	}

	// Handle index.html?prefix= format
	if strings.HasSuffix(parsed.Path, "index.html") {
		prefix = parsed.Query().Get("prefix")
		parsed.Path = ""
		parsed.RawQuery = ""
		return parsed.String(), prefix, nil
	}

	// Handle direct S3 path format: https://bucket.s3.amazonaws.com/path/to/prefix/
	bucketURL = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	prefix = strings.TrimPrefix(parsed.Path, "/")

	return bucketURL, prefix, nil
}

// indexS3Recursive builds the FTPDirectory tree by querying the S3 API.
func indexS3Recursive(bucketURL, prefix string, parent *FTPDirectory) (*FTPDirectory, error) {
	currentDir := &FTPDirectory{
		Parent:         parent,
		SubDirectories: nil,
		DownloadFiles:  nil,
	}

	var geoFiles []string
	var metaFiles []string
	var subPrefixes []string

	// Paginate through all results
	continuationToken := ""
	for {
		listURL := fmt.Sprintf("%s/?list-type=2&prefix=%s&delimiter=/",
			bucketURL, url.QueryEscape(prefix))
		if continuationToken != "" {
			listURL += "&continuation-token=" + url.QueryEscape(continuationToken)
		}

		resp, err := http.Get(listURL)
		if err != nil {
			return nil, fmt.Errorf("failed to list S3 bucket: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("S3 API error: %s", resp.Status)
		}

		var result S3ListBucketResult
		if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to parse S3 response: %w", err)
		}
		resp.Body.Close()

		// Process objects (files)
		for _, obj := range result.Contents {
			// Skip the prefix itself (directory marker)
			if obj.Key == prefix || obj.Size == 0 {
				continue
			}

			fileURL := fmt.Sprintf("%s/%s", bucketURL, obj.Key)
			ext := strings.ToLower(path.Ext(obj.Key))

			if GeoFileExtensions[ext] {
				geoFiles = append(geoFiles, fileURL)
			} else if GeoMetaFileExtensions[ext] {
				metaFiles = append(metaFiles, fileURL)
			}
		}

		// Collect subdirectories
		for _, commonPrefix := range result.CommonPrefixes {
			subPrefixes = append(subPrefixes, commonPrefix.Prefix)
		}

		// Check for more pages
		if !result.IsTruncated {
			break
		}
		continuationToken = result.NextContinuationToken
	}

	// Match geo files with metadata
	currentDir.DownloadFiles = matchGeoFilesWithMetadata(geoFiles, metaFiles)

	// Recursively process subdirectories
	for _, subPrefix := range subPrefixes {
		childDir, err := indexS3Recursive(bucketURL, subPrefix, currentDir)
		if err != nil {
			fmt.Printf("[WARN] Failed to index S3 prefix %s: %v\n", subPrefix, err)
			continue
		}

		// Only add if it contains files or has subdirectories with files
		if len(childDir.DownloadFiles) > 0 || len(childDir.SubDirectories) > 0 {
			currentDir.SubDirectories = append(currentDir.SubDirectories, childDir)
		}
	}

	return currentDir, nil
}

// Extract2 performs the actual HTTP GET for a node during the main crawl. It
// appends downloadable URLs to m.downloadURLs and returns any follow-on links
// for further crawling.
func (m *Manager) Extract2(node *WebNode) ([]WebNode, error) {
	var links []WebNode

	doc, resp, err := getDOMFromURL(node.URL)
	if err != nil {
		return nil, err
	}

	VisitNode(doc, &m.downloadURLs, resp, node, doc, "*m.searchQuery")

	return links, nil
}

func getDOMFromURL(URL string) (*html.Node, *http.Response, error) {
	resp, err := http.Get(URL)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("getting %s: %s", URL, resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("parsing %s as HTML: %v", URL, err)
	}
	return doc, resp, nil
}
