package gemini

import (
	"clerk_trades/utils"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

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

	// var aiFiles []*genai.File

	// for i, content := range fileContents {
	// 	link := links[i]
	// 	_, fileName := filepath.Split(link)

	// 	tmpFile, err := ioutil.TempFile(os.TempDir(), "")
	// 	if err != nil {
	// 		return fmt.Errorf("failed to create temp file: %v", err)
	// 	}

	// 	if _, err := tmpFile.Write(content); err != nil {
	// 		tmpFile.Close()
	// 		return fmt.Errorf("failed to write to temp file: %v", err)
	// 	}
	// 	tmpFile.Close()

	// 	if err := os.Rename(tmpFile.Name(), filepath.Join(os.TempDir(), fileName)); err != nil {
	// 		return fmt.Errorf("failed to rename temp file: %v", err)
	// 	}

	// 	uploadedFile, err := client.UploadFileFromPath(ctx, filepath.Join(os.TempDir(), fileName), nil)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to upload file %d: %v", i, err)
	// 	}
	// 	aiFiles = append(aiFiles, uploadedFile)

	// 	defer func(filePath string) {
	// 		if err := os.Remove(filePath); err != nil {
	// 			log.Printf("failed to delete temp file %s: %v", filePath, err)
	// 		}
	// 	}(filepath.Join(os.TempDir(), fileName))

	// 	log.Printf("uploaded %s\n", fileName)
	// }

	aiFiles, err := uploadFilesConcurrently(ctx, fileContents, links, client)
	if err != nil {
		return err
	}
	log.Printf("generating trade reports... ")

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.9)
	model.SetTopP(0.5)
	model.SetTopK(20)
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = genai.NewUserContent(genai.Text(`
It should read data from the uploaded PDF file and write data into the JSON array described below with some rules.
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
	for _, file := range aiFiles {
		parts = append(parts, genai.FileData{URI: file.URI})
	}

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
			// add better loqic here since, names and stock name can differ.
			// check data adn filing date adn type and size instead
			if newTrade == existingTrade {
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

func uploadFilesConcurrently(ctx context.Context, fileContents [][]byte, links []string, client *genai.Client) ([]*genai.File, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	var aiFiles []*genai.File
	errChan := make(chan error, len(fileContents))

	for i, content := range fileContents {
		wg.Add(1)
		go func(i int, content []byte) {
			defer wg.Done()

			link := links[i]
			_, fileName := filepath.Split(link)

			tmpFile, err := ioutil.TempFile(os.TempDir(), "")
			if err != nil {
				errChan <- fmt.Errorf("failed to create temp file: %v", err)
				return
			}

			if _, err := tmpFile.Write(content); err != nil {
				tmpFile.Close()
				errChan <- fmt.Errorf("failed to write to temp file: %v", err)
				return
			}
			tmpFile.Close()

			if err := os.Rename(tmpFile.Name(), filepath.Join(os.TempDir(), fileName)); err != nil {
				errChan <- fmt.Errorf("failed to rename temp file: %v", err)
				return
			}

			uploadedFile, err := client.UploadFileFromPath(ctx, filepath.Join(os.TempDir(), fileName), nil)
			if err != nil {
				errChan <- fmt.Errorf("failed to upload file %d: %v", i, err)
				return
			}

			mu.Lock()
			aiFiles = append(aiFiles, uploadedFile)
			mu.Unlock()

			defer func(filePath string) {
				if err := os.Remove(filePath); err != nil {
					log.Printf("failed to delete temp file %s: %v", filePath, err)
				}
			}(filepath.Join(os.TempDir(), fileName))

			log.Printf("uploaded %s\n", fileName)
		}(i, content)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return nil, fmt.Errorf("errors occurred during upload: %v", <-errChan)
	}

	return aiFiles, nil
}
