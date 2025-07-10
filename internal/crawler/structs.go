package crawler

type WebNode struct {
	Url    string
	Parent *WebNode // node is a parent if parentURL == "root"
	Depth  int
}

type Manager struct {
	secure       bool
	downloadPath *string
	searchQuery  *string
	downloadURLs []WebNode
	searchFrom   map[string]string
	linkChan     chan string
	smTokens     chan struct{}
	dlTokens     chan struct{}
	worklist     chan []WebNode
	done         chan bool
	seen         map[string]bool
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
