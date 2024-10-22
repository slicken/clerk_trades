package main

import (
	"clerk_trades/clerk"
	"clerk_trades/gemini"
	"clerk_trades/utils"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var links []string

func usage(code int) {
	fmt.Printf(`CLERK TRADES - U.S. Government Official Financial Report Tracker
Usage: %s [<num>] [<false|off>]

[Optional]
  num               List trades from last [num] reports
  false, off        Disable site scrape and exit after it printing trades
  help, -h, --help  Display this help menu
`, os.Args[0])
	os.Exit(code)
}

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
			usage(1)
		}
	}

	// add os.Args loop to capture args

	links, _ = utils.ReadJSON[[]string]("links.json")
	log.Printf("loaded %d reports.\n", len(links))

	// --

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := checkReports(); err != nil {
					log.Printf("Error: %v", err)
				}
			}
		}
	}()

	if err := checkReports(); err != nil {
		log.Printf("Error: %v", err)
	}

	if len(os.Args) > 2 {
		if os.Args[2] == "false" || os.Args[2] == "off" {
			return
		} else {
			usage(1)
		}
	}

	select {}
}

func checkReports() error {
	var err error
	var files []string

	if len(os.Args) > 2 {
		if os.Args[2] == "false" || os.Args[2] == "off" {
			log.Println("clerk disabled")
			files = links
		}
	} else {
		files, err = clerk.SearchSite(links)
		if err != nil {
			return err
		}
	}

	numFiles := len(links)
	if len(os.Args) > 1 {
		numFiles, err = strconv.Atoi(os.Args[1])
		if err != nil {
			log.Fatalf("Invalid number of files: %v", err)
		}
	}

	if len(files) > numFiles {
		files = files[len(files)-numFiles:]
	}

	// Store content in memory
	var fileContents [][]byte
	for _, file := range files {
		content, err := fetchFileContent(file)
		if err != nil {
			log.Printf("Failed to fetch content for file %s: %v", file, err)
			continue
		}
		fileContents = append(fileContents, content)
	}

	// Process the reports with the contents stored in memory
	gemini.ProsessRapports(fileContents, files)
	return err
}

// fetchFileContent retrieves the content of a file from a link
func fetchFileContent(link string) ([]byte, error) {
	resp, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch file: %s", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}
