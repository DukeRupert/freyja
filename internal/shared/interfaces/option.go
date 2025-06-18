// internal/shared/interfaces/option.go
package interfaces

import (
	"context"
	"time"
)

// =============================================================================
// Option Domain Types
// =============================================================================

// ProductOption represents a product option (e.g., "Size", "Color")
type ProductOption struct {
	ID        int32     `json:"id"`
	ProductID int32     `json:"product_id"`
	OptionKey string    `json:"option_key"`
	CreatedAt time.Time `json:"created_at"`
}

// ProductOptionValue represents a value for an option (e.g., "Large", "Red")
type ProductOptionValue struct {
	ID               int32     `json:"id"`
	ProductOptionID  int32     `json:"product_option_id"`
	Value            string    `json:"value"`
	CreatedAt        time.Time `json:"created_at"`
}

// ProductOptionWithValues includes the option and its values
type ProductOptionWithValues struct {
	ProductOption
	Values []ProductOptionValue `json:"values"`
}

// OptionCombination represents a combination of options available in stock
type OptionCombination struct {
	VariantID int32                    `json:"variant_id"`
	Stock     int32                    `json:"stock"`
	Options   []OptionSelectionDetail  `json:"options"`
}

// OptionSelectionDetail represents a specific option selection
type OptionSelectionDetail struct {
	OptionID    int32  `json:"option_id"`
	OptionKey   string `json:"option_key"`
	ValueID     int32  `json:"value_id"`
	Value       string `json:"value"`
}

// OptionUsageStats represents usage statistics for an option
type OptionUsageStats struct {
	OptionID      int32 `json:"option_id"`
	OptionKey     string `json:"option_key"`
	VariantCount  int32 `json:"variant_count"`
	ActiveVariants int32 `json:"active_variants"`
	CanDelete     bool  `json:"can_delete"`
}

// OptionPopularity represents popularity data for option values
type OptionPopularity struct {
	OptionID      int32  `json:"option_id"`
	OptionKey     string `json:"option_key"`
	ValueID       int32  `json:"value_id"`
	Value         string `json:"value"`
	VariantCount  int64  `json:"variant_count"`
	OrderCount    int64  `json:"order_count"`
	TotalSold     int64  `json:"total_sold"`
}

// =============================================================================
// Request/Response Types
// =============================================================================

// CreateProductOptionRequest represents the data for creating a product option
type CreateProductOptionRequest struct {
	ProductID int32  `json:"product_id" validate:"required,min=1"`
	OptionKey string `json:"option_key" validate:"required,min=1,max=50"`
}

// UpdateProductOptionRequest represents the data for updating a product option
type UpdateProductOptionRequest struct {
	OptionKey string `json:"option_key" validate:"required,min=1,max=50"`
}

// CreateOptionValueRequest represents the data for creating an option value
type CreateOptionValueRequest struct {
	OptionID int32  `json:"option_id" validate:"required,min=1"`
	Value    string `json:"value" validate:"required,min=1,max=100"`
}

// UpdateOptionValueRequest represents the data for updating an option value
type UpdateOptionValueRequest struct {
	Value string `json:"value" validate:"required,min=1,max=100"`
}

// FindVariantByOptionsRequest represents a request to find a variant by option selections
type FindVariantByOptionsRequest struct {
	ProductID      int32   `json:"product_id" validate:"required,min=1"`
	OptionValueIDs []int32 `json:"option_value_ids" validate:"required,min=1,max=8"`
}

// OptionPopularityFilters represents filters for option popularity queries
type OptionPopularityFilters struct {
	ProductID int32   `json:"product_id"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
}

// =============================================================================
// Service Interface
// =============================================================================

type OptionService interface {
	// Product Option Management
	CreateProductOption(ctx context.Context, req CreateProductOptionRequest) (*ProductOption, error)
	GetProductOptionByID(ctx context.Context, id int32) (*ProductOption, error)
	GetProductOptions(ctx context.Context, productID int32) ([]ProductOptionWithValues, error)
	UpdateProductOption(ctx context.Context, id int32, req UpdateProductOptionRequest) (*ProductOption, error)
	DeleteProductOption(ctx context.Context, id int32) error

	// Option Value Management
	CreateOptionValue(ctx context.Context, req CreateOptionValueRequest) (*ProductOptionValue, error)
	GetOptionValueByID(ctx context.Context, id int32) (*ProductOptionValue, error)
	GetOptionValues(ctx context.Context, optionID int32) ([]ProductOptionValue, error)
	UpdateOptionValue(ctx context.Context, id int32, req UpdateOptionValueRequest) (*ProductOptionValue, error)
	DeleteOptionValue(ctx context.Context, id int32) error

	// Customer-facing operations
	GetAvailableOptions(ctx context.Context, productID int32) ([]ProductOptionWithValues, error)
	GetOptionCombinationsInStock(ctx context.Context, productID int32) ([]OptionCombination, error)
	FindVariantByOptions(ctx context.Context, req FindVariantByOptionsRequest) (*ProductVariant, error)

	// Analytics and management
	GetOptionUsageStats(ctx context.Context, optionID int32) (*OptionUsageStats, error)
	GetOptionPopularity(ctx context.Context, filters OptionPopularityFilters) ([]OptionPopularity, error)
	GetOrphanedOptions(ctx context.Context) ([]ProductOption, error)
	GetOrphanedOptionValues(ctx context.Context) ([]ProductOptionValue, error)
}

// =============================================================================
// Repository Interface
// =============================================================================

type OptionRepository interface {
	// Product Option CRUD
	CreateProductOption(ctx context.Context, req CreateProductOptionRequest) (*ProductOption, error)
	GetProductOptionByID(ctx context.Context, id int32) (*ProductOption, error)
	GetProductOptionsByProduct(ctx context.Context, productID int32) ([]ProductOption, error)
	GetProductOptionByKey(ctx context.Context, productID int32, optionKey string) (*ProductOption, error)
	UpdateProductOption(ctx context.Context, id int32, req UpdateProductOptionRequest) (*ProductOption, error)
	DeleteProductOption(ctx context.Context, id int32) error

	// Option Value CRUD
	CreateOptionValue(ctx context.Context, req CreateOptionValueRequest) (*ProductOptionValue, error)
	GetOptionValueByID(ctx context.Context, id int32) (*ProductOptionValue, error)
	GetOptionValuesByOption(ctx context.Context, optionID int32) ([]ProductOptionValue, error)
	GetOptionValuesByProduct(ctx context.Context, productID int32) ([]ProductOptionValue, error)
	GetOptionValueByValue(ctx context.Context, optionID int32, value string) (*ProductOptionValue, error)
	UpdateOptionValue(ctx context.Context, id int32, req UpdateOptionValueRequest) (*ProductOptionValue, error)
	DeleteOptionValue(ctx context.Context, id int32) error

	// Variant Option Management
	CreateVariantOption(ctx context.Context, variantID, optionID, valueID int32) error
	GetVariantOptionsByVariant(ctx context.Context, variantID int32) ([]OptionSelectionDetail, error)
	GetVariantOptionsByProduct(ctx context.Context, productID int32) ([]OptionSelectionDetail, error)
	DeleteVariantOptionsByVariant(ctx context.Context, variantID int32) error

	// Complex queries
	GetVariantByOptionCombination(ctx context.Context, productID int32, optionValueIDs []int32) (*ProductVariant, error)
	GetAvailableOptionValues(ctx context.Context, productID int32) ([]OptionSelectionDetail, error)
	GetOptionCombinationsInStock(ctx context.Context, productID int32) ([]OptionCombination, error)

	// Validation and analytics
	CheckOptionUsage(ctx context.Context, optionID int32) (int32, error)
	CheckOptionValueUsage(ctx context.Context, valueID int32) (int32, error)
	GetOrphanedOptions(ctx context.Context) ([]ProductOption, error)
	GetOrphanedOptionValues(ctx context.Context) ([]ProductOptionValue, error)
	GetOptionPopularity(ctx context.Context, filters OptionPopularityFilters) ([]OptionPopularity, error)
}