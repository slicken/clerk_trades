package gemini

import (
	"clerk_trades/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

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
)

func ProsessRapports(fileContents [][]byte, links []string) error {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("environment variable GEMINI_API_KEY not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
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
	parts = append(parts, genai.Text("create JSON"))
	for _, data := range fileContents {
		parts = append(parts, genai.Blob{
			MIMEType: "application/pdf",
			Data:     data,
		})
	}

	log.Printf("generating trade reports...")

	resp, err := model.GenerateContent(ctx, parts...)
	if err != nil {
		return fmt.Errorf("failed to generate content: %v", err)
	}

	out := getResponse(resp)

	if err := json.Unmarshal([]byte(out), &newTrades); err != nil {
		return fmt.Errorf("error unmarshalling JSON: %v, Output: %s", err, out)
	}

	log.Printf("found %d trades in %d reports:\n", len(newTrades), len(links))

	tradesJSON, err := json.MarshalIndent(newTrades, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal new trades to JSON: %v", err)
	}
	fmt.Println(string(tradesJSON))

	return addWriteTrades(newTrades)
}

func getResponse(resp *genai.GenerateContentResponse) string {
	var str string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				str += fmt.Sprint(part)
			}
		}
	}
	return str
}

func addWriteTrades(newTrades []Trade) error {
	added := 0

	for _, newTrade := range newTrades {
		isUnique := true
		for _, existingTrade := range Trades {
			if newTrade.Ticker == existingTrade.Ticker &&
				newTrade.Type == existingTrade.Type &&
				newTrade.Date == existingTrade.Date &&
				newTrade.Filed == existingTrade.Filed &&
				newTrade.Amount == existingTrade.Amount &&
				newTrade.Cap == existingTrade.Cap {
				isUnique = false
				break
			}
		}
		if isUnique {
			Trades = append(Trades, newTrade)
			added++
		}
	}

	if err := utils.WriteJSON(FILE_TRADES, Trades); err != nil {
		return fmt.Errorf("failed to write unique trades to JSON: %w", err)
	}
	if added > 0 {
		log.Printf("updated %s with %d new trades.\n", FILE_TRADES, added)
	}

	return nil
}
