package domain

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// CART DOMAIN ERRORS
// =============================================================================

var (
	ErrCartNotFound     = &Error{Code: ENOTFOUND, Message: "Cart not found"}
	ErrCartItemNotFound = &Error{Code: ENOTFOUND, Message: "Cart item not found"}
	ErrSessionNotFound  = &Error{Code: ENOTFOUND, Message: "Session not found"}
	ErrInvalidQuantity  = &Error{Code: EINVALID, Message: "Quantity must be greater than 0"}
)

// CartService provides business logic for shopping cart operations.
// Implementations should be tenant-scoped.
type CartService interface {
	// GetOrCreateCart retrieves an existing cart or creates a new session and cart.
	// Returns the cart, session ID (new or existing), and any error.
	GetOrCreateCart(ctx context.Context, sessionID string) (*Cart, string, error)

	// GetCart retrieves an existing cart by session ID.
	GetCart(ctx context.Context, sessionID string) (*Cart, error)

	// AddItem adds a product SKU to the cart or updates quantity if already present.
	AddItem(ctx context.Context, cartID string, skuID string, quantity int) (*CartSummary, error)

	// UpdateItemQuantity updates the quantity of a cart item.
	// If quantity is 0, the item is removed.
	UpdateItemQuantity(ctx context.Context, cartID string, skuID string, quantity int) (*CartSummary, error)

	// RemoveItem removes a product SKU from the cart.
	RemoveItem(ctx context.Context, cartID string, skuID string) (*CartSummary, error)

	// GetCartSummary retrieves a cart with all items and calculated totals.
	GetCartSummary(ctx context.Context, cartID string) (*CartSummary, error)

	// ClearCart removes all items from a cart.
	ClearCart(ctx context.Context, cartID string) error
}

// Cart represents a lightweight cart view model.
type Cart struct {
	ID        pgtype.UUID
	TenantID  pgtype.UUID
	SessionID pgtype.UUID
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

// CartSummary aggregates cart information with items and calculated totals.
type CartSummary struct {
	Cart      Cart
	Items     []CartItem
	Subtotal  int32
	ItemCount int
}

// CartItem represents a cart line item with product details and calculated totals.
type CartItem struct {
	ID             pgtype.UUID
	SKUID          pgtype.UUID
	ProductName    string
	SKU            string
	WeightValue    string
	Grind          string
	Quantity       int32
	UnitPriceCents int32
	LineSubtotal   int32
	ImageURL       string
}
