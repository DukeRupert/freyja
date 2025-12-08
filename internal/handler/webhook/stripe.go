package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/telemetry"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v83"
)

// StripeHandler handles Stripe webhook events
type StripeHandler struct {
	provider            billing.Provider
	orderService        domain.OrderService
	subscriptionService domain.SubscriptionService
	config              StripeWebhookConfig
}

// StripeWebhookConfig contains configuration for Stripe webhook handling
type StripeWebhookConfig struct {
	// WebhookSecret is the webhook signing secret from Stripe dashboard
	WebhookSecret string

	// TenantID is used to scope payment intents (for multi-tenant isolation)
	// In production, this would come from the webhook endpoint URL or subdomain
	TenantID string

	// TestMode enables testing with Stripe CLI trigger commands.
	// When true, webhook handlers will:
	// - Log events without requiring tenant_id metadata match
	// - Skip order/subscription creation for events lacking required metadata
	// - Still validate webhook signatures
	// WARNING: Never enable in production - this bypasses tenant isolation checks
	TestMode bool
}

// NewStripeHandler creates a new Stripe webhook handler
func NewStripeHandler(provider billing.Provider, orderService domain.OrderService, subscriptionService domain.SubscriptionService, config StripeWebhookConfig) *StripeHandler {
	return &StripeHandler{
		provider:            provider,
		orderService:        orderService,
		subscriptionService: subscriptionService,
		config:              config,
	}
}

// HandleWebhook processes incoming Stripe webhook events
//
// Usage in main.go or router:
//
//	stripeHandler := webhook.NewStripeHandler(billingProvider, webhook.StripeWebhookConfig{
//	    WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
//	    TenantID:      os.Getenv("TENANT_ID"),
//	})
//	http.HandleFunc("/webhooks/stripe", stripeHandler.HandleWebhook)
//
// Stripe CLI testing:
//
//	stripe listen --forward-to localhost:3000/webhooks/stripe
//	stripe trigger payment_intent.succeeded
func (h *StripeHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("[WEBHOOK] Received request: %s %s", r.Method, r.URL.Path)

	// Only accept POST requests
	if r.Method != http.MethodPost {
		log.Printf("[WEBHOOK] Rejected: method %s not allowed", r.Method)
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Method not allowed"))
		return
	}

	// Read the request body
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[WEBHOOK] Error reading payload: %v", err)
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Error reading request body"))
		return
	}
	log.Printf("[WEBHOOK] Payload size: %d bytes", len(payload))

	// Get the signature from headers
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		log.Printf("[WEBHOOK] Missing Stripe-Signature header")
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Missing signature"))
		return
	}
	log.Printf("[WEBHOOK] Signature header present (length: %d)", len(signature))

	// Log webhook secret configuration (masked for security)
	secretLen := len(h.config.WebhookSecret)
	if secretLen > 0 {
		maskedSecret := h.config.WebhookSecret[:min(10, secretLen)] + "..."
		log.Printf("[WEBHOOK] Using webhook secret: %s (length: %d)", maskedSecret, secretLen)
	} else {
		log.Printf("[WEBHOOK] WARNING: Webhook secret is empty!")
	}

	// Verify the webhook signature
	err = h.provider.VerifyWebhookSignature(payload, signature, h.config.WebhookSecret)
	if err != nil {
		log.Printf("[WEBHOOK] Signature verification FAILED: %v", err)
		log.Printf("[WEBHOOK] Signature: %s", signature)
		handler.ErrorResponse(w, r, domain.Errorf(domain.EUNAUTHORIZED, "", "Invalid signature"))
		return
	}
	log.Printf("[WEBHOOK] Signature verification SUCCESS")

	// Parse the event
	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("Error parsing webhook JSON: %v", err)
		handler.ErrorResponse(w, r, domain.Errorf(domain.EINVALID, "", "Invalid JSON"))
		return
	}

	// Log the event for debugging
	log.Printf("Received Stripe webhook event: %s (ID: %s)", event.Type, event.ID)

	// Track webhook received
	tenantID := h.config.TenantID
	if telemetry.Business != nil {
		telemetry.Business.WebhookReceived.WithLabelValues(tenantID, string(event.Type)).Inc()
	}

	// Track processing time at the end
	defer func() {
		if telemetry.Business != nil {
			duration := time.Since(startTime).Seconds()
			telemetry.Business.WebhookLatency.WithLabelValues(tenantID, string(event.Type)).Observe(duration)
		}
	}()

	// Handle the event based on type
	switch event.Type {
	case "payment_intent.succeeded":
		h.handlePaymentIntentSucceeded(event)

	case "payment_intent.payment_failed":
		h.handlePaymentIntentFailed(event)

	case "payment_intent.canceled":
		h.handlePaymentIntentCanceled(event)

	case "payment_intent.created":
		// Usually no action needed - just for monitoring
		log.Printf("Payment intent created: %s", event.ID)

	// Subscription webhook events
	case "invoice.payment_succeeded":
		h.handleInvoicePaymentSucceeded(event)

	case "invoice.payment_failed":
		h.handleInvoicePaymentFailed(event)

	case "customer.subscription.updated":
		h.handleSubscriptionUpdated(event)

	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(event)

	default:
		// Log unhandled event types for future implementation
		log.Printf("Unhandled event type: %s", event.Type)
	}

	// Always return 200 to acknowledge receipt
	// Stripe will retry if we return an error
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"received": true}`))
}

// handlePaymentIntentSucceeded processes successful payment events
// Creates an order from a successful Stripe payment intent
func (h *StripeHandler) handlePaymentIntentSucceeded(event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Printf("Error parsing payment intent from webhook: %v", err)
		return
	}

	log.Printf("Payment succeeded for payment intent: %s (amount: %d %s)",
		paymentIntent.ID,
		paymentIntent.Amount,
		paymentIntent.Currency)

	// Extract metadata for logging
	tenantID := paymentIntent.Metadata["tenant_id"]
	cartID := paymentIntent.Metadata["cart_id"]
	orderType := paymentIntent.Metadata["order_type"]

	log.Printf("Creating order - tenant: %s, cart: %s, type: %s",
		tenantID, cartID, orderType)

	// Verify this payment intent belongs to our tenant
	if tenantID != h.config.TenantID {
		if h.config.TestMode {
			log.Printf("[TEST MODE] Tenant mismatch ignored - expected: %s, got: %s",
				h.config.TenantID, tenantID)
			log.Printf("[TEST MODE] Skipping order creation (missing required metadata)")
			log.Printf("[TEST MODE] ✓ Webhook received and parsed successfully")
			return
		}
		log.Printf("WARNING: Payment intent belongs to different tenant (expected: %s, got: %s)",
			h.config.TenantID, tenantID)
		return
	}

	// In test mode, validate required metadata before attempting order creation
	if h.config.TestMode && cartID == "" {
		log.Printf("[TEST MODE] Missing cart_id - skipping order creation")
		log.Printf("[TEST MODE] ✓ Webhook received and parsed successfully")
		return
	}

	// Create order from successful payment
	ctx := context.Background()
	order, err := h.orderService.CreateOrderFromPaymentIntent(ctx, paymentIntent.ID)
	if err != nil {
		// Check if this is an idempotency case (order already exists)
		if errors.Is(err, domain.ErrPaymentAlreadyProcessed) {
			log.Printf("Order already exists for payment intent %s (idempotent retry)", paymentIntent.ID)
			return
		}

		// Log error for investigation - this is a critical failure
		log.Printf("CRITICAL: Failed to create order from payment %s: %v", paymentIntent.ID, err)

		// Track failure in metrics and Sentry
		if telemetry.Business != nil {
			telemetry.Business.WebhookFailed.WithLabelValues(tenantID, "payment_intent.succeeded", "order_creation_failed").Inc()
		}
		telemetry.CaptureErrorWithTenant(err, tenantID, map[string]interface{}{
			"payment_intent_id": paymentIntent.ID,
			"amount":            paymentIntent.Amount,
			"cart_id":           cartID,
		})
		return
	}

	// Track successful order creation and revenue
	if telemetry.Business != nil {
		telemetry.Business.PaymentSucceeded.WithLabelValues(tenantID, orderType).Inc()
		telemetry.Business.OrdersCreated.WithLabelValues(tenantID, orderType).Inc()
		telemetry.Business.OrderValue.WithLabelValues(tenantID, orderType).Observe(float64(order.Order.TotalCents))
		telemetry.Business.RevenueCollected.WithLabelValues(tenantID, orderType).Add(float64(order.Order.TotalCents))
		telemetry.Business.WebhookProcessed.WithLabelValues(tenantID, "payment_intent.succeeded").Inc()
	}

	log.Printf("Order created successfully: %s (payment: %s, total: %d %s)",
		order.Order.OrderNumber,
		paymentIntent.ID,
		order.Order.TotalCents,
		order.Order.Currency)

	// TODO: Send order confirmation email to customer
	// TODO: Trigger fulfillment workflow (send to warehouse system)
	// TODO: Update analytics/reporting
}

// handlePaymentIntentFailed processes failed payment events
func (h *StripeHandler) handlePaymentIntentFailed(event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Printf("Error parsing payment intent from webhook: %v", err)
		return
	}

	log.Printf("Payment failed for payment intent: %s", paymentIntent.ID)

	// Extract metadata
	tenantID := paymentIntent.Metadata["tenant_id"]
	if tenantID == "" {
		tenantID = h.config.TenantID
	}

	failureReason := "unknown"
	if paymentIntent.LastPaymentError != nil {
		failureReason = string(paymentIntent.LastPaymentError.Code)
		log.Printf("Failure reason: %s (code: %s, decline_code: %s)",
			paymentIntent.LastPaymentError.Msg,
			paymentIntent.LastPaymentError.Code,
			paymentIntent.LastPaymentError.DeclineCode)
	}

	// Track payment failure
	if telemetry.Business != nil {
		telemetry.Business.PaymentFailed.WithLabelValues(tenantID, "one_time", failureReason).Inc()
		telemetry.Business.WebhookProcessed.WithLabelValues(tenantID, "payment_intent.payment_failed").Inc()
	}

	// TODO: Notify customer of payment failure
	// emailService.SendPaymentFailedEmail(customerEmail, paymentIntent.ID, failureReason)
}

// handlePaymentIntentCanceled processes canceled payment events
func (h *StripeHandler) handlePaymentIntentCanceled(event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Printf("Error parsing payment intent from webhook: %v", err)
		return
	}

	log.Printf("Payment intent canceled: %s", paymentIntent.ID)

	// Extract metadata
	tenantID := paymentIntent.Metadata["tenant_id"]
	if tenantID == "" {
		tenantID = h.config.TenantID
	}

	// Track checkout abandonment
	if telemetry.Business != nil {
		telemetry.Business.CheckoutAbandoned.WithLabelValues(tenantID).Inc()
		telemetry.Business.WebhookProcessed.WithLabelValues(tenantID, "payment_intent.canceled").Inc()
	}

	// TODO: Clean up abandoned cart
	// cartService.MarkAbandoned(paymentIntent.Metadata["cart_id"])
}

// getSubscriptionFromInvoice extracts subscription info from invoice using Stripe v83 API structure
// Returns nil if invoice is not for a subscription
func getSubscriptionFromInvoice(invoice *stripe.Invoice) *stripe.Subscription {
	if invoice.Parent == nil || invoice.Parent.SubscriptionDetails == nil {
		return nil
	}
	return invoice.Parent.SubscriptionDetails.Subscription
}

// handleInvoicePaymentSucceeded processes successful invoice payment events
// Creates orders for subscription renewals when invoice has a subscription parent
func (h *StripeHandler) handleInvoicePaymentSucceeded(event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("Error parsing invoice from webhook: %v", err)
		return
	}

	log.Printf("Invoice payment succeeded: %s (amount: %d %s)",
		invoice.ID,
		invoice.AmountPaid,
		invoice.Currency)

	// Check if this invoice is for a subscription (Stripe v83 API structure)
	subscription := getSubscriptionFromInvoice(&invoice)
	if subscription == nil || subscription.ID == "" {
		if h.config.TestMode {
			log.Printf("[TEST MODE] Invoice %s is not for a subscription (CLI trigger event)", invoice.ID)
			log.Printf("[TEST MODE] ✓ Webhook received and parsed successfully")
			return
		}
		log.Printf("Invoice %s is not for a subscription, skipping order creation", invoice.ID)
		return
	}

	log.Printf("Creating subscription renewal order for subscription: %s", subscription.ID)

	// Extract metadata for tenant validation
	var tenantID string
	if subscription.Metadata != nil {
		tenantID = subscription.Metadata["tenant_id"]
	}

	// Verify this subscription belongs to our tenant
	if tenantID != h.config.TenantID {
		if h.config.TestMode {
			log.Printf("[TEST MODE] Tenant mismatch ignored - expected: %s, got: %s",
				h.config.TenantID, tenantID)
			log.Printf("[TEST MODE] Skipping order creation (missing required metadata)")
			log.Printf("[TEST MODE] ✓ Webhook received and parsed successfully")
			return
		}
		log.Printf("WARNING: Subscription belongs to different tenant (expected: %s, got: %s)",
			h.config.TenantID, tenantID)
		return
	}

	// Convert tenant ID to pgtype.UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		log.Printf("ERROR: Invalid tenant ID format: %s", tenantID)
		return
	}

	// Create order from subscription invoice
	ctx := context.Background()
	order, err := h.subscriptionService.CreateOrderFromSubscriptionInvoice(ctx, invoice.ID, tenantUUID)
	if err != nil {
		// Check if this is an idempotency case (order already exists)
		if errors.Is(err, domain.ErrInvoiceAlreadyProcessed) {
			log.Printf("Order already exists for invoice %s (idempotent retry)", invoice.ID)
			return
		}

		// Log error for investigation - this is a critical failure
		log.Printf("CRITICAL: Failed to create order from subscription invoice %s: %v", invoice.ID, err)

		// TODO: Send alert to operations team
		// TODO: Queue for manual review
		return
	}

	log.Printf("Subscription renewal order created successfully: %s (invoice: %s, subscription: %s)",
		order.Order.OrderNumber,
		invoice.ID,
		subscription.ID)

	// TODO: Send subscription renewal email to customer
	// TODO: Trigger fulfillment workflow
}

// handleInvoicePaymentFailed processes failed invoice payment events
// Updates subscription status to past_due
func (h *StripeHandler) handleInvoicePaymentFailed(event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("Error parsing invoice from webhook: %v", err)
		return
	}

	log.Printf("Invoice payment failed: %s", invoice.ID)

	// Check if this invoice is for a subscription (Stripe v83 API structure)
	subscription := getSubscriptionFromInvoice(&invoice)
	if subscription == nil || subscription.ID == "" {
		if h.config.TestMode {
			log.Printf("[TEST MODE] Invoice %s is not for a subscription (CLI trigger event)", invoice.ID)
			log.Printf("[TEST MODE] ✓ Webhook received and parsed successfully")
			return
		}
		log.Printf("Invoice %s is not for a subscription, skipping", invoice.ID)
		return
	}

	log.Printf("Subscription %s payment failed", subscription.ID)

	// Extract metadata for tenant validation
	var tenantID string
	if subscription.Metadata != nil {
		tenantID = subscription.Metadata["tenant_id"]
	}

	// Verify this subscription belongs to our tenant
	if tenantID != h.config.TenantID {
		if h.config.TestMode {
			log.Printf("[TEST MODE] Tenant mismatch ignored - expected: %s, got: %s",
				h.config.TenantID, tenantID)
			log.Printf("[TEST MODE] ✓ Webhook received and parsed successfully")
			return
		}
		log.Printf("WARNING: Subscription belongs to different tenant (expected: %s, got: %s)",
			h.config.TenantID, tenantID)
		return
	}

	// Convert tenant ID to pgtype.UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		log.Printf("ERROR: Invalid tenant ID format: %s", tenantID)
		return
	}

	// Sync subscription status from Stripe (will update to past_due)
	ctx := context.Background()
	err := h.subscriptionService.SyncSubscriptionFromWebhook(ctx, domain.SyncSubscriptionParams{
		TenantID:               tenantUUID,
		ProviderSubscriptionID: subscription.ID,
		EventType:              string(event.Type),
		EventID:                event.ID,
	})
	if err != nil {
		log.Printf("ERROR: Failed to sync subscription %s: %v", subscription.ID, err)
		return
	}

	log.Printf("Subscription %s status updated to past_due", subscription.ID)

	// TODO: Send payment failure email to customer
	// TODO: Attempt retry logic based on dunning settings
}

// handleSubscriptionUpdated processes subscription update events
// Syncs subscription status and settings from Stripe
func (h *StripeHandler) handleSubscriptionUpdated(event stripe.Event) {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("Error parsing subscription from webhook: %v", err)
		return
	}

	log.Printf("Subscription updated: %s (status: %s)", subscription.ID, subscription.Status)

	// Extract metadata for tenant validation
	var tenantID string
	if subscription.Metadata != nil {
		tenantID = subscription.Metadata["tenant_id"]
	}

	// Verify this subscription belongs to our tenant
	if tenantID != h.config.TenantID {
		if h.config.TestMode {
			log.Printf("[TEST MODE] Tenant mismatch ignored - expected: %s, got: %s",
				h.config.TenantID, tenantID)
			log.Printf("[TEST MODE] ✓ Webhook received and parsed successfully")
			return
		}
		log.Printf("WARNING: Subscription belongs to different tenant (expected: %s, got: %s)",
			h.config.TenantID, tenantID)
		return
	}

	// Convert tenant ID to pgtype.UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		log.Printf("ERROR: Invalid tenant ID format: %s", tenantID)
		return
	}

	// Sync subscription from Stripe
	ctx := context.Background()
	err := h.subscriptionService.SyncSubscriptionFromWebhook(ctx, domain.SyncSubscriptionParams{
		TenantID:               tenantUUID,
		ProviderSubscriptionID: subscription.ID,
		EventType:              string(event.Type),
		EventID:                event.ID,
	})
	if err != nil {
		log.Printf("ERROR: Failed to sync subscription %s: %v", subscription.ID, err)
		return
	}

	log.Printf("Subscription %s synced successfully", subscription.ID)
}

// handleSubscriptionDeleted processes subscription deletion events
// Updates subscription status to expired/cancelled
func (h *StripeHandler) handleSubscriptionDeleted(event stripe.Event) {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("Error parsing subscription from webhook: %v", err)
		return
	}

	log.Printf("Subscription deleted: %s", subscription.ID)

	// Extract metadata for tenant validation
	var tenantID string
	if subscription.Metadata != nil {
		tenantID = subscription.Metadata["tenant_id"]
	}

	// Verify this subscription belongs to our tenant
	if tenantID != h.config.TenantID {
		if h.config.TestMode {
			log.Printf("[TEST MODE] Tenant mismatch ignored - expected: %s, got: %s",
				h.config.TenantID, tenantID)
			log.Printf("[TEST MODE] ✓ Webhook received and parsed successfully")
			return
		}
		log.Printf("WARNING: Subscription belongs to different tenant (expected: %s, got: %s)",
			h.config.TenantID, tenantID)
		return
	}

	// Convert tenant ID to pgtype.UUID
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		log.Printf("ERROR: Invalid tenant ID format: %s", tenantID)
		return
	}

	// Sync subscription from Stripe (will update to expired)
	ctx := context.Background()
	err := h.subscriptionService.SyncSubscriptionFromWebhook(ctx, domain.SyncSubscriptionParams{
		TenantID:               tenantUUID,
		ProviderSubscriptionID: subscription.ID,
		EventType:              string(event.Type),
		EventID:                event.ID,
	})
	if err != nil {
		log.Printf("ERROR: Failed to sync subscription %s: %v", subscription.ID, err)
		return
	}

	log.Printf("Subscription %s marked as expired", subscription.ID)
}
