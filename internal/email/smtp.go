package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPSender implements Sender using standard SMTP.
// This is the MVP implementation for development (Mailhog) and simple deployments.
type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSMTPSender creates a new SMTP email sender.
func NewSMTPSender(host string, port int, username, password, from string) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

// Send sends an email via SMTP.
func (s *SMTPSender) Send(ctx context.Context, email *Email) (string, error) {
	// Build message
	msg := s.buildMessage(email)

	// Determine sender address
	from := email.From
	if from == "" {
		from = s.from
	}

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	// For development (Mailhog), use plain SMTP
	// For production, use TLS
	var auth smtp.Auth
	if s.username != "" && s.password != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	// Send email
	err := smtp.SendMail(addr, auth, from, email.To, []byte(msg))
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	// SMTP doesn't provide message IDs, generate one
	return fmt.Sprintf("smtp-%d", len(email.To)), nil
}

// SendTemplate is not supported by basic SMTP.
func (s *SMTPSender) SendTemplate(ctx context.Context, templateID string, to []string, data map[string]interface{}) (string, error) {
	return "", fmt.Errorf("template emails not supported by SMTP sender")
}

// buildMessage constructs the email message with headers.
func (s *SMTPSender) buildMessage(email *Email) string {
	var msg strings.Builder

	// Headers
	msg.WriteString(fmt.Sprintf("From: %s\r\n", email.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))

	// Custom headers
	for key, value := range email.Headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Content type
	if email.HTMLBody != "" {
		msg.WriteString("MIME-Version: 1.0\r\n")
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	msg.WriteString("\r\n")

	// Body
	if email.HTMLBody != "" {
		msg.WriteString(email.HTMLBody)
	} else {
		msg.WriteString(email.TextBody)
	}

	return msg.String()
}

// SMTPTLSSender implements Sender using SMTP with TLS.
// This is for production use with services like Gmail, SendGrid SMTP relay, etc.
type SMTPTLSSender struct {
	*SMTPSender
}

// NewSMTPTLSSender creates a new SMTP sender with TLS support.
func NewSMTPTLSSender(host string, port int, username, password, from string) *SMTPTLSSender {
	return &SMTPTLSSender{
		SMTPSender: NewSMTPSender(host, port, username, password, from),
	}
}

// Send sends an email via SMTP with TLS.
func (s *SMTPTLSSender) Send(ctx context.Context, email *Email) (string, error) {
	// Build message
	msg := s.buildMessage(email)

	from := email.From
	if from == "" {
		from = s.from
	}

	// Connect with TLS
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	// TLS config
	tlsConfig := &tls.Config{
		ServerName: s.host,
	}

	// Connect
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return "", fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return "", fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Authenticate
	if s.username != "" && s.password != "" {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			return "", fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return "", fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, to := range email.To {
		if err := client.Rcpt(to); err != nil {
			return "", fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return "", fmt.Errorf("failed to open data writer: %w", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return "", fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close data writer: %w", err)
	}

	return fmt.Sprintf("smtp-tls-%d", len(email.To)), nil
}
