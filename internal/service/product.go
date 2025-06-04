// internal/service/product.go
package service

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/interfaces"
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
		"total_active":          totalActive,
		"total_inactive":        totalInactive,
		"total_products":        totalAll,
		"low_stock_count":       len(lowStockProducts),
		"out_of_stock_count":    len(outOfStockProducts),
		"low_stock_products":    lowStockProducts,
		"out_of_stock_products": outOfStockProducts,
	}, nil
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
func formatPrice(priceInCents int) string {
	dollars := float64(priceInCents) / 100
	return fmt.Sprintf("$%.2f", dollars)
}
