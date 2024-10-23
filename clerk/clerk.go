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
	URL    = "https://disclosures-clerk.house.gov/"
	SEARCH = "FinancialDisclosure#Search"
	pass   = "financial-pdfs"

	FILE_LINKS = "links.json"
)

var newLinks []string

// SearchSite navigates the clerk site, searches for filings from the current year, and returns the file links.
func SearchSite(links []string) ([]string, error) {
	var err error

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

	// click search form
	if err := page.Click(`button[aria-label="search button"]`); err != nil {
		return nil, fmt.Errorf("could not click search button: %v", err)
	}

	// wait for the results
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

	// loop throu all pages and look for new entries
	log.Printf("looking for new reports on %d pages of financial documents...\n", pageCount)
	for n := 1; n <= pageCount; n++ {
		if _, err := page.WaitForSelector(`#DataTables_Table_0 tbody tr`, playwright.PageWaitForSelectorOptions{
			State: playwright.WaitForSelectorStateVisible,
		}); err != nil {
			return nil, fmt.Errorf("could not wait for table rows on page %d: %v", n, err)
		}

		rows, err := page.QuerySelectorAll(`#DataTables_Table_0 tbody tr`)
		if err != nil {
			return nil, fmt.Errorf("could not query table rows: %v", err)
		}

		for _, row := range rows {
			linkElement, err := row.QuerySelector(`td.memberName a`)
			if err != nil {
				return nil, fmt.Errorf("could not find link in row: %v", err)
			}

			href, err := linkElement.GetAttribute("href")
			if err != nil {
				return nil, fmt.Errorf("could not get href attribute: %v", err)
			}

			if href == "" || strings.Contains(href, pass) {
				continue
			}

			// add is repost is new
			if utils.Contains(links, URL+href) {
				continue
			}
			links = append(links, URL+href)
			newLinks = append(newLinks, URL+href)
			fmt.Println(URL + href)
		}

		paginationButtons, err := page.QuerySelectorAll(`.paginate_button:not(.ellipsis):not(.next)`)
		if err != nil {
			return nil, fmt.Errorf("could not find pagination buttons: %v", err)
		}

		for _, button := range paginationButtons {
			buttonText, err := button.InnerText()
			if err != nil {
				return nil, fmt.Errorf("could not get button text: %v", err)
			}

			if buttonText == fmt.Sprintf("%d", n+1) {
				if err := button.Click(); err != nil {
					return nil, fmt.Errorf("could not click next page button: %v", err)
				}
				break
			}
		}
	}

	// cocurrently - does not work
	// links, newLinks, err := scrapePagesConcurrently(page, pageCount, pass, links)
	// if err != nil {
	// 	return nil, fmt.Errorf("scrapePage: %v", err)
	// }

	err = utils.WriteJSON[[]string](FILE_LINKS, links)
	if err != nil {
		return links, err
	}
	log.Printf("updated %q, contains %d reports.\n", FILE_LINKS, len(links))

	// return only new links, if any
	if len(newLinks) > 0 {
		return newLinks, nil
	}
	// or old links
	return links, nil
}

func scrapePagesConcurrently(page playwright.Page, pageCount int, pass string, links []string) ([]string, []string, error) {
	var newLinks []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for n := 1; n <= pageCount; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			// Wait for the table rows to be visible
			if _, err := page.WaitForSelector(`#DataTables_Table_0 tbody tr`, playwright.PageWaitForSelectorOptions{
				State: playwright.WaitForSelectorStateVisible,
			}); err != nil {
				fmt.Printf("could not wait for table rows on page %d: %v\n", n, err)
				return
			}

			// Query the rows
			rows, err := page.QuerySelectorAll(`#DataTables_Table_0 tbody tr`)
			if err != nil {
				fmt.Printf("could not query table rows on page %d: %v\n", n, err)
				return
			}

			for _, row := range rows {
				linkElement, err := row.QuerySelector(`td.memberName a`)
				if err != nil {
					fmt.Printf("could not find link in row on page %d: %v\n", n, err)
					continue
				}

				href, err := linkElement.GetAttribute("href")
				if err != nil {
					fmt.Printf("could not get href attribute on page %d: %v\n", n, err)
					continue
				}

				if href == "" || strings.Contains(href, pass) {
					continue
				}

				// Check if the link is new
				mu.Lock()
				if !utils.Contains(links, URL+href) {
					links = append(links, URL+href)
					newLinks = append(newLinks, URL+href)
					fmt.Println(URL + href)
				}
				mu.Unlock()
			}

			// Handle pagination
			paginationButtons, err := page.QuerySelectorAll(`.paginate_button:not(.ellipsis):not(.next)`)
			if err != nil {
				fmt.Printf("could not find pagination buttons on page %d: %v\n", n, err)
				return
			}

			for _, button := range paginationButtons {
				buttonText, err := button.InnerText()
				if err != nil {
					fmt.Printf("could not get button text on page %d: %v\n", n, err)
					continue
				}

				if buttonText == fmt.Sprintf("%d", n+1) {
					if err := button.Click(); err != nil {
						fmt.Printf("could not click next page button on page %d: %v\n", n, err)
					}
					break
				}
			}
			time.Sleep(100 * time.Microsecond)
		}(n)
	}

	wg.Wait()
	return links, newLinks, nil
}
