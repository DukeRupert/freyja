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
	"github.com/jackc/pgx/v5/pgtype"
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
	case interfaces.WebhookInvoicePaymentSucceeded:
		return s.handleInvoicePaymentSucceeded(ctx, event.Data, orderService, customerService)
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

// handleInvoicePaymentSucceeded processes successful recurring billing
func (s *StripeProvider) handleInvoicePaymentSucceeded(ctx context.Context, eventData map[string]interface{}, orderService interfaces.OrderService, customerService interfaces.CustomerService) error {
	log.Printf("Processing invoice.payment_succeeded webhook")

	// Extract invoice ID
	invoiceID, ok := eventData["id"].(string)
	if !ok {
		return fmt.Errorf("no invoice ID in invoice.payment_succeeded event")
	}

	// Extract customer ID from the invoice
	stripeCustomerID, ok := eventData["customer"].(string)
	if !ok {
		return fmt.Errorf("no customer ID in invoice.payment_succeeded event")
	}

	// Extract billing reason to distinguish subscription types
	billingReason, _ := eventData["billing_reason"].(string)

	// Log billing context
	log.Printf("Invoice %s: billing_reason=%s, customer=%s", invoiceID, billingReason, stripeCustomerID)

	// Only process subscription renewals, not initial subscriptions
	if billingReason != "subscription_cycle" {
		log.Printf("Skipping invoice %s - billing_reason '%s' is not a subscription renewal", invoiceID, billingReason)
		return nil
	}

	// Extract subscription ID to get line items
	subscriptionID, ok := eventData["subscription"].(string)
	if !ok {
		return fmt.Errorf("no subscription ID in invoice.payment_succeeded event")
	}

	// Extract amount paid
	amountPaidFloat, ok := eventData["amount_paid"].(float64)
	if !ok {
		return fmt.Errorf("no amount_paid in invoice.payment_succeeded event")
	}
	amountPaid := int32(amountPaidFloat)

	// Extract billing period
	periodStart, _ := eventData["period_start"].(float64)
	periodEnd, _ := eventData["period_end"].(float64)

	log.Printf("Processing subscription renewal: subscription=%s, amount=%d, period=%v to %v",
		subscriptionID, amountPaid, time.Unix(int64(periodStart), 0), time.Unix(int64(periodEnd), 0))

	// Find internal customer by Stripe customer ID
	internalCustomer, err := customerService.GetCustomerByStripeID(ctx, stripeCustomerID)
	if err != nil || internalCustomer == nil {
		log.Printf("Could not find internal customer for Stripe customer %s: %v", stripeCustomerID, err)
		return fmt.Errorf("customer not found for Stripe customer %s: %w", stripeCustomerID, err)
	}

	log.Printf("Found internal customer %d for Stripe customer %s", internalCustomer.ID, stripeCustomerID)

	// Get invoice line items to create order items
	lines, ok := eventData["lines"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no lines in invoice.payment_succeeded event")
	}

	lineData, ok := lines["data"].([]interface{})
	if !ok {
		return fmt.Errorf("no line data in invoice.payment_succeeded event")
	}

	// Create order items from invoice line items
	var orderItems []interfaces.CreateOrderItemRequest
	for _, line := range lineData {
		lineItem, ok := line.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract line item details
		quantity, _ := lineItem["quantity"].(float64)

		// Get price information
		price, ok := lineItem["price"].(map[string]interface{})
		if !ok {
			continue
		}

		stripePriceID, _ := price["id"].(string)
		unitAmountFloat, _ := price["unit_amount"].(float64)

		// Get metadata to find our internal variant ID and extract product details
		metadata, ok := price["metadata"].(map[string]interface{})
		if !ok {
			log.Printf("No metadata found for price %s, skipping line item", stripePriceID)
			continue
		}

		variantIDStr, ok := metadata["internal_variant_id"].(string)
		if !ok {
			log.Printf("No internal_variant_id in metadata for price %s, skipping line item", stripePriceID)
			continue
		}

		variantID, err := strconv.ParseInt(variantIDStr, 10, 32)
		if err != nil {
			log.Printf("Invalid variant ID '%s' for price %s, skipping line item", variantIDStr, stripePriceID)
			continue
		}

		// Get subscription interval from metadata
		subscriptionInterval := pgtype.Text{}
		if intervalStr, ok := metadata["subscription_days"].(string); ok && intervalStr != "" {
			subscriptionInterval = pgtype.Text{String: intervalStr, Valid: true}
		}

		// Extract product and variant names from metadata
		variantName, _ := metadata["variant_name"].(string)
		if variantName == "" {
			variantName = fmt.Sprintf("Subscription Item (Variant %d)", variantID)
		}

		// You might want to get the product name from metadata too
		productName, _ := metadata["product_name"].(string)
		if productName == "" {
			productName = variantName // Fallback to variant name
		}

		orderItems = append(orderItems, interfaces.CreateOrderItemRequest{
			ProductVariantID:     int32(variantID),
			Name:                 productName,
			VariantName:          variantName,
			Quantity:             int32(quantity),
			Price:                int32(unitAmountFloat),
			PurchaseType:         "subscription",
			SubscriptionInterval: subscriptionInterval,
			StripePriceID:        stripePriceID,
		})

		log.Printf("Added recurring order item: variant=%d, quantity=%d, interval=%s",
			variantID, int32(quantity), subscriptionInterval)
	}

	if len(orderItems) == 0 {
		log.Printf("No valid order items found in invoice %s", invoiceID)
		return fmt.Errorf("no valid order items found in invoice %s", invoiceID)
	}

	// Prepare billing period times
	var periodStartTime, periodEndTime *time.Time
	if periodStart > 0 {
		t := time.Unix(int64(periodStart), 0)
		periodStartTime = &t
	}
	if periodEnd > 0 {
		t := time.Unix(int64(periodEnd), 0)
		periodEndTime = &t
	}

	// Create order for the subscription renewal
	source := "subscription_renewal"
	createOrderReq := interfaces.CreateOrderRequest{
		CustomerID:           internalCustomer.ID,
		Status:               database.OrderStatusConfirmed, // Subscription renewals are auto-confirmed
		Total:                amountPaid,
		Items:                orderItems,
		Source:               &source,
		StripeInvoiceID:      &invoiceID,
		StripeSubscriptionID: &subscriptionID,
		BillingPeriodStart:   periodStartTime,
		BillingPeriodEnd:     periodEndTime,
		Metadata: map[string]string{
			"billing_reason": billingReason,
		},
	}

	order, err := orderService.CreateOrder(ctx, createOrderReq)
	if err != nil {
		log.Printf("Failed to create order for subscription renewal: %v", err)
		return fmt.Errorf("failed to create order for subscription renewal: %w", err)
	}

	log.Printf("Successfully created order %d for subscription renewal (invoice: %s, customer: %d)",
		order.ID, invoiceID, internalCustomer.ID)

	return nil
}
