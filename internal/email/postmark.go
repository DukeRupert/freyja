package email

import (
	"context"
	"fmt"
)

// PostmarkSender implements the Sender interface using Postmark API
type PostmarkSender struct {
	apiKey     string
	serverName string // Optional: for multiple Postmark servers
}

// NewPostmarkSender creates a new Postmark email sender
func NewPostmarkSender(apiKey string) *PostmarkSender {
	return &PostmarkSender{
		apiKey: apiKey,
	}
}

// Send sends an email via Postmark
func (p *PostmarkSender) Send(ctx context.Context, email *Email) (string, error) {
	// TODO: Implement Postmark API call
	// 1. Construct Postmark API request payload
	// 2. Make HTTP POST to https://api.postmarkapp.com/email
	// 3. Handle response and return message ID
	// 4. Handle errors (rate limits, invalid email, etc.)
	//
	// Example payload structure:
	// {
	//   "From": email.From,
	//   "To": strings.Join(email.To, ","),
	//   "Subject": email.Subject,
	//   "HtmlBody": email.HTMLBody,
	//   "TextBody": email.TextBody,
	//   "Headers": email.Headers,
	//   "Attachments": [...]
	// }
	//
	// Reference: https://postmarkapp.com/developer/api/email-api

	return "", fmt.Errorf("not implemented")
}

// SendTemplate sends an email using a Postmark template
func (p *PostmarkSender) SendTemplate(ctx context.Context, templateID string, to []string, data map[string]interface{}) (string, error) {
	// TODO: Implement Postmark template API call
	// 1. Construct Postmark template API request payload
	// 2. Make HTTP POST to https://api.postmarkapp.com/email/withTemplate
	// 3. Handle response and return message ID
	//
	// Example payload structure:
	// {
	//   "TemplateId": templateID,
	//   "To": strings.Join(to, ","),
	//   "TemplateModel": data
	// }
	//
	// Note: Using Postmark templates is optional. We can render templates
	// ourselves and use Send() instead.

	return "", fmt.Errorf("not implemented")
}
