// internal/server/service/order.go
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
	variantRepo interfaces.VariantRepository
	events      interfaces.EventPublisher
}

func NewOrderService(
	orderRepo interfaces.OrderRepository,
	cartService interfaces.CartService,
	variantRepo interfaces.VariantRepository,
	events interfaces.EventPublisher,
) interfaces.OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		cartService: cartService,
		variantRepo: variantRepo,
		events:      events,
	}
}

// =============================================================================
// Order Creation
// =============================================================================

// CreateOrderFromCart creates an order from cart items (main checkout flow)
func (s *OrderService) CreateOrderFromCart(ctx context.Context, customerID int32, cartID int32) (*interfaces.OrderWithItems, error) {
	// Validate cart for checkout
	validatedCart, err := s.cartService.ValidateCartForCheckout(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart validation failed: %w", err)
	}

	if len(validatedCart.Items) == 0 {
		return nil, fmt.Errorf("cannot create order from empty cart")
	}

	// Create order entity
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

	// Convert cart items to order items (now using variant IDs)
	var orderItems []interfaces.OrderItem
	for _, cartItem := range validatedCart.Items {
		orderItems = append(orderItems, interfaces.OrderItem{
			OrderID:              order.ID,
			ProductVariantID:     cartItem.ProductVariantID,
			Name:                 cartItem.VariantName,
			VariantName:          cartItem.VariantName,
			Quantity:             cartItem.Quantity,
			Price:                cartItem.Price,
			PurchaseType:         cartItem.PurchaseType,
			SubscriptionInterval: cartItem.SubscriptionInterval,
			StripePriceID:        cartItem.StripePriceID,
			CreatedAt:            time.Now(),
		})
	}

	// Create order items
	if err := s.orderRepo.CreateOrderItems(ctx, order.ID, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Decrement variant stock for each item
	for _, item := range orderItems {
		_, err := s.variantRepo.DecrementStock(ctx, item.ProductVariantID, item.Quantity)
		if err != nil {
			log.Printf("Failed to decrement stock for variant %d: %v", item.ProductVariantID, err)
			// Continue processing but log the error - this should be handled by eventual consistency
		}
	}

	// Clear the cart after successful order creation
	if err := s.cartService.Clear(ctx, cartID); err != nil {
		log.Printf("Failed to clear cart %d after order creation: %v", cartID, err)
		// Don't fail the order creation for this
	}

	// Publish order created event
	if err := s.publishOrderEvent(ctx, "order.created", order.ID, map[string]interface{}{
		"customer_id":    customerID,
		"total":          order.Total,
		"item_count":     len(orderItems),
		"cart_id":        cartID,
		"has_subscription": s.hasSubscriptionItems(orderItems),
	}); err != nil {
		log.Printf("Failed to publish order created event: %v", err)
	}

	// Get the complete order with items to return
	return s.orderRepo.GetWithItems(ctx, order.ID)
}

// CreateOrderFromPayment creates an order from Stripe payment information
// This is used when we receive a successful payment webhook but don't have cart context
func (s *OrderService) CreateOrderFromPayment(ctx context.Context, customerID int32, paymentIntentID string, amount int32) (*interfaces.OrderWithItems, error) {
	// Create order entity with payment information
	order := &interfaces.Order{
		CustomerID: customerID,
		Status:     database.OrderStatusConfirmed, // Payment already succeeded
		Total:      amount,
		StripePaymentIntentID: pgtype.Text{String: paymentIntentID, Valid: true},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// We need to get the cart for this customer to convert to order items
	// First, try to find the customer's cart
	cart, err := s.cartService.GetCustomerCart(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer cart for order creation: %w", err)
	}

	if cart == nil || len(cart.Items) == 0 {
		return nil, fmt.Errorf("no cart found for customer %d to create order from payment", customerID)
	}

	// Validate cart items before creating order items
	validatedCart, err := s.cartService.ValidateCartForCheckout(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("cart validation failed: %w", err)
	}

	// Convert cart items to order items
	var orderItems []interfaces.OrderItem
	for _, cartItem := range validatedCart.Items {
		orderItems = append(orderItems, interfaces.OrderItem{
			OrderID:              order.ID,
			ProductVariantID:     cartItem.ProductVariantID,
			Name:                 cartItem.ProductName,
			VariantName:          cartItem.VariantName,
			Quantity:             cartItem.Quantity,
			Price:                cartItem.Price,
			PurchaseType:         cartItem.PurchaseType,
			SubscriptionInterval: cartItem.SubscriptionInterval,
			StripePriceID:        cartItem.StripePriceID,
			CreatedAt:            time.Now(),
		})
	}

	// Create order items
	if err := s.orderRepo.CreateOrderItems(ctx, order.ID, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Decrement variant stock for each item
	for _, item := range orderItems {
		_, err := s.variantRepo.DecrementStock(ctx, item.ProductVariantID, item.Quantity)
		if err != nil {
			log.Printf("Failed to decrement stock for variant %d: %v", item.ProductVariantID, err)
		}
	}

	// Clear the cart after successful order creation
	if err := s.cartService.Clear(ctx, cart.ID); err != nil {
		log.Printf("Failed to clear cart %d after order creation: %v", cart.ID, err)
	}

	// Publish order created event
	if err := s.publishOrderEvent(ctx, "order.created", order.ID, map[string]interface{}{
		"customer_id":       customerID,
		"total":             order.Total,
		"item_count":        len(orderItems),
		"payment_intent_id": paymentIntentID,
		"creation_source":   "webhook",
		"has_subscription":  s.hasSubscriptionItems(orderItems),
	}); err != nil {
		log.Printf("Failed to publish order created event: %v", err)
	}

	// Get the complete order with items to return
	return s.orderRepo.GetWithItems(ctx, order.ID)
}

// CreateOrder creates an order from explicit request data (admin/API usage)
func (s *OrderService) CreateOrder(ctx context.Context, req interfaces.CreateOrderRequest) (*interfaces.OrderWithItems, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("order must have at least one item")
	}

	// Validate all variants exist and are available, AND collect variant details
	variantDetails := make(map[int32]*interfaces.ProductVariant)
	for _, reqItem := range req.Items {
		variant, err := s.variantRepo.GetByID(ctx, reqItem.ProductVariantID)
		if err != nil {
			return nil, fmt.Errorf("variant %d not found: %w", reqItem.ProductVariantID, err)
		}
		if !variant.Active {
			return nil, fmt.Errorf("variant %d is not active", reqItem.ProductVariantID)
		}
		if variant.Stock < reqItem.Quantity {
			return nil, fmt.Errorf("insufficient stock for variant %d: requested %d, available %d",
				reqItem.ProductVariantID, reqItem.Quantity, variant.Stock)
		}
		
		// Store variant details for later use
		variantDetails[reqItem.ProductVariantID] = variant
	}

	// Create order entity
	order := &interfaces.Order{
		CustomerID: req.CustomerID,
		Status:     req.Status,
		Total:      req.Total,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Set Stripe IDs if provided
	if req.StripeSessionID != nil {
		order.StripeSessionID = pgtype.Text{String: *req.StripeSessionID, Valid: true}
	}
	if req.StripePaymentIntentID != nil {
		order.StripePaymentIntentID = pgtype.Text{String: *req.StripePaymentIntentID, Valid: true}
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Convert request items to order items
	var orderItems []interfaces.OrderItem
	for _, reqItem := range req.Items {
		variant := variantDetails[reqItem.ProductVariantID]
		
		// Determine names: use provided names if available, otherwise fetch from variant
		productName := reqItem.Name
		variantName := reqItem.VariantName
		
		// If variant name not provided in request, use variant details
		if variantName == "" {
			variantName = variant.Name
		}
		
		// Product name should always be provided in request for explicit orders
		// But as fallback, use variant name if needed
		if productName == "" {
			productName = variant.Name
		}
		
		orderItems = append(orderItems, interfaces.OrderItem{
			OrderID:              order.ID,
			ProductVariantID:     reqItem.ProductVariantID,
			Name:                 productName,
			VariantName:          variantName,
			Quantity:             reqItem.Quantity,
			Price:                reqItem.Price,
			PurchaseType:         reqItem.PurchaseType,
			SubscriptionInterval: func() pgtype.Text {
				if reqItem.SubscriptionInterval != nil {
					return pgtype.Text{String: *reqItem.SubscriptionInterval, Valid: true}
				}
				return pgtype.Text{}
			}(),
			StripePriceID: reqItem.StripePriceID,
			CreatedAt:     time.Now(),
		})
	}

	// Create order items
	if err := s.orderRepo.CreateOrderItems(ctx, order.ID, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Decrement variant stock
	for _, item := range orderItems {
		_, err := s.variantRepo.DecrementStock(ctx, item.ProductVariantID, item.Quantity)
		if err != nil {
			log.Printf("Failed to decrement stock for variant %d: %v", item.ProductVariantID, err)
		}
	}

	// Publish order created event
	if err := s.publishOrderEvent(ctx, "order.created", order.ID, map[string]interface{}{
		"customer_id":      req.CustomerID,
		"total":            order.Total,
		"item_count":       len(orderItems),
		"creation_source":  "api",
		"has_subscription": s.hasSubscriptionItems(orderItems),
	}); err != nil {
		log.Printf("Failed to publish order created event: %v", err)
	}

	// Get the complete order with items to return
	return s.orderRepo.GetWithItems(ctx, order.ID)
}

// =============================================================================
// Order Retrieval
// =============================================================================

// GetByID retrieves an order with all its items by order ID
func (s *OrderService) GetByID(ctx context.Context, orderID int32) (*interfaces.OrderWithItems, error) {
	if orderID <= 0 {
		return nil, fmt.Errorf("invalid order ID: %d", orderID)
	}

	orderWithItems, err := s.orderRepo.GetWithItems(ctx, orderID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order %d: %w", orderID, err)
	}

	// Publish order accessed event for analytics
	if err := s.publishOrderEvent(ctx, "order.accessed", orderID, map[string]interface{}{
		"customer_id":  orderWithItems.CustomerID,
		"item_count":   len(orderWithItems.Items),
		"total_amount": orderWithItems.Total,
		"status":       orderWithItems.Status,
	}); err != nil {
		log.Printf("Failed to publish order access event: %v", err)
	}

	return orderWithItems, nil
}

// GetByCustomer retrieves all orders for a specific customer with items and filtering
func (s *OrderService) GetByCustomer(ctx context.Context, customerID int32, filters interfaces.OrderFilters) ([]interfaces.OrderWithItems, error) {
	if customerID <= 0 {
		return nil, fmt.Errorf("invalid customer ID: %d", customerID)
	}

	// Set customer ID in filters
	filters.CustomerID = &customerID

	// Set default pagination if not provided
	if filters.Limit == 0 {
		filters.Limit = 20 // Default limit for customer queries
	}
	if filters.Limit > 50 {
		filters.Limit = 50 // Max limit for customer queries
	}

	ordersWithItems, err := s.orderRepo.GetOrdersWithItems(ctx, customerID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for customer %d: %w", customerID, err)
	}

	// Publish customer order access event
	if err := s.publishOrderEvent(ctx, "orders.customer_accessed", 0, map[string]interface{}{
		"customer_id":  customerID,
		"total_orders": len(ordersWithItems),
		"filters":      filters,
	}); err != nil {
		log.Printf("Failed to publish customer order access event: %v", err)
	}

	return ordersWithItems, nil
}

// GetAll retrieves all orders with items based on filters (admin usage)
func (s *OrderService) GetAll(ctx context.Context, filters interfaces.OrderFilters) ([]interfaces.OrderWithItems, error) {
	// Set default pagination if not provided
	if filters.Limit == 0 {
		filters.Limit = 50 // Default limit for admin queries
	}
	if filters.Limit > 100 {
		filters.Limit = 100 // Max limit to prevent performance issues
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	// Get orders from repository
	orders, err := s.orderRepo.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	// Convert to OrderWithItems by getting items for each order
	var ordersWithItems []interfaces.OrderWithItems
	for _, order := range orders {
		orderWithItems, err := s.orderRepo.GetWithItems(ctx, order.ID)
		if err != nil {
			log.Printf("Failed to get items for order %d: %v", order.ID, err)
			continue // Skip this order but continue with others
		}
		ordersWithItems = append(ordersWithItems, *orderWithItems)
	}

	// Publish admin access event
	if err := s.publishOrderEvent(ctx, "orders.admin_accessed", 0, map[string]interface{}{
		"total_orders": len(ordersWithItems),
		"filters":      filters,
	}); err != nil {
		log.Printf("Failed to publish admin order access event: %v", err)
	}

	return ordersWithItems, nil
}

// =============================================================================
// Order Management
// =============================================================================

// UpdateStatus updates the status of an order
func (s *OrderService) UpdateStatus(ctx context.Context, orderID int32, status database.OrderStatus) error {
	if orderID <= 0 {
		return fmt.Errorf("invalid order ID: %d", orderID)
	}

	// Get current order for event data
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	oldStatus := order.Status

	// Update status
	if err := s.orderRepo.UpdateStatus(ctx, orderID, string(status)); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Publish status change event
	if err := s.publishOrderEvent(ctx, "order.status_changed", orderID, map[string]interface{}{
		"customer_id": order.CustomerID,
		"old_status":  string(oldStatus),
		"new_status":  string(status),
	}); err != nil {
		log.Printf("Failed to publish order status change event: %v", err)
	}

	return nil
}

// CancelOrder cancels an order (sets status to cancelled)
func (s *OrderService) CancelOrder(ctx context.Context, orderID int32) error {
	return s.UpdateStatus(ctx, orderID, database.OrderStatusCancelled)
}

// UpdateStripeChargeID updates an order's Stripe charge ID
func (s *OrderService) UpdateStripeChargeID(ctx context.Context, orderID int32, chargeID string) error {
	if orderID <= 0 {
		return fmt.Errorf("invalid order ID: %d", orderID)
	}
	if chargeID == "" {
		return fmt.Errorf("charge ID cannot be empty")
	}

	if err := s.orderRepo.UpdateStripeChargeID(ctx, orderID, chargeID); err != nil {
		return fmt.Errorf("failed to update Stripe charge ID: %w", err)
	}

	// Publish Stripe charge update event
	if err := s.publishOrderEvent(ctx, "order.stripe_charge_updated", orderID, map[string]interface{}{
		"stripe_charge_id": chargeID,
	}); err != nil {
		log.Printf("Failed to publish Stripe charge update event: %v", err)
	}

	return nil
}

// =============================================================================
// Analytics
// =============================================================================

// GetOrderSummary returns summary information for an order
func (s *OrderService) GetOrderSummary(ctx context.Context, orderID int32) (*interfaces.OrderSummary, error) {
	summary, err := s.orderRepo.GetOrderSummary(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order summary: %w", err)
	}

	return summary, nil
}

// GetAdminStats returns comprehensive statistics for admin dashboard
func (s *OrderService) GetAdminStats(ctx context.Context) (*interfaces.AdminOrderStats, error) {
	// Get order count by status
	statusCounts, err := s.orderRepo.GetOrderCountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get order count by status: %w", err)
	}

	// Calculate totals
	var totalOrders int
	var totalRevenue int32
	for _, count := range statusCounts {
		totalOrders += count
	}

	// Get revenue from confirmed/completed orders only
	filters := interfaces.OrderFilters{
		Limit: 1000, // Get a reasonable sample for revenue calculation
	}
	
	// You may want to add a specific revenue calculation method to the repository
	// For now, we'll calculate from recent orders
	recentOrders, err := s.orderRepo.GetAll(ctx, filters)
	if err != nil {
		log.Printf("Failed to get orders for revenue calculation: %v", err)
	} else {
		for _, order := range recentOrders {
			if order.Status == database.OrderStatusConfirmed || 
			   order.Status == database.OrderStatusShipped ||
			   order.Status == database.OrderStatusDelivered {
				totalRevenue += order.Total
			}
		}
	}

	// Calculate average order value
	var averageOrderValue int32
	if totalOrders > 0 {
		averageOrderValue = totalRevenue / int32(totalOrders)
	}

	stats := &interfaces.AdminOrderStats{
		TotalOrders:       totalOrders,
		TotalRevenue:      totalRevenue,
		AverageOrderValue: averageOrderValue,
		OrdersByStatus:    statusCounts,
		GeneratedAt:       time.Now(),
	}

	return stats, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// hasSubscriptionItems checks if any order items are subscriptions
func (s *OrderService) hasSubscriptionItems(items []interfaces.OrderItem) bool {
	for _, item := range items {
		if item.PurchaseType == "subscription" {
			return true
		}
	}
	return false
}

// publishOrderEvent publishes order-related events
func (s *OrderService) publishOrderEvent(ctx context.Context, eventType string, orderID int32, data map[string]interface{}) error {
	if s.events == nil {
		return nil // Events are optional
	}

	event := interfaces.Event{
		ID: generateEventID(),
		Type: eventType,
		AggregateID: fmt.Sprintf("order:%d", orderID),
		Data: data,
		Timestamp: time.Now(),
		Version: 1,
	}
	return s.events.PublishEvent(ctx, event)
}