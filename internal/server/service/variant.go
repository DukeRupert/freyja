// internal/server/service/variant.go
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type VariantService struct {
	variantRepo interfaces.VariantRepository
	productRepo interfaces.ProductRepository
	events      interfaces.EventPublisher
}

func NewVariantService(
	variantRepo interfaces.VariantRepository,
	productRepo interfaces.ProductRepository,
	events interfaces.EventPublisher,
) interfaces.VariantService {
	return &VariantService{
		variantRepo: variantRepo,
		productRepo: productRepo,
		events:      events,
	}
}

// =============================================================================
// Customer-facing operations
// =============================================================================

func (s *VariantService) GetByID(ctx context.Context, id int32) (*interfaces.ProductVariant, error) {
	variant, err := s.variantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get variant %d: %w", id, err)
	}

	// Check if variant is active and not archived
	if !variant.Active || variant.ArchivedAt.Valid {
		return nil, fmt.Errorf("variant %d is not available", id)
	}

	return variant, nil
}

func (s *VariantService) GetByIDWithOptions(ctx context.Context, id int32) (*interfaces.ProductVariantWithOptions, error) {
	variant, err := s.variantRepo.GetByIDWithOptions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get variant with options %d: %w", id, err)
	}

	// Check if variant is active and not archived
	if !variant.Active || variant.ArchivedAt.Valid {
		return nil, fmt.Errorf("variant %d is not available", id)
	}

	return variant, nil
}

func (s *VariantService) GetVariantsByProduct(ctx context.Context, productID int32) ([]interfaces.ProductVariant, error) {
	// Verify product exists and is active
	product, err := s.productRepo.GetByID(ctx, int32(productID))
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	if !product.Active {
		return nil, fmt.Errorf("product %d is not active", productID)
	}

	return s.variantRepo.GetVariantsByProduct(ctx, productID)
}

func (s *VariantService) GetActiveVariantsByProduct(ctx context.Context, productID int32) ([]interfaces.ProductVariant, error) {
	// Verify product exists and is active
	product, err := s.productRepo.GetByID(ctx, int32(productID))
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	if !product.Active {
		return []interfaces.ProductVariant{}, nil // Return empty slice for inactive products
	}

	return s.variantRepo.GetActiveVariantsByProduct(ctx, productID)
}

func (s *VariantService) SearchVariants(ctx context.Context, query string) ([]interfaces.ProductVariant, error) {
	if query == "" {
		return []interfaces.ProductVariant{}, nil
	}

	variants, err := s.variantRepo.SearchVariants(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search variants: %w", err)
	}

	// Filter out inactive or archived variants for customer-facing search
	activeVariants := make([]interfaces.ProductVariant, 0, len(variants))
	for _, variant := range variants {
		if variant.Active && !variant.ArchivedAt.Valid {
			activeVariants = append(activeVariants, variant)
		}
	}

	return activeVariants, nil
}

// =============================================================================
// Admin operations
// =============================================================================

func (s *VariantService) Create(ctx context.Context, req interfaces.CreateVariantRequest) (*interfaces.ProductVariant, error) {
	// Validate product exists
	product, err := s.productRepo.GetByID(ctx, int32(req.ProductID))
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", req.ProductID, err)
	}

	// Create the variant
	variant, err := s.variantRepo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create variant: %w", err)
	}

	// Publish event
	if err := s.events.PublishEvent(ctx, interfaces.Event{
		Type:        interfaces.EventVariantCreated,
		AggregateID: fmt.Sprintf("variant-%d", variant.ID),
		Data: map[string]interface{}{
			"variant_id": variant.ID,
			"product_id": variant.ProductID,
			"name":       variant.Name,
			"price":      variant.Price,
		},
	}); err != nil {
		log.Printf("Failed to publish variant.created event: %v", err)
	}

	log.Printf("✅ Created variant %d for product %d (%s)", variant.ID, product.ID, variant.Name)
	return variant, nil
}

func (s *VariantService) Update(ctx context.Context, id int32, req interfaces.UpdateVariantRequest) (*interfaces.ProductVariant, error) {
	// Get existing variant
	existingVariant, err := s.variantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get variant %d: %w", id, err)
	}

	// Track price changes for Stripe price updates
	priceChanged := req.Price != nil && *req.Price != existingVariant.Price

	// Update the variant
	variant, err := s.variantRepo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update variant: %w", err)
	}

	// Publish event
	eventData := map[string]interface{}{
		"variant_id":    variant.ID,
		"product_id":    variant.ProductID,
		"price_changed": priceChanged,
	}

	if priceChanged {
		eventData["old_price"] = existingVariant.Price
		eventData["new_price"] = variant.Price
	}

	if err := s.events.PublishEvent(ctx, interfaces.Event{
		Type:        interfaces.EventVariantUpdated,
		AggregateID: fmt.Sprintf("variant-%d", variant.ID),
		Data:        eventData,
	}); err != nil {
		log.Printf("Failed to publish variant.updated event: %v", err)
	}

	log.Printf("✅ Updated variant %d", variant.ID)
	return variant, nil
}

func (s *VariantService) Archive(ctx context.Context, id int32) (*interfaces.ProductVariant, error) {
	variant, err := s.variantRepo.Archive(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to archive variant: %w", err)
	}

	// Publish event
	if err := s.events.PublishEvent(ctx, interfaces.Event{
		Type:        interfaces.EventVariantArchived,
		AggregateID: fmt.Sprintf("variant-%d", variant.ID),
		Data: map[string]interface{}{
			"variant_id": variant.ID,
			"product_id": variant.ProductID,
		},
	}); err != nil {
		log.Printf("Failed to publish variant.archived event: %v", err)
	}

	log.Printf("✅ Archived variant %d", variant.ID)
	return variant, nil
}

func (s *VariantService) Activate(ctx context.Context, id int32) (*interfaces.ProductVariant, error) {
	variant, err := s.variantRepo.Activate(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to activate variant: %w", err)
	}

	// Publish event
	if err := s.events.PublishEvent(ctx, interfaces.Event{
		Type:        interfaces.EventVariantActivated,
		AggregateID: fmt.Sprintf("variant-%d", variant.ID),
		Data: map[string]interface{}{
			"variant_id": variant.ID,
			"product_id": variant.ProductID,
		},
	}); err != nil {
		log.Printf("Failed to publish variant.activated event: %v", err)
	}

	log.Printf("✅ Activated variant %d", variant.ID)
	return variant, nil
}

func (s *VariantService) Deactivate(ctx context.Context, id int32) (*interfaces.ProductVariant, error) {
	variant, err := s.variantRepo.Deactivate(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate variant: %w", err)
	}

	// Publish event
	if err := s.events.PublishEvent(ctx, interfaces.Event{
		Type:        interfaces.EventVariantDeactivated,
		AggregateID: fmt.Sprintf("variant-%d", variant.ID),
		Data: map[string]interface{}{
			"variant_id": variant.ID,
			"product_id": variant.ProductID,
		},
	}); err != nil {
		log.Printf("Failed to publish variant.deactivated event: %v", err)
	}

	log.Printf("✅ Deactivated variant %d", variant.ID)
	return variant, nil
}

// =============================================================================
// Stock management
// =============================================================================

func (s *VariantService) UpdateStock(ctx context.Context, id int32, stock int32) (*interfaces.ProductVariant, error) {
	if stock < 0 {
		return nil, fmt.Errorf("stock cannot be negative")
	}

	variant, err := s.variantRepo.UpdateStock(ctx, id, stock)
	if err != nil {
		return nil, fmt.Errorf("failed to update stock: %w", err)
	}

	// Publish event
	if err := s.events.PublishEvent(ctx, interfaces.Event{
		Type:        interfaces.EventVariantStockUpdated,
		AggregateID: fmt.Sprintf("variant-%d", variant.ID),
		Data: map[string]interface{}{
			"variant_id": variant.ID,
			"product_id": variant.ProductID,
			"new_stock":  variant.Stock,
		},
	}); err != nil {
		log.Printf("Failed to publish variant.stock_updated event: %v", err)
	}

	return variant, nil
}

func (s *VariantService) IncrementStock(ctx context.Context, id int32, delta int32) (*interfaces.ProductVariant, error) {
	if delta <= 0 {
		return nil, fmt.Errorf("increment delta must be positive")
	}

	variant, err := s.variantRepo.IncrementStock(ctx, id, delta)
	if err != nil {
		return nil, fmt.Errorf("failed to increment stock: %w", err)
	}

	// Publish event
	if err := s.events.PublishEvent(ctx, interfaces.Event{
		Type:        interfaces.EventVariantStockIncremented,
		AggregateID: fmt.Sprintf("variant-%d", variant.ID),
		Data: map[string]interface{}{
			"variant_id": variant.ID,
			"product_id": variant.ProductID,
			"delta":      delta,
			"new_stock":  variant.Stock,
		},
	}); err != nil {
		log.Printf("Failed to publish variant.stock_incremented event: %v", err)
	}

	return variant, nil
}

func (s *VariantService) DecrementStock(ctx context.Context, id int32, delta int32) (*interfaces.ProductVariant, error) {
	if delta <= 0 {
		return nil, fmt.Errorf("decrement delta must be positive")
	}

	variant, err := s.variantRepo.DecrementStock(ctx, id, delta)
	if err != nil {
		return nil, fmt.Errorf("failed to decrement stock: %w", err)
	}

	// Publish event
	if err := s.events.PublishEvent(ctx, interfaces.Event{
		Type:        interfaces.EventVariantStockDecremented,
		AggregateID: fmt.Sprintf("variant-%d", variant.ID),
		Data: map[string]interface{}{
			"variant_id": variant.ID,
			"product_id": variant.ProductID,
			"delta":      delta,
			"new_stock":  variant.Stock,
		},
	}); err != nil {
		log.Printf("Failed to publish variant.stock_decremented event: %v", err)
	}

	return variant, nil
}

// =============================================================================
// Utilities
// =============================================================================

func (s *VariantService) GetLowStockVariants(ctx context.Context, threshold int32) ([]interfaces.ProductVariant, error) {
	if threshold < 0 {
		return nil, fmt.Errorf("threshold cannot be negative")
	}

	return s.variantRepo.GetLowStockVariants(ctx, threshold)
}

func (s *VariantService) CheckAvailability(ctx context.Context, id int32, requestedQuantity int32) (bool, error) {
	if requestedQuantity <= 0 {
		return false, fmt.Errorf("requested quantity must be positive")
	}

	variant, err := s.variantRepo.GetByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to get variant: %w", err)
	}

	// Check if variant is available for purchase
	if !variant.Active || variant.ArchivedAt.Valid {
		return false, nil
	}

	// Check stock availability
	return variant.Stock >= requestedQuantity, nil
}

// =============================================================================
// Stripe Integration
// =============================================================================

func (s *VariantService) UpdateStripeIDs(ctx context.Context, variantID int32, stripeProductID string, priceIDs map[string]string) error {
	_, err := s.variantRepo.UpdateStripeIDs(ctx, variantID, stripeProductID, priceIDs)
	return err
}