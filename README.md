# clerk_trades - Unofficial\t !!UNDER WORK!!
This program lists trades made by U.S. government officials.
The financial reports official documents that the program scape fom [https://disclosures-clerk.house.gov/FinancialDisclosure](https://disclosures-clerk.house.gov/FinancialDisclosure) and then read by Google Gemini.

## Prepare
edit email/email.go and set your mailgun domain, if you want to use email future
<br>
## Usage
```bash
me@pc:~$ ./clerk --help
CLERK TRADES - U.S. Government Official Financial Report Tracker
Usage: ./app [<update_duration> | <list>]

Arguments:
  update_duration    Clerk site scrape duration, min 1h (e.g. 12h, 1d).
                     If not provided, site scraping will be disabled.
  list               Specify the number of reports to list their trades.
                     (type=int). This argument must be greater than 0.
                     If used, the program will exit after printing.

Note: Only one of these two arguments may be provided at a time.

OPTIONS:
  -e=<your@email.com>, --email=<your@email.com>
                     Email trades result to specified email address.
                     You will recive a email first where you must give mailgun
                     permission to send email to email adress.
  -v, --verbose      Enable verbose output for detailed logging and information.
  -h, --help         Display this help menu.
```

