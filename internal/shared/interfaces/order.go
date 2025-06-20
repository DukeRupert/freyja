// internal/shared/interfaces/order.go
package interfaces

import (
	"context"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// Order Domain Types
// =============================================================================

// Order represents the main order entity
type Order struct {
	ID                    int32                `json:"id"`
	CustomerID            int32                `json:"customer_id"`
	Status                database.OrderStatus `json:"status"`
	Total                 int32                `json:"total"`
	StripeSessionID       pgtype.Text          `json:"stripe_session_id"`
	StripePaymentIntentID pgtype.Text          `json:"stripe_payment_intent_id"`
	StripeChargeID        pgtype.Text          `json:"stripe_charge_id"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
}

// OrderItem represents an item within an order (now using product_variant_id)
type OrderItem struct {
	ID                   int32       `json:"id"`
	OrderID              int32       `json:"order_id"`
	ProductVariantID     int32       `json:"product_variant_id"`
	Name                 string      `json:"name"`
	VariantName          string      `json:"variant_name"`
	Quantity             int32       `json:"quantity"`
	Price                int32       `json:"price"`
	PurchaseType         string      `json:"purchase_type"`
	SubscriptionInterval pgtype.Text `json:"subscription_interval"`
	StripePriceID        string      `json:"stripe_price_id"`
	CreatedAt            time.Time   `json:"created_at"`
}

// OrderItemWithVariant includes variant and product information
type OrderItemWithVariant struct {
	ID                   int32       `json:"id"`
	OrderID              int32       `json:"order_id"`
	ProductVariantID     int32       `json:"product_variant_id"`
	Name                 string      `json:"name"`
	Description          string      `json:"description"`
	Quantity             int32       `json:"quantity"`
	Price                int32       `json:"price"`
	PurchaseType         string      `json:"purchase_type"`
	SubscriptionInterval pgtype.Text `json:"subscription_interval"`
	StripePriceID        string      `json:"stripe_price_id"`
	CreatedAt            time.Time   `json:"created_at"`

	// Variant information (current at time of viewing)
	VariantStock   int32       `json:"variant_stock"`
	VariantActive  bool        `json:"variant_active"`
	OptionsDisplay pgtype.Text `json:"options_display"`

	// Product information
	ProductID          int32       `json:"product_id"`
	ProductName        string      `json:"product_name"`
	ProductDescription pgtype.Text `json:"product_description"`
	ProductActive      bool        `json:"product_active"`
}

// OrderWithItems represents an order with all its items
type OrderWithItems struct {
	ID                    int32                  `json:"id"`
	CustomerID            int32                  `json:"customer_id"`
	Status                database.OrderStatus   `json:"status"`
	Total                 int32                  `json:"total"`
	StripeSessionID       *string                `json:"stripe_session_id"`
	StripePaymentIntentID *string                `json:"stripe_payment_intent_id"`
	StripeChargeID        *string                `json:"stripe_charge_id"`
	Items                 []OrderItemWithVariant `json:"items"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// CreateOrderRequest represents the data needed to create an order
type CreateOrderRequest struct {
	CustomerID            int32                    `json:"customer_id" validate:"required,min=1"`
	Status                database.OrderStatus     `json:"status" validate:"required"`
	Total                 int32                    `json:"total" validate:"required,min=1"`
	StripeSessionID       *string                  `json:"stripe_session_id,omitempty"`
	StripePaymentIntentID *string                  `json:"stripe_payment_intent_id,omitempty"`
	Items                 []CreateOrderItemRequest `json:"items" validate:"required,min=1"`
}

// CreateOrderItemRequest represents the data needed to create an order item
type CreateOrderItemRequest struct {
	ProductVariantID     int32   `json:"product_variant_id" validate:"required,min=1"`
	Name                 string  `json:"name" validate:"required,min=1"`
	VariantName          string  `json:"variant_name"`
	Quantity             int32   `json:"quantity" validate:"required,min=1"`
	Price                int32   `json:"price" validate:"required,min=1"`
	PurchaseType         string  `json:"purchase_type" validate:"required,oneof=one_time subscription"`
	SubscriptionInterval *string `json:"subscription_interval,omitempty"`
	StripePriceID        string  `json:"stripe_price_id" validate:"required"`
}

// OrderFilters represents filtering options for order queries
type OrderFilters struct {
	CustomerID *int32     `json:"customer_id,omitempty"`
	Status     *string    `json:"status,omitempty"`
	DateFrom   *time.Time `json:"date_from,omitempty"`
	DateTo     *time.Time `json:"date_to,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// OrderSummary provides aggregated order information
type OrderSummary struct {
	OrderID              int32 `json:"order_id"`
	ItemCount            int32 `json:"item_count"`
	Total                int32 `json:"total"`
	OneTimeTotal         int32 `json:"one_time_total"`
	SubscriptionTotal    int32 `json:"subscription_total"`
	UniqueVariants       int32 `json:"unique_variants"`
	HasSubscriptionItems bool  `json:"has_subscription_items"`
}

// =============================================================================
// API Response Types
// =============================================================================

// OrderResponse represents order data in API responses
type OrderResponse struct {
	ID                    int32               `json:"id"`
	CustomerID            int32               `json:"customer_id"`
	Status                string              `json:"status"`
	Total                 int32               `json:"total"`
	StripeSessionID       *string             `json:"stripe_session_id"`
	StripePaymentIntentID *string             `json:"stripe_payment_intent_id"`
	StripeChargeID        *string             `json:"stripe_charge_id"`
	Items                 []OrderItemResponse `json:"items"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`

	// Display helpers
	TotalDisplay string `json:"total_display"`
	ItemCount    int32  `json:"item_count"`
}

// OrderItemResponse represents order item data in API responses
type OrderItemResponse struct {
	ID                   int32     `json:"id"`
	ProductVariantID     int32     `json:"product_variant_id"`
	Name                 string    `json:"name"`
	Quantity             int32     `json:"quantity"`
	Price                int32     `json:"price"`
	PurchaseType         string    `json:"purchase_type"`
	SubscriptionInterval *string   `json:"subscription_interval"`
	CreatedAt            time.Time `json:"created_at"`

	// Product and variant details
	ProductID          int32       `json:"product_id"`
	ProductName        string      `json:"product_name"`
	ProductDescription *string     `json:"product_description"`
	VariantOptions     interface{} `json:"variant_options,omitempty"`

	// Display helpers
	LineTotal        int32  `json:"line_total"`
	PriceDisplay     string `json:"price_display"`
	LineTotalDisplay string `json:"line_total_display"`
}

// OrderListResponse represents paginated order list
type OrderListResponse struct {
	Orders []OrderResponse `json:"orders"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// AdminOrderStats provides comprehensive statistics for admin dashboard
type AdminOrderStats struct {
	TotalOrders       int                `json:"total_orders"`
	TotalRevenue      int32              `json:"total_revenue"`
	AverageOrderValue int32              `json:"average_order_value"`
	OrdersByStatus    map[string]int     `json:"orders_by_status"`
	GeneratedAt       time.Time          `json:"generated_at"`
}

// =============================================================================
// Repository Interfaces
// =============================================================================

type OrderRepository interface {
	// Order operations
	GetByID(ctx context.Context, id int32) (*Order, error)
	GetByStripePaymentIntentID(ctx context.Context, paymentIntentID string) (*Order, error)
	GetByStripeChargeID(ctx context.Context, chargeID string) (*Order, error)
	Create(ctx context.Context, order *Order) error
	UpdateStatus(ctx context.Context, id int32, status string) error
	UpdateStripeChargeID(ctx context.Context, id int32, chargeID string) error

	// Order queries
	GetAll(ctx context.Context, filters OrderFilters) ([]Order, error)
	GetByCustomerID(ctx context.Context, customerID int32, filters OrderFilters) ([]Order, error)
	GetByStatus(ctx context.Context, status string, limit, offset int32) ([]Order, error)

	// Order with items operations
	GetWithItems(ctx context.Context, id int32) (*OrderWithItems, error)
	GetOrdersWithItems(ctx context.Context, customerID int32, filters OrderFilters) ([]OrderWithItems, error)

	// Order item operations (now using product_variant_id)
	CreateOrderItems(ctx context.Context, orderID int32, items []OrderItem) error
	GetOrderItems(ctx context.Context, orderID int32) ([]OrderItemWithVariant, error)
	GetOrderItemsWithOptions(ctx context.Context, orderID int32) ([]OrderItemWithVariant, error)

	// Analytics and reporting
	GetOrderSummary(ctx context.Context, orderID int32) (*OrderSummary, error)
	GetOrderCountByStatus(ctx context.Context) (map[string]int, error)
}

// =============================================================================
// Service Interfaces
// =============================================================================

type OrderService interface {
	// Order creation
	CreateOrderFromCart(ctx context.Context, customerID int32, cartID int32) (*OrderWithItems, error)
	CreateOrderFromPayment(ctx context.Context, customerID int32, paymentIntentID string, amount int32) (*OrderWithItems, error)
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderWithItems, error)

	// Order retrieval
	GetByID(ctx context.Context, orderID int32) (*OrderWithItems, error)
	GetByCustomer(ctx context.Context, customerID int32, filters OrderFilters) ([]OrderWithItems, error)
	GetAll(ctx context.Context, filters OrderFilters) ([]OrderWithItems, error)

	// Order management
	UpdateStatus(ctx context.Context, orderID int32, status database.OrderStatus) error
	CancelOrder(ctx context.Context, orderID int32) error
	UpdateStripeChargeID(ctx context.Context, orderID int32, chargeID string) error

	// Analytics
	GetOrderSummary(ctx context.Context, orderID int32) (*OrderSummary, error)
	GetAdminStats(ctx context.Context) (*AdminOrderStats, error)
}

// =============================================================================
// Helper Functions
// =============================================================================

// ToOrderResponse converts OrderWithItems to API response format
func (o *OrderWithItems) ToOrderResponse() *OrderResponse {
	resp := &OrderResponse{
		ID:                    o.ID,
		CustomerID:            o.CustomerID,
		Status:                string(o.Status),
		Total:                 o.Total,
		StripeSessionID:       o.StripeSessionID,
		StripePaymentIntentID: o.StripePaymentIntentID,
		StripeChargeID:        o.StripeChargeID,
		CreatedAt:             o.CreatedAt,
		UpdatedAt:             o.UpdatedAt,
		TotalDisplay:          formatCurrency(o.Total),
		ItemCount:             int32(len(o.Items)),
	}

	// Convert items
	for _, item := range o.Items {
		resp.Items = append(resp.Items, *item.ToOrderItemResponse())
	}

	return resp
}

// ToOrderItemResponse converts OrderItemWithVariant to API response format
func (oi *OrderItemWithVariant) ToOrderItemResponse() *OrderItemResponse {
	resp := &OrderItemResponse{
		ID:               oi.ID,
		ProductVariantID: oi.ProductVariantID,
		Name:             oi.Name,
		Quantity:         oi.Quantity,
		Price:            oi.Price,
		PurchaseType:     oi.PurchaseType,
		CreatedAt:        oi.CreatedAt,
		ProductID:        oi.ProductID,
		ProductName:      oi.ProductName,
		LineTotal:        oi.Quantity * oi.Price,
		PriceDisplay:     formatCurrency(oi.Price),
		LineTotalDisplay: formatCurrency(oi.Quantity * oi.Price),
	}

	// Handle optional fields
	if oi.SubscriptionInterval.Valid {
		resp.SubscriptionInterval = &oi.SubscriptionInterval.String
	}

	if oi.ProductDescription.Valid {
		resp.ProductDescription = &oi.ProductDescription.String
	}

	if oi.OptionsDisplay.Valid && len(oi.OptionsDisplay.String) > 0 {
		resp.VariantOptions = oi.OptionsDisplay.String
	}

	return resp
}

func IsValidOrderStatus(status database.OrderStatus) bool {
	switch status {
	case database.OrderStatusPending,
		database.OrderStatusPaymentProcessing,
		database.OrderStatusConfirmed,
		database.OrderStatusProcessing,
		database.OrderStatusShipped,
		database.OrderStatusDelivered,
		database.OrderStatusCancelled,
		database.OrderStatusRefunded:
		return true
	default:
		return false
	}
}

func IsValidOrderStatusString(s string) bool {
    status := database.OrderStatus(s)
    return IsValidOrderStatus(status)
}
