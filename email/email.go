package email

import (
	"bufio"
	"bytes"
	"clerk_trades/gemini"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mailgun/mailgun-go/v4"

	"html/template"
)

const configFile = "mailgun.config"

type MailGun struct {
	APIKey  string
	Domain  string
	EmailTo []string
	Paid    bool
	*mailgun.MailgunImpl
}

var Mailgun = &MailGun{}

// load mailgun and its settings from config file
func LoadMailGun() error {
	file, err := os.Open(configFile)
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
		case "MAILGUN_EMAIL_TO":
			Mailgun.EmailTo = strings.Split(value, ",")
			var validEmails []string
			for _, email := range Mailgun.EmailTo {
				email = strings.TrimSpace(email)
				if email != "" && strings.Contains(email, "@") {
					validEmails = append(validEmails, email)
				}
			}
			Mailgun.EmailTo = validEmails
		case "MAILGUN_PAID":
			if value == "true" {
				Mailgun.Paid = true
			} else {
				Mailgun.Paid = false
			}
		default:
			return fmt.Errorf("unknown key: %s", key)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if len(Mailgun.EmailTo) == 0 {
		return fmt.Errorf("no emails in to send trade reports to")
	}

	if Mailgun.Domain == "" || Mailgun.APIKey == "" {
		return fmt.Errorf("missing required fields in config")
	}
	Mailgun.MailgunImpl = mailgun.NewMailgun(Mailgun.Domain, Mailgun.APIKey)
	// Mailgun.SetAPIBase(mailgun.APIBaseEU)

	return addEmailsToMailingList(Mailgun.EmailTo...)
}

// create mailing list
func createMailingList(ctx context.Context) error {
	listAddress := "clerk@" + Mailgun.Domain

	_, err := Mailgun.GetMailingList(ctx, listAddress)
	if err == nil {
		return nil // alredy exists
	}

	// If the list doesn't exist, create it
	if err != nil && strings.Contains(err.Error(), "not found") {
		_, err = Mailgun.CreateMailingList(ctx, mailgun.MailingList{
			Address:     listAddress,
			Name:        "clerk",
			Description: "clerk application",
		})
		if err != nil {
			return fmt.Errorf("failed to create mailing list: %w", err)
		}

		log.Println("clerk mailing list mailgun created successfully")
		return nil
	}

	return fmt.Errorf("failed to check mailing list: %w", err)
}

// add emails to mailing list
func addEmailsToMailingList(emails ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err := createMailingList(ctx)
	if err != nil {
		return err
	}
	var members []interface{}
	for _, email := range emails {
		members = append(members, mailgun.Member{
			Address: email,
		})
	}

	upsert := true
	err = Mailgun.CreateMemberList(ctx, &upsert, "clerk@"+Mailgun.Domain, members)
	if err != nil {
		return err
	}

	return nil
}

// send emails to mailing list (non-paid account)
func SendHTMLTo(body string) error {
	ctx := context.Background()
	it := Mailgun.ListMembers("clerk@"+Mailgun.Domain, nil)

	var members []mailgun.Member
	for it.Next(ctx, &members) {
		for _, member := range members {
			m := Mailgun.NewMessage(
				"clerk trades <mailgun@"+Mailgun.Domain+">", // From
				"TRADES", // Subject
				"",       // Body
			)
			// Set the HTML body
			m.SetHtml(body)
			m.AddRecipient(member.Address)

			_, _, err := Mailgun.Send(ctx, m)
			if err != nil {
				return fmt.Errorf("failed to send email: %v", err)
			}
		}
	}

	if it.Err() != nil {
		return it.Err()
	}

	return nil
}

// send emails to mailing list (paid account)
func SendHTMLToMailingList(body string) error {
	listAddress := "clerk@" + Mailgun.Domain

	m := Mailgun.NewMessage(
		"clerk trades <mailgun@"+Mailgun.Domain+">", // From
		"TRADES", // Subject
		"",       // Body
	)
	// Set the HTML body
	m.SetHtml(body)
	m.AddRecipient(listAddress)

	ctx := context.Background()
	_, _, err := Mailgun.Send(ctx, m)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}
	return nil
}

// generate html body
func GenerateEmailBody(trades []gemini.Trade) (string, error) {
	tmpl := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Trades</title>
			<style>
				table { width: 100%; border-collapse: collapse; }
				th, td { padding: 8px 12px; border: 1px solid #ddd; text-align: left; }
				th { background-color: #f4f4f4; }
			</style>
		</head>
		<body>
			<h1>Trade List</h1>
			<table>
				<thead>
					<tr>
						<th>Name</th>
						<th>Asset</th>
						<th>Ticker</th>
						<th>Type</th>
						<th>Date</th>
						<th>Filed</th>
						<th>Amount</th>
						<th>Cap</th>
					</tr>
				</thead>
				<tbody>
					{{range .}}
					<tr>
						<td>{{.Name}}</td>
						<td>{{.Asset}}</td>
						<td>{{.Ticker}}</td>
						<td>{{.Type}}</td>
						<td>{{.Date}}</td>
						<td>{{.Filed}}</td>
						<td>{{.Amount}}</td>
						<td>{{.Cap}}</td>
					</tr>
					{{end}}
				</tbody>
			</table>
		</body>
		</html>
	`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("error creating email template: %v", err)
	}

	var emailBodyBuffer bytes.Buffer
	err = t.Execute(&emailBodyBuffer, trades)
	if err != nil {
		return "", fmt.Errorf("error executing email template: %v", err)
	}

	emailBody := emailBodyBuffer.String()
	return emailBody, nil
}

// func ListEmailsInMailingList() error {
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
// 	defer cancel()

// 	it := Mailgun.ListMembers("clerk@"+Mailgun.Domain, nil)

// 	var members []mailgun.Member
// 	for it.Next(ctx, &members) {
// 		for _, member := range members {
// 			fmt.Println(member.Address)
// 		}
// 	}

// 	if it.Err() != nil {
// 		return it.Err()
// 	}

// 	return nil
// }
