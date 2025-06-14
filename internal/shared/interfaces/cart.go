// internal/interfaces/cart.go
package interfaces

import (
	"context"
	"time"

	"github.com/dukerupert/freyja/internal/database"
)

// Use database types directly for MVP simplicity
type Cart = database.Carts
type CartItem = database.CartItems
type CartItemWithProduct = database.GetCartItemsRow

// Cart with items and totals for API responses
type CartWithItems struct {
	ID         int32                 `json:"id"`
	CustomerID *int32                `json:"customer_id,omitempty"`
	SessionID  *string               `json:"session_id,omitempty"`
	Items      []CartItemWithProduct `json:"items"`
	Total      int32                 `json:"total"`
	ItemCount  int32                 `json:"item_count"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

// Request/Response types
type AddCartItemRequest struct {
	ProductID            int32   `json:"product_id" validate:"required,min=1"`
	Quantity             int32   `json:"quantity" validate:"required,min=1,max=100"`
	PurchaseType         string  `json:"purchase_type" validate:"required,oneof=one_time subscription"`
	SubscriptionInterval *string `json:"subscription_interval,omitempty" validate:"omitempty,oneof=14_day 21_day 30_day 60_day"`
}

type UpdateCartItemRequest struct {
	Quantity int32 `json:"quantity" validate:"required,min=1,max=100"`
}

type CartRepository interface {
	// Cart operations
	GetByID(ctx context.Context, id int32) (*Cart, error)
	GetByCustomerID(ctx context.Context, customerID int32) (*Cart, error)
	GetBySessionID(ctx context.Context, sessionID string) (*Cart, error)
	Create(ctx context.Context, customerID *int32, sessionID *string) (*Cart, error)
	UpdateTimestamp(ctx context.Context, cartID int32) (*Cart, error)
	Delete(ctx context.Context, id int32) error
	Clear(ctx context.Context, cartID int32) error

	// Cart item operations
	GetCartItems(ctx context.Context, cartID int32) ([]CartItemWithProduct, error)
	GetCartItem(ctx context.Context, itemID int32) (*CartItem, error)
	GetCartItemByProductID(ctx context.Context, cartID int32, productID int32) (*CartItem, error)
	GetCartItemsByProduct(ctx context.Context, cartID int32, productID int32) ([]CartItem, error)
	GetCartItemByProductAndType(ctx context.Context, cartID int32, productID int32, purchaseType string, subscriptionInterval *string) (*CartItem, error)
	AddItem(ctx context.Context, cartID int32, productID int32, quantity int32, price int32, purchaseType string, subscriptionInterval *string, stripePriceID string) (*CartItem, error)
	UpdateItem(ctx context.Context, itemID int32, quantity int32, price int32, stripePriceID string) (*CartItem, error)
	UpdateItemQuantity(ctx context.Context, itemID int32, quantity int32) (*CartItem, error)
	RemoveItem(ctx context.Context, itemID int32) error
	RemoveItemByProductID(ctx context.Context, cartID int32, productID int32) error

	// Cart totals
	GetCartTotal(ctx context.Context, cartID int32) (int32, error)
	GetCartItemCount(ctx context.Context, cartID int32) (int32, error)
}

type CartService interface {
	// Cart operations
	// GetByID(ctx context.Context, id int32) (*Cart, error)
	// GetByCustomerID(ctx context.Context, customerID int32) (*Cart, error)
	// GetBySessionID(ctx context.Context, sessionID string) (*Cart, error)
	// Create(ctx context.Context, customerID *int32, sessionID *string) (*Cart, error)
	GetOrCreateCart(ctx context.Context, customerID *int32, sessionID *string) (*CartWithItems, error)
	// UpdateTimestamp(ctx context.Context, cartID int32) (*Cart, error)
	// Delete(ctx context.Context, id int32) error
	Clear(ctx context.Context, cartID int32) error
	
	// Cart item operations
	GetCartItems(ctx context.Context, cartID int32) ([]CartItemWithProduct, error)
	// GetCartItem(ctx context.Context, itemID int32) (*CartItem, error)
	// GetCartItemByProductID(ctx context.Context, cartID int32, productID int32) (*CartItem, error)
	// GetCartItemsByProduct(ctx context.Context, cartID int32, productID int32) ([]CartItem, error)
	// GetCartItemByProductAndType(ctx context.Context, cartID int32, productID int32, purchaseType string, subscriptionInterval *string) (*CartItem, error)
	AddItem(ctx context.Context, cartID int32, productID int32, quantity int32, purchaseType string, subscriptionInterval *string) (*CartItem, error)
	// UpdateItem(ctx context.Context, itemID int32, quantity int32, price int32, stripePriceID string) (*CartItem, error)
	UpdateItemQuantity(ctx context.Context, itemID int32, quantity int32) (*CartItem, error)
	RemoveItem(ctx context.Context, itemID int32) error
	RemoveItemByProductID(ctx context.Context, cartID int32, productID int32) error

	// Cart totals and calculations
	// GetCartTotal(ctx context.Context, cartID int32) (int32, error)
	// GetCartItemCount(ctx context.Context, cartID int32) (int32, error)
	
	// Business logic methods
	// ConvertGuestCart(ctx context.Context, sessionID string, customerID int32) error
	ValidateCartForCheckout(ctx context.Context, cartID int32) (*CartWithItems, error)
}
