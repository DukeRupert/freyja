// internal/server/subscriber/order.go
package subscriber

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/rs/zerolog"
)

type OrderEventSubscriber struct {
	orderService interfaces.OrderService
	cartService  interfaces.CartService
	events       interfaces.EventPublisher
	logger       zerolog.Logger
}

func NewOrderEventSubscriber(
	orderService interfaces.OrderService,
	cartService interfaces.CartService,
	events interfaces.EventPublisher,
	logger zerolog.Logger,
) *OrderEventSubscriber {
	return &OrderEventSubscriber{
		orderService: orderService,
		cartService:  cartService,
		events:       events,
		logger:       logger.With().Str("component", "OrderEventSubscriber").Logger(),
	}
}

func (s *OrderEventSubscriber) Start(ctx context.Context) error {
	// Subscribe to checkout completion events
	if err := s.events.Subscribe(ctx, interfaces.EventCheckoutSessionCompleted, s.handleCheckoutCompleted); err != nil {
		return fmt.Errorf("failed to subscribe to checkout.session_completed events: %w", err)
	}

	s.logger.Info().Msg("[OK] Order event subscriber started")
	return nil
}

func (s *OrderEventSubscriber) handleCheckoutCompleted(ctx context.Context, event interfaces.Event) error {
	s.logger.Info().
		Str("event_id", event.ID).
		Str("session_id", fmt.Sprintf("%v", event.Data["stripe_session_id"])).
		Msg("Processing checkout completion for order creation")

	// Extract data from event
	sessionID, ok := event.Data["stripe_session_id"].(string)
	if !ok {
		return fmt.Errorf("missing stripe_session_id in event data")
	}

	customerIDFloat, ok := event.Data["customer_id"].(float64)
	if !ok {
		s.logger.Warn().Msg("No customer_id in checkout event - skipping order creation")
		return nil
	}
	customerID := int32(customerIDFloat)

	amountTotal, _ := event.Data["amount_total"].(float64)
	paymentIntentID, _ := event.Data["payment_intent_id"].(string)

	// Get customer's cart
	cart, err := s.cartService.GetCustomerCart(ctx, customerID)
	if err != nil {
		return fmt.Errorf("failed to get cart for customer %d: %w", customerID, err)
	}

	if len(cart.Items) == 0 {
		s.logger.Warn().
			Int32("customer_id", customerID).
			Str("session_id", sessionID).
			Msg("No items in cart - skipping order creation")
		return nil
	}

	// Create order from cart
	orderRequest := interfaces.CreateOrderRequest{
		CustomerID:            customerID,
		Status:                "pending", // You can adjust this based on your workflow
		Total:                 int32(amountTotal),
		StripeSessionID:       &sessionID,
		StripePaymentIntentID: &paymentIntentID,
		Items:                 s.convertCartItemsToOrderItems(cart.Items),
	}

	order, err := s.orderService.CreateOrder(ctx, orderRequest)
	if err != nil {
		return fmt.Errorf("failed to create order from checkout: %w", err)
	}

	s.logger.Info().
		Int32("order_id", order.ID).
		Int32("customer_id", customerID).
		Str("session_id", sessionID).
		Msg("✅ Successfully created order from checkout")

	// Clear cart after successful order creation
	if err := s.cartService.Clear(ctx, cart.ID); err != nil {
		s.logger.Warn().
			Err(err).
			Int32("customer_id", customerID).
			Msg("Failed to clear cart after order creation")
		// Don't fail the event processing for this
	}

	return nil
}

func (s *OrderEventSubscriber) convertCartItemsToOrderItems(cartItems []interfaces.CartItemWithVariant) []interfaces.CreateOrderItemRequest {
	orderItems := make([]interfaces.CreateOrderItemRequest, len(cartItems))
	for i, item := range cartItems {
		orderItems[i] = interfaces.CreateOrderItemRequest{
			ProductVariantID:     item.ProductVariantID,
			Name:                 item.ProductName,
			VariantName:          item.VariantName,
			Quantity:             item.Quantity,
			Price:                item.Price,
			PurchaseType:         item.PurchaseType,
			SubscriptionInterval: item.SubscriptionInterval,
			StripePriceID:        item.StripePriceID,
		}
	}
	return orderItems
}