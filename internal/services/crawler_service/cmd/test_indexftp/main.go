package main

import (
	"fmt"
	"os"

	"crawler_service/internal/crawler"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <URL>")
		fmt.Println("Example: go run main.go https://example.com/ftp/")
		fmt.Println("Example: go run main.go https://bucket.s3.amazonaws.com/prefix/")
		os.Exit(1)
	}

	url := os.Args[1]

	var result *crawler.FTPDirectory
	var err error

	// Auto-detect S3 URLs and use appropriate indexer
	if crawler.IsS3URL(url) {
		fmt.Println("Detected S3 URL, using IndexS3...")
		result, err = crawler.TestIndexS3(url)
	} else {
		fmt.Println("Using IndexFTP...")
		result, err = crawler.TestIndexFTP(url)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Index Result ===")
	crawler.PrintFTPDirectory(*result, "")
	fmt.Println("\n=== Test completed successfully ===")
}
