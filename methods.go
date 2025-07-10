package concurrent_scraper

import (
	"strings"
)

func Contains(value string, slice []string) int {
	for idx, wrd := range slice {
		if strings.Compare(strings.ToLower(wrd), strings.ToLower(value)) == 0 {
			return idx
		}
	}
	return -1
}
