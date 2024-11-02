package email

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mailgun/mailgun-go/v4"
)

const (
	emailDomain = "yourdomain.mailgun.org"
)

var apiKey string

func Init() error {
	apiKey = os.Getenv("MAILGUN_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("MAILGUN_API_KEY environment variable is not set")
	}
	return nil
}

func SendTrades(to string, body string) error {
	mg := mailgun.NewMailgun(emailDomain, apiKey)
	m := mg.NewMessage(
		"clerk trades <mailgun@"+emailDomain+">",
		"TRADES",
		body,
		to,
	)

	m.AddRecipient(to)
	ctx := context.Background()
	_, _, err := mg.Send(ctx, m)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

// not yet implemented
type MailgunConfig struct {
	APIKey    string
	Domain    string
	EmailList []string
}

var Mailgun = &MailgunConfig{}

func LoadMailgunConfig() error {
	file, err := os.Open("mailgun.config")
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid line format: %s", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "MAILGUN_API_KEY":
			Mailgun.APIKey = value
		case "MAILGUN_DOMAIN":
			Mailgun.Domain = value
		case "MAILGUN_EMAIL_LIST":
			Mailgun.EmailList = strings.Split(value, ",")
		default:
			return fmt.Errorf("unknown key: %s", key)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	fmt.Println("Mailgun API Key:", Mailgun.APIKey)
	fmt.Println("Mailgun Domain:", Mailgun.Domain)
	fmt.Println("Mailgun Email List:", Mailgun.EmailList)
	return nil

	// return scanner.Err()
}
