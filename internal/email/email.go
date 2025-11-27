package email

import "context"

// Email represents an email message to be sent.
type Email struct {
	To          []string          // Recipient email addresses
	From        string            // Sender email address
	Subject     string            // Email subject
	TextBody    string            // Plain text body
	HTMLBody    string            // HTML body (optional)
	Attachments []Attachment      // File attachments (optional)
	Headers     map[string]string // Custom headers (optional)
}

// Attachment represents a file attachment for an email.
type Attachment struct {
	Filename    string // Name of the file
	ContentType string // MIME type
	Content     []byte // File content
}

// Sender defines the interface for sending emails.
// Implementations can use SMTP, Postmark, Resend, SES, etc.
type Sender interface {
	// Send sends an email message.
	// Returns the message ID from the email provider (if available).
	Send(ctx context.Context, email *Email) (string, error)

	// SendTemplate sends an email using a provider-managed template.
	// templateID is the provider's template identifier.
	// data contains the template variables.
	SendTemplate(ctx context.Context, templateID string, to []string, data map[string]interface{}) (string, error)
}

// SendResult represents the result of sending an email.
type SendResult struct {
	MessageID string // Provider's message ID
	Status    string // Delivery status (sent, queued, etc.)
}
