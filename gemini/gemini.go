package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Define a struct to hold the JSON data
type Trade struct {
	Name   string `json:"Name"`
	Asset  string `json:"Asset"`
	Ticker string `json:"Ticker"`
	Type   string `json:"Type"`
	Date   string `json:"Date"`
	Filed  string `json:"Filed`
	Amount string `json:"Amount"`
	Cap    bool   `json:"Cap"`
}

var Trades []Trade

// ProsessRapports now accepts byte slices instead of file paths
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

	var aiFiles []*genai.File

	// Assuming you have a way to retrieve the original links
	for i, content := range fileContents {
		link := links[i]
		_, fileName := filepath.Split(link)

		tmpFile, err := ioutil.TempFile(os.TempDir(), "")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %v", err)
		}

		if _, err := tmpFile.Write(content); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write to temp file: %v", err)
		}
		tmpFile.Close()

		if err := os.Rename(tmpFile.Name(), filepath.Join(os.TempDir(), fileName)); err != nil {
			return fmt.Errorf("failed to rename temp file: %v", err)
		}

		uploadedFile, err := client.UploadFileFromPath(ctx, filepath.Join(os.TempDir(), fileName), nil)
		if err != nil {
			return fmt.Errorf("failed to upload file %d: %v", i, err)
		}
		aiFiles = append(aiFiles, uploadedFile)

		defer func(filePath string) {
			if err := os.Remove(filePath); err != nil {
				log.Printf("failed to delete temp file %s: %v", filePath, err)
			}
		}(filepath.Join(os.TempDir(), fileName))

		log.Printf("uploaded %s\n", filepath.Join(os.TempDir(), fileName))
	}

	log.Printf("generating trade reports... ")

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.9)
	model.SetTopP(0.5)
	model.SetTopK(20)
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = genai.NewUserContent(genai.Text(`
Application that reads data from JSON and writes into a JSON array of structs
Rules: strip "Hon. " from Name
Rules: for Transaction Type: if "P" input "Purchase", if "S" input "Sale"
[
	{
		"Name": "input the Name under Filer Information",
		"Asset": "input Asset fill name",
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

	if err := json.Unmarshal([]byte(out), &Trades); err != nil {
		return fmt.Errorf("error unmarshalling JSON: %v, Output: %s", err, out)
	}

	log.Printf("last %d trades:\n", len(Trades))

	tradesJSON, err := json.MarshalIndent(Trades, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trades to JSON: %v", err)
	}

	// Print only the structured JSON output
	fmt.Println(string(tradesJSON))

	return nil
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
