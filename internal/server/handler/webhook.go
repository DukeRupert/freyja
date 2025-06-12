// internal/handler/webhook.go
package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/dukerupert/freyja/internal/server/provider"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/labstack/echo/v4"
)

type WebhookHandler struct {
	paymentProvider interfaces.PaymentProvider
	orderService    interfaces.OrderService
	customerService interfaces.CustomerService
}

func NewWebhookHandler(
	paymentProvider interfaces.PaymentProvider,
	orderService interfaces.OrderService,
	customerService interfaces.CustomerService,
) *WebhookHandler {
	return &WebhookHandler{
		paymentProvider: paymentProvider,
		orderService:    orderService,
		customerService: customerService,
	}
}

// HandleStripeWebhook processes incoming Stripe webhook events
func (h *WebhookHandler) HandleStripeWebhook(c echo.Context) error {
	ctx := c.Request().Context()
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("Error reading webhook body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Use the payment provider to verify and parse the webhook
	event, err := h.paymentProvider.VerifyWebhook(body, c.Request().Header.Get("Stripe-Signature"))
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid signature"})
	}

	log.Printf("🔄 Processing Stripe webhook event: %s", event.Type)

	switch event.Type {
	case "payment_intent.succeeded":
		return h.handlePaymentIntentSucceeded(ctx, event.Data)
	default:
		// Log unhandled events but don't error - allows for easy extension
		log.Printf("📝 Passing webhook event to stripe provider to handle: %s", event.Type)
	}

	// Delegate webhook processing to the payment provider
	// This is where the magic happens - no more duplicated Stripe logic!
	if stripeProvider, ok := h.paymentProvider.(*provider.StripeProvider); ok {
		err = stripeProvider.HandleWebhookEvent(c.Request().Context(), event, h.orderService, h.customerService)
		if err != nil {
			log.Printf("Error processing webhook event: %v", err)
			// Still return 200 to Stripe to prevent retries for business logic errors
		}
	} else {
		log.Printf("Unsupported payment provider for webhook processing")
	}

	return c.JSON(http.StatusOK, map[string]bool{"received": true})
}

func (h *WebhookHandler) handlePaymentIntentSucceeded(ctx context.Context, eventData map[string]interface{}) error {
	// Extract Stripe customer ID
	stripeCustomerID, ok := eventData["customer"].(string)
	if !ok {
		return fmt.Errorf("no customer ID in payment intent")
	}

	// Extract Payment Intent ID
	paymentIntentID, ok := eventData["id"].(string)
	if !ok {
		return fmt.Errorf("no ID in payment intent")
	}

	// Extract Amount
	amountFloat, ok := eventData["amount"].(float64)
	if !ok {
		return fmt.Errorf("no amount in payment intent")
	}
	amount := int32(amountFloat)

	// First try to find by Stripe customer ID in your database
	internalCustomer, err := h.customerService.GetCustomerByStripeID(ctx, stripeCustomerID)
	if err == nil && internalCustomer != nil {
		log.Printf("✅ Found existing customer %d for Stripe customer %s", internalCustomer.ID, stripeCustomerID)

		// Found by Stripe ID, proceed with order creation
		_, err := h.orderService.CreateOrderFromPayment(ctx, internalCustomer.ID, paymentIntentID, amount)
		if err != nil {
			return fmt.Errorf("failed to create order from payment: %w", err)
		}

		log.Printf("📦 Order created successfully for customer %d (Payment Intent: %s)", internalCustomer.ID, paymentIntentID)
		return nil
	}

	// No internal customer found - this was likely a guest checkout
	log.Printf("🆕 Guest checkout detected, creating new customer for Stripe customer %s", stripeCustomerID)

	// Get full Stripe customer details to create internal customer
	stripeCustomer, err := h.paymentProvider.GetCustomer(ctx, stripeCustomerID)
	if err != nil {
		log.Printf("❌ Failed to get Stripe customer details for %s: %v", stripeCustomerID, err)
		return fmt.Errorf("failed to get Stripe customer details: %w", err)
	}

	// Create new internal customer from Stripe customer
	newCustomer, err := h.customerService.CreateCustomerFromStripe(ctx, stripeCustomerID, stripeCustomer.Email)
	if err != nil {
		log.Printf("❌ Failed to create customer from Stripe customer %s: %v", stripeCustomerID, err)
		return fmt.Errorf("failed to create customer from Stripe: %w", err)
	}

	log.Printf("✅ Created new customer %d from guest checkout (Stripe: %s, Email: %s)",
		newCustomer.ID, stripeCustomerID, stripeCustomer.Email)

	// Now create the order for the new customer
	_, err = h.orderService.CreateOrderFromPayment(ctx, newCustomer.ID, paymentIntentID, amount)
	if err != nil {
		log.Printf("❌ Failed to create order for new customer %d: %v", newCustomer.ID, err)
		return fmt.Errorf("failed to create order from payment: %w", err)
	}

	log.Printf("📦 Order created successfully for new customer %d (Payment Intent: %s)", newCustomer.ID, paymentIntentID)
	return nil
}
