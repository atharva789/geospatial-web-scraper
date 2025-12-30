package crawler

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// TestIndexFTP is a standalone function to test the IndexFTP function with a given URL.
// It fetches the URL, parses the HTML, and runs IndexFTP on it.
// Returns the FTPDirectory result and any error encountered.
func TestIndexFTP(url string) (*FTPDirectory, error) {
	fmt.Printf("Testing IndexFTP with URL: %s\n", url)

	// Fetch the URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Parse the HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Run IndexFTP
	result, err := IndexFTP(doc, resp)
	if err != nil {
		return result, fmt.Errorf("IndexFTP error: %w", err)
	}

	return result, nil
}

// TestIndexS3 tests the IndexS3 function with a given S3 URL.
// Returns the FTPDirectory result and any error encountered.
func TestIndexS3(s3URL string) (*FTPDirectory, error) {
	fmt.Printf("Testing IndexS3 with URL: %s\n", s3URL)
	return IndexS3(s3URL)
}

// IsS3URL checks if the given URL is an S3 bucket URL.
func IsS3URL(urlStr string) bool {
	return strings.Contains(urlStr, ".s3.") || strings.Contains(urlStr, "s3.amazonaws.com")
}

// PrintFTPDirectory is a helper function to pretty-print the FTPDirectory structure
func PrintFTPDirectory(dir FTPDirectory, indent string) {
	fmt.Printf("%sDownload Files (%d):\n", indent, len(dir.DownloadFiles))
	for i, file := range dir.DownloadFiles {
		fmt.Printf("%s  [%d] URL: %s\n", indent, i, file.URL)
		if file.Metadata != "" {
			fmt.Printf("%s      Metadata: %s\n", indent, file.Metadata)
		} else {
			fmt.Printf("%s      Metadata: (none)\n", indent)
		}
	}

	if len(dir.SubDirectories) > 0 {
		fmt.Printf("%sSubdirectories (%d):\n", indent, len(dir.SubDirectories))
		for i, subdir := range dir.SubDirectories {
			fmt.Printf("%s  [%d]:\n", indent, i)
			PrintFTPDirectory(*subdir, indent+"    ")
		}
	}
}
