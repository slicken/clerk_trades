package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const (
	Green    = "\033[32m"
	Yellow   = "\033[33m"
	Blue     = "\033[34m"
	Cyan     = "\033[36m"
	Magenta  = "\033[35m"
	Gray     = "\033[37m"
	DarkGray = "\033[90m"
	reset    = "\033[0m" // Reset to default
)

// utils.ColorPrintf(utils.Green, "This is a success message: %s\n", "Operation completed.")
// utils.ColorPrintf(utils.Yellow, "Warning: %s\n", "This is a warning.")
// utils.ColorPrintf(utils.Blue, "Info: %s\n", "This is some information.")
// utils.ColorPrintf(utils.Cyan, "Important: %s\n", "Check this out!")
// utils.ColorPrintf(utils.Magenta, "Attention: %s\n", "This needs your attention!")

// // ColorPrintf prints a formatted message in a specified color
// func ColorPrintf(color string, format string, a ...any) (n int, err error) {
// 	return fmt.Fprintf(os.Stdout, color+format+reset, a...)
// }

// // GrayPrintf prints a formatted message in dark gray color with a timestamp
// func GrayPrintf(format string, a ...any) (n int, err error) {
// 	// return fmt.Fprintf(os.Stdout, DarkGray+format+reset, a...)
// 	timestamp := time.Now().Format("2006-01-02 15:04:05")
// 	return fmt.Fprintf(os.Stdout, DarkGray+"%s "+format+reset, append([]any{timestamp}, a...)...)
// }

// ReadJSON to a generic struct T. If the file dont exist it will creates the file in JSON format []
func ReadJSON[T any](file string) (T, error) {
	var result T

	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the file and initialize with an empty JSON array
			if err := os.WriteFile(file, []byte("[]"), 0644); err != nil {
				return result, fmt.Errorf("failed to create file: %w", err)
			}
		} else {
			return result, fmt.Errorf("failed to open file: %w", err)
		}
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return result, fmt.Errorf("failed to read file: %w", err)
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return result, nil
}

// WriteJSON takes a generic struct T and writes it to a file as JSON
func WriteJSON[T any](file string, data T) error {
	bytes, err := json.MarshalIndent(data, "", "  ") // Pretty-print JSON
	if err != nil {
		return fmt.Errorf("failed to marshal struct: %w", err)
	}

	err = os.WriteFile(file, bytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	return nil
}

// check if links contains target
func Contains(slice []string, target string) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}
