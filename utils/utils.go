package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func ReadJSON[T any](file string) (T, error) {
	var result T

	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
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

// Helper function to check if a string exists in the slice
func Contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func EnsureValidJSON(input string) (string, error) {
	input = strings.TrimSpace(input)
	if !strings.HasSuffix(input, "}") && !strings.HasSuffix(input, "]") {
		input = input + "}"
	}
	var temp interface{}
	if err := json.Unmarshal([]byte(input), &temp); err != nil {
		return "", fmt.Errorf("invalid JSON format: %w", err)
	}
	return input, nil
}

func SafeUnmarshal(out string, target interface{}) error {
	validJSON, err := EnsureValidJSON(out)
	if err != nil {
		return fmt.Errorf("failed to ensure valid JSON: %v, output: %s", err, out)
	}
	if err := json.Unmarshal([]byte(validJSON), target); err != nil {
		return fmt.Errorf("failed to unmarshall JSON: %v, output: %s", err, validJSON)
	}
	return nil
}
