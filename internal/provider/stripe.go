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
	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/webhook"
)

type StripeProvider struct {
	apiKey string
	signing_secret string
}

func NewStripeProvider(apiKey string, signing_secret string) (*StripeProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Stripe API key is required")
	}

		if signing_secret == "" {
		return nil, fmt.Errorf("Stripe event signing secret is required")
	}

	provider := &StripeProvider{
		apiKey: apiKey,
		signing_secret: signing_secret,
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
		// *** FIX: Use existing Stripe Price ID instead of creating new PriceData ***
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
						Name: stripe.String(strconv.Itoa(int(item.ProductID))),
					},
					UnitAmount: stripe.Int64(int64(item.Price)),
				},
				Quantity: stripe.Int64(int64(item.Quantity)),
			})
		}
	}

	// *** FIX: Set mode based on whether we have subscription items ***
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
	case "checkout.session.completed":
		return s.handleCheckoutSessionCompleted(ctx, event.Data)
	case "payment_intent.succeeded":
		return s.handlePaymentIntentSucceeded(ctx, event.Data, orderService)
	case "payment_intent.payment_failed":
		return s.handlePaymentIntentFailed(ctx, event.Data)
	case "customer.created":
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
	
	// Extract customer info from metadata if present
	var customerID *int32
	if metadata, ok := eventData["metadata"].(map[string]interface{}); ok {
		if customerIDStr, exists := metadata["customer_id"].(string); exists {
			if id, err := strconv.ParseInt(customerIDStr, 10, 32); err == nil {
				customerID32 := int32(id)
				customerID = &customerID32
			}
		}
	}

	// You can publish events here or return data for the handler to process
	log.Printf("Customer ID from session: %v", customerID)
	return nil
}

func (s *StripeProvider) handlePaymentIntentSucceeded(ctx context.Context, eventData map[string]interface{}, orderService interfaces.OrderService) error {
	paymentIntentID, ok := eventData["id"].(string)
	if !ok {
		return fmt.Errorf("invalid payment intent ID in payment_intent.succeeded event")
	}

	amount, ok := eventData["amount"].(float64)
	if !ok {
		return fmt.Errorf("invalid amount in payment intent")
	}

	// Extract Stripe customer ID from the payment intent
	stripeCustomerID, ok := eventData["customer"].(string)
	if !ok {
		log.Printf("⚠️ No customer ID found in payment intent %s - skipping order creation", paymentIntentID)
		return nil
	}

	// Extract charge ID for linking to Stripe order
	var chargeID string
	if latestCharge, ok := eventData["latest_charge"].(string); ok {
		chargeID = latestCharge
	}

	log.Printf("💰 Payment succeeded: %s (Amount: %.0f cents, Customer: %s, Charge: %s)", 
		paymentIntentID, amount, stripeCustomerID, chargeID)

	// Find internal customer ID from Stripe customer ID
	customerID, err := s.findInternalCustomerID(ctx, stripeCustomerID)
	if err != nil {
		log.Printf("❌ Failed to find internal customer for Stripe customer %s: %v", stripeCustomerID, err)
		return fmt.Errorf("failed to find internal customer: %w", err)
	}

	if customerID == nil {
		log.Printf("⚠️ No internal customer found for Stripe customer %s - skipping order creation", stripeCustomerID)
		return nil
	}

	// Create order with enhanced payment information
	order, err := orderService.CreateOrderFromPayment(ctx, *customerID, paymentIntentID, int32(amount))
	if err != nil {
		log.Printf("❌ Failed to create order from payment: %v", err)
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Update order with Stripe charge ID for reference
	if chargeID != "" {
		if err := orderService.UpdateStripeChargeID(ctx, order.ID, chargeID); err != nil {
			log.Printf("⚠️ Failed to update order with charge ID: %v", err)
		}
	}

	log.Printf("📦 Order created successfully: ID %d (Payment Intent: %s, Charge: %s)", 
		order.ID, paymentIntentID, chargeID)

	return nil
}

// Helper method to find internal customer ID from Stripe customer ID
func (s *StripeProvider) findInternalCustomerID(ctx context.Context, stripeCustomerID string) (*int32, error) {
	// Get the Stripe customer to check metadata
	stripeCustomer, err := customer.Get(stripeCustomerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe customer: %w", err)
	}

	// Check if we have internal customer ID in metadata
	if internalIDStr, exists := stripeCustomer.Metadata["internal_customer_id"]; exists {
		if id, err := strconv.ParseInt(internalIDStr, 10, 32); err == nil {
			customerID := int32(id)
			return &customerID, nil
		}
	}

	// If no metadata, try to find by email (fallback)
	// This would require a customer service method to find by email
	// For now, return nil to indicate no customer found
	log.Printf("No internal_customer_id found in Stripe customer %s metadata", stripeCustomerID)
	return nil, nil
}

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