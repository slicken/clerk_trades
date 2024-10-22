package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func ReadJSON[T any](file string) (T, error) {
	var result T

	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the file and initialize with empty JSON array
			f, err = os.Create(file)
			if err != nil {
				return result, fmt.Errorf("failed to create file: %w", err)
			}
			defer f.Close()

			// Write an empty JSON array
			if _, err := f.Write([]byte("[]")); err != nil {
				return result, fmt.Errorf("failed to write to new file: %w", err)
			}
			// Reopen the file to read the empty content
			f, err = os.Open(file)
			if err != nil {
				return result, fmt.Errorf("failed to reopen file: %w", err)
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

// // ReadJSON reads a file and unmarshals its content into a generic struct T.
// func ReadJSON[T any](file string) (T, error) {
// 	var result T

// 	f, err := os.Open(file)
// 	if err != nil {
// 		return result, fmt.Errorf("failed to open file: %w", err)
// 	}
// 	defer f.Close()

// 	data, err := io.ReadAll(f)
// 	if err != nil {
// 		return result, fmt.Errorf("failed to read file: %w", err)
// 	}

// 	err = json.Unmarshal(data, &result)
// 	if err != nil {
// 		return result, fmt.Errorf("failed to unmarshal JSON: %w", err)
// 	}

// 	return result, nil
// }

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
