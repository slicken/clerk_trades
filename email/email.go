package email

import (
	"bufio"
	"bytes"
	"clerk_trades/gemini"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mailgun/mailgun-go/v4"

	"html/template"
)

const (
	emailDomain = "sandboxb7a349a65c174b62955c4190ef32fe43.mailgun.org"
)

var (
	mg     *mailgun.MailgunImpl
	apiKey string
)

func Init() error {
	apiKey = os.Getenv("MAILGUN_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("MAILGUN_API_KEY environment variable is not set")
	}
	mg = mailgun.NewMailgun(emailDomain, apiKey)
	return nil
}

func SendMail(to string, body string) error {
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

func SendHTML(to string, body string) error {
	m := mg.NewMessage(
		"clerk trades <mailgun@"+emailDomain+">", // From
		"TRADES", // Subject
		"",       // Plain-text body (empty in this case)
		to,       // Recipient
	)
	// Set the HTML body
	m.SetHtml(body)

	m.AddRecipient(to)

	ctx := context.Background()
	_, _, err := mg.Send(ctx, m)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}
	return nil
}

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

	mg = mailgun.NewMailgun(Mailgun.Domain, Mailgun.APIKey)

	for _, email := range Mailgun.EmailList {
		if err := checkAndAddEmail(email); err != nil {
			return err
		}
	}

	return nil
}

func checkAndAddEmail(to string) error {
	ctx := context.Background()

	mailingList, err := mg.GetMailingList(ctx, Mailgun.Domain)
	if err != nil {
		return fmt.Errorf("failed to retrieve mailing list from Mailgun: %w", err)
	}

	members, err := listMailingListMembers(ctx, mailingList.Address)
	if err != nil {
		return fmt.Errorf("failed to retrieve members from mailing list: %w", err)
	}

	for _, member := range members {
		if strings.EqualFold(member.Address, to) {
			return nil
		}
	}

	err = addMailingListMember(ctx, mailingList.Address, to)
	if err != nil {
		return fmt.Errorf("failed to add email to mailing list: %w", err)
	}

	return nil
}

func listMailingListMembers(ctx context.Context, address string) ([]mailgun.Member, error) {
	apiURL := fmt.Sprintf("https://api.mailgun.net/v3/lists/%s/members", address)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("api", apiKey)

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var membersResponse struct {
		Members []mailgun.Member `json:"items"`
	}
	err = json.NewDecoder(resp.Body).Decode(&membersResponse)
	if err != nil {
		return nil, err
	}

	return membersResponse.Members, nil
}

func addMailingListMember(ctx context.Context, address string, email string) error {
	apiURL := fmt.Sprintf("https://api.mailgun.net/v3/lists/%s/members", address)
	data := url.Values{}
	data.Set("address", email)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth("api", apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add member: %s", resp.Status)
	}

	return nil
}
