// internal/handler/webhook.go
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

type WebhookHandler struct {
	signing_secret  string
	orderService    interfaces.OrderService
	customerService interfaces.CustomerService
	eventPublisher  interfaces.EventPublisher
}

func NewWebhookHandler(
	signing_secret string,
	orderService interfaces.OrderService,
	customerService interfaces.CustomerService,
	eventPublisher interfaces.EventPublisher,
) *WebhookHandler {
	if signing_secret == "" {
		log.Printf("Missing STRIPE_WEBHOOK_SECRET environment variable")
	}

	return &WebhookHandler{
		signing_secret:  signing_secret,
		orderService:    orderService,
		customerService: customerService,
		eventPublisher:  eventPublisher,
	}
}

// HandleStripeWebhook processes incoming Stripe webhook events
func (h *WebhookHandler) HandleStripeWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("Error reading webhook body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Verify webhook signature
	event, err := webhook.ConstructEvent(body, c.Request().Header.Get("Stripe-Signature"), h.signing_secret)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid signature"})
	}

	// Handle the event based on its type
	switch event.Type {
	case "checkout.session.completed":
		return h.handleCheckoutSessionCompleted(c, event)
	case "payment_intent.succeeded":
		return h.handlePaymentIntentSucceeded(c, event)
	case "payment_intent.payment_failed":
		return h.handlePaymentIntentFailed(c, event)
	default:
		// Log unhandled events but return success
		log.Printf("📝 Received unhandled webhook event type: %s", event.Type)
		return c.JSON(http.StatusOK, map[string]bool{"received": true})
	}
}

// handleCheckoutSessionCompleted processes successful checkout session completion
func (h *WebhookHandler) handleCheckoutSessionCompleted(c echo.Context, event stripe.Event) error {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Printf("Error parsing checkout session: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid event data"})
	}

	log.Printf("✅ Checkout session completed: %s", session.ID)

	// Extract customer ID from metadata if present
	var customerID *int32
	if session.Metadata != nil {
		if customerIDStr, exists := session.Metadata["customer_id"]; exists {
			if id, err := parseCustomerID(customerIDStr); err == nil {
				customerID = &id
			}
		}
	}

	// Publish checkout completed event
	eventData := map[string]interface{}{
		"stripe_session_id": session.ID,
		"payment_status":    session.PaymentStatus,
		"amount_total":      session.AmountTotal,
		"currency":          session.Currency,
	}

	if customerID != nil {
		eventData["customer_id"] = *customerID
	}

	if err := h.publishEvent(c.Request().Context(), interfaces.EventCheckoutCompleted, fmt.Sprintf("session:%s", session.ID), eventData); err != nil {
		log.Printf("Failed to publish checkout completed event: %v", err)
		// Don't fail the webhook - Stripe expects 200 even if our internal processing fails
	}

	return c.JSON(http.StatusOK, map[string]bool{"received": true})
}

// handlePaymentIntentSucceeded processes successful payment confirmation
func (h *WebhookHandler) handlePaymentIntentSucceeded(c echo.Context, event stripe.Event) error {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Printf("Error parsing payment intent: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid event data"})
	}

	log.Printf("💰 Payment succeeded: %s (Amount: %d %s)", paymentIntent.ID, paymentIntent.Amount, paymentIntent.Currency)

	// Extract customer ID from metadata
	var customerID *int32
	if paymentIntent.Metadata != nil {
		if customerIDStr, exists := paymentIntent.Metadata["customer_id"]; exists {
			if id, err := parseCustomerID(customerIDStr); err == nil {
				customerID = &id
			}
		}
	}

	// If we have a customer ID, create an order
	if customerID != nil {
		order, err := h.orderService.CreateOrderFromPayment(
			c.Request().Context(),
			*customerID,
			paymentIntent.ID,
			int32(paymentIntent.Amount),
		)
		if err != nil {
			log.Printf("Failed to create order from payment: %v", err)
			// Log error but return success to Stripe - we don't want Stripe to retry
			// We can handle order creation through manual reconciliation if needed
		} else {
			log.Printf("📦 Order created successfully: ID %d", order.ID)
		}
	} else {
		log.Printf("⚠️ No customer ID found in payment intent metadata")
	}

	// Publish payment confirmed event
	eventData := map[string]interface{}{
		"payment_intent_id": paymentIntent.ID,
		"amount":            paymentIntent.Amount,
		"currency":          paymentIntent.Currency,
		"status":            paymentIntent.Status,
	}

	if customerID != nil {
		eventData["customer_id"] = *customerID
	}

	if err := h.publishEvent(c.Request().Context(), interfaces.EventPaymentConfirmed, fmt.Sprintf("payment:%s", paymentIntent.ID), eventData); err != nil {
		log.Printf("Failed to publish payment confirmed event: %v", err)
	}

	return c.JSON(http.StatusOK, map[string]bool{"received": true})
}

// handlePaymentIntentFailed processes failed payment attempts
func (h *WebhookHandler) handlePaymentIntentFailed(c echo.Context, event stripe.Event) error {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Printf("Error parsing payment intent: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid event data"})
	}

	log.Printf("❌ Payment failed: %s", paymentIntent.ID)

	// Extract error information
	var errorMessage string
	if paymentIntent.LastPaymentError != nil {
		errorMessage = paymentIntent.LastPaymentError.Msg
	} else {
		errorMessage = "Payment failed"
	}

	// Extract customer ID from metadata
	var customerID *int32
	if paymentIntent.Metadata != nil {
		if customerIDStr, exists := paymentIntent.Metadata["customer_id"]; exists {
			if id, err := parseCustomerID(customerIDStr); err == nil {
				customerID = &id
			}
		}
	}

	// Publish payment failed event
	eventData := map[string]interface{}{
		"payment_intent_id": paymentIntent.ID,
		"error_message":     errorMessage,
		"amount":            paymentIntent.Amount,
		"currency":          paymentIntent.Currency,
	}

	if customerID != nil {
		eventData["customer_id"] = *customerID
	}

	if err := h.publishEvent(c.Request().Context(), interfaces.EventPaymentFailed, fmt.Sprintf("payment:%s", paymentIntent.ID), eventData); err != nil {
		log.Printf("Failed to publish payment failed event: %v", err)
	}

	return c.JSON(http.StatusOK, map[string]bool{"received": true})
}

// Helper function to publish events
func (h *WebhookHandler) publishEvent(ctx context.Context, eventType string, aggregateID string, data map[string]interface{}) error {
	event := interfaces.Event{
		ID:          fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		Type:        eventType,
		AggregateID: aggregateID,
		Data:        data,
		Timestamp:   time.Now(),
	}

	return h.eventPublisher.PublishEvent(ctx, event)
}

// Helper function to parse customer ID from string
func parseCustomerID(customerIDStr string) (int32, error) {
	if customerIDStr == "" {
		return 0, fmt.Errorf("empty customer ID")
	}

	id, err := strconv.ParseInt(customerIDStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid customer ID format: %w", err)
	}

	return int32(id), nil
}
