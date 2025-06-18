// internal/server/repository/option.go
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresOptionRepository struct {
	db *database.DB
}

func NewPostgresOptionRepository(db *database.DB) interfaces.OptionRepository {
	return &PostgresOptionRepository{
		db: db,
	}
}

// =============================================================================
// Product Option CRUD
// =============================================================================

func (r *PostgresOptionRepository) CreateProductOption(ctx context.Context, req interfaces.CreateProductOptionRequest) (*interfaces.ProductOption, error) {
	option, err := r.db.Queries.CreateProductOption(ctx, database.CreateProductOptionParams{
		ProductID: req.ProductID,
		OptionKey: req.OptionKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create product option: %w", err)
	}

	return r.convertToProductOption(option), nil
}

func (r *PostgresOptionRepository) GetProductOptionByID(ctx context.Context, id int32) (*interfaces.ProductOption, error) {
	option, err := r.db.Queries.GetProductOption(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product option not found")
		}
		return nil, fmt.Errorf("failed to get product option: %w", err)
	}

	return r.convertToProductOption(option), nil
}

func (r *PostgresOptionRepository) GetProductOptionsByProduct(ctx context.Context, productID int32) ([]interfaces.ProductOption, error) {
	dbOptions, err := r.db.Queries.GetProductOptionsByProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product options: %w", err)
	}

	options := make([]interfaces.ProductOption, len(dbOptions))
	for i, dbOption := range dbOptions {
		options[i] = *r.convertToProductOption(dbOption)
	}

	return options, nil
}

func (r *PostgresOptionRepository) GetProductOptionByKey(ctx context.Context, productID int32, optionKey string) (*interfaces.ProductOption, error) {
	option, err := r.db.Queries.GetProductOptionByKey(ctx, database.GetProductOptionByKeyParams{
		ProductID: productID,
		OptionKey: optionKey,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product option not found")
		}
		return nil, fmt.Errorf("failed to get product option by key: %w", err)
	}

	return r.convertToProductOption(option), nil
}

func (r *PostgresOptionRepository) UpdateProductOption(ctx context.Context, id int32, req interfaces.UpdateProductOptionRequest) (*interfaces.ProductOption, error) {
	option, err := r.db.Queries.UpdateProductOption(ctx, database.UpdateProductOptionParams{
		ID:        id,
		OptionKey: req.OptionKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update product option: %w", err)
	}

	return r.convertToProductOption(option), nil
}

func (r *PostgresOptionRepository) DeleteProductOption(ctx context.Context, id int32) error {
	err := r.db.Queries.DeleteProductOption(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete product option: %w", err)
	}

	return nil
}

// =============================================================================
// Option Value CRUD
// =============================================================================

func (r *PostgresOptionRepository) CreateOptionValue(ctx context.Context, req interfaces.CreateOptionValueRequest) (*interfaces.ProductOptionValue, error) {
	value, err := r.db.Queries.CreateProductOptionValue(ctx, database.CreateProductOptionValueParams{
		ProductOptionID: req.OptionID,
		Value:           req.Value,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create option value: %w", err)
	}

	return r.convertToProductOptionValue(value), nil
}

func (r *PostgresOptionRepository) GetOptionValueByID(ctx context.Context, id int32) (*interfaces.ProductOptionValue, error) {
	value, err := r.db.Queries.GetProductOptionValue(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("option value not found")
		}
		return nil, fmt.Errorf("failed to get option value: %w", err)
	}

	return r.convertToProductOptionValue(value), nil
}

func (r *PostgresOptionRepository) GetOptionValuesByOption(ctx context.Context, optionID int32) ([]interfaces.ProductOptionValue, error) {
	dbValues, err := r.db.Queries.GetProductOptionValuesByOption(ctx, optionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option values: %w", err)
	}

	values := make([]interfaces.ProductOptionValue, len(dbValues))
	for i, dbValue := range dbValues {
		values[i] = *r.convertToProductOptionValue(dbValue)
	}

	return values, nil
}

func (r *PostgresOptionRepository) GetOptionValuesByProduct(ctx context.Context, productID int32) ([]interfaces.ProductOptionValue, error) {
	dbValues, err := r.db.Queries.GetProductOptionValuesByProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option values by product: %w", err)
	}

	values := make([]interfaces.ProductOptionValue, len(dbValues))
	for i, dbValue := range dbValues {
		values[i] = interfaces.ProductOptionValue{
			ID:              dbValue.ID,
			ProductOptionID: dbValue.ProductOptionID,
			Value:           dbValue.Value,
			CreatedAt:       dbValue.CreatedAt,
		}
	}

	return values, nil
}

func (r *PostgresOptionRepository) GetOptionValueByValue(ctx context.Context, optionID int32, value string) (*interfaces.ProductOptionValue, error) {
	optionValue, err := r.db.Queries.GetProductOptionValueByValue(ctx, database.GetProductOptionValueByValueParams{
		ProductOptionID: optionID,
		Value:           value,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("option value not found")
		}
		return nil, fmt.Errorf("failed to get option value by value: %w", err)
	}

	return r.convertToProductOptionValue(optionValue), nil
}

func (r *PostgresOptionRepository) UpdateOptionValue(ctx context.Context, id int32, req interfaces.UpdateOptionValueRequest) (*interfaces.ProductOptionValue, error) {
	value, err := r.db.Queries.UpdateProductOptionValue(ctx, database.UpdateProductOptionValueParams{
		ID:    id,
		Value: req.Value,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update option value: %w", err)
	}

	return r.convertToProductOptionValue(value), nil
}

func (r *PostgresOptionRepository) DeleteOptionValue(ctx context.Context, id int32) error {
	err := r.db.Queries.DeleteProductOptionValue(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete option value: %w", err)
	}

	return nil
}

// =============================================================================
// Variant Option Management
// =============================================================================

func (r *PostgresOptionRepository) CreateVariantOption(ctx context.Context, variantID, optionID, valueID int32) error {
	_, err := r.db.Queries.CreateVariantOption(ctx, database.CreateVariantOptionParams{
		ProductVariantID:     variantID,
		ProductOptionID:      optionID,
		ProductOptionValueID: valueID,
	})
	if err != nil {
		return fmt.Errorf("failed to create variant option: %w", err)
	}

	return nil
}

func (r *PostgresOptionRepository) GetVariantOptionsByVariant(ctx context.Context, variantID int32) ([]interfaces.OptionSelectionDetail, error) {
	dbOptions, err := r.db.Queries.GetVariantOptionsByVariant(ctx, variantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variant options: %w", err)
	}

	options := make([]interfaces.OptionSelectionDetail, len(dbOptions))
	for i, dbOption := range dbOptions {
		options[i] = interfaces.OptionSelectionDetail{
			OptionID:  dbOption.ProductOptionID,
			OptionKey: dbOption.OptionKey,
			ValueID:   dbOption.ProductOptionValueID,
			Value:     dbOption.Value,
		}
	}

	return options, nil
}

func (r *PostgresOptionRepository) GetVariantOptionsByProduct(ctx context.Context, productID int32) ([]interfaces.OptionSelectionDetail, error) {
	dbOptions, err := r.db.Queries.GetVariantOptionsByProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variant options by product: %w", err)
	}

	options := make([]interfaces.OptionSelectionDetail, len(dbOptions))
	for i, dbOption := range dbOptions {
		options[i] = interfaces.OptionSelectionDetail{
			OptionID:  dbOption.ProductOptionID,
			OptionKey: dbOption.OptionKey,
			ValueID:   dbOption.ProductOptionValueID,
			Value:     dbOption.Value,
		}
	}

	return options, nil
}

func (r *PostgresOptionRepository) DeleteVariantOptionsByVariant(ctx context.Context, variantID int32) error {
	err := r.db.Queries.DeleteVariantOptionsByVariant(ctx, variantID)
	if err != nil {
		return fmt.Errorf("failed to delete variant options: %w", err)
	}

	return nil
}

// =============================================================================
// Complex queries
// =============================================================================

func (r *PostgresOptionRepository) GetVariantByOptionCombination(ctx context.Context, productID int32, optionValueIDs []int32) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.GetVariantByOptionCombination(ctx, database.GetVariantByOptionCombinationParams{
		ProductID: productID,
		Column2:   optionValueIDs,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no variant found for option combination")
		}
		return nil, fmt.Errorf("failed to get variant by option combination: %w", err)
	}

	return r.convertVariantToProductVariant(variant), nil
}

func (r *PostgresOptionRepository) GetAvailableOptionValues(ctx context.Context, productID int32) ([]interfaces.OptionSelectionDetail, error) {
	dbOptions, err := r.db.Queries.GetAvailableOptionValues(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available option values: %w", err)
	}

	options := make([]interfaces.OptionSelectionDetail, len(dbOptions))
	for i, dbOption := range dbOptions {
		options[i] = interfaces.OptionSelectionDetail{
			OptionID:  dbOption.OptionID,
			OptionKey: dbOption.OptionKey,
			ValueID:   dbOption.ValueID,
			Value:     dbOption.Value,
		}
	}

	return options, nil
}

func (r *PostgresOptionRepository) GetOptionCombinationsInStock(ctx context.Context, productID int32) ([]interfaces.OptionCombination, error) {
	dbCombinations, err := r.db.Queries.GetOptionCombinationsInStock(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get option combinations in stock: %w", err)
	}

	combinations := make([]interfaces.OptionCombination, len(dbCombinations))
	for i, dbCombination := range dbCombinations {
		// Parse the JSON option combination
		// Note: You'll need to implement JSON parsing here based on your actual JSON structure
		// This is a simplified version
		combinations[i] = interfaces.OptionCombination{
			VariantID: dbCombination.VariantID,
			Stock:     dbCombination.Stock,
			Options:   []interfaces.OptionSelectionDetail{}, // Parse from JSON
		}
	}

	return combinations, nil
}

// =============================================================================
// Validation and analytics
// =============================================================================

func (r *PostgresOptionRepository) CheckOptionUsage(ctx context.Context, optionID int32) (int32, error) {
	usage, err := r.db.Queries.CheckOptionUsage(ctx, optionID)
	if err != nil {
		return 0, fmt.Errorf("failed to check option usage: %w", err)
	}

	return int32(usage), nil
}

func (r *PostgresOptionRepository) CheckOptionValueUsage(ctx context.Context, valueID int32) (int32, error) {
	usage, err := r.db.Queries.CheckOptionValueUsage(ctx, valueID)
	if err != nil {
		return 0, fmt.Errorf("failed to check option value usage: %w", err)
	}

	return int32(usage), nil
}

func (r *PostgresOptionRepository) GetOrphanedOptions(ctx context.Context) ([]interfaces.ProductOption, error) {
	dbOptions, err := r.db.Queries.GetOrphanedOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned options: %w", err)
	}

	options := make([]interfaces.ProductOption, len(dbOptions))
	for i, dbOption := range dbOptions {
		options[i] = *r.convertToProductOption(dbOption)
	}

	return options, nil
}

func (r *PostgresOptionRepository) GetOrphanedOptionValues(ctx context.Context) ([]interfaces.ProductOptionValue, error) {
	dbValues, err := r.db.Queries.GetOrphanedOptionValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned option values: %w", err)
	}

	values := make([]interfaces.ProductOptionValue, len(dbValues))
	for i, dbValue := range dbValues {
		values[i] = *r.convertToProductOptionValue(dbValue)
	}

	return values, nil
}

func (r *PostgresOptionRepository) GetOptionPopularity(ctx context.Context, filters interfaces.OptionPopularityFilters) ([]interfaces.OptionPopularity, error) {
	// Convert date strings to pgtype.Timestamp
	var startDate, endDate pgtype.Timestamp

	if filters.StartDate != nil {
		// Parse the date string and convert to pgtype.Timestamp
		// Assuming the date string is in RFC3339 format (e.g., "2023-01-01T00:00:00Z")
		if parsedTime, err := time.Parse(time.RFC3339, *filters.StartDate); err == nil {
			startDate = pgtype.Timestamp{Time: parsedTime, Valid: true}
		} else {
			return nil, fmt.Errorf("invalid start_date format: %w", err)
		}
	}
	// startDate remains zero value (Valid: false) if not provided

	if filters.EndDate != nil {
		// Parse the date string and convert to pgtype.Timestamp
		if parsedTime, err := time.Parse(time.RFC3339, *filters.EndDate); err == nil {
			endDate = pgtype.Timestamp{Time: parsedTime, Valid: true}
		} else {
			return nil, fmt.Errorf("invalid end_date format: %w", err)
		}
	}
	// endDate remains zero value (Valid: false) if not provided

	dbPopularity, err := r.db.Queries.GetOptionPopularity(ctx, database.GetOptionPopularityParams{
		ProductID: filters.ProductID,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get option popularity: %w", err)
	}

	popularity := make([]interfaces.OptionPopularity, len(dbPopularity))
	for i, dbPop := range dbPopularity {
		// Handle type assertion for TotalSold since COALESCE returns interface{}
		var totalSold int64
		if dbPop.TotalSold != nil {
			switch v := dbPop.TotalSold.(type) {
			case int64:
				totalSold = v
			case int32:
				totalSold = int64(v)
			case int:
				totalSold = int64(v)
			default:
				// Fallback to 0 if type assertion fails
				totalSold = 0
			}
		}
		popularity[i] = interfaces.OptionPopularity{
			OptionID:     dbPop.ID,
			OptionKey:    dbPop.OptionKey,
			ValueID:      dbPop.ValueID,
			Value:        dbPop.Value,
			VariantCount: dbPop.VariantCount,
			OrderCount:   dbPop.OrderCount,
			TotalSold:    totalSold,
		}
	}

	return popularity, nil
}

// =============================================================================
// Helper conversion methods
// =============================================================================

func (r *PostgresOptionRepository) convertToProductOption(dbOption database.ProductOptions) *interfaces.ProductOption {
	return &interfaces.ProductOption{
		ID:        dbOption.ID,
		ProductID: dbOption.ProductID,
		OptionKey: dbOption.OptionKey,
		CreatedAt: dbOption.CreatedAt,
	}
}

func (r *PostgresOptionRepository) convertToProductOptionValue(dbValue database.ProductOptionValues) *interfaces.ProductOptionValue {
	return &interfaces.ProductOptionValue{
		ID:              dbValue.ID,
		ProductOptionID: dbValue.ProductOptionID,
		Value:           dbValue.Value,
		CreatedAt:       dbValue.CreatedAt,
	}
}

func (r *PostgresOptionRepository) convertVariantToProductVariant(dbVariant database.ProductVariants) *interfaces.ProductVariant {
	return &interfaces.ProductVariant{
		ID:                   dbVariant.ID,
		ProductID:            dbVariant.ProductID,
		Name:                 dbVariant.Name,
		Price:                dbVariant.Price,
		Stock:                dbVariant.Stock,
		Active:               dbVariant.Active,
		IsSubscription:       dbVariant.IsSubscription,
		ArchivedAt:           dbVariant.ArchivedAt,
		CreatedAt:            dbVariant.CreatedAt,
		UpdatedAt:            dbVariant.UpdatedAt,
		StripeProductID:      dbVariant.StripeProductID,
		StripePriceOnetimeID: dbVariant.StripePriceOnetimeID,
		StripePrice14dayID:   dbVariant.StripePrice14dayID,
		StripePrice21dayID:   dbVariant.StripePrice21dayID,
		StripePrice30dayID:   dbVariant.StripePrice30dayID,
		StripePrice60dayID:   dbVariant.StripePrice60dayID,
		OptionsDisplay:       dbVariant.OptionsDisplay,
	}
}
