package email

import (
	"context"
	"fmt"
	"os"

	"github.com/mailgun/mailgun-go/v4"
)

const (
	emailDomain = "your.mailgun.domain"
)

var apiKey string

func Init() error {
	apiKey = os.Getenv("MAILGUN_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("Error: MAILGUN_API_KEY environment variable is not set.\n")
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
		return fmt.Errorf("Error sending email: %v\n", err)
	}

	return nil
}
