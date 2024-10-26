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
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

var links []string

func usage(code int) {
	fmt.Printf(`CLERK TRADES - U.S. Government Official Financial Report Tracker
Usage: %s [<update_duration> | <list>]

Arguments:
  <update_duration>    Clerk site scrape duration, min 1h (e.g. 12h, 1d).
                       If not provided, site scraping will be disabled.
  <list>               Specify the number of reports to list their trades.
                       (type=int). This argument must be greater than 0.
                       If used, the program will exit after printing.

Note: Only one argument may be provided at a time.

  help, -h, --help     Display this help menu.
`, os.Args[0])
	os.Exit(code)
}

func main() {
	var update time.Duration
	var listReports int

	for _, v := range os.Args[1:] {
		switch v {
		case "help", "--help", "-h":
			usage(0)
		default:
			if _, err := time.ParseDuration(v); err == nil {
				update, _ = time.ParseDuration(v)
			} else if n, err := strconv.Atoi(v); err == nil {
				listReports = n
			}
		}
	}
	if update == 0 && listReports == 0 || update != 0 && listReports != 0 {
		usage(1)
	}
	// fmt.Println("update", update)
	// fmt.Println("listReports", listReports)
	// os.Exit(0)
	// load stored links and trades
	links, _ = utils.ReadJSON[[]string](clerk.FILE_LINKS)
	log.Printf("loaded %d reports.\n", len(links))
	gemini.Trades, _ = utils.ReadJSON[[]gemini.Trade](gemini.FILE_TRADES)
	log.Printf("loaded %d trades.\n", len(gemini.Trades))

	if update != 0 {
		log.Printf("checking for new reports every %v.\n", update)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ticker := time.NewTicker(update)
		defer ticker.Stop()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := checkReports(update, listReports); err != nil {
						log.Printf("Error: %v", err)
					}
				}
			}
		}()
	}

	if err := checkReports(update, listReports); err != nil {
		log.Printf("Error: %v", err)
	}

	if update == 0 {
		return
	}

	select {}
}

func checkReports(update time.Duration, listReports int) error {
	var err error
	var files []string

	if update != 0 {
		log.Println("checking for new reports.")
		files, err = clerk.SiteScrape(links)
		if err != nil {
			return err
		}
	} else {
		files = links
	}

	if listReports > 0 {
		if len(files) > listReports {
			files = files[len(files)-listReports:] // Keep only the last listReports files
		}
	}

	// store content concurrently in memory
	var fileContents [][]byte
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, file := range files {

		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			content, err := fetchFileContent(file)
			if err != nil {
				log.Printf("Failed to fetch content for file %s: %v", file, err)
				return
			}

			log.Printf("stored %s in memory\n", filepath.Base(file))

			mu.Lock()
			fileContents = append(fileContents, content)
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	if len(fileContents) == 0 {
		return nil
	}
	return gemini.ProsessRapports(fileContents, files)
}

func fetchFileContent(link string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	// set User-Agent to mimic Google Chrome
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch file: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
