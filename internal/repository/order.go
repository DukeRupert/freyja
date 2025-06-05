// internal/repository/order.go
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresOrderRepository struct {
	db *database.DB
}

func NewPostgresOrderRepository(db *database.DB) interfaces.OrderRepository {
	return &PostgresOrderRepository{
		db: db,
	}
}

// =============================================================================
// Basic CRUD Operations
// =============================================================================

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id int32) (*interfaces.Order, error) {
	order, err := r.db.Queries.GetOrder(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return &order, nil
}

func (r *PostgresOrderRepository) GetWithItems(ctx context.Context, id int32) (*interfaces.OrderWithItems, error) {
	rows, err := r.db.Queries.GetOrderWithItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order with items: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("order not found")
	}

	// First row contains the order info (same across all rows)
	order := rows[0].Orders

	// Collect all order items (filter out null items from LEFT JOIN)
	var items []interfaces.OrderItem
	for _, row := range rows {
		if row.OrderItems.ID != 0 { // Check if order item exists
			items = append(items, row.OrderItems)
		}
	}

	// Build OrderWithItems from the embedded structs
	orderWithItems := &interfaces.OrderWithItems{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Status:     string(order.Status),
		Total:      order.Total,
		Items:      items,
		CreatedAt:  order.CreatedAt,
		UpdatedAt:  order.UpdatedAt,
	}

	// Handle optional Stripe fields
	if order.StripeSessionID.Valid {
		sessionID := order.StripeSessionID.String
		orderWithItems.StripeSessionID = &sessionID
	}

	if order.StripePaymentIntentID.Valid {
		paymentIntentID := order.StripePaymentIntentID.String
		orderWithItems.StripePaymentIntentID = &paymentIntentID
	}

	return orderWithItems, nil
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order *interfaces.Order) error {
	created, err := r.db.Queries.CreateOrder(ctx, database.CreateOrderParams{
		CustomerID:            order.CustomerID,
		Status:                order.Status,
		Total:                 order.Total,
		StripeSessionID:       order.StripeSessionID,
		StripePaymentIntentID: order.StripePaymentIntentID,
	})
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Update the order with the generated ID and timestamps
	order.ID = created.ID
	order.CreatedAt = created.CreatedAt
	order.UpdatedAt = created.UpdatedAt

	return nil
}

func (r *PostgresOrderRepository) Update(ctx context.Context, order *interfaces.Order) error {
	// For MVP, we'll focus on status updates
	// Full update can be added later when needed
	updated, err := r.db.Queries.UpdateOrderStatus(ctx, database.UpdateOrderStatusParams{
		ID:     order.ID,
		Status: order.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	// Update the order with fresh data
	order.Status = updated.Status
	order.UpdatedAt = updated.UpdatedAt

	return nil
}

func (r *PostgresOrderRepository) UpdateStatus(ctx context.Context, id int32, status database.OrderStatus) error {
	_, err := r.db.Queries.UpdateOrderStatus(ctx, database.UpdateOrderStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

func (r *PostgresOrderRepository) Delete(ctx context.Context, id int32) error {
	// For MVP, we'll implement soft delete by updating status
	// Hard delete can be added later if needed
	return r.UpdateStatus(ctx, id, database.OrderStatusCancelled)
}

// =============================================================================
// Query Operations
// =============================================================================

func (r *PostgresOrderRepository) GetByCustomerID(ctx context.Context, customerID int32, filters interfaces.OrderFilters) ([]interfaces.Order, error) {
	// Apply default pagination if not specified
	limit := int32(50)
	offset := int32(0)

	if filters.Limit > 0 {
		limit = int32(filters.Limit)
	}
	if filters.Offset > 0 {
		offset = int32(filters.Offset)
	}

	orders, err := r.db.Queries.GetOrdersByCustomerID(ctx, database.GetOrdersByCustomerIDParams{
		CustomerID: customerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by customer ID: %w", err)
	}

	// Convert to interface types
	result := make([]interfaces.Order, len(orders))
	for i, order := range orders {
		result[i] = order
	}

	return result, nil
}

func (r *PostgresOrderRepository) GetAll(ctx context.Context, filters interfaces.OrderFilters) ([]interfaces.Order, error) {
	// Apply default pagination
	limit := int32(50)
	offset := int32(0)

	if filters.Limit > 0 {
		limit = int32(filters.Limit)
	}
	if filters.Offset > 0 {
		offset = int32(filters.Offset)
	}

	var orders []database.Orders
	var err error

	// If status filter is specified, use GetOrdersByStatus
	if filters.Status != nil {
		orders, err = r.db.Queries.GetOrdersByStatus(ctx, database.GetOrdersByStatusParams{
			Status: database.OrderStatus(*filters.Status),
			Limit:  limit,
			Offset: offset,
		})
	} else {
		// Use GetRecentOrders for general listing
		orders, err = r.db.Queries.GetRecentOrders(ctx, database.GetRecentOrdersParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	// Convert to interface types
	result := make([]interfaces.Order, len(orders))
	for i, order := range orders {
		result[i] = order
	}

	return result, nil
}

func (r *PostgresOrderRepository) GetByStripeSessionID(ctx context.Context, sessionID string) (*interfaces.Order, error) {
	order, err := r.db.Queries.GetOrderByStripeSessionID(ctx, pgtype.Text{
		String: sessionID,
		Valid:  true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order not found for Stripe session ID")
		}
		return nil, fmt.Errorf("failed to get order by Stripe session ID: %w", err)
	}
	return &order, nil
}

func (r *PostgresOrderRepository) GetByStripePaymentIntentID(ctx context.Context, paymentIntentID string) (*interfaces.Order, error) {
	order, err := r.db.Queries.GetOrderByStripePaymentIntentID(ctx, pgtype.Text{
		String: paymentIntentID,
		Valid:  true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order not found for Stripe payment intent ID")
		}
		return nil, fmt.Errorf("failed to get order by Stripe payment intent ID: %w", err)
	}
	return &order, nil
}

// =============================================================================
// Order Items Operations
// =============================================================================

func (r *PostgresOrderRepository) GetOrderItems(ctx context.Context, orderID int32) ([]interfaces.OrderItem, error) {
	items, err := r.db.Queries.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	// Convert to interface types
	result := make([]interfaces.OrderItem, len(items))
	for i, item := range items {
		result[i] = item
	}

	return result, nil
}

func (r *PostgresOrderRepository) CreateOrderItems(ctx context.Context, orderID int32, items []interfaces.OrderItem) error {
	// Create each item individually
	// In a production system, you might want to use a batch insert
	for _, item := range items {
		_, err := r.db.Queries.CreateOrderItem(ctx, database.CreateOrderItemParams{
			OrderID:   orderID,
			ProductID: item.ProductID,
			Name:      item.Name,
			Quantity:  item.Quantity,
			Price:     item.Price,
		})
		if err != nil {
			return fmt.Errorf("failed to create order item for product %d: %w", item.ProductID, err)
		}
	}

	return nil
}

// =============================================================================
// Statistics and Reporting
// =============================================================================

func (r *PostgresOrderRepository) GetOrderCount(ctx context.Context, filters interfaces.OrderFilters) (int64, error) {
	if filters.CustomerID != nil {
		// Get count for specific customer
		return r.db.Queries.GetCustomerOrderCount(ctx, *filters.CustomerID)
	}

	// Get total order count
	count, err := r.db.Queries.GetTotalOrderCount(ctx)
	return int64(count), err
}

func (r *PostgresOrderRepository) GetTotalRevenue(ctx context.Context, filters interfaces.OrderFilters) (int64, error) {
	// For MVP, get total revenue from all completed orders
	// More complex filtering can be added later
	revenue, err := r.db.Queries.GetTotalRevenue(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get total revenue: %w", err)
	}
	return int64(revenue), nil
}

func (r *PostgresOrderRepository) GetOrdersByDateRange(ctx context.Context, from, to time.Time) ([]interfaces.Order, error) {
	// For MVP, this can be implemented later with a custom query
	// For now, return an error indicating it's not implemented
	return nil, fmt.Errorf("GetOrdersByDateRange not implemented in MVP")
}

// =============================================================================
// Helper Methods for Status Analysis
// =============================================================================

// GetOrderCountByStatus returns order counts grouped by status
func (r *PostgresOrderRepository) GetOrderCountByStatus(ctx context.Context) (map[string]int64, error) {
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