package main

import (
	"clerk_trades/clerk"
	"clerk_trades/email"
	"clerk_trades/gemini"
	"clerk_trades/utils"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func usage(code int) {
	fmt.Printf(`CLERK TRADES - U.S. Government Official Financial Report Tracker
Usage: %s [<ticker_duration> | <list>] [OPTIONS]

Arguments:
  ticker_duration     Duration for the application ticker to check for new
                      reports on Clerk website. Minimum 3h (e.g. 24h, 72h).
                      Only accepts 'h' for hours before the integer.
                      If not specified, it will not check for new reports.
  list                Specify the number of reports to list their trades.
                      (type=int). This argument must be betweengreater than
                      0 but less that 6.
                      If used, the program will exit after printing.

Note: Only one of these two arguments may be provided at a time.

OPTIONS:
  -n, --name <name>   List reports of a specific individual.
  -e, --email         Enable email notifications for trade results via Mailgun. 
                      Configure settings in 'gunmail.config' to activate.
  --log               Save logs to file.
  -v, --verbose       Enable verbose output for detailed logging and information.
  -h, --help          Display this help menu.
`, os.Args[0])
	os.Exit(code)
}

var (
	verbose bool
	mail    bool
	name    string
)

func main() {
	var update time.Duration
	var listReports int

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		switch {
		case arg == "-h" || arg == "--help" || arg == "help":
			usage(0)

		case arg == "-v" || arg == "--verbose" || arg == "verbose":
			verbose = true

		case arg == "--log":
			logName := time.Now().Format("01021504") + ".log"
			logFile, err := os.Create(logName)
			if err != nil {
				log.Fatalf("could not create logfile %q: %v", logFile.Name(), err)
			}
			log.SetOutput(io.MultiWriter(os.Stderr, logFile))
			log.Printf("successfully created logfile %q.\n", logFile.Name())

		case arg == "-e" || arg == "--email" || arg == "email":
			log.Printf("loading Mailgun settings..")
			err := email.LoadMailGun()
			if err != nil {
				log.Fatalln(err)
			}
			mail = true
			log.Printf("results will be sent to %v\n", email.Mailgun.EmailTo)

		case strings.HasPrefix(arg, "-n") || strings.HasPrefix(arg, "--name"):
			if strings.Contains(arg, "=") {
				name = strings.SplitN(arg, "=", 2)[1]
			} else if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				name = os.Args[i+1]
				i++
			} else {
				log.Fatalln("error: -n flag requires a name.")
			}

		default:
			if n, err := strconv.Atoi(arg); err == nil && n > 0 && n < 6 {
				listReports = n
			} else if duration, err := parseCustomDuration(arg); err == nil {
				if duration < 3*time.Hour {
					log.Fatalln("error: minimum duration must be 3h.")
				}
				update = duration
			} else {
				if name != "" {
					continue
				}
				log.Fatalln("error: invalid argument:", arg)
			}
		}
	}

	if name != "" {
		update = time.Duration(24 * time.Hour)
		listReports = 0

		if err := copyFile(clerk.FILE_LINKS, clerk.FILE_BACKUP); err != nil {
			log.Println("error copying file:", err)
		}
		if err := os.Remove(clerk.FILE_LINKS); err != nil {
			log.Println("error removing file:", err)
		}
		go HandleInterrupt()
	}

	if update == 0 && (listReports > 5 || listReports <= 0) || update != 0 && listReports != 0 {
		usage(1)
	}

	if verbose {
		log.Println("verbose is active.")
		gemini.SetVerbose(true)
		clerk.SetVerbose(true)
	}

	if update != 0 {
		log.Printf("ticker scheduled to check for new reports every %s.\n", fmt.Sprintf("%.0fh", update.Hours()))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ticker := time.NewTicker(update)
		defer ticker.Stop()

		go func() {
			for {
				select {
				case <-ctx.Done():
					if verbose {
						log.Println("ticker stopped.")
					}
					return
				case <-ticker.C:
					if err := checkReports(update, listReports, name); err != nil {
						log.Println("error:", err)
					}
				}
			}
		}()
	}

	if err := checkReports(update, listReports, name); err != nil {
		log.Println("error:", err)
	}

	if update == 0 {
		if verbose {
			log.Println("ticker is disabled. program exit.")
		}
		return
	}

	select {}
}

func checkReports(update time.Duration, listReports int, name string) error {
	var err error
	var files []string
	var links []string
	var loop bool

	links, _ = utils.ReadJSON[[]string](clerk.FILE_LINKS)
	if verbose {
		log.Printf("loaded %d reports.\n", len(links))
	}

	if update != 0 {
		if name != "" {
			log.Printf("checking for new reports of %s.\n", name)
		} else {
			log.Println("checking for new reports.")
		}
		files, err, loop = clerk.SiteCheck(links, name)
		if err != nil {
			return err
		}
	} else {
		if len(links) == 0 {
			log.Fatalln("no report links stored. run program with updater first, to links from clerk site.")
		}
		files = links
		if listReports > 0 {
			if len(files)-1 >= listReports {
				files = files[len(files)-listReports:] // Keep only the last listReports files
			}
		}
	}

	if len(files) > 0 {
		log.Printf("allocating space for %d report in memory.\n", len(files))
	}

	var fileContent [][]byte
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, file := range files {

		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			content, err := fetchFileContent(file)
			if err != nil {
				log.Printf("failed to fetch content for report %s: %v\n", file, err)
				return
			}

			mu.Lock()
			fileContent = append(fileContent, content)
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	if len(fileContent) == 0 {
		log.Println("nothing new to process.")
		return err
	}

	// Process reports
	trades, err := gemini.ProsessReports(fileContent, files)
	if err != nil {
		return err
	}

	if mail {
		emailBody, err := email.GenerateEmailBody(trades)
		if err != nil {
			return err
		}
		if email.Mailgun.Paid {
			if err := email.SendHTMLToMailingList(emailBody); err != nil {
				return err
			}
		} else {
			if err := email.SendHTMLTo(emailBody); err != nil {
				return err
			}
		}
		if verbose {
			log.Println("trade reports have been e-mailed.")
		}
	}

	if loop {
		defer func() {
			go checkReports(update, listReports, name)
		}()
	}

	return nil
}

func fetchFileContent(link string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.5938.62 Safari/537.36")

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

func parseCustomDuration(input string) (time.Duration, error) {
	if strings.HasSuffix(input, "h") {
		hours := strings.TrimSuffix(input, "h")
		if n, err := strconv.Atoi(hours); err == nil {
			return time.Duration(n) * time.Hour, nil
		}
	}
	return 0, fmt.Errorf("invalid duration format; only hours (h) are accepted")
}

func HandleInterrupt() {
	closeChan := make(chan os.Signal, 1)
	signal.Notify(closeChan, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	s := <-closeChan
	log.Printf("\n%+v recived. shutting down.\n", s)

	if err := copyFile(clerk.FILE_BACKUP, clerk.FILE_LINKS); err != nil {
		log.Println("error copying file:", err)
	}
	if err := os.Remove(clerk.FILE_BACKUP); err != nil {
		log.Println("error removing file:", err)
	}

	os.Exit(0)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
