package email

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wneessen/go-mail"
)

// SMTPConfig holds SMTP connection parameters.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string // optional - some servers allow unauthenticated relay
	Password string // optional
	From     string // default sender address
	FromName string // optional sender display name
}

// SMTPSender implements Sender using go-mail for robust SMTP support.
// Features:
// - Automatic TLS/STARTTLS detection based on port
// - Multiple auth methods (PLAIN, LOGIN, CRAM-MD5, SCRAM)
// - Proper MIME multipart message construction
// - Connection timeout handling
type SMTPSender struct {
	config *SMTPConfig
	logger *slog.Logger
}

// NewSMTPSender creates a new SMTP email sender using go-mail.
func NewSMTPSender(host string, port int, username, password, from, fromName string) *SMTPSender {
	return &SMTPSender{
		config: &SMTPConfig{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			From:     from,
			FromName: fromName,
		},
		logger: slog.Default(),
	}
}

// NewSMTPSenderFromConfig creates an SMTP sender from a config struct.
func NewSMTPSenderFromConfig(config *SMTPConfig) *SMTPSender {
	return &SMTPSender{
		config: config,
		logger: slog.Default(),
	}
}

// Send sends an email via SMTP using go-mail.
func (s *SMTPSender) Send(ctx context.Context, email *Email) (string, error) {
	s.logger.Info("smtp: preparing email",
		"to", email.To,
		"from", email.From,
		"subject", email.Subject,
		"host", s.config.Host,
		"port", s.config.Port,
	)

	// Create message
	msg := mail.NewMsg()

	// Set sender
	from := email.From
	if from == "" {
		from = s.config.From
	}
	if err := msg.From(from); err != nil {
		return "", fmt.Errorf("invalid from address: %w", err)
	}

	// Set recipients
	if err := msg.To(email.To...); err != nil {
		return "", fmt.Errorf("invalid to address: %w", err)
	}

	// Set subject
	msg.Subject(email.Subject)

	// Set body - prefer HTML with text fallback, or just text
	if email.HTMLBody != "" && email.TextBody != "" {
		msg.SetBodyString(mail.TypeTextPlain, email.TextBody)
		msg.AddAlternativeString(mail.TypeTextHTML, email.HTMLBody)
	} else if email.HTMLBody != "" {
		msg.SetBodyString(mail.TypeTextHTML, email.HTMLBody)
	} else {
		msg.SetBodyString(mail.TypeTextPlain, email.TextBody)
	}

	// Add custom headers
	for key, value := range email.Headers {
		msg.SetGenHeader(mail.Header(key), value)
	}

	// Add attachments
	for _, att := range email.Attachments {
		if err := msg.AttachReader(att.Filename, &bytesReader{data: att.Content},
			mail.WithFileContentType(mail.ContentType(att.ContentType))); err != nil {
			return "", fmt.Errorf("failed to attach file %s: %w", att.Filename, err)
		}
	}

	// Create client with appropriate options
	opts := s.buildClientOptions()

	client, err := mail.NewClient(s.config.Host, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to create SMTP client: %w", err)
	}

	// Send the message
	if err := client.DialAndSend(msg); err != nil {
		s.logger.Error("smtp: failed to send email", "error", err)
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("smtp: email sent successfully", "to", email.To)

	// Generate a message ID (SMTP doesn't provide one reliably)
	messageID := fmt.Sprintf("smtp-%d-%d", time.Now().UnixNano(), len(email.To))
	return messageID, nil
}

// SendTemplate is not supported by SMTP sender.
func (s *SMTPSender) SendTemplate(ctx context.Context, templateID string, to []string, data map[string]interface{}) (string, error) {
	return "", ErrNotImplemented
}

// buildClientOptions returns go-mail client options based on configuration.
func (s *SMTPSender) buildClientOptions() []mail.Option {
	opts := []mail.Option{
		mail.WithPort(s.config.Port),
		mail.WithTimeout(30 * time.Second),
	}

	// TLS mode based on port (go-mail auto-detects, but we can be explicit)
	switch s.config.Port {
	case 465:
		// Implicit TLS (SMTPS)
		opts = append(opts, mail.WithSSL())
	case 587:
		// STARTTLS (submission port)
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	case 25:
		// Plain SMTP or opportunistic STARTTLS
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	default:
		// For other ports (like 1025 for Mailhog), try opportunistic TLS
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	}

	// Authentication if credentials provided
	if s.config.Username != "" && s.config.Password != "" {
		opts = append(opts,
			mail.WithUsername(s.config.Username),
			mail.WithPassword(s.config.Password),
			mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		)
	}

	return opts
}

// TestConnection verifies SMTP connectivity and authentication without sending email.
func (s *SMTPSender) TestConnection() error {
	return TestSMTPConnection(s.config.Host, s.config.Port, s.config.Username, s.config.Password)
}

// TestSMTPConnection verifies SMTP connectivity and authentication.
// This is a standalone function for use in the admin handler.
func TestSMTPConnection(host string, port int, username, password string) error {
	opts := []mail.Option{
		mail.WithPort(port),
		mail.WithTimeout(10 * time.Second),
	}

	// TLS mode based on port
	switch port {
	case 465:
		opts = append(opts, mail.WithSSL())
	case 587:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	case 25:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	default:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	}

	// Authentication if credentials provided
	if username != "" && password != "" {
		opts = append(opts,
			mail.WithUsername(username),
			mail.WithPassword(password),
			mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		)
	}

	client, err := mail.NewClient(host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	// Dial to test connection and authentication
	if err := client.DialWithContext(context.Background()); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer client.Close()

	return nil
}

// bytesReader wraps a byte slice to implement io.Reader for attachments.
type bytesReader struct {
	data   []byte
	offset int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}
