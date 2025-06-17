// internal/shared/interfaces/cart.go
package interfaces

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// Cart Domain Types
// =============================================================================

// Cart represents the shopping cart entity
type Cart struct {
	ID         int32       `json:"id"`
	CustomerID pgtype.Int4 `json:"customer_id"`
	SessionID  pgtype.Text `json:"session_id"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// CartItem represents a single item in the cart (now using product_variant_id)
type CartItem struct {
	ID                   int32       `json:"id"`
	CartID               int32       `json:"cart_id"`
	ProductVariantID     int32       `json:"product_variant_id"`
	Quantity             int32       `json:"quantity"`
	Price                int32       `json:"price"`
	PurchaseType         string      `json:"purchase_type"`
	SubscriptionInterval pgtype.Text `json:"subscription_interval"`
	StripePriceID        string      `json:"stripe_price_id"`
	CreatedAt            time.Time   `json:"created_at"`
}

// CartItemWithVariant includes variant and product information
type CartItemWithVariant struct {
	ID                   int32       `json:"id"`
	CartID               int32       `json:"cart_id"`
	ProductVariantID     int32       `json:"product_variant_id"`
	Quantity             int32       `json:"quantity"`
	Price                int32       `json:"price"`
	PurchaseType         string      `json:"purchase_type"`
	SubscriptionInterval pgtype.Text `json:"subscription_interval"`
	StripePriceID        string      `json:"stripe_price_id"`
	CreatedAt            time.Time   `json:"created_at"`
	
	// Variant information
	VariantName    string      `json:"variant_name"`
	VariantStock   int32       `json:"variant_stock"`
	VariantActive  bool        `json:"variant_active"`
	OptionsDisplay pgtype.Text `json:"options_display"`
	
	// Product information
	ProductID          int32       `json:"product_id"`
	ProductName        string      `json:"product_name"`
	ProductDescription pgtype.Text `json:"product_description"`
	ProductActive      bool        `json:"product_active"`
	
	// For validation/error reporting
	IssueType          string      `json:"issue_type,omitempty"`
}

// CartItemWithOptions includes detailed option information
type CartItemWithOptions struct {
	CartItemWithVariant
	VariantOptions string `json:"variant_options"` // JSON string of option details
}

// CartWithItems represents a cart with all its items and totals
type CartWithItems struct {
	ID         int32                   `json:"id"`
	CustomerID *int32                  `json:"customer_id"`
	SessionID  *string                 `json:"session_id"`
	Items      []CartItemWithVariant   `json:"items"`
	Total      int32                   `json:"total"`
	ItemCount  int32                   `json:"item_count"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  time.Time               `json:"updated_at"`
}

// CartSummary provides cart totals and summary information
type CartSummary struct {
	CartID                int32                       `json:"cart_id"`
	Total                 int32                       `json:"total"`
	ItemCount             int32                       `json:"item_count"`
	OneTimeTotal          int32                       `json:"one_time_total"`
	SubscriptionSummaries []CartSubscriptionSummary   `json:"subscription_summaries"`
}

// CartSubscriptionSummary groups subscription items by interval
type CartSubscriptionSummary struct {
	SubscriptionInterval pgtype.Text `json:"subscription_interval"`
	ItemCount            int64       `json:"item_count"`
	TotalQuantity        int64       `json:"total_quantity"`
	TotalAmount          int64       `json:"total_amount"`
}

// CartItemValidation represents validation status of cart items
type CartItemValidation struct {
	CartItemID        int32  `json:"cart_item_id"`
	ProductVariantID  int32  `json:"product_variant_id"`
	RequestedQuantity int32  `json:"requested_quantity"`
	AvailableStock    int32  `json:"available_stock"`
	VariantActive     bool   `json:"variant_active"`
	ProductActive     bool   `json:"product_active"`
	ValidationStatus  string `json:"validation_status"`
}

// VariantAvailability represents variant availability check results
type VariantAvailability struct {
	VariantID     int32 `json:"variant_id"`
	Stock         int32 `json:"stock"`
	Active        bool  `json:"active"`
	ProductActive bool  `json:"product_active"`
	IsAvailable   bool  `json:"is_available"`
}

// =============================================================================
// Request/Response Types for API
// =============================================================================

// AddCartItemRequest represents the request to add an item to cart
type AddCartItemRequest struct {
	ProductVariantID     int32   `json:"product_variant_id" validate:"required,min=1"`
	Quantity             int32   `json:"quantity" validate:"required,min=1,max=100"`
	PurchaseType         string  `json:"purchase_type" validate:"required,oneof=one_time subscription"`
	SubscriptionInterval *string `json:"subscription_interval,omitempty" validate:"omitempty,oneof=14_day 21_day 30_day 60_day"`
}

// UpdateCartItemRequest represents the request to update cart item quantity
type UpdateCartItemRequest struct {
	Quantity int32 `json:"quantity" validate:"required,min=1,max=100"`
}

// CartResponse represents cart data in API responses
type CartResponse struct {
	ID         int32                      `json:"id"`
	CustomerID *int32                     `json:"customer_id"`
	SessionID  *string                    `json:"session_id"`
	Items      []CartItemResponse         `json:"items"`
	Total      int32                      `json:"total"`
	ItemCount  int32                      `json:"item_count"`
	CreatedAt  time.Time                  `json:"created_at"`
	UpdatedAt  time.Time                  `json:"updated_at"`
}

// CartItemResponse represents cart item data in API responses
type CartItemResponse struct {
	ID                   int32                  `json:"id"`
	ProductVariantID     int32                  `json:"product_variant_id"`
	Quantity             int32                  `json:"quantity"`
	Price                int32                  `json:"price"`
	PurchaseType         string                 `json:"purchase_type"`
	SubscriptionInterval *string                `json:"subscription_interval"`
	CreatedAt            time.Time              `json:"created_at"`
	
	// Product and variant details
	ProductID          int32                   `json:"product_id"`
	ProductName        string                  `json:"product_name"`
	ProductDescription *string                 `json:"product_description"`
	VariantName        string                  `json:"variant_name"`
	VariantOptions     interface{}             `json:"variant_options,omitempty"`
	
	// Display helpers
	LineTotal          int32                   `json:"line_total"`
	PriceDisplay       string                  `json:"price_display"`
	LineTotalDisplay   string                  `json:"line_total_display"`
}

// CartSummaryResponse represents cart summary in API responses
type CartSummaryResponse struct {
	CartID                int32                           `json:"cart_id"`
	Total                 int32                           `json:"total"`
	ItemCount             int32                           `json:"item_count"`
	OneTimeTotal          int32                           `json:"one_time_total"`
	SubscriptionTotal     int32                           `json:"subscription_total"`
	SubscriptionBreakdown []SubscriptionBreakdownResponse `json:"subscription_breakdown"`
	
	// Display helpers
	TotalDisplay            string `json:"total_display"`
	OneTimeTotalDisplay     string `json:"one_time_total_display"`
	SubscriptionTotalDisplay string `json:"subscription_total_display"`
}

// SubscriptionBreakdownResponse shows subscription totals by interval
type SubscriptionBreakdownResponse struct {
	Interval     string `json:"interval"`
	ItemCount    int64  `json:"item_count"`
	TotalAmount  int64  `json:"total_amount"`
	AmountDisplay string `json:"amount_display"`
}

// =============================================================================
// Repository Interfaces
// =============================================================================

type CartRepository interface {
	// Cart operations
	GetByID(ctx context.Context, id int32) (*Cart, error)
	GetByCustomerID(ctx context.Context, customerID int32) (*Cart, error)
	GetBySessionID(ctx context.Context, sessionID string) (*Cart, error)
	Create(ctx context.Context, customerID *int32, sessionID *string) (*Cart, error)
	UpdateTimestamp(ctx context.Context, cartID int32) (*Cart, error)
	Delete(ctx context.Context, id int32) error
	Clear(ctx context.Context, cartID int32) error

	// Cart item operations (now using product_variant_id)
	GetCartItems(ctx context.Context, cartID int32) ([]CartItemWithVariant, error)
	GetCartItemsWithOptions(ctx context.Context, cartID int32) ([]CartItemWithOptions, error)
	GetCartItem(ctx context.Context, itemID int32) (*CartItem, error)
	GetCartItemsByVariant(ctx context.Context, cartID int32, productVariantID int32) ([]CartItem, error)
	AddItem(ctx context.Context, cartID int32, productVariantID int32, quantity int32, price int32, purchaseType string, subscriptionInterval *string, stripePriceID string) (*CartItem, error)
	UpdateItem(ctx context.Context, itemID int32, quantity int32, price int32, stripePriceID string) (*CartItem, error)
	UpdateItemQuantity(ctx context.Context, itemID int32, quantity int32) (*CartItem, error)
	IncrementItemQuantity(ctx context.Context, itemID int32, delta int32) (*CartItem, error)
	RemoveItem(ctx context.Context, itemID int32) error
	RemoveItemByVariantID(ctx context.Context, cartID int32, productVariantID int32) error
	RemoveItemByVariantAndType(ctx context.Context, cartID int32, productVariantID int32, purchaseType string, subscriptionInterval *string) error

	// Cart totals and analytics
	GetCartTotal(ctx context.Context, cartID int32) (int32, error)
	GetCartItemCount(ctx context.Context, cartID int32) (int32, error)
	GetCartTotalByPurchaseType(ctx context.Context, cartID int32, purchaseType string) (int32, error)
	GetCartSubscriptionSummary(ctx context.Context, cartID int32) ([]CartSubscriptionSummary, error)

	// Cart validation and maintenance
	ValidateCartItems(ctx context.Context, cartID int32) ([]CartItemValidation, error)
	GetInvalidCartItems(ctx context.Context, cartID int32) ([]CartItemWithVariant, error)
	CheckVariantAvailability(ctx context.Context, productVariantID int32, requestedQuantity int32) (*VariantAvailability, error)
	RemoveUnavailableItems(ctx context.Context, cartID int32) error
	UpdateCartItemPrices(ctx context.Context, cartID int32) error
}

// =============================================================================
// Service Interfaces
// =============================================================================

type CartService interface {
	// Cart operations
	GetOrCreateCart(ctx context.Context, customerID *int32, sessionID *string) (*CartWithItems, error)
	GetCart(ctx context.Context, cartID int32) (*CartWithItems, error)
	GetCartItems(ctx context.Context, cartID int32) ([]CartItemWithVariant, error)
	GetCartItemsWithOptions(ctx context.Context, cartID int32) ([]CartItemWithOptions, error)

	// Cart item management (now using variants)
	AddItem(ctx context.Context, cartID int32, productVariantID int32, quantity int32, purchaseType string, subscriptionInterval *string) (*CartItem, error)
	UpdateItemQuantity(ctx context.Context, itemID int32, quantity int32) (*CartItem, error)
	RemoveItem(ctx context.Context, itemID int32) error
	RemoveItemByVariantID(ctx context.Context, cartID int32, productVariantID int32) error
	Clear(ctx context.Context, cartID int32) error

	// Cart validation and maintenance
	ValidateCartForCheckout(ctx context.Context, cartID int32) (*CartWithItems, error)
	CleanupInvalidItems(ctx context.Context, cartID int32) ([]CartItemWithVariant, error)

	// Cart summary
	GetCartSummary(ctx context.Context, cartID int32) (*CartSummary, error)
}

// =============================================================================
// Helper Functions
// =============================================================================

// ToCartResponse converts CartWithItems to API response format
func (c *CartWithItems) ToCartResponse() *CartResponse {
	resp := &CartResponse{
		ID:         c.ID,
		CustomerID: c.CustomerID,
		SessionID:  c.SessionID,
		Total:      c.Total,
		ItemCount:  c.ItemCount,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}

	// Convert items
	for _, item := range c.Items {
		resp.Items = append(resp.Items, *item.ToCartItemResponse())
	}

	return resp
}

// ToCartItemResponse converts CartItemWithVariant to API response format
func (c *CartItemWithVariant) ToCartItemResponse() *CartItemResponse {
	resp := &CartItemResponse{
		ID:               c.ID,
		ProductVariantID: c.ProductVariantID,
		Quantity:         c.Quantity,
		Price:            c.Price,
		PurchaseType:     c.PurchaseType,
		CreatedAt:        c.CreatedAt,
		ProductID:        c.ProductID,
		ProductName:      c.ProductName,
		VariantName:      c.VariantName,
		LineTotal:        c.Quantity * c.Price,
		PriceDisplay:     formatCurrency(c.Price),
		LineTotalDisplay: formatCurrency(c.Quantity * c.Price),
	}

	// Handle optional fields
	if c.SubscriptionInterval.Valid {
		resp.SubscriptionInterval = &c.SubscriptionInterval.String
	}

	if c.ProductDescription.Valid {
		resp.ProductDescription = &c.ProductDescription.String
	}

	if c.OptionsDisplay.Valid && len(c.OptionsDisplay.String) > 0 {
		// This could be parsed JSON, but for now include as string
		resp.VariantOptions = c.OptionsDisplay.String
	}

	return resp
}

// ToCartSummaryResponse converts CartSummary to API response format
func (c *CartSummary) ToCartSummaryResponse() *CartSummaryResponse {
	resp := &CartSummaryResponse{
		CartID:                   c.CartID,
		Total:                    c.Total,
		ItemCount:                c.ItemCount,
		OneTimeTotal:            c.OneTimeTotal,
		TotalDisplay:            formatCurrency(c.Total),
		OneTimeTotalDisplay:     formatCurrency(c.OneTimeTotal),
	}

	// Calculate subscription total and breakdown
	var subscriptionTotal int32
	for _, sub := range c.SubscriptionSummaries {
		amount := int32(sub.TotalAmount)
		subscriptionTotal += amount
		
		resp.SubscriptionBreakdown = append(resp.SubscriptionBreakdown, SubscriptionBreakdownResponse{
			Interval:      sub.SubscriptionInterval.String,
			ItemCount:     sub.ItemCount,
			TotalAmount:   sub.TotalAmount,
			AmountDisplay: formatCurrency(amount),
		})
	}

	resp.SubscriptionTotal = subscriptionTotal
	resp.SubscriptionTotalDisplay = formatCurrency(subscriptionTotal)

	return resp
}