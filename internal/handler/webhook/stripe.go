package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v83"
)

// StripeHandler handles Stripe webhook events
type StripeHandler struct {
	provider             billing.Provider
	orderService         service.OrderService
	subscriptionService  service.SubscriptionService
	config               StripeWebhookConfig
}

// StripeWebhookConfig contains configuration for Stripe webhook handling
type StripeWebhookConfig struct {
	// WebhookSecret is the webhook signing secret from Stripe dashboard
	WebhookSecret string

	// TenantID is used to scope payment intents (for multi-tenant isolation)
	// In production, this would come from the webhook endpoint URL or subdomain
	TenantID string
}

// NewStripeHandler creates a new Stripe webhook handler
func NewStripeHandler(provider billing.Provider, orderService service.OrderService, subscriptionService service.SubscriptionService, config StripeWebhookConfig) *StripeHandler {
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
	log.Printf("[WEBHOOK] Received request: %s %s", r.Method, r.URL.Path)

	// Only accept POST requests
	if r.Method != http.MethodPost {
		log.Printf("[WEBHOOK] Rejected: method %s not allowed", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[WEBHOOK] Error reading payload: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	log.Printf("[WEBHOOK] Payload size: %d bytes", len(payload))

	// Get the signature from headers
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		log.Printf("[WEBHOOK] Missing Stripe-Signature header")
		http.Error(w, "Missing signature", http.StatusBadRequest)
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
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}
	log.Printf("[WEBHOOK] Signature verification SUCCESS")

	// Parse the event
	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("Error parsing webhook JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log the event for debugging
	log.Printf("Received Stripe webhook event: %s (ID: %s)", event.Type, event.ID)

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
	w.Write([]byte(`{"received": true}`))
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
		log.Printf("WARNING: Payment intent belongs to different tenant (expected: %s, got: %s)",
			h.config.TenantID, tenantID)
		return
	}

	// Create order from successful payment
	ctx := context.Background()
	order, err := h.orderService.CreateOrderFromPaymentIntent(ctx, paymentIntent.ID)
	if err != nil {
		// Check if this is an idempotency case (order already exists)
		if errors.Is(err, service.ErrPaymentAlreadyProcessed) {
			log.Printf("Order already exists for payment intent %s (idempotent retry)", paymentIntent.ID)
			return
		}

		// Log error for investigation - this is a critical failure
		log.Printf("CRITICAL: Failed to create order from payment %s: %v", paymentIntent.ID, err)

		// TODO: Send alert to operations team
		// TODO: Queue for manual review
		return
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
	// TODO: Implement failure handling
	//
	// Steps:
	// 1. Parse payment intent from event
	// 2. Extract failure reason and error code
	// 3. Log failure for debugging
	// 4. Send email to customer with:
	//    - What went wrong (card declined, insufficient funds, etc.)
	//    - Instructions to retry with different payment method
	// 5. Update cart status to "payment_failed"
	// 6. Optionally: Implement retry logic with exponential backoff

	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Printf("Error parsing payment intent from webhook: %v", err)
		return
	}

	log.Printf("Payment failed for payment intent: %s", paymentIntent.ID)

	if paymentIntent.LastPaymentError != nil {
		log.Printf("Failure reason: %s (code: %s, decline_code: %s)",
			paymentIntent.LastPaymentError.Msg,
			paymentIntent.LastPaymentError.Code,
			paymentIntent.LastPaymentError.DeclineCode)
	}

	// TODO: Notify customer of payment failure
	// emailService.SendPaymentFailedEmail(customerEmail, paymentIntent.ID, failureReason)
}

// handlePaymentIntentCanceled processes canceled payment events
func (h *StripeHandler) handlePaymentIntentCanceled(event stripe.Event) {
	// TODO: Implement cancellation handling
	//
	// Steps:
	// 1. Parse payment intent from event
	// 2. Mark cart as abandoned
	// 3. Optionally: Send "complete your order" reminder email after 24 hours
	// 4. Clean up any reserved inventory

	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Printf("Error parsing payment intent from webhook: %v", err)
		return
	}

	log.Printf("Payment intent canceled: %s", paymentIntent.ID)

	// TODO: Clean up abandoned cart
	// cartService.MarkAbandoned(paymentIntent.Metadata["cart_id"])
}

// handleInvoicePaymentSucceeded processes successful invoice payment events
// Creates orders for subscription renewals when invoice.subscription is set
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

	// Check if this invoice is for a subscription
	if invoice.Subscription == nil || invoice.Subscription.ID == "" {
		log.Printf("Invoice %s is not for a subscription, skipping order creation", invoice.ID)
		return
	}

	log.Printf("Creating subscription renewal order for subscription: %s", invoice.Subscription.ID)

	// Extract metadata for tenant validation
	var tenantID string
	if invoice.Subscription.Metadata != nil {
		tenantID = invoice.Subscription.Metadata["tenant_id"]
	}

	// Verify this subscription belongs to our tenant
	if tenantID != h.config.TenantID {
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
		if errors.Is(err, service.ErrInvoiceAlreadyProcessed) {
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
		invoice.Subscription.ID)

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

	// Check if this invoice is for a subscription
	if invoice.Subscription == nil || invoice.Subscription.ID == "" {
		log.Printf("Invoice %s is not for a subscription, skipping", invoice.ID)
		return
	}

	log.Printf("Subscription %s payment failed", invoice.Subscription.ID)

	// Extract metadata for tenant validation
	var tenantID string
	if invoice.Subscription.Metadata != nil {
		tenantID = invoice.Subscription.Metadata["tenant_id"]
	}

	// Verify this subscription belongs to our tenant
	if tenantID != h.config.TenantID {
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
	err := h.subscriptionService.SyncSubscriptionFromWebhook(ctx, service.SyncSubscriptionParams{
		TenantID:               tenantUUID,
		ProviderSubscriptionID: invoice.Subscription.ID,
		EventType:              event.Type,
		EventID:                event.ID,
	})
	if err != nil {
		log.Printf("ERROR: Failed to sync subscription %s: %v", invoice.Subscription.ID, err)
		return
	}

	log.Printf("Subscription %s status updated to past_due", invoice.Subscription.ID)

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
	err := h.subscriptionService.SyncSubscriptionFromWebhook(ctx, service.SyncSubscriptionParams{
		TenantID:               tenantUUID,
		ProviderSubscriptionID: subscription.ID,
		EventType:              event.Type,
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
	err := h.subscriptionService.SyncSubscriptionFromWebhook(ctx, service.SyncSubscriptionParams{
		TenantID:               tenantUUID,
		ProviderSubscriptionID: subscription.ID,
		EventType:              event.Type,
		EventID:                event.ID,
	})
	if err != nil {
		log.Printf("ERROR: Failed to sync subscription %s: %v", subscription.ID, err)
		return
	}

	log.Printf("Subscription %s marked as expired", subscription.ID)
}
