// internal/interfaces/product.go
package interfaces

import (
	"context"

	"github.com/dukerupert/freyja/internal/database"
)

// Use database types directly for MVP simplicity
type Product = database.Products

type ProductFilters struct {
	Active *bool `json:"active,omitempty"`
	Limit  int   `json:"limit,omitempty"`
	Offset int   `json:"offset,omitempty"`
}

type ProductRepository interface {
	GetByID(ctx context.Context, id int32) (*Product, error)
	GetByName(ctx context.Context, name string) (*Product, error)
	GetByStripeProductID(ctx context.Context, stripeProductID string) (*Product, error)
	GetAll(ctx context.Context, filters ProductFilters) ([]Product, error)
	SearchProducts(ctx context.Context, query string) ([]Product, error)
	GetInStock(ctx context.Context) ([]Product, error)
	GetLowStock(ctx context.Context, threshold int32) ([]Product, error)
	GetProductsWithoutStripeSync(ctx context.Context, limit, offset int) ([]Product, error)
	Create(ctx context.Context, req CreateProductRequest) (*Product, error)
	Update(ctx context.Context, product *Product) error
	UpdateStock(ctx context.Context, id int32, stock int32) error
	UpdateStripeProductID(ctx context.Context, id int32, stripeProductID string) error
	UpdateStripePriceIDs(ctx context.Context, id int32, priceIDs map[string]string) error
	Delete(ctx context.Context, id int32) error
	GetCount(ctx context.Context, activeOnly bool) (int64, error)
	GetTotalValue(ctx context.Context) (int32, error)
}

type ProductService interface {
	// Product retrieval
	GetByID(ctx context.Context, id int) (*Product, error)
	GetByName(ctx context.Context, name string) (*Product, error)
	GetByStripeProductID(ctx context.Context, stripeProductID string) (*Product, error)
	GetAll(ctx context.Context, filters ProductFilters) ([]Product, error)
	GetActiveProducts(ctx context.Context) ([]Product, error)
	SearchProducts(ctx context.Context, query string) ([]Product, error)
	GetInStock(ctx context.Context) ([]Product, error)
	GetLowStock(ctx context.Context, threshold int) ([]Product, error)
	GetProductsWithoutStripeSync(ctx context.Context, limit, offset int) ([]Product, error)

	// Product management
	CreateProduct(ctx context.Context, req CreateProductRequest) (*Product, error)
	UpdateProduct(ctx context.Context, productID int32, req UpdateProductRequest) (*Product, error)
	UpdateStock(ctx context.Context, id int, stock int32) error
	ReduceStock(ctx context.Context, id int32, quantity int32) error
	DeactivateProduct(ctx context.Context, id int) error
	ActivateProduct(ctx context.Context, id int) error
	DeleteProduct(ctx context.Context, id int) error

	// Stripe integration
	UpdateStripeProductID(ctx context.Context, productID int32, stripeProductID string) error
	UpdateStripePriceIDs(ctx context.Context, productID int32, priceIDs map[string]string) error
	EnsureStripeProduct(ctx context.Context, productID int32) error

	// Validation and utilities
	ValidateProduct(product *Product) error
	IsAvailable(ctx context.Context, id int, quantity int) (bool, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
	GetCount(ctx context.Context, activeOnly bool) (int64, error)
	GetTotalValue(ctx context.Context) (int64, error)
}

// Request/Response types for Product Service
type CreateProductFormRequest struct {
	Name        string  `json:"name" form:"name" validate:"required"`
	Description string  `json:"description" form:"description"`
	Price       float64 `json:"price" form:"price" validate:"required,min=0.01"`
	Stock       int32   `json:"stock" form:"stock" validate:"min=0"`
	Active      string  `json:"active" form:"active"`
}

// Convert to the service request struct
func (f CreateProductFormRequest) ToCreateProductRequest() CreateProductRequest {
	return CreateProductRequest{
		Name:        f.Name,
		Description: f.Description,
		Price:       int32(f.Price * 100), // Convert dollars to cents
		Stock:       f.Stock,
		Active:      f.Active == "on" || f.Active == "true" || f.Active == "1",
	}
}

type CreateProductRequest struct {
	Name        string `json:"name" form:"name" validate:"required"`
	Description string `json:"description" form:"description"`
	Price       int32  `json:"price" form:"price" validate:"required,min=1"`
	Stock       int32  `json:"stock" form:"stock" validate:"min=0"`
	Active      bool   `json:"active" form:"active"`
}

type UpdateProductRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Price       *int32  `json:"price,omitempty"`
	Stock       *int32  `json:"stock,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}

// Stripe Price Configuration for products
type StripePriceConfig struct {
	ProductID          int32             `json:"product_id"`
	OneTimePrice       *string           `json:"onetime_price,omitempty"`
	SubscriptionPrices map[string]string `json:"subscription_prices,omitempty"` // "14day": "price_id", etc.
}

// Product availability info
type ProductAvailability struct {
	ProductID    int32  `json:"product_id"`
	Available    bool   `json:"available"`
	Stock        int32  `json:"stock"`
	RequestedQty int32  `json:"requested_quantity"`
	AvailableQty int32  `json:"available_quantity"`
	Reason       string `json:"reason,omitempty"`
}
