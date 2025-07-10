package downloader

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	urlFlag := flag.String("u", "", "pass a valid direct-download URL")
	bytesFlag := flag.String("b", "", "pass bytes")
	dirFlag := flag.String("d", "", "the directory the downloads will go to")

	flag.Parse()

	if *dirFlag == "" || *urlFlag == "" || *bytesFlag == "" {
		fmt.Println("ERROR: Bytes, url, and download directories must be specified. Exiting")
		flag.Usage()
		os.Exit(1)
	}

	byteData := []byte(*bytesFlag)
	DownloadBytes(byteData, *urlFlag, *dirFlag)

}
