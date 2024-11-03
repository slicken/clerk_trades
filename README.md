# clerk_trades - Unofficial !!UNDER WORK!!
This program lists trades made by U.S. government officials.
The financial reports official documents that the program scape fom [https://disclosures-clerk.house.gov/FinancialDisclosure](https://disclosures-clerk.house.gov/FinancialDisclosure) and then read by Google Gemini.

## Prepare
edit email/email.go and set your mailgun domain, if you want to use email future
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
                     (type=int). This argument must be greater than 0.
                     If used, the program will exit after printing.

Note: Only one of these two arguments may be provided at a time.

OPTIONS:
  -e=<your@email.com>, --email=<your@email.com>
                     Email trades result to specified email address.
                     You will recive a email first where you must give mailgun
                     permission to send email to email adress.
  --log              Save logs to file.
  -v, --verbose      Enable verbose output for detailed logging and information.
  -h, --help         Display this help menu.
```
<br>
## Todo
implement mailgun.config file where you can set your domain name and maby api key and
email adresses to mail when we find new trades.

