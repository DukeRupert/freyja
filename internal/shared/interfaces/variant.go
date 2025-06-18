// internal/shared/interfaces/variant.go
package interfaces

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// Variant Domain Types
// =============================================================================

// ProductVariant represents a product variant entity
type ProductVariant struct {
	ID                   int32            `json:"id"`
	ProductID            int32            `json:"product_id"`
	Name                 string           `json:"name"`
	Price                int32            `json:"price"`
	Stock                int32            `json:"stock"`
	Active               bool             `json:"active"`
	IsSubscription       bool             `json:"is_subscription"`
	ArchivedAt           pgtype.Timestamp `json:"archived_at"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
	StripeProductID      pgtype.Text      `json:"stripe_product_id"`
	StripePriceOnetimeID pgtype.Text      `json:"stripe_price_onetime_id"`
	StripePrice14dayID   pgtype.Text      `json:"stripe_price_14day_id"`
	StripePrice21dayID   pgtype.Text      `json:"stripe_price_21day_id"`
	StripePrice30dayID   pgtype.Text      `json:"stripe_price_30day_id"`
	StripePrice60dayID   pgtype.Text      `json:"stripe_price_60day_id"`
	OptionsDisplay       pgtype.Text      `json:"options_display"`
}

// ProductVariantWithOptions includes detailed option information
type ProductVariantWithOptions struct {
	ProductVariant
	Options []VariantOption `json:"options"`
}

// VariantOption represents an option selection for a variant
type VariantOption struct {
	OptionID    int32  `json:"option_id"`
	OptionKey   string `json:"option_key"`
	ValueID     int32  `json:"value_id"`
	Value       string `json:"value"`
}

// CreateVariantRequest represents the data needed to create a variant
type CreateVariantRequest struct {
	ProductID      int32   `json:"product_id" validate:"required,min=1"`
	Name           string  `json:"name" validate:"required,min=1,max=500"`
	Price          int32   `json:"price" validate:"required,min=1"`
	Stock          int32   `json:"stock" validate:"min=0"`
	Active         bool    `json:"active"`
	IsSubscription bool    `json:"is_subscription"`
	OptionsDisplay string  `json:"options_display"`
	OptionValueIDs []int32 `json:"option_value_ids"` // For setting variant options
}

// UpdateVariantRequest represents the data for updating a variant
type UpdateVariantRequest struct {
	Name           *string `json:"name,omitempty" validate:"omitempty,min=1,max=500"`
	Price          *int32  `json:"price,omitempty" validate:"omitempty,min=1"`
	Stock          *int32  `json:"stock,omitempty" validate:"omitempty,min=0"`
	Active         *bool   `json:"active,omitempty"`
	IsSubscription *bool   `json:"is_subscription,omitempty"`
	OptionsDisplay *string `json:"options_display,omitempty"`
}

// VariantFilters represents filtering options for variant queries
type VariantFilters struct {
	ProductID *int32 `json:"product_id,omitempty"`
	Active    *bool  `json:"active,omitempty"`
	InStock   *bool  `json:"in_stock,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

// =============================================================================
// Repository Interface
// =============================================================================

type VariantRepository interface {
	// Basic variant operations
	GetByID(ctx context.Context, id int32) (*ProductVariant, error)
	GetByIDWithOptions(ctx context.Context, id int32) (*ProductVariantWithOptions, error)
	GetByStripeProductID(ctx context.Context, stripeProductID string) (*ProductVariant, error)
	Create(ctx context.Context, req CreateVariantRequest) (*ProductVariant, error)
	Update(ctx context.Context, id int32, req UpdateVariantRequest) (*ProductVariant, error)
	Archive(ctx context.Context, id int32) (*ProductVariant, error)
	Unarchive(ctx context.Context, id int32) (*ProductVariant, error)
	Activate(ctx context.Context, id int32) (*ProductVariant, error)
	Deactivate(ctx context.Context, id int32) (*ProductVariant, error)

	// Product-specific variant operations
	GetVariantsByProduct(ctx context.Context, productID int32) ([]ProductVariant, error)
	GetActiveVariantsByProduct(ctx context.Context, productID int32) ([]ProductVariant, error)
	GetVariantsInStock(ctx context.Context, productID int32) ([]ProductVariant, error)
	GetVariantsWithOptions(ctx context.Context, productID int32) ([]ProductVariantWithOptions, error)

	// Stock management
	UpdateStock(ctx context.Context, id int32, stock int32) (*ProductVariant, error)
	IncrementStock(ctx context.Context, id int32, delta int32) (*ProductVariant, error)
	DecrementStock(ctx context.Context, id int32, delta int32) (*ProductVariant, error)

	// Search and filtering
	SearchVariants(ctx context.Context, query string) ([]ProductVariant, error)
	GetVariantsByPriceRange(ctx context.Context, minPrice, maxPrice int32) ([]ProductVariant, error)
	GetSubscriptionVariants(ctx context.Context) ([]ProductVariant, error)
	GetLowStockVariants(ctx context.Context, threshold int32) ([]ProductVariant, error)

	// Stripe integration
	UpdateStripeIDs(ctx context.Context, id int32, stripeProductID string, priceIDs map[string]string) (*ProductVariant, error)
	GetVariantsNeedingStripeSync(ctx context.Context, limit, offset int32) ([]ProductVariant, error)
	GetVariantsWithStripeProducts(ctx context.Context, limit, offset int32) ([]ProductVariant, error)

	// Admin utilities
	GetVariantSalesStats(ctx context.Context, dateFrom, dateTo *time.Time, limit, offset int32) ([]VariantSalesStats, error)
	GetTopSellingVariants(ctx context.Context, dateFrom, dateTo *time.Time, limit, offset int32) ([]TopSellingVariant, error)
}

// =============================================================================
// Analytics Types
// =============================================================================

type VariantSalesStats struct {
	VariantID      int32  `json:"variant_id"`
	VariantName    string `json:"variant_name"`
	OptionsDisplay string `json:"options_display"`
	ProductName    string `json:"product_name"`
	OrderCount     int64  `json:"order_count"`
	TotalSold      int64  `json:"total_sold"`
	TotalRevenue   int64  `json:"total_revenue"`
	AvgPrice       int64  `json:"avg_price"`
}

type TopSellingVariant struct {
	VariantID      int32  `json:"variant_id"`
	VariantName    string `json:"variant_name"`
	OptionsDisplay string `json:"options_display"`
	ProductName    string `json:"product_name"`
	TotalSold      int64  `json:"total_sold"`
	TotalRevenue   int64  `json:"total_revenue"`
}

// =============================================================================
// Service Interface
// =============================================================================

type VariantService interface {
	// Customer-facing operations
	GetByID(ctx context.Context, id int32) (*ProductVariant, error)
	GetByIDWithOptions(ctx context.Context, id int32) (*ProductVariantWithOptions, error)
	GetVariantsByProduct(ctx context.Context, productID int32) ([]ProductVariant, error)
	GetActiveVariantsByProduct(ctx context.Context, productID int32) ([]ProductVariant, error)
	SearchVariants(ctx context.Context, query string) ([]ProductVariant, error)

		// Stripe integration
	UpdateStripeIDs(ctx context.Context, variantID int32, stripeProductID string, priceIDs map[string]string) error

	// Admin operations
	Create(ctx context.Context, req CreateVariantRequest) (*ProductVariant, error)
	Update(ctx context.Context, id int32, req UpdateVariantRequest) (*ProductVariant, error)
	Archive(ctx context.Context, id int32) (*ProductVariant, error)
	Activate(ctx context.Context, id int32) (*ProductVariant, error)
	Deactivate(ctx context.Context, id int32) (*ProductVariant, error)

	// Stock management
	UpdateStock(ctx context.Context, id int32, stock int32) (*ProductVariant, error)
	IncrementStock(ctx context.Context, id int32, delta int32) (*ProductVariant, error)
	DecrementStock(ctx context.Context, id int32, delta int32) (*ProductVariant, error)

	// Utilities
	GetLowStockVariants(ctx context.Context, threshold int32) ([]ProductVariant, error)
	CheckAvailability(ctx context.Context, id int32, requestedQuantity int32) (bool, error)
}