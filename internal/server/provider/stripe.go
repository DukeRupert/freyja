// internal/provider/stripe.go
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/webhook"
)

type StripeProvider struct {
	apiKey         string
	signing_secret string
	events         interfaces.EventPublisher
}

func NewStripeProvider(apiKey string, signing_secret string, events interfaces.EventPublisher) (*StripeProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Stripe API key is required")
	}

	if signing_secret == "" {
		return nil, fmt.Errorf("Stripe event signing secret is required")
	}

	provider := &StripeProvider{
		apiKey:         apiKey,
		signing_secret: signing_secret,
		events:         events,
	}

	// Set the global Stripe API key
	stripe.Key = apiKey

	// Perform health check
	if err := provider.HealthCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("Stripe health check failed: %w", err)
	}

	log.Println("✅ Stripe provider initialized and health check passed")
	return provider, nil
}

// HealthCheck verifies the API key and Stripe service connectivity
func (s *StripeProvider) HealthCheck(ctx context.Context) error {
	// Attempt to retrieve account information to verify API key and connectivity
	_, err := account.Get()
	if err != nil {
		return fmt.Errorf("failed to connect to Stripe: %w", err)
	}

	log.Printf("Stripe health check passed - API key valid and service reachable")
	return nil
}

// CreateCheckoutSession creates a Stripe Checkout session
func (s *StripeProvider) CreateCheckoutSession(ctx context.Context, req interfaces.CheckoutSessionRequest) (*interfaces.CheckoutSessionResponse, error) {
	// Convert cart items to Stripe line items using existing Price IDs
	var lineItems []*stripe.CheckoutSessionLineItemParams
	var mode stripe.CheckoutSessionMode = stripe.CheckoutSessionModePayment

	for _, item := range req.Items {
		// Use existing Stripe Price ID (should always be present after sync)
		if item.StripePriceID != "" {
			// Use the existing Stripe Price ID (this will link to the real product)
			lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
				Price:    stripe.String(item.StripePriceID),
				Quantity: stripe.Int64(int64(item.Quantity)),
			})

			// Check if this is a subscription item (if any item has recurring pricing)
			if item.PurchaseType == "subscription" {
				mode = stripe.CheckoutSessionModeSubscription
			}
		} else {
			// Fallback to PriceData if no Stripe Price ID (shouldn't happen after sync)
			log.Printf("Warning: Cart item %d missing Stripe Price ID, falling back to PriceData", item.ID)
			lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(fmt.Sprintf("Product Variant %d", item.ProductVariantID)), // Fixed: Use ProductVariantID
					},
					UnitAmount: stripe.Int64(int64(item.Price)),
				},
				Quantity: stripe.Int64(int64(item.Quantity)),
			})
		}
	}

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems:          lineItems,
		Mode:               stripe.String(string(mode)),
		SuccessURL:         stripe.String(req.SuccessURL),
		CancelURL:          stripe.String(req.CancelURL),
		Metadata: map[string]string{
			"source": "ecommerce_api",
		},
	}

	// Add customer information if available
	if req.CustomerID != nil {
		params.Metadata["customer_id"] = strconv.Itoa(int(*req.CustomerID))
	}

	// Set customer email if provided
	if req.CustomerEmail != nil {
		params.CustomerEmail = stripe.String(*req.CustomerEmail)
	}

	// Create the session
	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe checkout session: %w", err)
	}

	log.Printf("✅ Created Stripe checkout session: %s (mode: %s)", sess.ID, mode)

	return &interfaces.CheckoutSessionResponse{
		SessionID:   sess.ID,
		CheckoutURL: sess.URL,
	}, nil
}

// VerifyWebhook verifies the webhook signature and parses the event
func (s *StripeProvider) VerifyWebhook(payload []byte, signature string) (*interfaces.PaymentWebhookEvent, error) {

	event, err := webhook.ConstructEvent(payload, signature, s.signing_secret)
	if err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	// Convert Stripe event to our interface format
	eventData := make(map[string]interface{})
	if err := json.Unmarshal(event.Data.Raw, &eventData); err != nil {
		return nil, fmt.Errorf("failed to parse event data: %w", err)
	}

	return &interfaces.PaymentWebhookEvent{
		Type:      string(event.Type),
		ID:        event.ID,
		Data:      eventData,
		CreatedAt: time.Unix(event.Created, 0),
	}, nil
}

// CreateCustomer creates a customer in Stripe
func (s *StripeProvider) CreateCustomer(ctx context.Context, cust database.Customers) (*interfaces.PaymentCustomer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(cust.Email),
		Metadata: map[string]string{
			"internal_customer_id": strconv.Itoa(int(cust.ID)),
		},
	}

	// Add name if available
	if cust.FirstName.Valid || cust.LastName.Valid {
		name := ""
		if cust.FirstName.Valid {
			name += cust.FirstName.String
		}
		if cust.LastName.Valid {
			if name != "" {
				name += " "
			}
			name += cust.LastName.String
		}
		if name != "" {
			params.Name = stripe.String(name)
		}
	}

	stripeCustomer, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	log.Printf("✅ Created Stripe customer: %s for email: %s", stripeCustomer.ID, cust.Email)

	return &interfaces.PaymentCustomer{
		ID:    stripeCustomer.ID,
		Email: stripeCustomer.Email,
	}, nil
}

// GetCustomer retrieves a customer from Stripe
func (s *StripeProvider) GetCustomer(ctx context.Context, customerID string) (*interfaces.PaymentCustomer, error) {
	stripeCustomer, err := customer.Get(customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe customer: %w", err)
	}

	return &interfaces.PaymentCustomer{
		ID:    stripeCustomer.ID,
		Email: stripeCustomer.Email,
	}, nil
}

// HandleWebhookEvent processes a verified webhook event and performs business logic
func (s *StripeProvider) HandleWebhookEvent(ctx context.Context, event *interfaces.PaymentWebhookEvent, orderService interfaces.OrderService, customerService interfaces.CustomerService) error {
	log.Printf("🔄 Processing Stripe webhook event: %s", event.Type)

	switch event.Type {
	case interfaces.WebhookCheckoutSessionCompleted:
		return s.handleCheckoutSessionCompleted(ctx, event.Data)
	case interfaces.WebhookPaymentIntentFailed:
		return s.handlePaymentIntentFailed(ctx, event.Data)
	case interfaces.WebhookCustomerCreated:
		return s.handleCustomerCreated(ctx, event.Data, customerService)
	default:
		// Log unhandled events but don't error - allows for easy extension
		log.Printf("📝 Received unhandled Stripe webhook event type: %s", event.Type)
		return nil
	}
}

// Internal webhook handlers

func (s *StripeProvider) handleCheckoutSessionCompleted(ctx context.Context, eventData map[string]interface{}) error {
	sessionID, ok := eventData["id"].(string)
	if !ok {
		return fmt.Errorf("invalid session ID in checkout.session.completed event")
	}

	log.Printf("✅ Checkout session completed: %s", sessionID)

	// Extract customer info from metadata
	var customerID *int32
	if metadata, ok := eventData["metadata"].(map[string]interface{}); ok {
		if customerIDStr, exists := metadata["customer_id"].(string); exists {
			if id, err := strconv.ParseInt(customerIDStr, 10, 32); err == nil {
				customerID32 := int32(id)
				customerID = &customerID32
			}
		}
	}

	// Extract payment details
	var amountTotal int32
	if amount, ok := eventData["amount_total"].(float64); ok {
		amountTotal = int32(amount)
	}

	var paymentIntentID *string
	if piID, ok := eventData["payment_intent"].(string); ok {
		paymentIntentID = &piID
	}

	// Publish checkout completed event to event bus
	if s.events != nil {
		event := interfaces.Event{
			ID:          interfaces.GenerateEventID(),
			Type:        interfaces.EventCheckoutSessionCompleted,
			AggregateID: fmt.Sprintf("checkout:%s", sessionID),
			Data: map[string]interface{}{
				"stripe_session_id": sessionID,
				"customer_id":       customerID,
				"amount_total":      amountTotal,
				"payment_intent_id": paymentIntentID,
				"payment_status":    eventData["payment_status"],
				"completed_at":      time.Now().Unix(),
			},
			Timestamp: time.Now(),
			Version:   1,
		}

		if err := s.events.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish checkout.session_completed event: %v", err)
			return err
		}

		log.Printf("📤 Published checkout.session_completed event for session %s", sessionID)
	}

	return nil
}

// Helper method to find internal customer ID from Stripe customer ID

func (s *StripeProvider) handlePaymentIntentFailed(ctx context.Context, eventData map[string]interface{}) error {
	paymentIntentID, ok := eventData["id"].(string)
	if !ok {
		return fmt.Errorf("invalid payment intent ID in payment_intent.payment_failed event")
	}

	log.Printf("❌ Payment failed: %s", paymentIntentID)

	// Extract error information
	var errorMessage string
	if lastPaymentError, ok := eventData["last_payment_error"].(map[string]interface{}); ok {
		if msg, exists := lastPaymentError["message"].(string); exists {
			errorMessage = msg
		}
	}
	if errorMessage == "" {
		errorMessage = "Payment failed"
	}

	log.Printf("Error details: %s", errorMessage)
	return nil
}

func (s *StripeProvider) handleCustomerCreated(ctx context.Context, eventData map[string]interface{}, customerService interfaces.CustomerService) error {
	stripeCustomerID, ok := eventData["id"].(string)
	if !ok {
		return fmt.Errorf("invalid customer ID in customer.created event")
	}

	log.Printf("👤 Stripe customer created: %s", stripeCustomerID)

	// You could sync this back to your customer service if needed
	// This is useful if customers are created directly in Stripe dashboard

	return nil
}

// RefundPayment creates a refund for a payment
func (s *StripeProvider) RefundPayment(ctx context.Context, paymentID string, amount int) (*interfaces.RefundResponse, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentID),
	}

	// If amount is specified and not full refund
	if amount > 0 {
		params.Amount = stripe.Int64(int64(amount))
	}

	stripeRefund, err := refund.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe refund: %w", err)
	}

	log.Printf("✅ Created Stripe refund: %s for payment: %s", stripeRefund.ID, paymentID)

	return &interfaces.RefundResponse{
		ID:     stripeRefund.ID,
		Amount: int(stripeRefund.Amount),
		Status: string(stripeRefund.Status),
	}, nil
}
