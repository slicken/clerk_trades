package clerk

import (
	"clerk_trades/utils"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

const (
	URL        = "https://disclosures-clerk.house.gov/"
	SEARCH     = "FinancialDisclosure#Search"
	pass       = "financial-pdfs"
	FILE_LINKS = "links.json"
)

var (
	newLinks []string
	verbose  bool
)

func SetVerbose(v bool) {
	verbose = v
}

func SiteCheck(links []string) ([]string, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start Playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %v", err)
	}

	_, err = page.Goto(URL + SEARCH)
	if err != nil {
		return nil, fmt.Errorf("failed to go to URL: %v", err)
	}

	// select the current year
	thisYear := fmt.Sprintf("%d", time.Now().Year())
	selectYear := page.Locator("#FilingYear")
	_, err = selectYear.SelectOption(playwright.SelectOptionValues{
		Values: &[]string{thisYear},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to select Filing Year %s: %v", thisYear, err)
	}

	// click search form and wait for result table
	if err := page.Click(`button[aria-label="search button"]`); err != nil {
		return nil, fmt.Errorf("failed to click search button: %v", err)
	}
	if _, err = page.WaitForSelector(`#DataTables_Table_0`, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return nil, fmt.Errorf("failed to wait for results table to load: %v", err)
	}

	// get number of pages
	lastPaginationButtonText, err := page.Locator(`.paginate_button:not(.ellipsis):not(.next):last-child`).InnerText()
	if err != nil {
		return nil, fmt.Errorf("failed to find the last pagination button: %v", err)
	}
	pageCount, err := strconv.Atoi(lastPaginationButtonText)
	if err != nil {
		return nil, fmt.Errorf("failed to convert page count to integer: %v", err)
	}

	if verbose {
		log.Printf("looking through %d pages.\n", pageCount)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for n := 1; n <= pageCount; n++ {
		wg.Add(1)
		go func(pageNum int) {
			defer wg.Done()
			err := scrapeLinks(pageNum, page, &mu, links, newLinks)
			if err != nil && verbose {
				log.Printf("failed to get data from page %d: %v", pageNum, err)
			}
		}(n)
	}

	wg.Wait()

	if len(newLinks) != 0 {
		links = append(links, newLinks...)
		err = utils.WriteJSON[[]string](FILE_LINKS, links)
		if err != nil {
			return links, err
		}
		log.Printf("updated %s. contains %d reports.\n", FILE_LINKS, len(links))
	}

	return newLinks, nil
}

// scrapeLinks scrapes the links from a specific page.
func scrapeLinks(pageNum int, page playwright.Page, mu *sync.Mutex, links []string, newLinks []string) error {
	_, err := page.Goto(fmt.Sprintf("%s%s&page=%d", URL, SEARCH, pageNum))
	if err != nil {
		return fmt.Errorf("failed to go to page %d: %v", pageNum, err)
	}

	// wait for the results to load
	if _, err := page.WaitForSelector(`#DataTables_Table_0`, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return fmt.Errorf("failed to wait for results table on page %d: %v", pageNum, err)
	}

	rows, err := page.QuerySelectorAll(`#DataTables_Table_0 tbody tr`)
	if err != nil {
		return fmt.Errorf("failed to query table rows on page %d: %v", pageNum, err)
	}

	for _, row := range rows {
		linkElement, err := row.QuerySelector(`td.memberName a`)
		if err != nil {
			log.Printf("failed to find link in row on page %d: %v", pageNum, err)
			continue
		}

		href, err := linkElement.GetAttribute("href")
		if err != nil {
			log.Printf("failed to get href attribute on page %d: %v", pageNum, err)
			continue
		}

		if href == "" || strings.Contains(href, pass) {
			continue
		}

		mu.Lock()
		if !utils.Contains(links, URL+href) {
			newLinks = append(newLinks, URL+href)
			log.Println(URL + href)
		}
		mu.Unlock()
	}

	return nil
}
