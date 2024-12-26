package gemini

import (
	"clerk_trades/utils"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Trade struct {
	Name   string `json:"Name"`
	Asset  string `json:"Asset"`
	Ticker string `json:"Ticker"`
	Type   string `json:"Type"`
	Date   string `json:"Date"`
	Filed  string `json:"Filed"`
	Amount string `json:"Amount"`
	Cap    bool   `json:"Cap"`
}

var verbose bool

func SetVerbose(v bool) {
	verbose = v
}

func ProsessReports(fileContents [][]byte, links []string) ([]Trade, error) {
	var Trades []Trade
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatalln("error: environment variable GEMINI_API_KEY not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to creating client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	//model := client.GenerativeModel("gemini-2.0-flash-exp")
	model.SetTemperature(0)
	model.SetTopP(0)
	model.SetTopK(0)
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = genai.NewUserContent(genai.Text(`
It should read data from the PDF file and write data into the JSON array described below with some rules.
Rule1: Name can be obtained under Filer Information. Input First Name and Last Name only! Dont include "Hon.", "Mrs", "Mr", etc.
Rule2: in Type field (Transaction Type): if "P" input "Purchase", if "S" input "Sale".
[
	{
		"Name": "input First Name and Last Name only",
		"Asset": "input Full Asset Name",
		"Ticker": "input Ticker for the Asset",
		"Type": "input Transaction Type",
		"Date": "input Date",
		"Filed": "input Date under Notification Date",
		"Amount": "input Amount",
		"Cap":  True or False (boolean)
	}
]
`))

	var parts []genai.Part
	parts = append(parts, genai.Text("create JSON with the very important instructions"))
	for _, data := range fileContents {
		parts = append(parts, genai.Blob{
			MIMEType: "application/pdf",
			Data:     data,
		})
	}

	log.Printf("processing trade report...")

	resp, err := model.GenerateContent(ctx, parts...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %v", err)
	}

	out := getResponse(resp)
	if len(out) == 0 {
		return nil, fmt.Errorf("no output data from gemini")
	}
	if err := utils.SafeUnmarshal(out, &Trades); err != nil {
		log.Fatalf("safe unmarshal failed: %v", err)
	}

	// print trades
	strTrades := PrintTrades(Trades)
	log.Print("\r\n", strTrades)

	// Trades = checkTrades(Trades)
	if verbose {
		log.Printf("%d trades in %d reports.\n", len(Trades), len(links))
	}

	return Trades, nil
}

func getResponse(resp *genai.GenerateContentResponse) string {
	var str string
	for _, c := range resp.Candidates {
		if c.Content != nil {
			for _, part := range c.Content.Parts {
				str += fmt.Sprint(part)
			}
		}
	}
	return str
}

func PrintTrades(trades []Trade) string {
	output := "\n"
	for _, trade := range trades {
		output += fmt.Sprintf("Name:    %-20s\n", trade.Name)
		output += fmt.Sprintf("Asset:   %-20s\n", trade.Asset)
		output += fmt.Sprintf("Ticker:  %-20s\n", trade.Ticker)
		output += fmt.Sprintf("Type:    %-20s\n", trade.Type)
		output += fmt.Sprintf("Date:    %-20s\n", trade.Date)
		output += fmt.Sprintf("Filed:   %-20s\n", trade.Filed)
		output += fmt.Sprintf("Amount:  %-20s\n", trade.Amount)
		output += fmt.Sprintf("Cap:     %-20v\n\n", trade.Cap)
	}
	return output
}

// func checkTrades(Trades []Trade) []Trade {
// 	var count int
// 	var trades []Trade

// 	for _, newTrade := range Trades {
// 		// empty fileds are not accepted
// 		if newTrade.Ticker == "" {
// 			count++
// 			continue
// 		}
// 		if newTrade.Type == "" {
// 			count++
// 			continue
// 		}
// 		if newTrade.Date == "" {
// 			count++
// 			continue
// 		}
// 		if newTrade.Filed == "" {
// 			count++
// 			continue
// 		}
// 		trades = append(trades, newTrade)
// 	}

// 	if count == 0 {
// 		return Trades
// 	}

// 	if verbose {
// 		log.Printf("removed 3 trades has bad gemini data.\n")
// 	}

// 	return trades
// }

// func hasMatchingWord(new, old string) bool {
// 	if new == "" || old == "" {
// 		return false
// 	}
// 	newWords := strings.Fields(new)
// 	existingWords := strings.Fields(old)

// 	for _, newWord := range newWords {
// 		for _, existingWord := range existingWords {
// 			if newWord == existingWord {
// 				return true
// 			}
// 		}
// 	}
// 	return false
// }
