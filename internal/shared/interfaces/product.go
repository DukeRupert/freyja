// internal/shared/interfaces/product.go
package interfaces

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// Service Interfaces
// =============================================================================

type ProductService interface {
	// Customer-facing operations (returns ProductSummary with variant data)
	GetByID(ctx context.Context, id int) (*ProductSummary, error)
	GetAll(ctx context.Context, filters ProductFilters) ([]ProductSummary, error)
	GetInStock(ctx context.Context) ([]ProductSummary, error)
	SearchProducts(ctx context.Context, query string) ([]ProductSummary, error)

	// Admin operations (returns basic Product data)
	GetBasicProductByID(ctx context.Context, id int) (*Product, error)
	GetByName(ctx context.Context, name string) (*Product, error)
	Create(ctx context.Context, req CreateProductRequest) (*Product, error)
	Update(ctx context.Context, product *Product) error
	Activate(ctx context.Context, id int) (*Product, error)
	Deactivate(ctx context.Context, id int) (*Product, error)
	Delete(ctx context.Context, id int) error

	// Admin utilities
	GetProductsWithoutVariants(ctx context.Context, limit, offset int) ([]Product, error)
	RefreshProductSummary(ctx context.Context) error
}

// =============================================================================
// HTTP Request/Response Types
// =============================================================================

// ProductResponse represents a product in API responses (customer-facing)
type ProductResponse struct {
	ID               int32       `json:"id"`
	Name             string      `json:"name"`
	Description      *string     `json:"description"`
	TotalStock       int32       `json:"total_stock"`
	VariantsInStock  int32       `json:"variants_in_stock"`
	TotalVariants    int32       `json:"total_variants"`
	MinPrice         int32       `json:"min_price"`
	MaxPrice         int32       `json:"max_price"`
	HasStock         bool        `json:"has_stock"`
	StockStatus      string      `json:"stock_status"`
	PriceDisplay     string      `json:"price_display"`
	AvailableOptions interface{} `json:"available_options,omitempty"`
}

// AdminProductResponse represents a product in admin API responses
type AdminProductResponse struct {
	ID          int32     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProductListResponse represents paginated product list
type ProductListResponse struct {
	Products []ProductResponse `json:"products"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// =============================================================================
// Helper Functions
// =============================================================================

// ToProductResponse converts ProductSummary to customer-facing API response
func (ps *ProductSummary) ToProductResponse() *ProductResponse {
	resp := &ProductResponse{
		ID:              ps.ProductID,
		Name:            ps.Name,
		TotalStock:      ps.TotalStock,
		VariantsInStock: ps.VariantsInStock,
		TotalVariants:   ps.TotalVariants,
		MinPrice:        ps.MinPrice,
		MaxPrice:        ps.MaxPrice,
		HasStock:        ps.HasStock,
		StockStatus:     ps.StockStatus,
		PriceDisplay:    formatPriceDisplay(ps.MinPrice, ps.MaxPrice),
	}

	// Handle optional description
	if ps.Description.Valid {
		resp.Description = &ps.Description.String
	}

	// Handle available options JSON
	if len(ps.AvailableOptions) > 0 {
		// This would be parsed JSON, but for now just include as raw
		resp.AvailableOptions = ps.AvailableOptions
	}

	return resp
}

// ToAdminProductResponse converts Product to admin API response
func (p *Product) ToAdminProductResponse() *AdminProductResponse {
	resp := &AdminProductResponse{
		ID:        p.ID,
		Name:      p.Name,
		Active:    p.Active,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}

	// Handle optional description
	if p.Description.Valid {
		resp.Description = &p.Description.String
	}

	return resp
}

// formatPriceDisplay creates a user-friendly price display string
func formatPriceDisplay(minPrice, maxPrice int32) string {
	if minPrice == maxPrice {
		return formatCurrency(minPrice)
	}
	return formatCurrency(minPrice) + " - " + formatCurrency(maxPrice)
}

// formatCurrency converts cents to dollar display
func formatCurrency(cents int32) string {
	dollars := float64(cents) / 100
	return fmt.Sprintf("$%.2f", dollars)
}

// =============================================================================
// Product Domain Types
// =============================================================================

// Product represents the basic product entity (for admin/management operations)
type Product struct {
	ID          int32       `json:"id"`
	Name        string      `json:"name"`
	Description pgtype.Text `json:"description"`
	Active      bool        `json:"active"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ProductSummary represents aggregated product information with variant data
// This is what customers see when browsing products
type ProductSummary struct {
	ProductID        int32     `json:"product_id"`
	Name             string    `json:"name"`
	Description      pgtype.Text `json:"description"`
	ProductActive    bool      `json:"product_active"`
	TotalStock       int32     `json:"total_stock"`
	VariantsInStock  int32     `json:"variants_in_stock"`
	TotalVariants    int32     `json:"total_variants"`
	MinPrice         int32     `json:"min_price"`
	MaxPrice         int32     `json:"max_price"`
	HasStock         bool      `json:"has_stock"`
	StockStatus      string    `json:"stock_status"`
	AvailableOptions []byte    `json:"available_options"`
}

// CreateProductRequest represents the data needed to create a new product
type CreateProductRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=1000"`
	Active      bool   `json:"active"`
}

// ProductFilters represents filtering options for product queries
type ProductFilters struct {
	Active *bool `json:"active,omitempty"`
	Limit  int   `json:"limit,omitempty"`
	Offset int   `json:"offset,omitempty"`
}

// =============================================================================
// Repository Interfaces
// =============================================================================

type ProductRepository interface {
	// Basic product operations (for admin/management)
	GetByID(ctx context.Context, id int32) (*Product, error)
	GetByName(ctx context.Context, name string) (*Product, error)
	Create(ctx context.Context, req CreateProductRequest) (*Product, error)
	Update(ctx context.Context, product *Product) error
	Activate(ctx context.Context, id int32) (*Product, error)
	Deactivate(ctx context.Context, id int32) (*Product, error)
	Delete(ctx context.Context, id int32) error

	// Product summary operations (using materialized view for customer-facing)
	GetProductWithSummary(ctx context.Context, id int32) (*ProductSummary, error)
	GetAllWithSummary(ctx context.Context, filters ProductFilters) ([]ProductSummary, error)
	GetProductsInStock(ctx context.Context) ([]ProductSummary, error)
	SearchProductsWithOptions(ctx context.Context, query string) ([]ProductSummary, error)

	// Admin utilities
	GetProductsWithoutVariants(ctx context.Context, limit, offset int32) ([]Product, error)
	RefreshProductStockSummary(ctx context.Context) error
}