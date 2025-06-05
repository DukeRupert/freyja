// internal/service/order.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/jackc/pgx/v5/pgtype"
)

type OrderService struct {
	orderRepo   interfaces.OrderRepository
	cartService *CartService
	events      interfaces.EventPublisher
}

func NewOrderService(
	orderRepo interfaces.OrderRepository,
	cartService *CartService,
	events interfaces.EventPublisher,
) *OrderService {
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
	// Get cart with items
	cart, err := s.cartService.GetCart(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if len(cart.Items) == 0 {
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

	// Validate cart total matches payment amount
	if cart.Total != amount {
		return nil, fmt.Errorf("cart total (%d) does not match payment amount (%d)", cart.Total, amount)
	}

	// Create order with payment information
	order := &interfaces.Order{
		CustomerID: customerID,
		Status:     database.OrderStatusConfirmed,
		Total:      amount,
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
			OrderID:   order.ID,
			ProductID: cartItem.ProductID,
			Name:      cartItem.ProductName,
			Quantity:  cartItem.Quantity,
			Price:     cartItem.Price,
			CreatedAt: time.Now(),
		})
	}

	if err := s.orderRepo.CreateOrderItems(ctx, order.ID, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Clear the cart after successful order creation
	if err := s.cartService.ClearCart(ctx, cart.ID); err != nil {
		// Log error but don't fail - order was created successfully
		fmt.Printf("Failed to clear cart after order creation: %v\n", err)
	}

	// Publish order confirmed event
	if err := s.publishOrderEvent(ctx, interfaces.EventOrderConfirmed, order.ID, map[string]interface{}{
		"customer_id":        customerID,
		"payment_intent_id":  paymentIntentID,
		"total":              order.Total,
		"item_count":         len(orderItems),
	}); err != nil {
		fmt.Printf("Failed to publish order confirmed event: %v\n", err)
	}

	return order, nil
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

func (s *OrderService) GetByID(ctx context.Context, orderID int32) (*interfaces.OrderWithItems, error) {
	if orderID <= 0 {
		return nil, fmt.Errorf("invalid order ID: %d", orderID)
	}

	order, err := s.orderRepo.GetWithItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order %d: %w", orderID, err)
	}

	return order, nil
}

func (s *OrderService) GetByCustomer(ctx context.Context, customerID int32, filters interfaces.OrderFilters) ([]interfaces.OrderWithItems, error) {
	if customerID <= 0 {
		return nil, fmt.Errorf("invalid customer ID: %d", customerID)
	}

	// Set customer ID in filters
	filters.CustomerID = &customerID

	// Get orders from repository
	orders, err := s.orderRepo.GetByCustomerID(ctx, customerID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for customer %d: %w", customerID, err)
	}

	// Convert to OrderWithItems (for MVP, we'll get items separately if needed)
	var ordersWithItems []interfaces.OrderWithItems
	for _, order := range orders {
		orderWithItems, err := s.orderRepo.GetWithItems(ctx, order.ID)
		if err != nil {
			// Log error but continue with other orders
			fmt.Printf("Failed to get items for order %d: %v\n", order.ID, err)
			continue
		}
		ordersWithItems = append(ordersWithItems, *orderWithItems)
	}

	return ordersWithItems, nil
}

func (s *OrderService) GetAll(ctx context.Context, filters interfaces.OrderFilters) ([]interfaces.OrderWithItems, error) {
	// Get orders from repository
	orders, err := s.orderRepo.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	// Convert to OrderWithItems
	var ordersWithItems []interfaces.OrderWithItems
	for _, order := range orders {
		orderWithItems, err := s.orderRepo.GetWithItems(ctx, order.ID)
		if err != nil {
			// Log error but continue with other orders
			fmt.Printf("Failed to get items for order %d: %v\n", order.ID, err)
			continue
		}
		ordersWithItems = append(ordersWithItems, *orderWithItems)
	}

	return ordersWithItems, nil
}

// =============================================================================
// Order Management
// =============================================================================

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID int32, req interfaces.UpdateOrderStatusRequest) error {
	if orderID <= 0 {
		return fmt.Errorf("invalid order ID: %d", orderID)
	}

	// Validate request
	if err := interfaces.ValidateUpdateStatusRequest(req); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	// Get current order to check status transition
	currentOrder, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get current order: %w", err)
	}

	newStatus := database.OrderStatus(req.Status)
	
	// Validate status transition
	if !interfaces.CanTransitionTo(currentOrder.Status, newStatus) {
		return fmt.Errorf("cannot transition from %s to %s", currentOrder.Status, newStatus)
	}

	// Update order status
	if err := s.orderRepo.UpdateStatus(ctx, orderID, newStatus); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Publish status change event
	eventData := map[string]interface{}{
		"old_status": string(currentOrder.Status),
		"new_status": req.Status,
		"order_id":   orderID,
	}

	if req.TrackingNumber != nil {
		eventData["tracking_number"] = *req.TrackingNumber
	}

	if req.Notes != nil {
		eventData["notes"] = *req.Notes
	}

	// Determine event type based on new status
	var eventType string
	switch newStatus {
	case database.OrderStatusShipped:
		eventType = interfaces.EventOrderShipped
	case database.OrderStatusDelivered:
		eventType = interfaces.EventOrderDelivered
	case database.OrderStatusCancelled:
		eventType = interfaces.EventOrderCancelled
	default:
		eventType = "order.status_updated"
	}

	if err := s.publishOrderEvent(ctx, eventType, orderID, eventData); err != nil {
		fmt.Printf("Failed to publish order status event: %v\n", err)
	}

	return nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID int32, reason string) error {
	if orderID <= 0 {
		return fmt.Errorf("invalid order ID: %d", orderID)
	}

	// Get current order
	currentOrder, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Check if order can be cancelled
	if !interfaces.CanTransitionTo(currentOrder.Status, database.OrderStatusCancelled) {
		return fmt.Errorf("order with status %s cannot be cancelled", currentOrder.Status)
	}

	// Update to cancelled status
	if err := s.orderRepo.UpdateStatus(ctx, orderID, database.OrderStatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	// Publish cancellation event
	if err := s.publishOrderEvent(ctx, interfaces.EventOrderCancelled, orderID, map[string]interface{}{
		"reason":     reason,
		"old_status": string(currentOrder.Status),
	}); err != nil {
		fmt.Printf("Failed to publish order cancelled event: %v\n", err)
	}

	return nil
}

func (s *OrderService) RefundOrder(ctx context.Context, orderID int32, amount *int32, reason string) error {
	// For MVP, this is a placeholder - full refund logic would integrate with payment provider
	return fmt.Errorf("refund functionality not implemented in MVP")
}

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

func (s *OrderService) GetOrderStats(ctx context.Context, filters interfaces.OrderFilters) (*interfaces.OrderStats, error) {
	// Get basic counts and revenue
	totalOrders, err := s.orderRepo.GetOrderCount(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get order count: %w", err)
	}

	totalRevenue, err := s.orderRepo.GetTotalRevenue(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get total revenue: %w", err)
	}

	// Calculate average order value
	var averageOrder float64
	if totalOrders > 0 {
		averageOrder = float64(totalRevenue) / float64(totalOrders)
	}

	// Get order counts by status (if repository supports it)
	var ordersByStatus map[string]int64
	if repo, ok := s.orderRepo.(*repository.PostgresOrderRepository); ok {
		if statusCounts, err := repo.GetOrderCountByStatus(ctx); err == nil {
			ordersByStatus = statusCounts
		}
	}

	return &interfaces.OrderStats{
		TotalOrders:    totalOrders,
		TotalRevenue:   totalRevenue,
		AverageOrder:   averageOrder,
		OrdersByStatus: ordersByStatus,
		// RevenueByDay can be implemented later with date range queries
	}, nil
}

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

func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}