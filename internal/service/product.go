package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProductService provides business logic for product operations
type ProductService interface {
	ListProducts(ctx context.Context) ([]repository.ListActiveProductsRow, error)
	GetProductDetail(ctx context.Context, slug string) (*ProductDetail, error)
	GetProductPrice(ctx context.Context, skuID string) (*ProductPrice, error)
}

// ProductDetail aggregates product information with SKUs, pricing, and images
type ProductDetail struct {
	Product repository.Product
	SKUs    []ProductSKU
	Images  []repository.ProductImage
}

// ProductSKU combines SKU information with resolved pricing
type ProductSKU struct {
	SKU              repository.ProductSku
	PriceCents       int32
	CompareAtCents   pgtype.Int4
	InventoryMessage string
}

// ProductPrice contains pricing information for a specific SKU
type ProductPrice struct {
	SKUID       pgtype.UUID
	PriceCents  int32
	PriceListID pgtype.UUID
}

type productService struct {
	repo     repository.Querier
	tenantID pgtype.UUID
}

// NewProductService creates a new ProductService instance
func NewProductService(repo repository.Querier, tenantID string) (ProductService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	return &productService{
		repo:     repo,
		tenantID: tenantUUID,
	}, nil
}

// ListProducts returns all active public products for the tenant
func (s *productService) ListProducts(ctx context.Context) ([]repository.ListActiveProductsRow, error) {
	products, err := s.repo.ListActiveProducts(ctx, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	return products, nil
}

// GetProductDetail retrieves a product with all its SKUs, pricing, and images
func (s *productService) GetProductDetail(ctx context.Context, slug string) (*ProductDetail, error) {
	product, err := s.repo.GetProductBySlug(ctx, repository.GetProductBySlugParams{
		TenantID: s.tenantID,
		Slug:     slug,
	})
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product by slug: %w", err)
	}

	skus, err := s.repo.GetProductSKUs(ctx, product.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product SKUs: %w", err)
	}

	images, err := s.repo.GetProductImages(ctx, product.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product images: %w", err)
	}

	priceList, err := s.repo.GetDefaultPriceList(ctx, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default price list: %w", err)
	}

	productSKUs := make([]ProductSKU, 0, len(skus))
	for _, sku := range skus {
		price, err := s.repo.GetPriceForSKU(ctx, repository.GetPriceForSKUParams{
			PriceListID:  priceList.ID,
			ProductSkuID: sku.ID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("failed to get price for SKU %s: %w", sku.Sku, err)
		}

		inventoryMsg := s.getInventoryMessage(sku)

		productSKUs = append(productSKUs, ProductSKU{
			SKU:              sku,
			PriceCents:       price.PriceCents,
			CompareAtCents:   price.CompareAtPriceCents,
			InventoryMessage: inventoryMsg,
		})
	}

	return &ProductDetail{
		Product: product,
		SKUs:    productSKUs,
		Images:  images,
	}, nil
}

// GetProductPrice retrieves pricing information for a specific SKU
func (s *productService) GetProductPrice(ctx context.Context, skuID string) (*ProductPrice, error) {
	var skuUUID pgtype.UUID
	if err := skuUUID.Scan(skuID); err != nil {
		return nil, fmt.Errorf("invalid SKU ID: %w", err)
	}

	sku, err := s.repo.GetSKUByID(ctx, skuUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSKUNotFound
		}
		return nil, fmt.Errorf("failed to get SKU: %w", err)
	}

	priceList, err := s.repo.GetDefaultPriceList(ctx, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default price list: %w", err)
	}

	price, err := s.repo.GetPriceForSKU(ctx, repository.GetPriceForSKUParams{
		PriceListID:  priceList.ID,
		ProductSkuID: sku.ID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPriceNotFound
		}
		return nil, fmt.Errorf("failed to get price for SKU: %w", err)
	}

	return &ProductPrice{
		SKUID:       sku.ID,
		PriceCents:  price.PriceCents,
		PriceListID: priceList.ID,
	}, nil
}

// getInventoryMessage returns an advisory message about inventory status
func (s *productService) getInventoryMessage(sku repository.ProductSku) string {
	if sku.InventoryQuantity <= 0 {
		if sku.InventoryPolicy == "allow" {
			return "Available on backorder"
		}
		return "Out of stock"
	}

	if sku.LowStockThreshold.Valid && sku.InventoryQuantity <= sku.LowStockThreshold.Int32 {
		return "Low stock"
	}

	return "In stock"
}
