package domain

import (
	"context"

	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProductService provides business logic for product catalog operations.
// Implementations should be tenant-scoped.
type ProductService interface {
	// ListProducts returns all active public products for the tenant.
	ListProducts(ctx context.Context) ([]repository.ListActiveProductsRow, error)

	// GetProductDetail retrieves a product with SKUs, pricing, and images.
	GetProductDetail(ctx context.Context, slug string) (*ProductDetail, error)

	// GetProductPrice retrieves pricing for a specific SKU.
	GetProductPrice(ctx context.Context, skuID string) (*ProductPrice, error)

	// GetSKUForCheckout retrieves SKU details with product info for checkout display.
	GetSKUForCheckout(ctx context.Context, skuID string) (*SKUCheckoutDetail, error)
}

// ProductDetail aggregates product information with SKUs, pricing, and images.
type ProductDetail struct {
	Product repository.Product
	SKUs    []ProductSKU
	Images  []repository.ProductImage
}

// ProductSKU combines SKU information with resolved pricing.
type ProductSKU struct {
	SKU              repository.ProductSku
	PriceCents       int32
	CompareAtCents   pgtype.Int4
	InventoryMessage string
}

// ProductPrice contains pricing information for a specific SKU.
type ProductPrice struct {
	SKUID       pgtype.UUID
	PriceCents  int32
	PriceListID pgtype.UUID
}

// SKUCheckoutDetail contains SKU and product info for checkout display.
type SKUCheckoutDetail struct {
	SKUID                   pgtype.UUID
	SKU                     string
	WeightValue             string
	WeightUnit              string
	Grind                   string
	PriceCents              int32
	ProductName             string
	ProductSlug             string
	ProductShortDescription string
	ProductOrigin           string
	ProductRoastLevel       string
	ProductImageURL         string
}
