// internal/handler/webhook.go
package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/dukerupert/freyja/internal/server/middleware"
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
	logger := middleware.GetLogger(c).With().
		Str("component", "WebhookHandler").
		Str("function", "HandleStripeWebhook").
		Str("provider", "stripe").
		Logger()

	logger.Info().Msg("Processing incoming Stripe webhook")

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		logger.Error().Err(err).Msg("Error reading webhook body")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	logger = logger.With().Int("body_size", len(body)).Logger()

	// Use the payment provider to verify and parse the webhook
	event, err := h.paymentProvider.VerifyWebhook(body, c.Request().Header.Get("Stripe-Signature"))
	if err != nil {
		logger.Error().Err(err).Msg("Webhook signature verification failed")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid signature"})
	}

	logger = logger.With().
		Str("event_type", event.Type).
		Str("event_id", event.ID).
		Time("event_created_at", event.CreatedAt).
		Logger()

	logger.Info().Msg("Processing Stripe webhook event")

	switch event.Type {
	case "payment_intent.succeeded":
		logger.Info().Msg("Routing to payment intent succeeded handler")
		return h.handlePaymentIntentSucceeded(c, event.Data)
	default:
		// Log unhandled events but don't error - allows for easy extension
		logger.Info().Msg("Passing webhook event to stripe provider to handle")
	}

	// Delegate webhook processing to the payment provider
	// This is where the magic happens - no more duplicated Stripe logic!
	if stripeProvider, ok := h.paymentProvider.(*provider.StripeProvider); ok {
		logger.Info().Msg("Delegating webhook event to Stripe provider")
		err = stripeProvider.HandleWebhookEvent(c.Request().Context(), event, h.orderService, h.customerService)
		if err != nil {
			logger.Error().Err(err).Msg("Error processing webhook event")
			// Still return 200 to Stripe to prevent retries for business logic errors
		} else {
			logger.Info().Msg("Successfully processed webhook event")
		}
	} else {
		logger.Error().Msg("Unsupported payment provider for webhook processing")
	}

	return c.JSON(http.StatusOK, map[string]bool{"received": true})
}

func (h *WebhookHandler) handlePaymentIntentSucceeded(c echo.Context, eventData map[string]interface{}) error {
	// Get logger from context (should have request ID from middleware)
	ctx := c.Request().Context()
	logger := middleware.GetLogger(c).With().
		Str("component", "WebhookHandler").
		Str("function", "handlePaymentIntentSucceeded").
		Str("event_type", "payment_intent.succeeded").
		Logger()

	logger.Info().Msg("Processing payment intent succeeded webhook")

	// Extract Stripe customer ID
	stripeCustomerID, ok := eventData["customer"].(string)
	if !ok {
		logger.Error().Msg("No customer ID in payment intent")
		return fmt.Errorf("no customer ID in payment intent")
	}

	// Extract Payment Intent ID
	paymentIntentID, ok := eventData["id"].(string)
	if !ok {
		logger.Error().Str("stripe_customer_id", stripeCustomerID).Msg("No ID in payment intent")
		return fmt.Errorf("no ID in payment intent")
	}

	// Extract Amount
	amountFloat, ok := eventData["amount"].(float64)
	if !ok {
		logger.Error().
			Str("stripe_customer_id", stripeCustomerID).
			Str("payment_intent_id", paymentIntentID).
			Msg("No amount in payment intent")
		return fmt.Errorf("no amount in payment intent")
	}
	amount := int32(amountFloat)

	logger = logger.With().
		Str("stripe_customer_id", stripeCustomerID).
		Str("payment_intent_id", paymentIntentID).
		Int32("amount", amount).
		Logger()

	logger.Info().Msg("Extracted payment intent details")

	// First try to find by Stripe customer ID in your database
	internalCustomer, err := h.customerService.GetCustomerByStripeID(ctx, stripeCustomerID)
	if err == nil && internalCustomer != nil {
		logger.Info().
			Int32("internal_customer_id", internalCustomer.ID).
			Msg("Found existing customer for Stripe customer")

		// Found by Stripe ID, proceed with order creation
		_, err := h.orderService.CreateOrderFromPayment(ctx, internalCustomer.ID, paymentIntentID, amount)
		if err != nil {
			logger.Error().
				Err(err).
				Int32("internal_customer_id", internalCustomer.ID).
				Msg("Failed to create order from payment")
			return fmt.Errorf("failed to create order from payment: %w", err)
		}

		logger.Info().
			Int32("internal_customer_id", internalCustomer.ID).
			Msg("Order created successfully for existing customer")
		return nil
	}

	// No internal customer found - this was likely a guest checkout
	logger.Info().Msg("Guest checkout detected, creating new customer for Stripe customer")

	// Get full Stripe customer details to create internal customer
	stripeCustomer, err := h.paymentProvider.GetCustomer(ctx, stripeCustomerID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get Stripe customer details")
		return fmt.Errorf("failed to get Stripe customer details: %w", err)
	}

	logger = logger.With().Str("customer_email", stripeCustomer.Email).Logger()

	// Create new internal customer from Stripe customer
	newCustomer, err := h.customerService.CreateCustomerFromStripe(ctx, stripeCustomerID, stripeCustomer.Email)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create customer from Stripe customer")
		return fmt.Errorf("failed to create customer from Stripe: %w", err)
	}

	logger.Info().
		Int32("internal_customer_id", newCustomer.ID).
		Msg("Created new customer from guest checkout")

	// Now create the order for the new customer
	_, err = h.orderService.CreateOrderFromPayment(ctx, newCustomer.ID, paymentIntentID, amount)
	if err != nil {
		logger.Error().
			Err(err).
			Int32("internal_customer_id", newCustomer.ID).
			Msg("Failed to create order for new customer")
		return fmt.Errorf("failed to create order from payment: %w", err)
	}

	logger.Info().
		Int32("internal_customer_id", newCustomer.ID).
		Msg("Order created successfully for new customer")
	return nil
}
