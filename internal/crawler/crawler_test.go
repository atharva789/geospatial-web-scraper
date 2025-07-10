package crawler

import (
	"log"
	"testing"
)

func TestBreadthFirst(t *testing.T) {
	t.Run("Site with no links", func(t *testing.T) {
		url := "https://water.usgs.gov/GIS/wbd_huc8.pdf"
		jobs := []string{url}
		got, err := BreadthFirst(jobs, "")
		want := []string{url}

		if !SlicesEqualUnordered(got, want) || err != nil {
			t.Errorf("got %v, want %v", got, want)
		}
	})
	t.Run("1-level depth site-map", func(t *testing.T) {
		url := "https://httpbin.org/links/10/0"
		got, err := BreadthFirst([]string{url}, "")
		want := []string{"https://httpbin.org/links/10/9", "https://httpbin.org/links/10/0", "https://httpbin.org/links/10/1", "https://httpbin.org/links/10/2", "https://httpbin.org/links/10/3", "https://httpbin.org/links/10/4", "https://httpbin.org/links/10/5", "https://httpbin.org/links/10/6", "https://httpbin.org/links/10/7", "https://httpbin.org/links/10/8"}
		if !SlicesEqualUnordered(got, want) || err != nil {
			t.Errorf("got %v, want %v", got, want)
		}
	})
	t.Run("Direct-download-link test", func(t *testing.T) {
		url := "https://www.nass.usda.gov/Research_and_Science/Cropland/Release/datasets/2014_30m_cdls.zip"
		got, err := BreadthFirst([]string{url}, "")
		want := []string{url}
		if !SlicesEqualUnordered(got, want) || err != nil {
			t.Errorf("got %v, want %v", got, want)
		}

	})
}

func BenchmarkBreadthFirst(b *testing.B) {
	// url := "https://httpbin.org/links/10/9"
	const JobSize = 10
	var scrapeQueue []string
	i := 0
	for key, _ := range PublicGeospatialDataSeedsMap {
		if i > JobSize-1 {
			break
		}
		scrapeQueue = append(scrapeQueue, key)
		i++
	}

	// scrapeQueue := []string{"https://www.ncei.noaa.gov/products/arctic-antarctic-products-data-information"}
	log.Printf("To-scrape: %v", scrapeQueue)
	// url := "https://www.nass.usda.gov/Research_and_Science/Cropland/Release/index.php"
	var uniqueLinks []string
	dList, _ := BreadthFirst(scrapeQueue, "/Users/thorbthorb/Downloads/scraped-data")
	for _, url := range dList {
		if Contains(url, scrapeQueue) == -1 {
			uniqueLinks = append(uniqueLinks, url)
			log.Println("to Download: ", url)
		}
	}
	log.Printf("Num New URLs: %d, %d", len(uniqueLinks), len(scrapeQueue))

}
