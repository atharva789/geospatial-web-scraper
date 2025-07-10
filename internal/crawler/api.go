package crawler

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Replace this with actual download logic
func downloadFile(url, outputDir string, secure bool) error {
	filename := extractFilename(url)
	outputPath := fmt.Sprintf("%s/%s", strings.TrimRight(outputDir, "/"), filename)

	if secure {
		cmd := exec.Command(
			"firejail",
			"--private="+outputDir,
			"--net=none",
			"--caps.drop=all",
			"--seccomp",
			"--shell=none",
			"--quiet",
			"wget", "-O", outputPath, url,
		)
		return cmd.Run()
	} else {
		cmd := exec.Command("wget", "-O", outputPath, url)
		return cmd.Run()
	}
}

func extractFilename(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("file-%d.dat", rand.Intn(100000))
}

func main() {
	// Flags
	searchPtr := flag.String("s", "", "Search query for dataset. Required.")
	downloadDir := flag.String("download", "", "Directory to download datasets to. If empty, only prints URLs.")
	noSec := flag.Bool("nosec", false, "Disable security sandboxing (enabled by default).")

	flag.Parse()

	// Validate search query
	if strings.TrimSpace(*searchPtr) == "" {
		fmt.Println("ERROR: Search query (-s) is required.")
		flag.Usage()
		os.Exit(1)
	}

	var mg Manager
	// Begin search
	var downloadableLinks []string
	fmt.Printf("Searching for: \"%s\"\n", *searchPtr)

	if *downloadDir == "" {
		dir := ""
		mg = Manager{
			secure:       false,
			downloadPath: &dir,
			searchQuery:  searchPtr,
			downloadURLs: []string{},
			searchFrom:   PublicGeospatialDataSeedsMap,
			linkChan:     make(chan string, 1),
			smTokens:     make(chan struct{}, 40),
			dlTokens:     make(chan struct{}, 40),
			worklist:     make(chan []WebNode),
			done:         make(chan bool),
		}
		downloadableLinks = mg.findLinks()
		fmt.Println("Found URLs:")
		for _, link := range downloadableLinks {
			fmt.Println(link)
		}
		return
	}

	// Create output directory if not exists
	if _, err := os.Stat(*downloadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(*downloadDir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", *downloadDir, err)
		}
	}

	// Download files
	fmt.Printf("Downloading %d files to %s (security: %v)\n", len(downloadableLinks), *downloadDir, !*noSec)
	for _, link := range downloadableLinks {
		fmt.Printf("Downloading %s...\n", link)
		err := downloadFile(link, *downloadDir, !*noSec)
		if err != nil {
			log.Printf("Failed to download %s: %v", link, err)
		} else {
			fmt.Printf("Downloaded: %s\n", link)
		}
	}
}
