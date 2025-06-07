// internal/interfaces/product_service.go
package interfaces

import (
	"context"
)

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
}

// Request/Response types for Product Service
type CreateProductRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Price       int32  `json:"price" validate:"required,min=1"`
	Stock       int32  `json:"stock" validate:"min=0"`
	Active      bool   `json:"active"`
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
