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

var newLinks []string

// SiteScrape
func SiteScrape(links []string) ([]string, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start Playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %v", err)
	}

	_, err = page.Goto(URL + SEARCH)
	if err != nil {
		return nil, fmt.Errorf("could not go to URL: %v", err)
	}

	// Select the current year
	thisYear := fmt.Sprintf("%d", time.Now().Year())
	selectYear := page.Locator("#FilingYear")
	_, err = selectYear.SelectOption(playwright.SelectOptionValues{
		Values: &[]string{thisYear},
	})
	if err != nil {
		return nil, fmt.Errorf("could not select Filing Year %s: %v", thisYear, err)
	}

	// click search form and wait for result table
	if err := page.Click(`button[aria-label="search button"]`); err != nil {
		return nil, fmt.Errorf("could not click search button: %v", err)
	}
	if _, err = page.WaitForSelector(`#DataTables_Table_0`, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return nil, fmt.Errorf("could not wait for results table to load: %v", err)
	}

	// get number of pages
	lastPaginationButtonText, err := page.Locator(`.paginate_button:not(.ellipsis):not(.next):last-child`).InnerText()
	if err != nil {
		return nil, fmt.Errorf("could not find the last pagination button: %v", err)
	}
	pageCount, err := strconv.Atoi(lastPaginationButtonText)
	if err != nil {
		return nil, fmt.Errorf("could not convert page count to integer: %v", err)
	}

	// log.Printf("looking through %d pages.", pageCount)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for n := 1; n <= pageCount; n++ {
		wg.Add(1)
		go func(pageNum int) {
			defer wg.Done()
			err := scrapeLinks(pageNum, page, &mu, links, newLinks)
			if err != nil {
				log.Printf("could not scrape %d: %v", pageNum, err)
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
		log.Printf("updated %q, contains %d reports.\n", FILE_LINKS, len(links))
	}

	return newLinks, nil
}

// scrapeLinks scrapes the links from a specific page.
func scrapeLinks(pageNum int, page playwright.Page, mu *sync.Mutex, links []string, newLinks []string) error {
	_, err := page.Goto(fmt.Sprintf("%s%s&page=%d", URL, SEARCH, pageNum))
	if err != nil {
		return fmt.Errorf("could not go to page %d: %v", pageNum, err)
	}

	// Wait for the results to load
	if _, err := page.WaitForSelector(`#DataTables_Table_0`, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return fmt.Errorf("could not wait for results table on page %d: %v", pageNum, err)
	}

	rows, err := page.QuerySelectorAll(`#DataTables_Table_0 tbody tr`)
	if err != nil {
		return fmt.Errorf("could not query table rows on page %d: %v", pageNum, err)
	}

	// fmt.Print(".")
	for _, row := range rows {
		linkElement, err := row.QuerySelector(`td.memberName a`)
		if err != nil {
			log.Printf("could not find link in row on page %d: %v", pageNum, err)
			continue
		}

		href, err := linkElement.GetAttribute("href")
		if err != nil {
			log.Printf("could not get href attribute on page %d: %v", pageNum, err)
			continue
		}

		if href == "" || strings.Contains(href, pass) {
			continue
		}

		mu.Lock()
		if !utils.Contains(links, URL+href) {
			newLinks = append(newLinks, URL+href)
			fmt.Println(URL + href)
		}
		mu.Unlock()
	}

	return nil
}
