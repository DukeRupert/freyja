// internal/repository/order.go
package repository

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/jackc/pgx/v5/pgtype"
)

type OrderRepository struct {
	queries *database.Queries
}

func NewOrderRepository(queries *database.Queries) *OrderRepository {
	return &OrderRepository{
		queries: queries,
	}
}

// Create creates a new order
func (r *OrderRepository) Create(ctx context.Context, order *interfaces.Order) error {
	created, err := r.queries.CreateOrder(ctx, database.CreateOrderParams{
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
	dbOrder, err := r.queries.GetOrder(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return r.convertToOrder(dbOrder), nil
}

// GetByCustomerID retrieves orders for a customer
func (r *OrderRepository) GetByCustomerID(ctx context.Context, customerID int32, limit, offset int) ([]interfaces.Order, error) {
	dbOrders, err := r.queries.GetOrdersByCustomerID(ctx, database.GetOrdersByCustomerIDParams{
		CustomerID: customerID,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by customer ID: %w", err)
	}

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

	_, err = r.queries.UpdateOrderStatus(ctx, database.UpdateOrderStatusParams{
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
	_, err := r.queries.UpdateStripeChargeID(ctx, database.UpdateStripeChargeIDParams{
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
		_, err := r.queries.CreateOrderItem(ctx, database.CreateOrderItemParams{
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
	dbItems, err := r.queries.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	items := make([]interfaces.OrderItem, len(dbItems))
	for i, dbItem := range dbItems {
		items[i] = r.convertToOrderItem(dbItem)
	}

	return items, nil
}

// GetOrdersByStatus retrieves orders by status
func (r *OrderRepository) GetOrdersByStatus(ctx context.Context, status string, limit, offset int) ([]interfaces.Order, error) {
	s, err := parseOrderStatus(status)
	if err != nil {
		return nil, fmt.Errorf("invalid order status: %w", err)
	}

	orders, err := r.queries.GetOrdersByStatus(ctx, database.GetOrdersByStatusParams{
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
	results, err := r.queries.GetOrderCountByStatus(ctx)
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