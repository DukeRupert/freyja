// internal/interfaces/order.go
package interfaces

import (
	"context"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/database"
)

type Order = database.Orders
type OrderItem = database.OrderItems

type OrderRepository interface {
	// Basic CRUD operations
	GetByID(ctx context.Context, id int32) (*Order, error)
	GetWithItems(ctx context.Context, id int32) (*OrderWithItems, error)
	Create(ctx context.Context, order *Order) error
	Update(ctx context.Context, order *Order) error
	UpdateStatus(ctx context.Context, id int32, status database.OrderStatus) error
	Delete(ctx context.Context, id int32) error
	
	// Query operations
	GetByCustomerID(ctx context.Context, customerID int32, filters OrderFilters) ([]Order, error)
	GetAll(ctx context.Context, filters OrderFilters) ([]Order, error)
	GetByStripeSessionID(ctx context.Context, sessionID string) (*Order, error)
	GetByStripePaymentIntentID(ctx context.Context, paymentIntentID string) (*Order, error)
	
	// Order items operations
	GetOrderItems(ctx context.Context, orderID int32) ([]OrderItem, error)
	CreateOrderItems(ctx context.Context, orderID int32, items []OrderItem) error
	UpdateStripeChargeID(ctx context.Context, orderID int32, chargeID string) error
	
	// Statistics and reporting
	GetOrderCount(ctx context.Context, filters OrderFilters) (int64, error)
	GetOrderCountByStatus(ctx context.Context) (map[string]int64, error)
	GetTotalRevenue(ctx context.Context, filters OrderFilters) (int64, error)
	GetOrdersByDateRange(ctx context.Context, from, to time.Time) ([]Order, error)
}

type OrderWithItems struct {
	ID                    int32       `json:"id"`
	CustomerID            int32       `json:"customer_id"`
	Status                string      `json:"status"`
	Total                 int32       `json:"total"`
	StripeSessionID       *string     `json:"stripe_session_id,omitempty"`
	StripePaymentIntentID *string     `json:"stripe_payment_intent_id,omitempty"`
	Items                 []OrderItem `json:"items"`
	CreatedAt             time.Time   `json:"created_at"`
	UpdatedAt             time.Time   `json:"updated_at"`
}

type CreateOrderRequest struct {
	CustomerID int32                  `json:"customer_id" validate:"required"`
	Items      []CreateOrderItemRequest `json:"items" validate:"required,min=1"`
	Total      int32                  `json:"total" validate:"required,min=1"`
}

type CreateOrderItemRequest struct {
	ProductID int32  `json:"product_id" validate:"required"`
	Name      string `json:"name" validate:"required"`
	Quantity  int32  `json:"quantity" validate:"required,min=1"`
	Price     int32  `json:"price" validate:"required,min=1"`
}

type UpdateOrderStatusRequest struct {
	Status        string  `json:"status" validate:"required"`
	TrackingNumber *string `json:"tracking_number,omitempty"`
	Notes         *string `json:"notes,omitempty"`
}

type OrderFilters struct {
	Status     *string `json:"status,omitempty"`
	CustomerID *int32  `json:"customer_id,omitempty"`
	DateFrom   *time.Time `json:"date_from,omitempty"`
	DateTo     *time.Time `json:"date_to,omitempty"`
	Limit      int     `json:"limit,omitempty"`
	Offset     int     `json:"offset,omitempty"`
}

type OrderService interface {
	// Order creation
	CreateOrderFromCart(ctx context.Context, customerID int32, cartID int32) (*Order, error)
	CreateOrderFromPayment(ctx context.Context, customerID int32, paymentIntentID string, amount int32) (*Order, error)
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error)
	
	// Order retrieval
	GetByID(ctx context.Context, orderID int32) (*OrderWithItems, error)
	GetByCustomer(ctx context.Context, customerID int32, filters OrderFilters) ([]OrderWithItems, error)
	GetAll(ctx context.Context, filters OrderFilters) ([]OrderWithItems, error)
	
	// Order management
	UpdateOrderStatus(ctx context.Context, orderID int32, req UpdateOrderStatusRequest) error
	UpdateStripeChargeID(ctx context.Context, orderID int32, chargeID string) error
	CancelOrder(ctx context.Context, orderID int32, reason string) error
	RefundOrder(ctx context.Context, orderID int32, amount *int32, reason string) error
	
	// Order validation
	ValidateOrderForPayment(ctx context.Context, orderID int32) error
	ValidateOrderForShipping(ctx context.Context, orderID int32) error
	
	// Order statistics
	GetOrderStats(ctx context.Context, filters OrderFilters) (*OrderStats, error)
}

// =============================================================================
// Statistics Types
// =============================================================================

type OrderStats struct {
	TotalOrders    int64   `json:"total_orders"`
	TotalRevenue   int64   `json:"total_revenue"`
	AverageOrder   float64 `json:"average_order_value"`
	OrdersByStatus map[string]int64 `json:"orders_by_status"`
	RevenueByDay   []DailyRevenue   `json:"revenue_by_day,omitempty"`
}

type DailyRevenue struct {
	Date    time.Time `json:"date"`
	Revenue int64     `json:"revenue"`
	Orders  int64     `json:"orders"`
}

// =============================================================================
// Order Status Helpers
// =============================================================================

// GetValidOrderStatuses returns all valid order statuses
func GetValidOrderStatuses() []database.OrderStatus {
	return []database.OrderStatus{
		database.OrderStatusPending,
		database.OrderStatusPaymentProcessing,
		database.OrderStatusConfirmed,
		database.OrderStatusProcessing,
		database.OrderStatusShipped,
		database.OrderStatusDelivered,
		database.OrderStatusCancelled,
		database.OrderStatusRefunded,
	}
}

// IsValidOrderStatus checks if a status is valid
func IsValidOrderStatus(status string) bool {
	for _, validStatus := range GetValidOrderStatuses() {
		if string(validStatus) == status {
			return true
		}
	}
	return false
}

// CanTransitionTo checks if an order can transition from one status to another
func CanTransitionTo(from, to database.OrderStatus) bool {
	transitions := map[database.OrderStatus][]database.OrderStatus{
		database.OrderStatusPending: {
			database.OrderStatusPaymentProcessing,
			database.OrderStatusConfirmed,
			database.OrderStatusCancelled,
		},
		database.OrderStatusPaymentProcessing: {
			database.OrderStatusConfirmed,
			database.OrderStatusCancelled,
		},
		database.OrderStatusConfirmed: {
			database.OrderStatusProcessing,
			database.OrderStatusCancelled,
			database.OrderStatusRefunded,
		},
		database.OrderStatusProcessing: {
			database.OrderStatusShipped,
			database.OrderStatusCancelled,
		},
		database.OrderStatusShipped: {
			database.OrderStatusDelivered,
		},
		database.OrderStatusDelivered: {
			database.OrderStatusRefunded,
		},
		// Terminal states - no transitions allowed
		database.OrderStatusCancelled: {},
		database.OrderStatusRefunded:  {},
	}

	allowedTransitions, exists := transitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}
	return false
}

// IsTerminalStatus checks if a status is terminal (no further transitions)
func IsTerminalStatus(status database.OrderStatus) bool {
	return status == database.OrderStatusDelivered ||
		   status == database.OrderStatusCancelled ||
		   status == database.OrderStatusRefunded
}

// RequiresPayment checks if a status requires payment to be completed
func RequiresPayment(status database.OrderStatus) bool {
	return status == database.OrderStatusConfirmed ||
		   status == database.OrderStatusProcessing ||
		   status == database.OrderStatusShipped ||
		   status == database.OrderStatusDelivered
}

// =============================================================================
// Order Validation Helpers
// =============================================================================

// ValidateCreateOrderRequest validates a create order request
func ValidateCreateOrderRequest(req CreateOrderRequest) error {
	if req.CustomerID <= 0 {
		return fmt.Errorf("customer ID must be positive")
	}
	
	if len(req.Items) == 0 {
		return fmt.Errorf("order must have at least one item")
	}
	
	if req.Total <= 0 {
		return fmt.Errorf("order total must be positive")
	}
	
	// Validate each item
	calculatedTotal := int32(0)
	for i, item := range req.Items {
		if item.ProductID <= 0 {
			return fmt.Errorf("item %d: product ID must be positive", i)
		}
		if item.Name == "" {
			return fmt.Errorf("item %d: product name is required", i)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("item %d: quantity must be positive", i)
		}
		if item.Price <= 0 {
			return fmt.Errorf("item %d: price must be positive", i)
		}
		
		calculatedTotal += item.Price * item.Quantity
	}
	
	// Verify total matches items
	if calculatedTotal != req.Total {
		return fmt.Errorf("order total (%d) does not match sum of items (%d)", req.Total, calculatedTotal)
	}
	
	return nil
}

// ValidateUpdateStatusRequest validates an update status request
func ValidateUpdateStatusRequest(req UpdateOrderStatusRequest) error {
	if req.Status == "" {
		return fmt.Errorf("status is required")
	}
	
	if !IsValidOrderStatus(req.Status) {
		return fmt.Errorf("invalid order status: %s", req.Status)
	}
	
	return nil
}