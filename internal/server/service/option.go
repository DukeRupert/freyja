// internal/server/service/option.go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type OptionService struct {
	optionRepo    interfaces.OptionRepository
	variantRepo   interfaces.VariantRepository
	productRepo   interfaces.ProductRepository
	events        interfaces.EventPublisher
}

func NewOptionService(
	optionRepo interfaces.OptionRepository, 
	variantRepo interfaces.VariantRepository,
	productRepo interfaces.ProductRepository,
	events interfaces.EventPublisher,
) interfaces.OptionService {
	return &OptionService{
		optionRepo:  optionRepo,
		variantRepo: variantRepo,
		productRepo: productRepo,
		events:      events,
	}
}

// =============================================================================
// Product Option Management
// =============================================================================

func (s *OptionService) CreateProductOption(ctx context.Context, req interfaces.CreateProductOptionRequest) (*interfaces.ProductOption, error) {
	// Validate product exists
	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", req.ProductID, err)
	}

	// Check if option key already exists for this product
	existingOption, err := s.optionRepo.GetProductOptionByKey(ctx, req.ProductID, req.OptionKey)
	if err == nil && existingOption != nil {
		return nil, fmt.Errorf("option with key '%s' already exists for product %d", req.OptionKey, req.ProductID)
	}

	// Create the option
	option, err := s.optionRepo.CreateProductOption(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create product option: %w", err)
	}

	// Publish event
	if err := s.publishOptionEvent(ctx, "option.created", int(option.ID), map[string]interface{}{
		"option_id":   option.ID,
		"product_id":  option.ProductID,
		"option_key":  option.OptionKey,
		"product_name": product.Name,
	}); err != nil {
		log.Printf("Failed to publish option.created event: %v", err)
	}

	log.Printf("✅ Created option %d for product %d (%s: %s)", option.ID, product.ID, product.Name, option.OptionKey)
	return option, nil
}

func (s *OptionService) GetProductOptionByID(ctx context.Context, id int32) (*interfaces.ProductOption, error) {
	option, err := s.optionRepo.GetProductOptionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get product option %d: %w", id, err)
	}

	return option, nil
}

func (s *OptionService) GetProductOptions(ctx context.Context, productID int32) ([]interfaces.ProductOptionWithValues, error) {
	// Validate product exists
	_, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	// Get options for the product
	options, err := s.optionRepo.GetProductOptionsByProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product options: %w", err)
	}

	// Get values for each option
	optionsWithValues := make([]interfaces.ProductOptionWithValues, len(options))
	for i, option := range options {
		values, err := s.optionRepo.GetOptionValuesByOption(ctx, option.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get values for option %d: %w", option.ID, err)
		}

		optionsWithValues[i] = interfaces.ProductOptionWithValues{
			ProductOption: option,
			Values:        values,
		}
	}

	return optionsWithValues, nil
}

func (s *OptionService) UpdateProductOption(ctx context.Context, id int32, req interfaces.UpdateProductOptionRequest) (*interfaces.ProductOption, error) {
	// Get existing option
	existingOption, err := s.optionRepo.GetProductOptionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing option: %w", err)
	}

	// Check if new option key conflicts with existing options for the same product
	if req.OptionKey != existingOption.OptionKey {
		conflictingOption, err := s.optionRepo.GetProductOptionByKey(ctx, existingOption.ProductID, req.OptionKey)
		if err == nil && conflictingOption != nil && conflictingOption.ID != id {
			return nil, fmt.Errorf("option with key '%s' already exists for this product", req.OptionKey)
		}
	}

	// Update the option
	option, err := s.optionRepo.UpdateProductOption(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update product option: %w", err)
	}

	// Publish event
	if err := s.publishOptionEvent(ctx, "option.updated", int(option.ID), map[string]interface{}{
		"option_id":     option.ID,
		"product_id":    option.ProductID,
		"option_key":    option.OptionKey,
		"old_key":       existingOption.OptionKey,
		"key_changed":   req.OptionKey != existingOption.OptionKey,
	}); err != nil {
		log.Printf("Failed to publish option.updated event: %v", err)
	}

	log.Printf("✅ Updated option %d (%s)", option.ID, option.OptionKey)
	return option, nil
}

func (s *OptionService) DeleteProductOption(ctx context.Context, id int32) error {
	// Get the option first
	option, err := s.optionRepo.GetProductOptionByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get option %d: %w", id, err)
	}

	// Check if option is being used by any active variants
	usage, err := s.optionRepo.CheckOptionUsage(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check option usage: %w", err)
	}

	if usage > 0 {
		return fmt.Errorf("cannot delete option '%s': it is used by %d active variants. Archive those variants first", option.OptionKey, usage)
	}

	// Delete the option (this will cascade delete option values due to FK constraints)
	err = s.optionRepo.DeleteProductOption(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete product option: %w", err)
	}

	// Publish event
	if err := s.publishOptionEvent(ctx, "option.deleted", int(option.ID), map[string]interface{}{
		"option_id":   option.ID,
		"product_id":  option.ProductID,
		"option_key":  option.OptionKey,
		"usage_count": usage,
	}); err != nil {
		log.Printf("Failed to publish option.deleted event: %v", err)
	}

	log.Printf("✅ Deleted option %d (%s)", option.ID, option.OptionKey)
	return nil
}

// =============================================================================
// Option Value Management
// =============================================================================

func (s *OptionService) CreateOptionValue(ctx context.Context, req interfaces.CreateOptionValueRequest) (*interfaces.ProductOptionValue, error) {
	// Validate option exists
	option, err := s.optionRepo.GetProductOptionByID(ctx, req.OptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option %d: %w", req.OptionID, err)
	}

	// Check if value already exists for this option
	existingValue, err := s.optionRepo.GetOptionValueByValue(ctx, req.OptionID, req.Value)
	if err == nil && existingValue != nil {
		return nil, fmt.Errorf("value '%s' already exists for option '%s'", req.Value, option.OptionKey)
	}

	// Create the value
	value, err := s.optionRepo.CreateOptionValue(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create option value: %w", err)
	}

	// Publish event
	if err := s.publishOptionEvent(ctx, "option_value.created", int(value.ID), map[string]interface{}{
		"value_id":   value.ID,
		"option_id":  option.ID,
		"option_key": option.OptionKey,
		"value":      value.Value,
		"product_id": option.ProductID,
	}); err != nil {
		log.Printf("Failed to publish option_value.created event: %v", err)
	}

	log.Printf("✅ Created option value %d for option %s (%s: %s)", value.ID, option.OptionKey, option.OptionKey, value.Value)
	return value, nil
}

func (s *OptionService) GetOptionValueByID(ctx context.Context, id int32) (*interfaces.ProductOptionValue, error) {
	value, err := s.optionRepo.GetOptionValueByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get option value %d: %w", id, err)
	}

	return value, nil
}

func (s *OptionService) GetOptionValues(ctx context.Context, optionID int32) ([]interfaces.ProductOptionValue, error) {
	// Validate option exists
	_, err := s.optionRepo.GetProductOptionByID(ctx, optionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option %d: %w", optionID, err)
	}

	values, err := s.optionRepo.GetOptionValuesByOption(ctx, optionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option values: %w", err)
	}

	return values, nil
}

func (s *OptionService) UpdateOptionValue(ctx context.Context, id int32, req interfaces.UpdateOptionValueRequest) (*interfaces.ProductOptionValue, error) {
	// Get existing value
	existingValue, err := s.optionRepo.GetOptionValueByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing option value: %w", err)
	}

	// Get the option for validation
	option, err := s.optionRepo.GetProductOptionByID(ctx, existingValue.ProductOptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option: %w", err)
	}

	// Check if new value conflicts with existing values for the same option
	if req.Value != existingValue.Value {
		conflictingValue, err := s.optionRepo.GetOptionValueByValue(ctx, existingValue.ProductOptionID, req.Value)
		if err == nil && conflictingValue != nil && conflictingValue.ID != id {
			return nil, fmt.Errorf("value '%s' already exists for option '%s'", req.Value, option.OptionKey)
		}
	}

	// Update the value
	value, err := s.optionRepo.UpdateOptionValue(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update option value: %w", err)
	}

	// Publish event
	if err := s.publishOptionEvent(ctx, "option_value.updated", int(value.ID), map[string]interface{}{
		"value_id":      value.ID,
		"option_id":     option.ID,
		"option_key":    option.OptionKey,
		"value":         value.Value,
		"old_value":     existingValue.Value,
		"value_changed": req.Value != existingValue.Value,
		"product_id":    option.ProductID,
	}); err != nil {
		log.Printf("Failed to publish option_value.updated event: %v", err)
	}

	log.Printf("✅ Updated option value %d (%s: %s)", value.ID, option.OptionKey, value.Value)
	return value, nil
}

func (s *OptionService) DeleteOptionValue(ctx context.Context, id int32) error {
	// Get the value first
	value, err := s.optionRepo.GetOptionValueByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get option value %d: %w", id, err)
	}

	// Get the option for context
	option, err := s.optionRepo.GetProductOptionByID(ctx, value.ProductOptionID)
	if err != nil {
		return fmt.Errorf("failed to get option: %w", err)
	}

	// Check if value is being used by any active variants
	usage, err := s.optionRepo.CheckOptionValueUsage(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check option value usage: %w", err)
	}

	if usage > 0 {
		return fmt.Errorf("cannot delete option value '%s' for '%s': it is used by %d active variants. Archive those variants first", value.Value, option.OptionKey, usage)
	}

	// Delete the value
	err = s.optionRepo.DeleteOptionValue(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete option value: %w", err)
	}

	// Publish event
	if err := s.publishOptionEvent(ctx, "option_value.deleted", int(value.ID), map[string]interface{}{
		"value_id":    value.ID,
		"option_id":   option.ID,
		"option_key":  option.OptionKey,
		"value":       value.Value,
		"usage_count": usage,
		"product_id":  option.ProductID,
	}); err != nil {
		log.Printf("Failed to publish option_value.deleted event: %v", err)
	}

	log.Printf("✅ Deleted option value %d (%s: %s)", value.ID, option.OptionKey, value.Value)
	return nil
}

// =============================================================================
// Customer-facing operations
// =============================================================================

func (s *OptionService) GetAvailableOptions(ctx context.Context, productID int32) ([]interfaces.ProductOptionWithValues, error) {
	// Validate product exists and is active
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	if !product.Active {
		return []interfaces.ProductOptionWithValues{}, nil // Return empty slice for inactive products
	}

	// Get available option values (only those used by active variants)
	availableOptions, err := s.optionRepo.GetAvailableOptionValues(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available options: %w", err)
	}

	// Group by option
	optionMap := make(map[int32]*interfaces.ProductOptionWithValues)
	for _, detail := range availableOptions {
		if _, exists := optionMap[detail.OptionID]; !exists {
			optionMap[detail.OptionID] = &interfaces.ProductOptionWithValues{
				ProductOption: interfaces.ProductOption{
					ID:        detail.OptionID,
					ProductID: productID,
					OptionKey: detail.OptionKey,
				},
				Values: []interfaces.ProductOptionValue{},
			}
		}

		optionMap[detail.OptionID].Values = append(optionMap[detail.OptionID].Values, interfaces.ProductOptionValue{
			ID:              detail.ValueID,
			ProductOptionID: detail.OptionID,
			Value:           detail.Value,
		})
	}

	// Convert map to slice
	result := make([]interfaces.ProductOptionWithValues, 0, len(optionMap))
	for _, option := range optionMap {
		result = append(result, *option)
	}

	return result, nil
}

func (s *OptionService) GetOptionCombinationsInStock(ctx context.Context, productID int32) ([]interfaces.OptionCombination, error) {
	// Validate product exists and is active
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	if !product.Active {
		return []interfaces.OptionCombination{}, nil
	}

	combinations, err := s.optionRepo.GetOptionCombinationsInStock(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option combinations: %w", err)
	}

	return combinations, nil
}

func (s *OptionService) FindVariantByOptions(ctx context.Context, req interfaces.FindVariantByOptionsRequest) (*interfaces.ProductVariant, error) {
	// Validate product exists and is active
	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", req.ProductID, err)
	}

	if !product.Active {
		return nil, fmt.Errorf("product %d is not active", req.ProductID)
	}

	// Find variant by option combination
	variant, err := s.optionRepo.GetVariantByOptionCombination(ctx, req.ProductID, req.OptionValueIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to find variant by options: %w", err)
	}

	// Check if variant is active and not archived
	if !variant.Active || variant.ArchivedAt.Valid {
		return nil, fmt.Errorf("variant is not available")
	}

	return variant, nil
}

// =============================================================================
// Analytics and management
// =============================================================================

func (s *OptionService) GetOptionUsageStats(ctx context.Context, optionID int32) (*interfaces.OptionUsageStats, error) {
	// Get the option
	option, err := s.optionRepo.GetProductOptionByID(ctx, optionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option %d: %w", optionID, err)
	}

	// Get usage count
	variantCount, err := s.optionRepo.CheckOptionUsage(ctx, optionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check option usage: %w", err)
	}

	// Count active variants (you might need to add this method to repository)
	// For now, we'll use the total variant count
	activeVariants := variantCount // Simplified - you may want to query active variants separately

	stats := &interfaces.OptionUsageStats{
		OptionID:       option.ID,
		OptionKey:      option.OptionKey,
		VariantCount:   variantCount,
		ActiveVariants: activeVariants,
		CanDelete:      variantCount == 0,
	}

	return stats, nil
}

func (s *OptionService) GetOptionPopularity(ctx context.Context, filters interfaces.OptionPopularityFilters) ([]interfaces.OptionPopularity, error) {
	// Validate product exists
	_, err := s.productRepo.GetByID(ctx, filters.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d: %w", filters.ProductID, err)
	}

	popularity, err := s.optionRepo.GetOptionPopularity(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get option popularity: %w", err)
	}

	return popularity, nil
}

func (s *OptionService) GetOrphanedOptions(ctx context.Context) ([]interfaces.ProductOption, error) {
	options, err := s.optionRepo.GetOrphanedOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned options: %w", err)
	}

	return options, nil
}

func (s *OptionService) GetOrphanedOptionValues(ctx context.Context) ([]interfaces.ProductOptionValue, error) {
	values, err := s.optionRepo.GetOrphanedOptionValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned option values: %w", err)
	}

	return values, nil
}

// =============================================================================
// Event Publishing
// =============================================================================

func (s *OptionService) publishOptionEvent(ctx context.Context, eventType string, optionID int, data map[string]interface{}) error {
	if s.events == nil {
		return nil // Events are optional
	}

	event := interfaces.Event{
		ID:          s.generateEventID(),
		Type:        eventType,
		AggregateID: fmt.Sprintf("option-%d", optionID),
		Data:        data,
		Timestamp:   time.Now(),
		Version:     1,
	}

	return s.events.PublishEvent(ctx, event)
}

func (s *OptionService) generateEventID() string {
	// Simple event ID generation - you might want to use a UUID library
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}