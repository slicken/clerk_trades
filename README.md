# clerk_trades - Unofficial\t !!UNDER WORK!!
This program lists trades made by U.S. government officials.
The financial reports official documents that the program scape fom [https://disclosures-clerk.house.gov/FinancialDisclosure](https://disclosures-clerk.house.gov/FinancialDisclosure) and then read by Google Gemini.

## Usage
```bash
me@pc:~$ ./clerk --help
CLERK TRADES - U.S. Government Official Financial Report Tracker
Usage: ./app [<update_duration> | <list>]

Arguments:
  <update_duration>    Duration to update (e.g., 3h, 2m, 1s). If not provided,
                       site scraping will be disabled.
  <list>               Specify the number of reports to list trades from 
                       (type=int). If this argument is used, the program 
                       will exit after printing. (This value can not be 0)

  help, -h, --help     Display this help menu.
```

