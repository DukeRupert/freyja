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
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	stripePrice "github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/webhook"
)

type StripeProvider struct {
	apiKey         string
	signing_secret string
	events         interfaces.EventPublisher
	logger         zerolog.Logger
}

func NewStripeProvider(apiKey string, signing_secret string, events interfaces.EventPublisher, logger zerolog.Logger) (*StripeProvider, error) {
	// Create contextual logger for this provider
	providerLogger := logger.With().
		Str("component", "StripeProvider").
		Logger()

	if apiKey == "" {
		providerLogger.Error().Msg("Stripe API key is required")
		return nil, fmt.Errorf("stripe API key is required")
	}

	if signing_secret == "" {
		providerLogger.Error().Msg("Stripe webhook signing secret is required")
		return nil, fmt.Errorf("stripe event signing secret is required")
	}

	provider := &StripeProvider{
		apiKey:         apiKey,
		signing_secret: signing_secret,
		events:         events,
		logger:         providerLogger, // Use the contextual logger
	}

	// Set the global Stripe API key
	stripe.Key = apiKey
	providerLogger.Debug().Msg("Stripe API key configured")

	// Perform health check
	providerLogger.Info().Msg("Performing Stripe health check")
	if err := provider.HealthCheck(context.Background()); err != nil {
		providerLogger.Error().Err(err).Msg("Stripe health check failed")
		return nil, fmt.Errorf("stripe health check failed: %w", err)
	}

	providerLogger.Info().Msg("Stripe provider initialized successfully")
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
	logger := s.logger.With().
		Str("function", "HandleWebhookEvent").
		Str("event_type", event.Type).
		Str("event_id", event.ID).
		Time("event_created_at", event.CreatedAt).
		Logger()

	logger.Info().Msg("Processing Stripe webhook event")

	switch event.Type {
	case interfaces.WebhookCheckoutSessionCompleted:
		logger.Info().Msg("Routing to checkout session completed handler")
		return s.handleCheckoutSessionCompleted(ctx, event.Data)
		
	case interfaces.WebhookPaymentIntentFailed:
		logger.Info().Msg("Routing to payment intent failed handler")
		return s.handlePaymentIntentFailed(ctx, event.Data)
		
	case interfaces.WebhookCustomerCreated:
		logger.Info().Msg("Routing to customer created handler")
		return s.handleCustomerCreated(ctx, event.Data, customerService)
		
	case interfaces.WebhookInvoicePaymentSucceeded:
		logger.Info().Msg("Routing to invoice payment succeeded handler")
		return s.handleInvoicePaymentSucceeded(ctx, event.Data, orderService, customerService)
		
	default:
		// Log unhandled events but don't error - allows for easy extension
		logger.Info().Msg("Received unhandled Stripe webhook event type - skipping processing")
		return nil
	}
}

// Internal webhook handlers

func (s *StripeProvider) handleCheckoutSessionCompleted(ctx context.Context, eventData map[string]interface{}) error {
	logger := s.logger.With().
		Str("function", "handleCheckoutSessionCompleted").
		Str("event_type", "checkout.session.completed").
		Logger()

	logger.Info().Msg("Processing checkout session completed webhook")

	sessionID, ok := eventData["id"].(string)
	if !ok {
		logger.Error().Msg("Invalid session ID in checkout session completed event")
		return fmt.Errorf("invalid session ID in checkout.session.completed event")
	}

	logger = logger.With().Str("session_id", sessionID).Logger()

	logger.Info().Msg("Checkout session completed")

	// Extract customer info from metadata
	var customerID *int32
	if metadata, ok := eventData["metadata"].(map[string]interface{}); ok {
		if customerIDStr, exists := metadata["customer_id"].(string); exists {
			if id, err := strconv.ParseInt(customerIDStr, 10, 32); err == nil {
				customerID32 := int32(id)
				customerID = &customerID32
				logger = logger.With().Int32("customer_id", *customerID).Logger()
			} else {
				logger.Warn().
					Str("customer_id_str", customerIDStr).
					Err(err).
					Msg("Failed to parse customer ID from metadata")
			}
		} else {
			logger.Debug().Msg("No customer ID found in session metadata")
		}
	} else {
		logger.Debug().Msg("No metadata found in checkout session")
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

	// Extract additional session details
	paymentStatus, _ := eventData["payment_status"].(string)
	mode, _ := eventData["mode"].(string)
	stripeCustomerID, _ := eventData["customer"].(string)

	logger = logger.With().
		Int32("amount_total", amountTotal).
		Str("payment_status", paymentStatus).
		Str("mode", mode).
		Str("stripe_customer_id", stripeCustomerID).
		Logger()

	if paymentIntentID != nil {
		logger = logger.With().Str("payment_intent_id", *paymentIntentID).Logger()
	}

	logger.Info().Msg("Extracted checkout session details")

	// Publish checkout completed event to event bus
	if s.events != nil {
		event := interfaces.Event{
			ID:          interfaces.GenerateEventID(),
			Type:        interfaces.EventCheckoutSessionCompleted,
			AggregateID: fmt.Sprintf("checkout:%s", sessionID),
			Data: map[string]interface{}{
				"stripe_session_id":   sessionID,
				"customer_id":         customerID,
				"amount_total":        amountTotal,
				"payment_intent_id":   paymentIntentID,
				"payment_status":      paymentStatus,
				"mode":               mode,
				"stripe_customer_id": stripeCustomerID,
				"completed_at":       time.Now().Unix(),
			},
			Timestamp: time.Now(),
			Version:   1,
		}

		logger.Info().
			Str("event_id", event.ID).
			Str("aggregate_id", event.AggregateID).
			Msg("Publishing checkout session completed event")

		if err := s.events.PublishEvent(ctx, event); err != nil {
			logger.Error().
				Err(err).
				Str("event_id", event.ID).
				Msg("Failed to publish checkout session completed event")
			return err
		}

		logger.Info().
			Str("event_id", event.ID).
			Msg("Successfully published checkout session completed event")
	} else {
		logger.Warn().Msg("Event publisher not available, skipping event publication")
	}

	return nil
}

func (s *StripeProvider) handlePaymentIntentFailed(ctx context.Context, eventData map[string]interface{}) error {
	logger := s.logger.With().
		Str("function", "handlePaymentIntentFailed").
		Str("event_type", "payment_intent.payment_failed").
		Logger()

	logger.Info().Msg("Processing payment intent failed webhook")

	paymentIntentID, ok := eventData["id"].(string)
	if !ok {
		logger.Error().Msg("Invalid payment intent ID in payment intent failed event")
		return fmt.Errorf("invalid payment intent ID in payment_intent.payment_failed event")
	}

	logger = logger.With().Str("payment_intent_id", paymentIntentID).Logger()

	// Extract error information
	var errorMessage, errorCode, errorType string
	if lastPaymentError, ok := eventData["last_payment_error"].(map[string]interface{}); ok {
		if msg, exists := lastPaymentError["message"].(string); exists {
			errorMessage = msg
		}
		if code, exists := lastPaymentError["code"].(string); exists {
			errorCode = code
		}
		if errType, exists := lastPaymentError["type"].(string); exists {
			errorType = errType
		}
	}

	// Set default error message if none provided
	if errorMessage == "" {
		errorMessage = "Payment failed"
	}

	// Extract additional payment context
	amount, _ := eventData["amount"].(float64)
	currency, _ := eventData["currency"].(string)
	stripeCustomerID, _ := eventData["customer"].(string)
	status, _ := eventData["status"].(string)

	logger.Error().
		Str("error_message", errorMessage).
		Str("error_code", errorCode).
		Str("error_type", errorType).
		Float64("amount", amount).
		Str("currency", currency).
		Str("stripe_customer_id", stripeCustomerID).
		Str("status", status).
		Msg("Payment intent failed")

	// Log detailed error information for debugging
	logger.Debug().
		Interface("last_payment_error", eventData["last_payment_error"]).
		Interface("charges", eventData["charges"]).
		Msg("Detailed payment failure information")

	// TODO: Implement business logic for payment failures
	// - Update order status to failed
	// - Notify customer about payment failure
	// - Trigger retry logic or dunning management
	// - Send failure notifications to admin

	logger.Info().Msg("Payment failure processed - business logic implementation needed")

	return nil
}

func (s *StripeProvider) handleCustomerCreated(ctx context.Context, eventData map[string]interface{}, customerService interfaces.CustomerService) error {
	logger := s.logger.With().
		Str("function", "handleCustomerCreated").
		Str("event_type", "customer.created").
		Logger()

	logger.Info().Msg("Processing customer created webhook")

	stripeCustomerID, ok := eventData["id"].(string)
	if !ok {
		logger.Error().Msg("Invalid customer ID in customer created event")
		return fmt.Errorf("invalid customer ID in customer.created event")
	}

	// Extract customer email - required for creating internal customer
	email, ok := eventData["email"].(string)
	if !ok || email == "" {
		logger.Error().Str("stripe_customer_id", stripeCustomerID).Msg("No email found in customer created event")
		return fmt.Errorf("no email found in customer.created event for customer %s", stripeCustomerID)
	}

	logger = logger.With().
		Str("stripe_customer_id", stripeCustomerID).
		Str("email", email).
		Logger()

	// Extract additional customer data
	name, _ := eventData["name"].(string)
	created, _ := eventData["created"].(float64)

	logger.Info().
		Str("name", name).
		Time("created_at", time.Unix(int64(created), 0)).
		Msg("Stripe customer created - syncing to internal database")

	// Create or update customer in internal database
	customer, err := customerService.CreateCustomerFromStripe(ctx, stripeCustomerID, email)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create internal customer from Stripe customer")
		return fmt.Errorf("failed to create internal customer from Stripe customer %s: %w", stripeCustomerID, err)
	}

	logger.Info().
		Int32("internal_customer_id", customer.ID).
		Msg("Successfully synced Stripe customer to internal database")

	return nil
}

// handleInvoicePaymentSucceeded processes successful recurring billing
func (s *StripeProvider) handleInvoicePaymentSucceeded(ctx context.Context, eventData map[string]interface{}, orderService interfaces.OrderService, customerService interfaces.CustomerService) error {
	logger := s.logger.With().
		Str("function", "handleInvoicePaymentSucceeded").
		Str("event_type", "invoice.payment_succeeded").
		Logger()

	logger.Info().Msg("Processing invoice payment succeeded webhook")

	// Extract invoice ID
	invoiceID, ok := eventData["id"].(string)
	if !ok {
		logger.Error().Msg("No invoice ID in invoice payment succeeded event")
		return fmt.Errorf("no invoice ID in invoice.payment_succeeded event")
	}

	// Extract customer ID from the invoice
	stripeCustomerID, ok := eventData["customer"].(string)
	if !ok {
		logger.Error().Str("invoice_id", invoiceID).Msg("No customer ID in invoice payment succeeded event")
		return fmt.Errorf("no customer ID in invoice.payment_succeeded event")
	}

	// Extract billing reason to distinguish subscription types
	billingReason, _ := eventData["billing_reason"].(string)

	// Add context to logger
	logger = logger.With().
		Str("invoice_id", invoiceID).
		Str("stripe_customer_id", stripeCustomerID).
		Str("billing_reason", billingReason).
		Logger()

	logger.Info().Msg("Invoice details extracted")

	// Only process subscription renewals, not initial subscriptions
	if billingReason != "subscription_cycle" {
		logger.Info().Msg("Skipping invoice - billing reason is not a subscription renewal")
		return nil
	}

	// Extract subscription ID to get line items (make this optional)
	subscriptionID, ok := eventData["subscription"].(string)
	if !ok {
		logger.Warn().Msg("No subscription ID found in invoice, will process without subscription context")
		subscriptionID = ""
	}

	// Extract amount paid
	amountPaidFloat, ok := eventData["amount_paid"].(float64)
	if !ok {
		logger.Error().Msg("No amount_paid in invoice payment succeeded event")
		return fmt.Errorf("no amount_paid in invoice.payment_succeeded event")
	}
	amountPaid := int32(amountPaidFloat)

	// Extract billing period
	periodStart, _ := eventData["period_start"].(float64)
	periodEnd, _ := eventData["period_end"].(float64)

	logger.Info().
		Str("subscription_id", subscriptionID).
		Int32("amount_paid", amountPaid).
		Time("period_start", time.Unix(int64(periodStart), 0)).
		Time("period_end", time.Unix(int64(periodEnd), 0)).
		Msg("Processing subscription renewal")

	// Find internal customer by Stripe customer ID
	internalCustomer, err := customerService.GetCustomerByStripeID(ctx, stripeCustomerID)
	if err != nil || internalCustomer == nil {
		logger.Error().Err(err).Msg("Could not find internal customer for Stripe customer")
		return fmt.Errorf("customer not found for Stripe customer %s: %w", stripeCustomerID, err)
	}

	logger.Info().Int32("internal_customer_id", internalCustomer.ID).Msg("Found internal customer")

	// Get invoice line items to create order items
	lines, ok := eventData["lines"].(map[string]interface{})
	if !ok {
		logger.Error().Msg("No lines object in invoice payment succeeded event")
		return fmt.Errorf("no lines in invoice.payment_succeeded event")
	}

	lineData, ok := lines["data"].([]interface{})
	if !ok {
		logger.Error().Msg("No line data array in invoice payment succeeded event")
		return fmt.Errorf("no line data in invoice.payment_succeeded event")
	}

	logger.Info().Int("line_item_count", len(lineData)).Msg("Found line items in invoice")

	// Create order items from invoice line items
	var orderItems []interfaces.CreateOrderItemRequest
	for i, line := range lineData {
		lineLogger := logger.With().Int("line_item_index", i).Logger()

		lineItem, ok := line.(map[string]interface{})
		if !ok {
			lineLogger.Warn().Msg("Line item is not a map, skipping")
			continue
		}

		lineLogger.Debug().Interface("line_item_data", lineItem).Msg("Processing line item")

		// Extract line item details
		quantity, _ := lineItem["quantity"].(float64)
		if quantity <= 0 {
			lineLogger.Warn().Float64("quantity", quantity).Msg("Line item has invalid quantity, skipping")
			continue
		}

		// Get price information - check both possible locations
		var price map[string]interface{}
		var stripePriceID string
		var unitAmountFloat float64

		// First try the direct price field
		if priceObj, ok := lineItem["price"].(map[string]interface{}); ok {
			price = priceObj
			stripePriceID, _ = price["id"].(string)
			unitAmountFloat, _ = price["unit_amount"].(float64)
		} else if pricing, ok := lineItem["pricing"].(map[string]interface{}); ok {
			// Try the nested pricing structure
			if priceDetails, ok := pricing["price_details"].(map[string]interface{}); ok {
				if priceIDStr, ok := priceDetails["price"].(string); ok {
					stripePriceID = priceIDStr
					// Get unit amount from the pricing object
					if unitAmountDecimal, ok := pricing["unit_amount_decimal"].(string); ok {
						if parsed, err := strconv.ParseFloat(unitAmountDecimal, 64); err == nil {
							unitAmountFloat = parsed
						}
					}
					// Create a minimal price object for metadata fetching
					price = map[string]interface{}{
						"id":          stripePriceID,
						"unit_amount": unitAmountFloat,
					}
				}
			}
		}

		if price == nil {
			lineLogger.Warn().Msg("Line item has no price object in either location, skipping")
			continue
		}

		lineLogger = lineLogger.With().
			Str("stripe_price_id", stripePriceID).
			Float64("unit_amount", unitAmountFloat).
			Logger()

		lineLogger.Debug().Msg("Extracted price information")

		if stripePriceID == "" {
			lineLogger.Warn().Msg("Line item has no Stripe price ID, skipping")
			continue
		}

		if unitAmountFloat <= 0 {
			lineLogger.Warn().Msg("Line item has invalid unit amount, skipping")
			continue
		}

		// Get metadata to find our internal variant ID and extract product details
		var metadata map[string]interface{}

		if priceMetadata, ok := price["metadata"].(map[string]interface{}); ok {
			// Metadata is available directly in the price object
			metadata = priceMetadata
			lineLogger.Debug().Interface("metadata", metadata).Msg("Using metadata from price object")
		} else if stripePriceID != "" {
			// Need to fetch the price from Stripe to get metadata
			lineLogger.Info().Msg("Fetching price from Stripe to get metadata")

			// Use Stripe SDK to fetch the price with metadata
			stripePrice, err := stripePrice.Get(stripePriceID, nil)
			if err != nil {
				lineLogger.Error().Err(err).Msg("Failed to fetch price from Stripe")
				continue
			}

			// Convert Stripe metadata to our format
			metadata = make(map[string]interface{})
			for key, value := range stripePrice.Metadata {
				metadata[key] = value
			}

			// Also update the unit amount from the fetched price if it wasn't available
			if unitAmountFloat == 0 && stripePrice.UnitAmount > 0 {
				unitAmountFloat = float64(stripePrice.UnitAmount)
				lineLogger = lineLogger.With().Float64("updated_unit_amount", unitAmountFloat).Logger()
			}

			lineLogger.Debug().Interface("metadata", metadata).Msg("Fetched metadata from Stripe API")
		} else {
			lineLogger.Warn().Msg("No price ID available for line item, skipping")
			continue
		}

		variantIDStr, ok := metadata["internal_variant_id"].(string)
		if !ok {
			lineLogger.Warn().Msg("No internal variant ID in metadata, skipping line item")
			continue
		}

		variantID, err := strconv.ParseInt(variantIDStr, 10, 32)
		if err != nil {
			lineLogger.Error().Err(err).Str("variant_id_str", variantIDStr).Msg("Invalid variant ID, skipping line item")
			continue
		}

		// Get subscription interval from metadata and format for database constraint
		subscriptionInterval := pgtype.Text{}
		if intervalStr, ok := metadata["subscription_days"].(string); ok && intervalStr != "" {
			// Convert "14" to "14_day" format as expected by database constraint
			formattedInterval := intervalStr + "_day"
			subscriptionInterval = pgtype.Text{String: formattedInterval, Valid: true}
		}

		// Extract product and variant names from metadata
		variantName, _ := metadata["variant_name"].(string)
		if variantName == "" {
			variantName = fmt.Sprintf("Subscription Item (Variant %d)", variantID)
		}

		// Use variant_name as product name for now
		productName, _ := metadata["variant_name"].(string)
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

		lineLogger.Info().
			Int64("variant_id", variantID).
			Int32("quantity", int32(quantity)).
			Str("subscription_interval", subscriptionInterval.String).
			Str("variant_name", variantName).
			Msg("Added recurring order item")
	}

	if len(orderItems) == 0 {
		logger.Warn().Msg("No valid order items found in invoice")
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
		CustomerID:         internalCustomer.ID,
		Status:             database.OrderStatusConfirmed, // Subscription renewals are auto-confirmed
		Total:              amountPaid,
		Items:              orderItems,
		Source:             &source,
		StripeInvoiceID:    &invoiceID,
		BillingPeriodStart: periodStartTime,
		BillingPeriodEnd:   periodEndTime,
		Metadata: map[string]string{
			"billing_reason": billingReason,
		},
	}

	// Only add subscription ID if we have one
	if subscriptionID != "" {
		createOrderReq.StripeSubscriptionID = &subscriptionID
	}

	logger.Info().
		Int32("customer_id", internalCustomer.ID).
		Int("order_item_count", len(orderItems)).
		Int32("total_amount", amountPaid).
		Msg("Creating order for subscription renewal")

	order, err := orderService.CreateOrder(ctx, createOrderReq)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create order for subscription renewal")
		return fmt.Errorf("failed to create order for subscription renewal: %w", err)
	}

	logger.Info().
		Int32("order_id", order.ID).
		Int32("customer_id", internalCustomer.ID).
		Str("source", "subscription_renewal").
		Msg("Successfully created order for subscription renewal")

	return nil
}

// RefundPayment creates a refund for a payment ** Untested **
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
