package saas

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/webhook"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/service"
)

// WebhookHandler handles Stripe webhooks for SaaS subscriptions
type WebhookHandler struct {
	onboardingService service.OnboardingService
	billingProvider   billing.Provider
	webhookSecret     string // Stripe webhook signing secret for SaaS events
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(
	onboardingService service.OnboardingService,
	billingProvider billing.Provider,
	webhookSecret string,
) *WebhookHandler {
	return &WebhookHandler{
		onboardingService: onboardingService,
		billingProvider:   billingProvider,
		webhookSecret:     webhookSecret,
	}
}

// HandleStripeWebhook handles POST /webhooks/stripe/saas
// Processes Stripe webhook events for platform subscriptions
func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("saas webhook: failed to read body",
			"error", err,
		)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Verify webhook signature
	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, signatureHeader, h.webhookSecret)
	if err != nil {
		slog.Error("saas webhook: signature verification failed",
			"error", err,
		)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	slog.Info("saas webhook: received event",
		"type", event.Type,
		"id", event.ID,
	)

	// Handle event types
	switch event.Type {
	case "checkout.session.completed":
		h.handleCheckoutSessionCompleted(ctx, w, event)

	case "invoice.paid":
		h.handleInvoicePaid(ctx, w, event)

	case "invoice.payment_failed":
		h.handleInvoicePaymentFailed(ctx, w, event)

	case "customer.subscription.updated":
		h.handleSubscriptionUpdated(ctx, w, event)

	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(ctx, w, event)

	default:
		slog.Debug("saas webhook: unhandled event type",
			"type", event.Type,
		)
		w.WriteHeader(http.StatusOK)
	}
}

func (h *WebhookHandler) handleCheckoutSessionCompleted(ctx context.Context, w http.ResponseWriter, event stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		slog.Error("saas webhook: failed to parse checkout session",
			"error", err,
		)
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	// Extract business name from custom fields
	var businessName string
	for _, field := range session.CustomFields {
		if field.Key == "business_name" && field.Text != nil {
			businessName = field.Text.Value
			break
		}
	}

	// If no business name from custom field, use customer name or email
	if businessName == "" && session.CustomerDetails != nil {
		businessName = session.CustomerDetails.Name
		if businessName == "" {
			businessName = session.CustomerDetails.Email
		}
	}

	// Get email
	var email string
	if session.CustomerDetails != nil {
		email = session.CustomerDetails.Email
	}
	if email == "" && session.Customer != nil {
		email = session.Customer.Email
	}

	// Get customer ID
	var customerID string
	if session.Customer != nil {
		customerID = session.Customer.ID
	}

	checkoutData := service.CheckoutSession{
		ID:           session.ID,
		CustomerID:   customerID,
		Email:        email,
		BusinessName: businessName,
		AmountTotal:  session.AmountTotal,
	}

	tenantID, operatorID, err := h.onboardingService.ProcessCheckoutCompleted(ctx, checkoutData)
	if err != nil {
		slog.Error("saas webhook: failed to process checkout completed",
			"session_id", session.ID,
			"error", err,
		)
		// Return 200 to acknowledge receipt - we've logged the error
		// Stripe will retry if we return error, but the data may be invalid
		w.WriteHeader(http.StatusOK)
		return
	}

	slog.Info("saas webhook: checkout completed processed",
		"session_id", session.ID,
		"tenant_id", tenantID,
		"operator_id", operatorID,
	)

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleInvoicePaid(ctx context.Context, w http.ResponseWriter, event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		slog.Error("saas webhook: failed to parse invoice",
			"error", err,
		)
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	err := h.onboardingService.ProcessInvoicePaid(ctx, invoice.ID)
	if err != nil {
		slog.Error("saas webhook: failed to process invoice paid",
			"invoice_id", invoice.ID,
			"error", err,
		)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleInvoicePaymentFailed(ctx context.Context, w http.ResponseWriter, event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		slog.Error("saas webhook: failed to parse invoice",
			"error", err,
		)
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	err := h.onboardingService.ProcessInvoicePaymentFailed(ctx, invoice.ID)
	if err != nil {
		slog.Error("saas webhook: failed to process invoice payment failed",
			"invoice_id", invoice.ID,
			"error", err,
		)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleSubscriptionUpdated(ctx context.Context, w http.ResponseWriter, event stripe.Event) {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		slog.Error("saas webhook: failed to parse subscription",
			"error", err,
		)
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	err := h.onboardingService.ProcessSubscriptionUpdated(ctx, subscription.ID)
	if err != nil {
		slog.Error("saas webhook: failed to process subscription updated",
			"subscription_id", subscription.ID,
			"error", err,
		)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleSubscriptionDeleted(ctx context.Context, w http.ResponseWriter, event stripe.Event) {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		slog.Error("saas webhook: failed to parse subscription",
			"error", err,
		)
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	err := h.onboardingService.ProcessSubscriptionDeleted(ctx, subscription.ID)
	if err != nil {
		slog.Error("saas webhook: failed to process subscription deleted",
			"subscription_id", subscription.ID,
			"error", err,
		)
	}

	w.WriteHeader(http.StatusOK)
}
