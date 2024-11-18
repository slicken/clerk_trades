# clerk_trades - Unofficial !!UNDER WORK!!
This program lists trades made by U.S. government officials.
The financial reports official documents that the program scape fom [https://disclosures-clerk.house.gov/FinancialDisclosure](https://disclosures-clerk.house.gov/FinancialDisclosure) and then read by Google Gemini.

## Prepare
install the package Playwright browsers and OS dependencies
```
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps
# Or
go install github.com/playwright-community/playwright-go/cmd/playwright@latest
playwright install --with-deps
```
if you want the trades to be email to you and your friends you can create a free gunmail account
on www.gunmail.com.
edit the gunmail.config file to enable this future
<br>

## Usage
```
me@pc:~$ ./clerk --help
CLERK TRADES - U.S. Government Official Financial Report Tracker
Usage: %s [<ticker_duration> | <list>] [OPTIONS]

Arguments:
  ticker_duration    Duration for the application ticker to check for new
                     reports on Clerk website. Minimum 3h (e.g. 24h, 72h).
                     Only accepts 'h' for hours before the integer.
                     If not specified, it will not check for new reports.
  list               Specify the number of reports to list their trades.
                     (type=int). This argument must be greater than
                     0 but less that 6.
                     If used, the program will exit after printing.

Note: Only one of these two arguments may be provided at a time.

OPTIONS:
  -e, --email        Enable email notifications for trade results via Mailgun. 
                     Configure settings in 'gunmail.config' to activate.
  --log              Save logs to file.
  -v, --verbose      Enable verbose output for detailed logging and information.
  -h, --help         Display this help menu.
```
<br>
