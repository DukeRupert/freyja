// internal/repository/order.go
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/jackc/pgx/v5/pgtype"
)

type OrderRepository struct {
	db *database.DB
}

func NewOrderRepository(db *database.DB) interfaces.OrderRepository {
	return &OrderRepository{
		db: db,
	}
}

// Create creates a new order
func (r *OrderRepository) Create(ctx context.Context, order *interfaces.Order) error {
	created, err := r.db.Queries.CreateOrder(ctx, database.CreateOrderParams{
		CustomerID:            order.CustomerID,
		Status:                order.Status,
		Total:                 order.Total,
		StripeSessionID:       order.StripeSessionID,
		StripePaymentIntentID: order.StripePaymentIntentID,
		StripeChargeID:        order.StripeChargeID,
	})
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Update the order with the created data
	order.ID = created.ID
	order.CreatedAt = created.CreatedAt
	order.UpdatedAt = created.UpdatedAt

	return nil
}

// GetByID retrieves an order by ID
func (r *OrderRepository) GetByID(ctx context.Context, id int32) (*interfaces.Order, error) {
	dbOrder, err := r.db.Queries.GetOrder(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return r.convertToOrder(dbOrder), nil
}

// GetByCustomerID retrieves orders for a customer with filters
func (r *OrderRepository) GetByCustomerID(ctx context.Context, customerID int32, filters interfaces.OrderFilters) ([]interfaces.Order, error) {
	limit := int32(10)
	offset := int32(0)

	if filters.Limit > 0 {
		limit = int32(filters.Limit)
	}
	if filters.Offset >= 0 {
		offset = int32(filters.Offset)
	}

	// Execute the appropriate query using helper function
	dbOrders, err := r.executeOrderQuery(ctx, customerID, filters, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	// Convert results
	orders := make([]interfaces.Order, len(dbOrders))
	for i, dbOrder := range dbOrders {
		orders[i] = *r.convertToOrder(dbOrder)
	}

	return orders, nil
}

// GetAll retrieves all orders with comprehensive filtering using generated SQLC interface
func (r *OrderRepository) GetAll(ctx context.Context, filters interfaces.OrderFilters) ([]interfaces.Order, error) {
	// Set default pagination
	limit := int32(filters.Limit)
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset := int32(filters.Offset)
	if offset < 0 {
		offset = 0
	}

	// Prepare customer ID - use 0 for "no filter" since SQLC generated it as int32
	customerID := int32(0)
	if filters.CustomerID != nil {
		customerID = *filters.CustomerID
	}

	// Prepare status parameter - use empty string for "no filter"
	status := ""
	if filters.Status != nil && *filters.Status != "" {
		status = *filters.Status
	}

	// Prepare date parameters using pgtype.Timestamptz
	var dateFrom pgtype.Timestamptz
	if filters.DateFrom != nil {
		dateFrom = pgtype.Timestamptz{
			Time:  *filters.DateFrom,
			Valid: true,
		}
	}
	// If filters.DateFrom is nil, dateFrom.Valid remains false (equivalent to NULL)

	var dateTo pgtype.Timestamptz
	if filters.DateTo != nil {
		dateTo = pgtype.Timestamptz{
			Time:  *filters.DateTo,
			Valid: true,
		}
	}
	// If filters.DateTo is nil, dateTo.Valid remains false (equivalent to NULL)

	// Call the generated SQLC method
	dbOrders, err := r.db.Queries.GetAllOrdersWithFilters(ctx, database.GetAllOrdersWithFiltersParams{
		CustomerID:  customerID,
		Status:      status,
		DateFrom:    dateFrom,
		DateTo:      dateTo,
		LimitCount:  limit,
		OffsetCount: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get orders with filters: %w", err)
	}

	// Convert database orders to interface orders
	orders := make([]interfaces.Order, len(dbOrders))
	for i, dbOrder := range dbOrders {
		orders[i] = *r.convertToOrder(dbOrder)
	}

	return orders, nil
}

// UpdateStatus updates an order's status
func (r *OrderRepository) UpdateStatus(ctx context.Context, id int32, status string) error {
	s, err := parseOrderStatus(status)
	if err != nil {
		return fmt.Errorf("Invalid order status: %w", err)
	}

	_, err = r.db.Queries.UpdateOrderStatus(ctx, database.UpdateOrderStatusParams{
		ID:     id,
		Status: s,
	})
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// UpdateStripeChargeID updates an order's Stripe charge ID
func (r *OrderRepository) UpdateStripeChargeID(ctx context.Context, orderID int32, chargeID string) error {
	_, err := r.db.Queries.UpdateStripeChargeID(ctx, database.UpdateStripeChargeIDParams{
		ID: orderID,
		StripeChargeID: pgtype.Text{
			String: chargeID,
			Valid:  true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update stripe charge ID: %w", err)
	}

	return nil
}

// CreateOrderItems creates order items for an order
func (r *OrderRepository) CreateOrderItems(ctx context.Context, orderID int32, items []interfaces.OrderItem) error {
	for _, item := range items {
		_, err := r.db.Queries.CreateOrderItem(ctx, database.CreateOrderItemParams{
			OrderID:              orderID,
			ProductID:            item.ProductID,
			Name:                 item.Name,
			Quantity:             item.Quantity,
			Price:                item.Price,
			PurchaseType:         item.PurchaseType,
			SubscriptionInterval: item.SubscriptionInterval,
			StripePriceID:        item.StripePriceID,
		})
		if err != nil {
			return fmt.Errorf("failed to create order item: %w", err)
		}
	}

	return nil
}

// GetOrderItems retrieves all items for an order
func (r *OrderRepository) GetOrderItems(ctx context.Context, orderID int32) ([]interfaces.OrderItem, error) {
	dbItems, err := r.db.Queries.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	items := make([]interfaces.OrderItem, len(dbItems))
	for i, dbItem := range dbItems {
		items[i] = r.convertToOrderItem(dbItem)
	}

	return items, nil
}

// GetWithItems retrieves an order with all its items
func (r *OrderRepository) GetWithItems(ctx context.Context, id int32) (*interfaces.OrderWithItems, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid order ID: %d", id)
	}

	// Get the order first
	order, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order %d: %w", id, err)
	}

	// Get the order items using existing method
	orderItems, err := r.GetOrderItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get items for order %d: %w", id, err)
	}

	// Convert pgtype fields to *string for OrderWithItems
	var stripeSessionID *string
	if order.StripeSessionID.Valid {
		stripeSessionID = &order.StripeSessionID.String
	}

	var stripePaymentIntentID *string
	if order.StripePaymentIntentID.Valid {
		stripePaymentIntentID = &order.StripePaymentIntentID.String
	}

	// Build OrderWithItems
	orderWithItems := &interfaces.OrderWithItems{
		ID:                    order.ID,
		CustomerID:            order.CustomerID,
		Status:                string(order.Status),
		Total:                 order.Total,
		StripeSessionID:       stripeSessionID,
		StripePaymentIntentID: stripePaymentIntentID,
		Items:                 orderItems,
		CreatedAt:             order.CreatedAt,
		UpdatedAt:             order.UpdatedAt,
	}

	return orderWithItems, nil
}

// GetOrdersByStatus retrieves orders by status
func (r *OrderRepository) GetOrdersByStatus(ctx context.Context, status string, limit, offset int) ([]interfaces.Order, error) {
	s, err := parseOrderStatus(status)
	if err != nil {
		return nil, fmt.Errorf("invalid order status: %w", err)
	}

	orders, err := r.db.Queries.GetOrdersByStatus(ctx, database.GetOrdersByStatusParams{
		Status: s,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by status: %w", err)
	}

	return orders, nil
}

func (r *OrderRepository) GetOrderCountByStatus(ctx context.Context) (map[string]int64, error) {
	results, err := r.db.Queries.GetOrderCountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get order count by status: %w", err)
	}

	statusCounts := make(map[string]int64)
	for _, result := range results {
		statusCounts[string(result.Status)] = int64(result.Count)
	}

	return statusCounts, nil
}

// Helper methods to convert between database and interface types

func (r *OrderRepository) convertToOrder(dbOrder database.Orders) *interfaces.Order {
	return &interfaces.Order{
		ID:                    dbOrder.ID,
		CustomerID:            dbOrder.CustomerID,
		Status:                dbOrder.Status,
		Total:                 dbOrder.Total,
		StripeSessionID:       dbOrder.StripeSessionID,
		StripePaymentIntentID: dbOrder.StripePaymentIntentID,
		StripeChargeID:        dbOrder.StripeChargeID,
		CreatedAt:             dbOrder.CreatedAt,
		UpdatedAt:             dbOrder.UpdatedAt,
	}
}

func (r *OrderRepository) convertToOrderItem(dbItem database.GetOrderItemsRow) interfaces.OrderItem {
	return interfaces.OrderItem{
		ID:                   dbItem.ID,
		OrderID:              dbItem.OrderID,
		ProductID:            dbItem.ProductID,
		Name:                 dbItem.Name,
		Quantity:             dbItem.Quantity,
		Price:                dbItem.Price,
		PurchaseType:         dbItem.PurchaseType,
		SubscriptionInterval: dbItem.SubscriptionInterval,
		StripePriceID:        dbItem.StripePriceID,
		CreatedAt:            dbItem.CreatedAt,
	}
}

// convert string into OrderStatus
func parseOrderStatus(s string) (database.OrderStatus, error) {
	switch s {
	case "pending":
		return database.OrderStatusPending, nil
	case "payment_processing":
		return database.OrderStatusPaymentProcessing, nil
	case "confirmed":
		return database.OrderStatusConfirmed, nil
	case "processing":
		return database.OrderStatusProcessing, nil
	case "shipped":
		return database.OrderStatusShipped, nil
	case "delivered":
		return database.OrderStatusDelivered, nil
	case "cancelled":
		return database.OrderStatusCancelled, nil
	case "refunded":
		return database.OrderStatusRefunded, nil
	default:
		return "", fmt.Errorf("invalid order status: %s", s)
	}
}

// Helper function to execute the appropriate query based on filters
func (r *OrderRepository) executeOrderQuery(ctx context.Context, customerID int32, filters interfaces.OrderFilters, limit, offset int32) ([]database.Orders, error) {
	switch {
	case filters.Status != nil && filters.DateFrom != nil && filters.DateTo != nil:
		return r.getOrdersWithStatusAndDateRange(ctx, customerID, *filters.Status, *filters.DateFrom, *filters.DateTo, limit, offset)

	case filters.Status != nil:
		return r.getOrdersWithStatus(ctx, customerID, *filters.Status, limit, offset)

	case filters.DateFrom != nil && filters.DateTo != nil:
		return r.getOrdersWithDateRange(ctx, customerID, *filters.DateFrom, *filters.DateTo, limit, offset)

	default:
		return r.getOrdersBasic(ctx, customerID, limit, offset)
	}
}

// Individual query methods
func (r *OrderRepository) getOrdersWithStatusAndDateRange(ctx context.Context, customerID int32, statusStr string, dateFrom, dateTo time.Time, limit, offset int32) ([]database.Orders, error) {
	status, err := parseOrderStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	return r.db.Queries.GetOrdersByCustomerIDWithStatusAndDateRange(ctx, database.GetOrdersByCustomerIDWithStatusAndDateRangeParams{
		CustomerID:  customerID,
		Status:      status,
		After:       pgtype.Timestamptz{Time: dateFrom, Valid: true},
		Before:      pgtype.Timestamptz{Time: dateTo, Valid: true},
		LimitCount:  limit,
		OffsetCount: offset,
	})
}

func (r *OrderRepository) getOrdersWithStatus(ctx context.Context, customerID int32, statusStr string, limit, offset int32) ([]database.Orders, error) {
	status, err := parseOrderStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	return r.db.Queries.GetOrdersByCustomerIDAndStatus(ctx, database.GetOrdersByCustomerIDAndStatusParams{
		CustomerID: customerID,
		Status:     status,
		Limit:      limit,
		Offset:     offset,
	})
}

func (r *OrderRepository) getOrdersWithDateRange(ctx context.Context, customerID int32, dateFrom, dateTo time.Time, limit, offset int32) ([]database.Orders, error) {
	return r.db.Queries.GetOrdersByCustomerIDAndDateRange(ctx, database.GetOrdersByCustomerIDAndDateRangeParams{
		CustomerID:  customerID,
		After:       pgtype.Timestamptz{Time: dateFrom, Valid: true},
		Before:      pgtype.Timestamptz{Time: dateTo, Valid: true},
		LimitCount:  limit,
		OffsetCount: offset,
	})
}

func (r *OrderRepository) getOrdersBasic(ctx context.Context, customerID int32, limit, offset int32) ([]database.Orders, error) {
	return r.db.Queries.GetOrdersByCustomerID(ctx, database.GetOrdersByCustomerIDParams{
		CustomerID: customerID,
		Limit:      limit,
		Offset:     offset,
	})
}
