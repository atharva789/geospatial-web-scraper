package concurrent_scraper

type WebNode struct {
	Url    string
	Parent *WebNode // node is a parent if parentURL == "root"
	Depth  int
}

type Manager struct {
	downloadPath string
	searchQuery  string
	downloadURLs []string
	searchFrom   []string //configure a set of URLs to enable scraping from
}

func SlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	freq := make(map[string]int)
	for _, x := range a {
		freq[x]++
	}
	for _, y := range b {
		freq[y]--
	}

	for _, count := range freq {
		if count != 0 {
			return false
		}
	}
	return true
}
