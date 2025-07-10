package downloader

import (
	"log"
	"net/url"
	"os"
	"path"
)

func DownloadBytes(data []byte, urlString string, downloadDir string) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		log.Printf("Error parsing URL: %v", err)
		return
	}
	filename := path.Base(parsedURL.Path)
	filepath := path.Join(downloadDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Printf("Error writing data: %v", err)
	}
}
