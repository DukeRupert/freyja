// internal/handler/webhook.go
package handler

import (
	"io"
	"log"
	"net/http"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/dukerupert/freyja/internal/provider"
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