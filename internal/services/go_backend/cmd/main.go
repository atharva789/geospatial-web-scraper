package main

import "geospatial-web-scraper/internal/services/go_backend/internal/crawler"

// main is the entry point for the command line tool. It delegates all work to
// the crawler package.
func main() {
	crawler.Run()
}
