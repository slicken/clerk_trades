package gemini

import (
	"clerk_trades/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const FILE_TRADES = "trades.json"

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

var (
	Trades    []Trade
	newTrades []Trade
	verbose   bool
)

func SetVerbose(v bool) {
	verbose = v
}

func ProsessReports(fileContents [][]byte, links []string) ([]Trade, error) {
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
	if err := json.Unmarshal([]byte(out), &newTrades); err != nil {
		return nil, fmt.Errorf("failed to unmarshalling JSON: %v, output: %s", err, out)
	}

	// print new trades to stdout
	strTrades := PrintTrades(newTrades)
	log.Println(strTrades)

	if verbose {
		log.Printf("%d trades in %d reports.\n", len(newTrades), len(links))
	}

	err = addNewTrades(newTrades)

	return newTrades, err
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

func addNewTrades(newTrades []Trade) error {
	var add int

	for _, newTrade := range newTrades {
		isUnique := true
		for _, existingTrade := range Trades {
			// does this work correctly?
			if hasMatchingWord(newTrade.Name, existingTrade.Name) &&
				(newTrade.Ticker == existingTrade.Ticker || newTrade.Ticker == "") &&
				(newTrade.Type == existingTrade.Type || newTrade.Type == "") &&
				(newTrade.Date == existingTrade.Date || newTrade.Date == "") &&
				(newTrade.Filed == existingTrade.Filed || newTrade.Filed == "") {
				isUnique = false
				break
			}
		}
		if isUnique {
			Trades = append(Trades, newTrade)
			add++
		}
	}

	if add == 0 {
		if verbose {
			log.Printf("no new trades.\n")
		}
		return nil
	}

	if err := utils.WriteJSON(FILE_TRADES, Trades); err != nil {
		return fmt.Errorf("failed to write unique trades to JSON: %w", err)
	}
	log.Printf("updated %s with %d new trades.\n", FILE_TRADES, add)

	return nil
}

func hasMatchingWord(new, old string) bool {
	if new == "" || old == "" {
		return false
	}
	newWords := strings.Fields(new)
	existingWords := strings.Fields(old)

	for _, newWord := range newWords {
		for _, existingWord := range existingWords {
			if newWord == existingWord {
				return true
			}
		}
	}
	return false
}

func PrintTrades(trades []Trade) string {
	output := "\r"
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
