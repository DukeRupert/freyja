package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
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
	htmlBody, textBody, err := s.renderTemplate(data.TemplateName(), data)
	if err != nil {
		return fmt.Errorf("failed to render password reset template: %w", err)
	}

	email := &Email{
		To:       []string{data.Email},
		From:     fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress),
		Subject:  data.Subject(),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	_, err = s.sender.Send(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	return nil
}

// SendOrderConfirmation sends an order confirmation email
func (s *Service) SendOrderConfirmation(ctx context.Context, data OrderConfirmationEmail) error {
	htmlBody, textBody, err := s.renderTemplate(data.TemplateName(), data)
	if err != nil {
		return fmt.Errorf("failed to render order confirmation template: %w", err)
	}

	email := &Email{
		To:       []string{data.CustomerName},
		From:     fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress),
		Subject:  data.Subject(),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	_, err = s.sender.Send(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to send order confirmation email: %w", err)
	}

	return nil
}

// SendShippingConfirmation sends a shipping confirmation email
func (s *Service) SendShippingConfirmation(ctx context.Context, data ShippingConfirmationEmail) error {
	htmlBody, textBody, err := s.renderTemplate(data.TemplateName(), data)
	if err != nil {
		return fmt.Errorf("failed to render shipping confirmation template: %w", err)
	}

	email := &Email{
		To:       []string{data.CustomerName},
		From:     fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress),
		Subject:  data.Subject(),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	_, err = s.sender.Send(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to send shipping confirmation email: %w", err)
	}

	return nil
}

// SendSubscriptionWelcome sends a subscription welcome email
func (s *Service) SendSubscriptionWelcome(ctx context.Context, data SubscriptionWelcomeEmail) error {
	htmlBody, textBody, err := s.renderTemplate(data.TemplateName(), data)
	if err != nil {
		return fmt.Errorf("failed to render subscription welcome template: %w", err)
	}

	email := &Email{
		To:       []string{data.CustomerName},
		From:     fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress),
		Subject:  data.Subject(),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	_, err = s.sender.Send(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to send subscription welcome email: %w", err)
	}

	return nil
}

// SendSubscriptionPaymentFailed sends a subscription payment failed email
func (s *Service) SendSubscriptionPaymentFailed(ctx context.Context, data SubscriptionPaymentFailedEmail) error {
	htmlBody, textBody, err := s.renderTemplate(data.TemplateName(), data)
	if err != nil {
		return fmt.Errorf("failed to render subscription payment failed template: %w", err)
	}

	email := &Email{
		To:       []string{data.CustomerName},
		From:     fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress),
		Subject:  data.Subject(),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	_, err = s.sender.Send(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to send subscription payment failed email: %w", err)
	}

	return nil
}

// SendSubscriptionCancelled sends a subscription cancelled email
func (s *Service) SendSubscriptionCancelled(ctx context.Context, data SubscriptionCancelledEmail) error {
	htmlBody, textBody, err := s.renderTemplate(data.TemplateName(), data)
	if err != nil {
		return fmt.Errorf("failed to render subscription cancelled template: %w", err)
	}

	email := &Email{
		To:       []string{data.CustomerName},
		From:     fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress),
		Subject:  data.Subject(),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	_, err = s.sender.Send(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to send subscription cancelled email: %w", err)
	}

	return nil
}

// Helper method to render a template
func (s *Service) renderTemplate(templateName string, data interface{}) (string, string, error) {
	var htmlBuf bytes.Buffer
	err := s.templateCache.ExecuteTemplate(&htmlBuf, "email_layout", data)
	if err != nil {
		return "", "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	htmlBody := htmlBuf.String()

	plainText := generatePlainText(htmlBody)

	return htmlBody, plainText, nil
}

// generatePlainText creates a simple plain text version from HTML
func generatePlainText(html string) string {
	text := html

	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n\n")
	text = strings.ReplaceAll(text, "</div>", "\n")
	text = strings.ReplaceAll(text, "</h1>", "\n\n")
	text = strings.ReplaceAll(text, "</h2>", "\n\n")
	text = strings.ReplaceAll(text, "</h3>", "\n\n")

	for strings.Contains(text, "<") && strings.Contains(text, ">") {
		start := strings.Index(text, "<")
		end := strings.Index(text, ">")
		if start >= 0 && end > start {
			text = text[:start] + text[end+1:]
		} else {
			break
		}
	}

	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}
