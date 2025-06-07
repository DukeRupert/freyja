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
	Create(ctx context.Context, product *Product) error
	Update(ctx context.Context, product *Product) error
	UpdateStock(ctx context.Context, id int32, stock int32) error
	UpdateStripeProductID(ctx context.Context, id int32, stripeProductID string) error
	UpdateStripePriceIDs(ctx context.Context, id int32, priceIDs map[string]string) error
	Delete(ctx context.Context, id int32) error
	GetCount(ctx context.Context, activeOnly bool) (int64, error)
	GetTotalValue(ctx context.Context) (int32, error)
}
