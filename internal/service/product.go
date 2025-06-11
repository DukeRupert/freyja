// internal/service/product.go
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/jackc/pgx/v5/pgtype"
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
// Product Retrieval
// =============================================================================

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

// GetByStripeProductID retrieves a product by its Stripe product ID
func (s *ProductService) GetByStripeProductID(ctx context.Context, stripeProductID string) (*interfaces.Product, error) {
	if stripeProductID == "" {
		return nil, fmt.Errorf("Stripe product ID cannot be empty")
	}

	product, err := s.repo.GetByStripeProductID(ctx, stripeProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product by Stripe ID '%s': %w", stripeProductID, err)
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

// GetProductsWithoutStripeSync retrieves products that haven't been synced to Stripe
func (s *ProductService) GetProductsWithoutStripeSync(ctx context.Context, limit, offset int) ([]interfaces.Product, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	products, err := s.repo.GetProductsWithoutStripeSync(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get products without Stripe sync: %w", err)
	}

	return products, nil
}

// =============================================================================
// Product Management
// =============================================================================

// CreateProduct creates a product and publishes creation event
func (s *ProductService) CreateProduct(ctx context.Context, req interfaces.CreateProductRequest) (*interfaces.Product, error) {
	// Validate product data
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create product
	product := &interfaces.Product{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Price:       req.Price,
		Stock:       req.Stock,
		Active:      req.Active,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Publish product created event
	if err := s.publishProductEvent(ctx, interfaces.EventProductCreated, product.ID, map[string]interface{}{
		"name":   product.Name,
		"price":  product.Price,
		"active": product.Active,
	}); err != nil {
		// Log error but don't fail product creation
		fmt.Printf("Failed to publish product created event: %v\n", err)
	}

	return product, nil
}

// UpdateProduct updates a product and publishes update event
func (s *ProductService) UpdateProduct(ctx context.Context, productID int32, req interfaces.UpdateProductRequest) (*interfaces.Product, error) {
	// Get existing product
	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	oldPrice := product.Price
	oldActive := product.Active

	// Update fields
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = pgtype.Text{String: *req.Description, Valid: *req.Description != ""}
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Stock != nil {
		product.Stock = *req.Stock
	}
	if req.Active != nil {
		product.Active = *req.Active
	}

	// Update in database
	if err := s.repo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Determine what changed and publish appropriate events
	eventData := map[string]interface{}{
		"name": product.Name,
	}

	// Check if price changed (triggers Stripe price update)
	if oldPrice != product.Price {
		eventData["price_changed"] = true
		eventData["old_price"] = oldPrice
		eventData["new_price"] = product.Price
	}

	// Check if product was deactivated
	if oldActive && !product.Active {
		// Publish deactivation event
		if err := s.publishProductEvent(ctx, interfaces.EventProductDeactivated, product.ID, eventData); err != nil {
			fmt.Printf("Failed to publish product deactivated event: %v\n", err)
		}
	} else {
		// Publish general update event
		if err := s.publishProductEvent(ctx, interfaces.EventProductUpdated, product.ID, eventData); err != nil {
			fmt.Printf("Failed to publish product updated event: %v\n", err)
		}
	}

	return product, nil
}

// UpdateStock updates product stock
func (s *ProductService) UpdateStock(ctx context.Context, id int, stock int32) error {
	if id <= 0 {
		return fmt.Errorf("invalid product ID: %d", id)
	}
	if stock < 0 {
		return fmt.Errorf("stock cannot be negative")
	}

	return s.repo.UpdateStock(ctx, int32(id), stock)
}

// ReduceStock reduces product stock by the specified quantity
func (s *ProductService) ReduceStock(ctx context.Context, id int32, quantity int32) error {
	if id <= 0 {
		return fmt.Errorf("invalid product ID: %d", id)
	}
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive: %d", quantity)
	}

	// Get current product to check stock
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get product %d: %w", id, err)
	}

	// Check if we have enough stock
	if product.Stock < quantity {
		return fmt.Errorf("insufficient stock for product %d: have %d, need %d", id, product.Stock, quantity)
	}

	// Calculate new stock level
	newStock := product.Stock - quantity

	// Update stock using existing method
	return s.repo.UpdateStock(ctx, id, newStock)
}

// DeactivateProduct deactivates a product
func (s *ProductService) DeactivateProduct(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid product ID: %d", id)
	}

	// Get current product
	product, err := s.repo.GetByID(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	if !product.Active {
		return nil // Already deactivated
	}

	// Update to inactive
	req := interfaces.UpdateProductRequest{
		Active: &[]bool{false}[0],
	}

	_, err = s.UpdateProduct(ctx, int32(id), req)
	return err
}

// ActivateProduct activates a product
func (s *ProductService) ActivateProduct(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid product ID: %d", id)
	}

	// Get current product
	product, err := s.repo.GetByID(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	if product.Active {
		return nil // Already active
	}

	// Update to active
	req := interfaces.UpdateProductRequest{
		Active: &[]bool{true}[0],
	}

	_, err = s.UpdateProduct(ctx, int32(id), req)
	return err
}

// DeleteProduct deletes a product
func (s *ProductService) DeleteProduct(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid product ID: %d", id)
	}

	return s.repo.Delete(ctx, int32(id))
}

// =============================================================================
// Stripe Integration
// =============================================================================

// UpdateStripeProductID updates a product's Stripe product ID
func (s *ProductService) UpdateStripeProductID(ctx context.Context, productID int32, stripeProductID string) error {
	if productID <= 0 {
		return fmt.Errorf("invalid product ID: %d", productID)
	}
	if stripeProductID == "" {
		return fmt.Errorf("Stripe product ID cannot be empty")
	}

	return s.repo.UpdateStripeProductID(ctx, productID, stripeProductID)
}

// UpdateStripePriceIDs updates a product's Stripe price IDs
func (s *ProductService) UpdateStripePriceIDs(ctx context.Context, productID int32, priceIDs map[string]string) error {
	if productID <= 0 {
		return fmt.Errorf("invalid product ID: %d", productID)
	}
	if len(priceIDs) == 0 {
		return fmt.Errorf("price IDs map cannot be empty")
	}

	return s.repo.UpdateStripePriceIDs(ctx, productID, priceIDs)
}

// EnsureStripeProduct ensures a product has Stripe Product and Price objects
func (s *ProductService) EnsureStripeProduct(ctx context.Context, productID int32) error {
	if productID <= 0 {
		return fmt.Errorf("invalid product ID: %d", productID)
	}

	// Publish sync request event - the subscriber will handle the actual sync
	if err := s.publishProductEvent(ctx, interfaces.EventProductStripeSync, productID, map[string]interface{}{
		"sync_requested": true,
	}); err != nil {
		return fmt.Errorf("failed to publish Stripe sync event: %w", err)
	}

	return nil
}

// =============================================================================
// Validation and Utilities
// =============================================================================

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

	// Get products without Stripe sync
	unsyncedProducts, err := s.repo.GetProductsWithoutStripeSync(ctx, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get unsynced products: %w", err)
	}

	return map[string]interface{}{
		"total_active":                    totalActive,
		"total_inactive":                  totalInactive,
		"total_products":                  totalAll,
		"total_inventory_value":           totalValue,
		"total_inventory_value_formatted": formatPrice(totalValue),
		"low_stock_count":                 len(lowStockProducts),
		"out_of_stock_count":              len(outOfStockProducts),
		"unsynced_stripe_count":           len(unsyncedProducts),
		"low_stock_products":              lowStockProducts,
		"out_of_stock_products":           outOfStockProducts,
		"unsynced_stripe_products":        unsyncedProducts,
	}, nil
}

// GetCount returns the total count of products
func (s *ProductService) GetCount(ctx context.Context, activeOnly bool) (int64, error) {
	return s.repo.GetCount(ctx, activeOnly)
}

// GetTotalValue returns the total inventory value
func (s *ProductService) GetTotalValue(ctx context.Context) (int64, error) {
	value, err := s.repo.GetTotalValue(ctx)
	if err != nil {
		return 0, err
	}
	return int64(value), nil
}

// Event Handler Methods

// HandleInventoryReductionEvent processes inventory.reduce_stock events
func (s *ProductService) HandleInventoryReductionEvent(ctx context.Context, data map[string]interface{}) error {
    orderID, ok := data["order_id"].(float64) // JSON numbers come as float64
    if !ok {
        return fmt.Errorf("invalid order_id in inventory reduction event")
    }

    items, ok := data["items"].([]interface{})
    if !ok {
        return fmt.Errorf("invalid items in inventory reduction event")
    }

    log.Printf("Processing inventory reduction for order %d", int32(orderID))

    // Process each item
    for _, itemData := range items {
        item, ok := itemData.(map[string]interface{})
        if !ok {
            log.Printf("⚠️ Invalid item data in inventory reduction event for order %d", int32(orderID))
            continue
        }

        productID, ok := item["product_id"].(float64)
        if !ok {
            log.Printf("⚠️ Invalid product_id in inventory reduction event for order %d", int32(orderID))
            continue
        }

        quantity, ok := item["quantity"].(float64)
        if !ok {
            log.Printf("⚠️ Invalid quantity in inventory reduction event for order %d", int32(orderID))
            continue
        }

        // Reduce stock for this item
        if err := s.ReduceStock(ctx, int32(productID), int32(quantity)); err != nil {
            log.Printf("⚠️ Failed to reduce stock for product %d (order %d): %v", int32(productID), int32(orderID), err)
            // Continue processing other items even if one fails
        } else {
            log.Printf("✅ Reduced stock for product %d by %d units (order %d)", int32(productID), int32(quantity), int32(orderID))
        }
    }

    return nil
}

// =============================================================================
// Helper Methods
// =============================================================================

func (s *ProductService) publishProductEvent(ctx context.Context, eventType string, productID int32, data map[string]interface{}) error {
	event := interfaces.BuildProductEvent(eventType, productID, data)
	return s.events.PublishEvent(ctx, event)
}

func (s *ProductService) validateCreateRequest(req interfaces.CreateProductRequest) error {
	if req.Name == "" {
		return fmt.Errorf("product name is required")
	}
	if req.Price <= 0 {
		return fmt.Errorf("product price must be positive")
	}
	if req.Stock < 0 {
		return fmt.Errorf("product stock cannot be negative")
	}
	return nil
}

// Helper function to format price (same as in handler)
func formatPrice(priceInCents int32) string {
	dollars := float64(priceInCents) / 100
	return fmt.Sprintf("$%.2f", dollars)
}
