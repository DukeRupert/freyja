// internal/server/service/product.go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type ProductService struct {
	repo   interfaces.ProductRepository
	events interfaces.EventPublisher
}

func NewProductService(repo interfaces.ProductRepository, events interfaces.EventPublisher) interfaces.ProductService {
	return &ProductService{
		repo:   repo,
		events: events,
	}
}

// =============================================================================
// Product Retrieval (Now using materialized view)
// =============================================================================

// GetByID retrieves a product with aggregated variant information
func (s *ProductService) GetByID(ctx context.Context, id int) (*interfaces.ProductSummary, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid product ID: %d", id)
	}

	productSummary, err := s.repo.GetProductWithSummary(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", id, err)
	}

	return productSummary, nil
}

// GetBasicProductByID retrieves just the basic product info (for admin operations)
func (s *ProductService) GetBasicProductByID(ctx context.Context, id int) (*interfaces.Product, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid product ID: %d", id)
	}

	product, err := s.repo.GetByID(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", id, err)
	}

	return product, nil
}

// GetByName retrieves a basic product by its name
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

// GetAll retrieves product summaries with optional filtering (uses materialized view)
func (s *ProductService) GetAll(ctx context.Context, filters interfaces.ProductFilters) ([]interfaces.ProductSummary, error) {
	// Apply default filters for public API - only show active products
	if filters.Active == nil {
		active := true
		filters.Active = &active
	}

	productSummaries, err := s.repo.GetAllWithSummary(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	// Publish product list viewed event for analytics
	if err := s.publishProductEvent(ctx, "product.list_viewed", 0, map[string]interface{}{
		"filter_active": filters.Active,
		"limit":         filters.Limit,
		"offset":        filters.Offset,
		"result_count":  len(productSummaries),
	}); err != nil {
		log.Printf("Failed to publish product list viewed event: %v", err)
	}

	return productSummaries, nil
}

// GetInStock retrieves products that have stock available (uses materialized view)
func (s *ProductService) GetInStock(ctx context.Context) ([]interfaces.ProductSummary, error) {
	products, err := s.repo.GetProductsInStock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get in-stock products: %w", err)
	}

	return products, nil
}

// SearchProducts searches products by name, description, or variant options
func (s *ProductService) SearchProducts(ctx context.Context, query string) ([]interfaces.ProductSummary, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Use the enhanced search that includes variant options
	products, err := s.repo.SearchProductsWithOptions(ctx, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	// Publish search event for analytics
	if err := s.publishProductEvent(ctx, "product.searched", 0, map[string]interface{}{
		"query":        query,
		"result_count": len(products),
	}); err != nil {
		log.Printf("Failed to publish product search event: %v", err)
	}

	return products, nil
}

// =============================================================================
// Product Management (Admin Operations)
// =============================================================================

// Create creates a new product (basic product only, variants created separately)
func (s *ProductService) Create(ctx context.Context, req interfaces.CreateProductRequest) (*interfaces.Product, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("product name is required")
	}

	// Validate that the product name is unique
	existingProduct, err := s.repo.GetByName(ctx, req.Name)
	if err == nil && existingProduct != nil {
		return nil, fmt.Errorf("product with name '%s' already exists", req.Name)
	}

	product, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Publish product created event
	if err := s.publishProductEvent(ctx, "product.created", int(product.ID), map[string]interface{}{
		"name":   product.Name,
		"active": product.Active,
	}); err != nil {
		log.Printf("Failed to publish product created event: %v", err)
	}

	return product, nil
}

// Update updates basic product information (name, description, active status)
func (s *ProductService) Update(ctx context.Context, product *interfaces.Product) error {
	if product.ID <= 0 {
		return fmt.Errorf("invalid product ID: %d", product.ID)
	}

	if product.Name == "" {
		return fmt.Errorf("product name is required")
	}

	// Check if product exists
	existing, err := s.repo.GetByID(ctx, product.ID)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	err = s.repo.Update(ctx, product)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	// Refresh materialized view if the product was activated/deactivated
	if existing.Active != product.Active {
		if err := s.repo.RefreshProductStockSummary(ctx); err != nil {
			log.Printf("Failed to refresh product stock summary: %v", err)
		}
	}

	// Publish product updated event
	if err := s.publishProductEvent(ctx, "product.updated", int(product.ID), map[string]interface{}{
		"name":           product.Name,
		"active":         product.Active,
		"status_changed": existing.Active != product.Active,
	}); err != nil {
		log.Printf("Failed to publish product updated event: %v", err)
	}

	return nil
}

// Activate sets a product as active
func (s *ProductService) Activate(ctx context.Context, id int) (*interfaces.Product, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid product ID: %d", id)
	}

	product, err := s.repo.Activate(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("failed to activate product: %w", err)
	}

	// Refresh materialized view
	if err := s.repo.RefreshProductStockSummary(ctx); err != nil {
		log.Printf("Failed to refresh product stock summary: %v", err)
	}

	// Publish product activated event
	if err := s.publishProductEvent(ctx, "product.activated", id, map[string]interface{}{
		"name": product.Name,
	}); err != nil {
		log.Printf("Failed to publish product activated event: %v", err)
	}

	return product, nil
}

// Deactivate sets a product as inactive
func (s *ProductService) Deactivate(ctx context.Context, id int) (*interfaces.Product, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid product ID: %d", id)
	}

	product, err := s.repo.Deactivate(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate product: %w", err)
	}

	// Refresh materialized view
	if err := s.repo.RefreshProductStockSummary(ctx); err != nil {
		log.Printf("Failed to refresh product stock summary: %v", err)
	}

	// Publish product deactivated event
	if err := s.publishProductEvent(ctx, "product.deactivated", id, map[string]interface{}{
		"name": product.Name,
	}); err != nil {
		log.Printf("Failed to publish product deactivated event: %v", err)
	}

	return product, nil
}

// Delete removes a product and all its variants
func (s *ProductService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid product ID: %d", id)
	}

	// Get product info before deletion for event
	product, err := s.repo.GetByID(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	err = s.repo.Delete(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	// Refresh materialized view
	if err := s.repo.RefreshProductStockSummary(ctx); err != nil {
		log.Printf("Failed to refresh product stock summary: %v", err)
	}

	// Publish product deleted event
	if err := s.publishProductEvent(ctx, "product.deleted", id, map[string]interface{}{
		"name": product.Name,
	}); err != nil {
		log.Printf("Failed to publish product deleted event: %v", err)
	}

	return nil
}

// =============================================================================
// Admin Utilities
// =============================================================================

// GetProductsWithoutVariants finds products that need default variants created
func (s *ProductService) GetProductsWithoutVariants(ctx context.Context, limit, offset int) ([]interfaces.Product, error) {
	products, err := s.repo.GetProductsWithoutVariants(ctx, int32(limit), int32(offset))
	if err != nil {
		return nil, fmt.Errorf("failed to get products without variants: %w", err)
	}

	return products, nil
}

// RefreshProductSummary manually refreshes the materialized view
func (s *ProductService) RefreshProductSummary(ctx context.Context) error {
	err := s.repo.RefreshProductStockSummary(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh product stock summary: %w", err)
	}

	// Publish refresh event
	if err := s.publishProductEvent(ctx, "product.summary_refreshed", 0, map[string]interface{}{
		"timestamp": "manual_refresh",
	}); err != nil {
		log.Printf("Failed to publish summary refresh event: %v", err)
	}

	return nil
}

// =============================================================================
// Event Publishing
// =============================================================================

func (s *ProductService) publishProductEvent(ctx context.Context, eventType string, productID int, data map[string]interface{}) error {
	if s.events == nil {
		return nil // Events are optional
	}

	event := interfaces.Event{
		ID:          generateEventID(),
		Type:        eventType,
		AggregateID: fmt.Sprintf("product:%d", productID),
		Data:        data,
		Timestamp:   time.Now(),
		Version:     1,
	}
	
	return s.events.PublishEvent(ctx, event)
}