// internal/service/checkout.go
package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/dukerupert/freyja/internal/interfaces"
)

type CheckoutService struct {
	cartService     *CartService
	orderService    *OrderService
	paymentProvider interfaces.PaymentProvider
	eventPublisher  interfaces.EventPublisher
}

func NewCheckoutService(
	cartService *CartService,
	orderService *OrderService,
	paymentProvider interfaces.PaymentProvider,
	eventPublisher interfaces.EventPublisher,
) *CheckoutService {
	return &CheckoutService{
		cartService:     cartService,
		orderService:    orderService,
		paymentProvider: paymentProvider,
		eventPublisher:  eventPublisher,
	}
}

func (s *CheckoutService) CreateCheckoutSession(ctx context.Context, customerID *int32, sessionID *string, successURL, cancelURL string) (*interfaces.CheckoutSessionResponse, error) {
	// Get or create cart
	cart, err := s.cartService.GetOrCreateCart(ctx, customerID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// Validate cart for checkout
	validatedCart, err := s.cartService.ValidateCartForCheckout(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("cart validation failed: %w", err)
	}

	if len(validatedCart.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// *** FIX: Convert cart items with complete information including Stripe Price IDs ***
	var checkoutItems []interfaces.CartItem
	for _, item := range validatedCart.Items {
		checkoutItems = append(checkoutItems, interfaces.CartItem{
			ID:                   item.ID,
			ProductID:            item.ProductID,
			Quantity:             item.Quantity,
			Price:                item.Price,
			PurchaseType:         item.PurchaseType,         // *** Include purchase type ***
			SubscriptionInterval: item.SubscriptionInterval, // *** Include subscription interval ***
			StripePriceID:        item.StripePriceID,        // *** Include Stripe Price ID ***
		})
	}

	// Prepare checkout session request
	req := interfaces.CheckoutSessionRequest{
		CustomerID: customerID,
		Items:      checkoutItems,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
	}

	// Create Stripe checkout session
	session, err := s.paymentProvider.CreateCheckoutSession(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment session: %w", err)
	}

	// Publish checkout initiated event
	eventData := map[string]interface{}{
		"cart_id":           cart.ID,
		"stripe_session_id": session.SessionID,
		"total_amount":      validatedCart.Total,
		"item_count":        validatedCart.ItemCount,
	}

	if customerID != nil {
		eventData["customer_id"] = *customerID
	}

	if sessionID != nil {
		eventData["session_id"] = *sessionID
	}

	event := interfaces.Event{
		ID:          generateEventID(),
		Type:        "checkout.session_created",
		AggregateID: fmt.Sprintf("cart:%d", cart.ID),
		Data:        eventData,
		Timestamp:   time.Now(),
	}

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		// Log error but don't fail the checkout
		fmt.Printf("Failed to publish checkout event: %v\n", err)
	}

	return session, nil
}

func (s *CheckoutService) HandleWebhookEvent(ctx context.Context, eventType string, eventData map[string]interface{}) error {
	switch eventType {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, eventData)
	case "payment_intent.succeeded":
		return s.handlePaymentSucceeded(ctx, eventData)
	case "payment_intent.payment_failed":
		return s.handlePaymentFailed(ctx, eventData)
	default:
		// Log unhandled event types but don't error
		fmt.Printf("Unhandled webhook event type: %s\n", eventType)
		return nil
	}
}

func (s *CheckoutService) handleCheckoutCompleted(ctx context.Context, eventData map[string]interface{}) error {
	sessionID, ok := eventData["id"].(string)
	if !ok {
		return fmt.Errorf("invalid session ID in event")
	}

	// Extract customer info from metadata
	metadata, ok := eventData["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing metadata in checkout session")
	}

	var customerID *int32
	if customerIDStr, exists := metadata["customer_id"].(string); exists {
		if id, err := strconv.Atoi(customerIDStr); err == nil {
			customerID32 := int32(id)
			customerID = &customerID32
		}
	}

	// Publish payment processing event
	event := interfaces.Event{
		ID:          generateEventID(),
		Type:        "payment.processing",
		AggregateID: fmt.Sprintf("session:%s", sessionID),
		Data: map[string]interface{}{
			"stripe_session_id": sessionID,
			"customer_id":       customerID,
		},
		Timestamp: time.Now(),
	}

	return s.eventPublisher.PublishEvent(ctx, event)
}

func (s *CheckoutService) handlePaymentSucceeded(ctx context.Context, eventData map[string]interface{}) error {
	paymentIntentID, ok := eventData["id"].(string)
	if !ok {
		return fmt.Errorf("invalid payment intent ID")
	}

	amount, ok := eventData["amount"].(float64)
	if !ok {
		return fmt.Errorf("invalid amount in payment intent")
	}

	// Extract customer info
	metadata, _ := eventData["metadata"].(map[string]interface{})
	var customerID *int32
	if metadata != nil {
		if customerIDStr, exists := metadata["customer_id"].(string); exists {
			if id, err := strconv.Atoi(customerIDStr); err == nil {
				customerID32 := int32(id)
				customerID = &customerID32
			}
		}
	}

	// Create order from successful payment
	// This is where you'd convert the cart to an order
	if customerID != nil {
		order, err := s.orderService.CreateOrderFromPayment(ctx, *customerID, paymentIntentID, int32(amount))
		if err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		// Clear the cart after successful order creation
		if err := s.cartService.ClearCart(ctx, order.ID); err != nil {
			// Log error but don't fail - order was created successfully
			fmt.Printf("Failed to clear cart after order creation: %v\n", err)
		}
	}

	// Publish payment confirmed event
	event := interfaces.Event{
		ID:          generateEventID(),
		Type:        "payment.confirmed",
		AggregateID: fmt.Sprintf("payment:%s", paymentIntentID),
		Data: map[string]interface{}{
			"payment_intent_id": paymentIntentID,
			"amount":            int(amount),
			"customer_id":       customerID,
		},
		Timestamp: time.Now(),
	}

	return s.eventPublisher.PublishEvent(ctx, event)
}

func (s *CheckoutService) handlePaymentFailed(ctx context.Context, eventData map[string]interface{}) error {
	paymentIntentID, ok := eventData["id"].(string)
	if !ok {
		return fmt.Errorf("invalid payment intent ID")
	}

	// Extract error information
	lastPaymentError, _ := eventData["last_payment_error"].(map[string]interface{})
	errorMessage := "Payment failed"
	if lastPaymentError != nil {
		if msg, ok := lastPaymentError["message"].(string); ok {
			errorMessage = msg
		}
	}

	// Publish payment failed event
	event := interfaces.Event{
		ID:          generateEventID(),
		Type:        "payment.failed",
		AggregateID: fmt.Sprintf("payment:%s", paymentIntentID),
		Data: map[string]interface{}{
			"payment_intent_id": paymentIntentID,
			"error_message":     errorMessage,
		},
		Timestamp: time.Now(),
	}

	return s.eventPublisher.PublishEvent(ctx, event)
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}
