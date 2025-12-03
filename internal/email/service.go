package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"path/filepath"
)

// Service handles email composition and sending
type Service struct {
	sender        Sender
	fromAddress   string
	fromName      string
	templateCache *template.Template
}

// NewService creates a new email service
func NewService(sender Sender, fromAddress, fromName, templateDir string) (*Service, error) {
	// Load all email templates
	tmpl, err := template.ParseGlob(filepath.Join(templateDir, "email", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse email templates: %w", err)
	}

	return &Service{
		sender:        sender,
		fromAddress:   fromAddress,
		fromName:      fromName,
		templateCache: tmpl,
	}, nil
}

// SendPasswordReset sends a password reset email
func (s *Service) SendPasswordReset(ctx context.Context, data PasswordResetEmail) error {
	// TODO: Render template and send email
	// 1. Execute template with data
	// 2. Generate both HTML and plain text versions
	// 3. Call s.sender.Send()
	return fmt.Errorf("not implemented")
}

// SendOrderConfirmation sends an order confirmation email
func (s *Service) SendOrderConfirmation(ctx context.Context, data OrderConfirmationEmail) error {
	// TODO: Render template and send email
	// 1. Execute template with data
	// 2. Generate both HTML and plain text versions
	// 3. Call s.sender.Send()
	return fmt.Errorf("not implemented")
}

// SendShippingConfirmation sends a shipping confirmation email
func (s *Service) SendShippingConfirmation(ctx context.Context, data ShippingConfirmationEmail) error {
	// TODO: Render template and send email
	// 1. Execute template with data
	// 2. Generate both HTML and plain text versions
	// 3. Call s.sender.Send()
	return fmt.Errorf("not implemented")
}

// SendSubscriptionWelcome sends a subscription welcome email
func (s *Service) SendSubscriptionWelcome(ctx context.Context, data SubscriptionWelcomeEmail) error {
	// TODO: Render template and send email
	// 1. Execute template with data
	// 2. Generate both HTML and plain text versions
	// 3. Call s.sender.Send()
	return fmt.Errorf("not implemented")
}

// SendSubscriptionPaymentFailed sends a subscription payment failed email
func (s *Service) SendSubscriptionPaymentFailed(ctx context.Context, data SubscriptionPaymentFailedEmail) error {
	// TODO: Render template and send email
	// 1. Execute template with data
	// 2. Generate both HTML and plain text versions
	// 3. Call s.sender.Send()
	return fmt.Errorf("not implemented")
}

// SendSubscriptionCancelled sends a subscription cancelled email
func (s *Service) SendSubscriptionCancelled(ctx context.Context, data SubscriptionCancelledEmail) error {
	// TODO: Render template and send email
	// 1. Execute template with data
	// 2. Generate both HTML and plain text versions
	// 3. Call s.sender.Send()
	return fmt.Errorf("not implemented")
}

// Helper method to render a template
func (s *Service) renderTemplate(templateName string, data interface{}) (string, string, error) {
	// TODO: Execute template and generate both HTML and plain text versions
	// This is a helper that the Send* methods will use
	var htmlBuf bytes.Buffer
	err := s.templateCache.ExecuteTemplate(&htmlBuf, templateName, data)
	if err != nil {
		return "", "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	// TODO: Generate plain text version from HTML
	// For now, return placeholder
	plainText := "TODO: Generate plain text version"

	return htmlBuf.String(), plainText, nil
}
