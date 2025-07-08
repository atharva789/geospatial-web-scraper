package concurrent_scraper

import (
	"fmt"
	"strings"
)

func (m *Manager) findLinks() []string {
	return []string{}
}

func (m *Manager) getDatasets() bool {
	downloadLinks := m.findLinks()
	for _, link := range downloadLinks {
		fmt.Printf("Downloading: %s", link)

	}
	return false
}

func Contains(value string, slice []string) bool {
	for _, val := range slice {
		if strings.Compare(val, value) == 0 {
			return true
		}
	}
	return false
}
