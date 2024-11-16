package clerk

import (
	"clerk_trades/utils"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

const (
	URL        = "https://disclosures-clerk.house.gov/"
	SEARCH     = "FinancialDisclosure#Search"
	pass       = "financial-pdfs"
	FILE_LINKS = "links.json"
)

var verbose bool

func SetVerbose(v bool) {
	verbose = v
}

func SiteCheck(links []string) ([]string, error) {
	var newLinks []string

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

	// scrape links
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		// Wait for the results table to be visible before scraping
		if _, err := page.WaitForSelector(`#DataTables_Table_0`, playwright.PageWaitForSelectorOptions{
			State: playwright.WaitForSelectorStateVisible,
		}); err != nil {
			return nil, fmt.Errorf("failed to wait for results table on page %d: %v", pageNum, err)
		}

		// Scrape the rows
		rows, err := page.QuerySelectorAll(`#DataTables_Table_0 tbody tr`)
		if err != nil {
			return nil, fmt.Errorf("failed to query table rows on page %d: %v", pageNum, err)
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

			if !utils.Contains(links, URL+href) {
				newLinks = append(newLinks, URL+href)
				log.Println(URL + href)
			}
		}

		if pageNum >= pageCount {
			break
		}

		// get next page data
		next := pageNum + 1
		dataDtIdxLocator := page.Locator(fmt.Sprintf(".paginate_button:has-text('%d')", next)).GetByText(fmt.Sprintf("%d", next), playwright.LocatorGetByTextOptions{
			Exact: playwright.Bool(true),
		})

		dataDtIdxText, err := dataDtIdxLocator.GetAttribute("data-dt-idx")
		if err != nil {
			log.Printf("failed to get data-dt-idx attribute for page %d button: %v", next, err)
			break
		}

		nextPageButtonLocator := page.Locator(fmt.Sprintf(".paginate_button[data-dt-idx='%s']", dataDtIdxText))
		if err := nextPageButtonLocator.Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(60000), // 60 seconds timeout
		}); err != nil {
			log.Printf("failed to click next page button on page %d: %v", next, err)
			break
		}
	}

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
