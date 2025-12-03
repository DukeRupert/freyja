package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PostmarkSender implements the Sender interface using Postmark API
type PostmarkSender struct {
	apiKey     string
	serverName string // Optional: for multiple Postmark servers
}

type postmarkEmail struct {
	From        string            `json:"From"`
	To          string            `json:"To"`
	Subject     string            `json:"Subject"`
	HtmlBody    string            `json:"HtmlBody,omitempty"`
	TextBody    string            `json:"TextBody,omitempty"`
	Headers     []postmarkHeader  `json:"Headers,omitempty"`
	Attachments []postmarkAttach  `json:"Attachments,omitempty"`
}

type postmarkHeader struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type postmarkAttach struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"`
	ContentType string `json:"ContentType"`
}

type postmarkResponse struct {
	To        string `json:"To"`
	MessageID string `json:"MessageID"`
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
}

// NewPostmarkSender creates a new Postmark email sender
func NewPostmarkSender(apiKey string) *PostmarkSender {
	return &PostmarkSender{
		apiKey: apiKey,
	}
}

// Send sends an email via Postmark
func (p *PostmarkSender) Send(ctx context.Context, email *Email) (string, error) {

	payload := postmarkEmail{
		From:     email.From,
		To:       strings.Join(email.To, ","),
		Subject:  email.Subject,
		HtmlBody: email.HTMLBody,
		TextBody: email.TextBody,
	}

	if len(email.Headers) > 0 {
		headers := make([]postmarkHeader, 0, len(email.Headers))
		for name, value := range email.Headers {
			headers = append(headers, postmarkHeader{Name: name, Value: value})
		}
		payload.Headers = headers
	}

	if len(email.Attachments) > 0 {
		attachments := make([]postmarkAttach, len(email.Attachments))
		for i, att := range email.Attachments {
			attachments[i] = postmarkAttach{
				Name:        att.Filename,
				Content:     base64.StdEncoding.EncodeToString(att.Content),
				ContentType: att.ContentType,
			}
		}
		payload.Attachments = attachments
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.postmarkapp.com/email", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Postmark-Server-Token", p.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("postmark API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result postmarkResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.ErrorCode != 0 {
		return "", fmt.Errorf("postmark error %d: %s", result.ErrorCode, result.Message)
	}

	return result.MessageID, nil
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
