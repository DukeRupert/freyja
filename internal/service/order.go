// internal/service/order.go
package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/jackc/pgx/v5/pgtype"
)

type OrderService struct {
	orderRepo   interfaces.OrderRepository
	cartService interfaces.CartService
	events      interfaces.EventPublisher
}

func NewOrderService(
	orderRepo interfaces.OrderRepository,
	cartService interfaces.CartService,
	events interfaces.EventPublisher,
) interfaces.OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		cartService: cartService,
		events:      events,
	}
}

// =============================================================================
// Order Creation
// =============================================================================

func (s *OrderService) CreateOrderFromCart(ctx context.Context, customerID int32, cartID int32) (*interfaces.Order, error) {
	// Confirm cart has items
	cartItems, err := s.cartService.GetCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get items for cart: %w", err)
	}

	if len(cartItems) == 0 {
		return nil, fmt.Errorf("cannot create order from empty cart")
	}

	// Validate cart for checkout one more time
	validatedCart, err := s.cartService.ValidateCartForCheckout(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart validation failed: %w", err)
	}

	// Create order
	order := &interfaces.Order{
		CustomerID: customerID,
		Status:     database.OrderStatusPending,
		Total:      validatedCart.Total,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Create order items from cart items
	var orderItems []interfaces.OrderItem
	for _, cartItem := range validatedCart.Items {
		orderItems = append(orderItems, interfaces.OrderItem{
			OrderID:   order.ID,
			ProductID: cartItem.ProductID,
			Name:      cartItem.ProductName, // Snapshot the name
			Quantity:  cartItem.Quantity,
			Price:     cartItem.Price,
			CreatedAt: time.Now(),
		})
	}

	if err := s.orderRepo.CreateOrderItems(ctx, order.ID, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Publish order created event
	if err := s.publishOrderEvent(ctx, interfaces.EventOrderCreated, order.ID, map[string]interface{}{
		"customer_id": customerID,
		"total":       order.Total,
		"item_count":  len(orderItems),
		"cart_id":     cartID,
	}); err != nil {
		// Log error but don't fail order creation
		fmt.Printf("Failed to publish order created event: %v\n", err)
	}

	return order, nil
}

func (s *OrderService) CreateOrderFromPayment(ctx context.Context, customerID int32, paymentIntentID string, amount int32) (*interfaces.Order, error) {
	// Get customer's cart
	cart, err := s.cartService.GetOrCreateCart(ctx, &customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer cart: %w", err)
	}

	// Validate cart has items
	if len(cart.Items) == 0 {
		return nil, fmt.Errorf("cannot create order from empty cart")
	}

	// Validate cart total matches payment amount
	if cart.Total != amount {
		log.Printf("⚠️ Cart total (%d) does not match payment amount (%d) - using payment amount", cart.Total, amount)
	}

	// Create order with payment information
	order := &interfaces.Order{
		CustomerID: customerID,
		Status:     database.OrderStatusConfirmed, // Confirmed since payment succeeded
		Total:      amount,                        // Use payment amount as authoritative
		StripePaymentIntentID: pgtype.Text{
			String: paymentIntentID,
			Valid:  true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Create order items from cart items
	var orderItems []interfaces.OrderItem
	for _, cartItem := range cart.Items {
		orderItems = append(orderItems, interfaces.OrderItem{
			OrderID:              order.ID,
			ProductID:            cartItem.ProductID,
			Name:                 cartItem.ProductName,
			Quantity:             cartItem.Quantity,
			Price:                cartItem.Price,
			PurchaseType:         cartItem.PurchaseType,         // *** Include purchase type ***
			SubscriptionInterval: cartItem.SubscriptionInterval, // *** Include subscription info ***
			StripePriceID:        cartItem.StripePriceID,        // *** Include Stripe Price ID ***
			CreatedAt:            time.Now(),
		})
	}

	if err := s.orderRepo.CreateOrderItems(ctx, order.ID, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Clear the cart after successful order creation
	if err := s.cartService.Clear(ctx, cart.ID); err != nil {
		// Log error but don't fail - order was created successfully
		log.Printf("Failed to clear cart after order creation: %v", err)
	}

	// Publish inventory reduction event
	inventoryItems := make([]map[string]interface{}, len(orderItems))
	for i, item := range orderItems {
		inventoryItems[i] = map[string]interface{}{
			"product_id": item.ProductID,
			"quantity":   item.Quantity,
		}
	}

	if err := s.publishOrderEvent(ctx, "inventory.reduce_stock", order.ID, map[string]interface{}{
		"order_id": order.ID,
		"items":    inventoryItems,
	}); err != nil {
		log.Printf("⚠️ Failed to publish inventory reduction event for order %d: %v", order.ID, err)
	}

	// Publish order confirmed event with enhanced data
	if err := s.publishOrderEvent(ctx, interfaces.EventOrderConfirmed, order.ID, map[string]interface{}{
		"customer_id":       customerID,
		"payment_intent_id": paymentIntentID,
		"total":             order.Total,
		"item_count":        len(orderItems),
		"has_subscription":  s.hasSubscriptionItems(orderItems),
		"cart_cleared":      true,
	}); err != nil {
		// Log error but don't fail order creation
		log.Printf("Failed to publish order confirmed event: %v", err)
	}

	log.Printf("✅ Order %d created from payment %s (Customer: %d, Total: $%.2f)",
		order.ID, paymentIntentID, customerID, float64(amount)/100)

	return order, nil
}

// UpdateStripeChargeID updates an order with the Stripe charge ID for reference
func (s *OrderService) UpdateStripeChargeID(ctx context.Context, orderID int32, chargeID string) error {
	return s.orderRepo.UpdateStripeChargeID(ctx, orderID, chargeID)
}

// Helper method to check if order contains subscription items
func (s *OrderService) hasSubscriptionItems(items []interfaces.OrderItem) bool {
	for _, item := range items {
		if item.PurchaseType == "subscription" {
			return true
		}
	}
	return false
}

func (s *OrderService) CreateOrder(ctx context.Context, req interfaces.CreateOrderRequest) (*interfaces.Order, error) {
	// Validate request
	if err := interfaces.ValidateCreateOrderRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Create order
	order := &interfaces.Order{
		CustomerID: req.CustomerID,
		Status:     database.OrderStatusPending,
		Total:      req.Total,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Create order items
	var orderItems []interfaces.OrderItem
	for _, reqItem := range req.Items {
		orderItems = append(orderItems, interfaces.OrderItem{
			OrderID:   order.ID,
			ProductID: reqItem.ProductID,
			Name:      reqItem.Name,
			Quantity:  reqItem.Quantity,
			Price:     reqItem.Price,
			CreatedAt: time.Now(),
		})
	}

	if err := s.orderRepo.CreateOrderItems(ctx, order.ID, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Publish order created event
	if err := s.publishOrderEvent(ctx, interfaces.EventOrderCreated, order.ID, map[string]interface{}{
		"customer_id": req.CustomerID,
		"total":       order.Total,
		"item_count":  len(orderItems),
	}); err != nil {
		fmt.Printf("Failed to publish order created event: %v\n", err)
	}

	return order, nil
}

// =============================================================================
// Order Retrieval
// =============================================================================

// GetByID retrieves an order with all its items by order ID
func (s *OrderService) GetByID(ctx context.Context, orderID int32) (*interfaces.OrderWithItems, error) {
	if orderID <= 0 {
		return nil, fmt.Errorf("invalid order ID: %d", orderID)
	}

	// Use the repository's GetWithItems method to get order with items
	orderWithItems, err := s.orderRepo.GetWithItems(ctx, orderID)
	if err != nil {
		// Check if it's a "not found" error and provide appropriate message
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order %d: %w", orderID, err)
	}

	// Publish order accessed event for analytics (optional)
	if err := s.publishOrderEvent(ctx, "order.accessed", orderID, map[string]interface{}{
		"order_id":     orderID,
		"customer_id":  orderWithItems.CustomerID,
		"item_count":   len(orderWithItems.Items),
		"total_amount": orderWithItems.Total,
		"accessed_at":  time.Now(),
	}); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to publish order access event: %v\n", err)
	}

	return orderWithItems, nil
}

// GetByCustomer retrieves all orders for a specific customer with items and filtering
func (s *OrderService) GetByCustomer(ctx context.Context, customerID int32, filters interfaces.OrderFilters) ([]interfaces.OrderWithItems, error) {
	if customerID <= 0 {
		return nil, fmt.Errorf("invalid customer ID: %d", customerID)
	}

	// Ensure customer ID is set in filters
	filters.CustomerID = &customerID

	// Set default pagination if not provided
	if filters.Limit == 0 {
		filters.Limit = 50 // Default limit
	}
	if filters.Limit > 100 {
		filters.Limit = 100 // Max limit
	}

	// Get orders from repository (this returns basic order info)
	orders, err := s.orderRepo.GetByCustomerID(ctx, customerID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for customer %d: %w", customerID, err)
	}

	// If no orders found, return empty slice
	if len(orders) == 0 {
		return []interfaces.OrderWithItems{}, nil
	}

	// Convert to OrderWithItems by getting items for each order
	var ordersWithItems []interfaces.OrderWithItems
	for _, order := range orders {
		orderWithItems, err := s.orderRepo.GetWithItems(ctx, order.ID)
		if err != nil {
			// Log error but continue with other orders to provide partial results
			fmt.Printf("Failed to get items for order %d: %v\n", order.ID, err)

			// Create OrderWithItems with empty items slice as fallback
			var stripeSessionID *string
			if order.StripeSessionID.Valid {
				stripeSessionID = &order.StripeSessionID.String
			}

			var stripePaymentIntentID *string
			if order.StripePaymentIntentID.Valid {
				stripePaymentIntentID = &order.StripePaymentIntentID.String
			}

			// Create OrderWithItems with empty items slice as fallback
			fallbackOrder := &interfaces.OrderWithItems{
				ID:                    order.ID,
				CustomerID:            order.CustomerID,
				Status:                string(order.Status),
				Total:                 order.Total,
				StripeSessionID:       stripeSessionID,
				StripePaymentIntentID: stripePaymentIntentID,
				Items:                 []interfaces.OrderItem{}, // Empty items
				CreatedAt:             order.CreatedAt,
				UpdatedAt:             order.UpdatedAt,
			}
			ordersWithItems = append(ordersWithItems, *fallbackOrder)
			continue
		}
		ordersWithItems = append(ordersWithItems, *orderWithItems)
	}

	// Publish customer order accessed event for analytics
	if err := s.publishOrderEvent(ctx, "customer.orders_accessed", customerID, map[string]interface{}{
		"customer_id": customerID,
		"order_count": len(ordersWithItems),
		"filters":     filters,
		"accessed_at": time.Now(),
	}); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to publish order access event: %v\n", err)
	}

	return ordersWithItems, nil
}

// GetAll retrieves all orders with items based on filters
func (s *OrderService) GetAll(ctx context.Context, filters interfaces.OrderFilters) ([]interfaces.OrderWithItems, error) {
	// Set default pagination if not provided
	if filters.Limit == 0 {
		filters.Limit = 50 // Default limit
	}
	if filters.Limit > 100 {
		filters.Limit = 100 // Max limit to prevent performance issues
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	// Get orders from repository (this returns basic order info)
	orders, err := s.orderRepo.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	// If no orders found, return empty slice
	if len(orders) == 0 {
		return []interfaces.OrderWithItems{}, nil
	}

	// Convert to OrderWithItems by getting items for each order
	var ordersWithItems []interfaces.OrderWithItems
	var failedOrderIDs []int32

	for _, order := range orders {
		orderWithItems, err := s.orderRepo.GetWithItems(ctx, order.ID)
		if err != nil {
			// Log error but continue with other orders to provide partial results
			fmt.Printf("Failed to get items for order %d: %v\n", order.ID, err)
			failedOrderIDs = append(failedOrderIDs, order.ID)

			// Create OrderWithItems with empty items slice as fallback
			var stripeSessionID *string
			if order.StripeSessionID.Valid {
				stripeSessionID = &order.StripeSessionID.String
			}

			var stripePaymentIntentID *string
			if order.StripePaymentIntentID.Valid {
				stripePaymentIntentID = &order.StripePaymentIntentID.String
			}

			fallbackOrder := &interfaces.OrderWithItems{
				ID:                    order.ID,
				CustomerID:            order.CustomerID,
				Status:                string(order.Status),
				Total:                 order.Total,
				StripeSessionID:       stripeSessionID,
				StripePaymentIntentID: stripePaymentIntentID,
				Items:                 []interfaces.OrderItem{}, // Empty items
				CreatedAt:             order.CreatedAt,
				UpdatedAt:             order.UpdatedAt,
			}
			ordersWithItems = append(ordersWithItems, *fallbackOrder)
			continue
		}
		ordersWithItems = append(ordersWithItems, *orderWithItems)
	}

	// Publish analytics event for admin access patterns
	if err := s.publishOrderEvent(ctx, "orders.bulk_accessed", 0, map[string]interface{}{
		"total_orders":     len(ordersWithItems),
		"failed_orders":    len(failedOrderIDs),
		"failed_order_ids": failedOrderIDs,
		"filters":          filters,
		"accessed_at":      time.Now(),
	}); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to publish bulk order access event: %v\n", err)
	}

	return ordersWithItems, nil
}

// =============================================================================
// Order Management
// =============================================================================

// =============================================================================
// Order Validation
// =============================================================================

func (s *OrderService) ValidateOrderForPayment(ctx context.Context, orderID int32) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != database.OrderStatusPending {
		return fmt.Errorf("order status must be pending for payment, current status: %s", order.Status)
	}

	if order.Total <= 0 {
		return fmt.Errorf("order total must be positive")
	}

	return nil
}

func (s *OrderService) ValidateOrderForShipping(ctx context.Context, orderID int32) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if !interfaces.RequiresPayment(order.Status) {
		return fmt.Errorf("order must be paid before shipping, current status: %s", order.Status)
	}

	if order.Status == database.OrderStatusShipped || order.Status == database.OrderStatusDelivered {
		return fmt.Errorf("order already shipped or delivered")
	}

	return nil
}

// =============================================================================
// Order Statistics
// =============================================================================

// =============================================================================
// Helper Methods
// =============================================================================

func (s *OrderService) publishOrderEvent(ctx context.Context, eventType string, orderID int32, data map[string]interface{}) error {
	event := interfaces.Event{
		ID:          generateEventID(),
		Type:        eventType,
		AggregateID: fmt.Sprintf("order:%d", orderID),
		Data:        data,
		Timestamp:   time.Now(),
		Version:     1,
	}

	return s.events.PublishEvent(ctx, event)
}
