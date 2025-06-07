// internal/service/product.go
package service

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v82"
	stripePrice "github.com/stripe/stripe-go/v82/price"
	stripeProduct "github.com/stripe/stripe-go/v82/product"
)

type ProductService struct {
	repo interfaces.ProductRepository
	// Note: cache and events will be added later
}

func NewProductService(repo interfaces.ProductRepository) *ProductService {
	return &ProductService{
		repo: repo,
	}
}

// GetByID retrieves a product by its ID
func (s *ProductService) GetByID(ctx context.Context, id int) (*interfaces.Product, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid product ID: %d", id)
	}

	product, err := s.repo.GetByID(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", id, err)
	}

	return product, nil
}

// GetByName retrieves a product by its name
func (s *ProductService) GetByName(ctx context.Context, name string) (*interfaces.Product, error) {
	if name == "" {
		return nil, fmt.Errorf("product name cannot be empty")
	}

	product, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get product by name '%s': %w", name, err)
	}

	return product, nil
}

// GetAll retrieves products with optional filtering
func (s *ProductService) GetAll(ctx context.Context, filters interfaces.ProductFilters) ([]interfaces.Product, error) {
	// Apply default filters for public API
	if filters.Active == nil {
		active := true
		filters.Active = &active
	}

	// Apply reasonable limits
	if filters.Limit <= 0 {
		filters.Limit = 50 // Default limit
	}
	if filters.Limit > 100 {
		filters.Limit = 100 // Max limit
	}

	products, err := s.repo.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	return products, nil
}

// GetActiveProducts retrieves all active products
func (s *ProductService) GetActiveProducts(ctx context.Context) ([]interfaces.Product, error) {
	active := true
	filters := interfaces.ProductFilters{
		Active: &active,
	}
	return s.GetAll(ctx, filters)
}

// SearchProducts searches for products by name and description
func (s *ProductService) SearchProducts(ctx context.Context, query string) ([]interfaces.Product, error) {
	if query == "" {
		return s.GetActiveProducts(ctx)
	}

	// For MVP, we'll use the repository's search method
	// In the future, this could integrate with Elasticsearch
	products, err := s.repo.SearchProducts(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	return products, nil
}

// GetInStock retrieves all products that are currently in stock
func (s *ProductService) GetInStock(ctx context.Context) ([]interfaces.Product, error) {
	products, err := s.repo.GetInStock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get in-stock products: %w", err)
	}

	return products, nil
}

// GetLowStock retrieves products below the specified stock threshold
func (s *ProductService) GetLowStock(ctx context.Context, threshold int) ([]interfaces.Product, error) {
	if threshold < 0 {
		threshold = 10 // Default threshold
	}

	products, err := s.repo.GetLowStock(ctx, int32(threshold))
	if err != nil {
		return nil, fmt.Errorf("failed to get low-stock products: %w", err)
	}

	return products, nil
}

// GetStats returns product statistics for admin dashboard
func (s *ProductService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// Get total active products
	totalActive, err := s.repo.GetCount(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get active product count: %w", err)
	}

	// Get total inactive products
	totalAll, err := s.repo.GetCount(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get total product count: %w", err)
	}
	totalInactive := totalAll - totalActive

	// Get total inventory value
	totalValue, err := s.repo.GetTotalValue(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total inventory value: %w", err)
	}

	// Get low stock products (threshold: 10)
	lowStockProducts, err := s.repo.GetLowStock(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get low stock products: %w", err)
	}

	// Get out of stock products
	outOfStockProducts, err := s.repo.GetLowStock(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get out of stock products: %w", err)
	}

	return map[string]interface{}{
		"total_active":                    totalActive,
		"total_inactive":                  totalInactive,
		"total_products":                  totalAll,
		"total_inventory_value":           totalValue,
		"total_inventory_value_formatted": formatPrice(totalValue),
		"low_stock_count":                 len(lowStockProducts),
		"out_of_stock_count":              len(outOfStockProducts),
		"low_stock_products":              lowStockProducts,
		"out_of_stock_products":           outOfStockProducts,
	}, nil
}

// EnsureStripeProduct ensures a product has Stripe Product and Price objects
func (s *ProductService) EnsureStripeProduct(ctx context.Context, productID int32) error {
	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	// Create Stripe Product if it doesn't exist
	if product.StripeProductID.String == "" {
		stripeProduct, err := s.createStripeProduct(product)
		if err != nil {
			return fmt.Errorf("failed to create Stripe product: %w", err)
		}

		// Update product with Stripe Product ID
		if err := s.repo.UpdateStripeProductID(ctx, productID, stripeProduct.ID); err != nil {
			return fmt.Errorf("failed to update product with Stripe ID: %w", err)
		}

		product.StripeProductID = pgtype.Text{String: stripeProduct.ID, Valid: true}
	}

	// Create all Price objects if they don't exist
	return s.ensureStripePrices(ctx, product)
}

func (s *ProductService) createStripeProduct(product *interfaces.Product) (*stripe.Product, error) {
	params := &stripe.ProductParams{
		Name:        stripe.String(product.Name),
		Description: stripe.String(product.Description.String),
		Active:      stripe.Bool(product.Active),
		Metadata: map[string]string{
			"internal_product_id": fmt.Sprintf("%d", product.ID),
		},
	}

	return stripeProduct.New(params)
}

func (s *ProductService) ensureStripePrices(ctx context.Context, product *interfaces.Product) error {
	priceUpdates := make(map[string]string)

	// One-time purchase price
	if product.StripePriceOnetimeID.String == "" {
		price, err := s.createStripePrice(product, nil) // nil = one-time
		if err != nil {
			return err
		}
		priceUpdates["onetime"] = price.ID
	}

	// Subscription prices for each interval
	intervals := map[string]int{"14day": 14, "21day": 21, "30day": 30, "60day": 60}
	currentPrices := map[string]string{
		"14day": product.StripePrice14dayID.String,
		"21day": product.StripePrice21dayID.String,
		"30day": product.StripePrice30dayID.String,
		"60day": product.StripePrice60dayID.String,
	}

	for interval, days := range intervals {
		if currentPrices[interval] == "" {
			price, err := s.createStripePrice(product, &days)
			if err != nil {
				return err
			}
			priceUpdates[interval] = price.ID
		}
	}

	// Update all price IDs in database
	if len(priceUpdates) > 0 {
		return s.repo.UpdateStripePriceIDs(ctx, product.ID, priceUpdates)
	}

	return nil
}

func (s *ProductService) createStripePrice(product *interfaces.Product, recurringDays *int) (*stripe.Price, error) {
	params := &stripe.PriceParams{
		Product:    stripe.String(product.StripeProductID.String),
		UnitAmount: stripe.Int64(int64(product.Price)),
		Currency:   stripe.String("usd"),
		Metadata: map[string]string{
			"internal_product_id": fmt.Sprintf("%d", product.ID),
		},
	}

	if recurringDays != nil {
		// Subscription price
		params.Recurring = &stripe.PriceRecurringParams{
			Interval:      stripe.String("day"),
			IntervalCount: stripe.Int64(int64(*recurringDays)),
		}
		params.Metadata["subscription_days"] = fmt.Sprintf("%d", *recurringDays)
	} else {
		// One-time price
		params.Metadata["type"] = "onetime"
	}

	return stripePrice.New(params)
}

// ValidateProduct validates product data
func (s *ProductService) ValidateProduct(product *interfaces.Product) error {
	if product == nil {
		return fmt.Errorf("product cannot be nil")
	}

	if product.Name == "" {
		return fmt.Errorf("product name is required")
	}

	if len(product.Name) > 255 {
		return fmt.Errorf("product name cannot exceed 255 characters")
	}

	if product.Price < 0 {
		return fmt.Errorf("product price cannot be negative")
	}

	if product.Stock < 0 {
		return fmt.Errorf("product stock cannot be negative")
	}

	return nil
}

// IsAvailable checks if a product is available for purchase
func (s *ProductService) IsAvailable(ctx context.Context, id int, quantity int) (bool, error) {
	product, err := s.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	if !product.Active {
		return false, fmt.Errorf("product is not active")
	}

	if product.Stock < int32(quantity) {
		return false, fmt.Errorf("insufficient stock: requested %d, available %d", quantity, product.Stock)
	}

	return true, nil
}

// Helper function to format price (same as in handler)
func formatPrice(priceInCents int32) string {
	dollars := float64(priceInCents) / 100
	return fmt.Sprintf("$%.2f", dollars)
}
