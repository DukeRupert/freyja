// internal/server/repository/variant.go
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

type PostgresVariantRepository struct {
	db *database.DB
}

func NewPostgresVariantRepository(db *database.DB) interfaces.VariantRepository {
	return &PostgresVariantRepository{
		db: db,
	}
}

// =============================================================================
// Basic variant operations
// =============================================================================

func (r *PostgresVariantRepository) GetByID(ctx context.Context, id int32) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.GetVariant(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("variant not found")
		}
		return nil, fmt.Errorf("failed to get variant: %w", err)
	}

	return r.convertToVariant(variant), nil
}

func (r *PostgresVariantRepository) GetByIDWithOptions(ctx context.Context, id int32) (*interfaces.ProductVariantWithOptions, error) {
	variantWithOptions, err := r.db.Queries.GetVariantWithOptions(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("variant not found")
		}
		return nil, fmt.Errorf("failed to get variant with options: %w", err)
	}

	return r.convertToVariantWithOptions(variantWithOptions), nil
}

func (r *PostgresVariantRepository) GetByStripeProductID(ctx context.Context, stripeProductID string) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.GetVariantByStripeProductID(ctx, stringToPgText(stripeProductID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("variant not found")
		}
		return nil, fmt.Errorf("failed to get variant by Stripe product ID: %w", err)
	}

	return r.convertToVariant(variant), nil
}

func (r *PostgresVariantRepository) Create(ctx context.Context, req interfaces.CreateVariantRequest) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.CreateVariant(ctx, database.CreateVariantParams{
		ProductID:      req.ProductID,
		Name:           req.Name,
		Price:          req.Price,
		Stock:          req.Stock,
		Active:         req.Active,
		IsSubscription: req.IsSubscription,
		OptionsDisplay: pgtype.Text{String: req.OptionsDisplay, Valid: req.OptionsDisplay != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create variant: %w", err)
	}

	created := r.convertToVariant(variant)

	// Create variant option associations if provided
	if len(req.OptionValueIDs) > 0 {
		for _, valueID := range req.OptionValueIDs {
			// Get the option ID for this value
			optionValue, err := r.db.Queries.GetProductOptionValue(ctx, valueID)
			if err != nil {
				return nil, fmt.Errorf("failed to get option value %d: %w", valueID, err)
			}

			_, err = r.db.Queries.CreateVariantOption(ctx, database.CreateVariantOptionParams{
				ProductVariantID:     created.ID,
				ProductOptionID:      optionValue.ProductOptionID,
				ProductOptionValueID: valueID,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create variant option: %w", err)
			}
		}
	}

	return created, nil
}

func (r *PostgresVariantRepository) Update(ctx context.Context, id int32, req interfaces.UpdateVariantRequest) (*interfaces.ProductVariant, error) {
	// Build update parameters
	params := database.UpdateVariantParams{
		ID: id,
	}

	if req.Name != nil {
		params.Name = pgtype.Text{String: *req.Name, Valid: true}
	}
	if req.Price != nil {
		params.Price = pgtype.Int4{Int32: *req.Price, Valid: true}
	}
	if req.Stock != nil {
		params.Stock = pgtype.Int4{Int32: *req.Stock, Valid: true}
	}
	if req.Active != nil {
		params.Active = *req.Active
	}
	if req.IsSubscription != nil {
		params.IsSubscription = *req.IsSubscription
	}
	if req.OptionsDisplay != nil {
		params.OptionsDisplay = pgtype.Text{String: *req.OptionsDisplay, Valid: true}
	}

	variant, err := r.db.Queries.UpdateVariant(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update variant: %w", err)
	}

	return r.convertToVariant(variant), nil
}

func (r *PostgresVariantRepository) Archive(ctx context.Context, id int32) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.ArchiveVariant(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to archive variant: %w", err)
	}

	return r.convertToVariant(variant), nil
}

func (r *PostgresVariantRepository) Unarchive(ctx context.Context, id int32) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.UnarchiveVariant(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to unarchive variant: %w", err)
	}

	return r.convertToVariant(variant), nil
}

func (r *PostgresVariantRepository) Activate(ctx context.Context, id int32) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.ActivateVariant(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to activate variant: %w", err)
	}

	return r.convertToVariant(variant), nil
}

func (r *PostgresVariantRepository) Deactivate(ctx context.Context, id int32) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.DeactivateVariant(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate variant: %w", err)
	}

	return r.convertToVariant(variant), nil
}

// =============================================================================
// Product-specific variant operations
// =============================================================================

func (r *PostgresVariantRepository) GetVariantsByProduct(ctx context.Context, productID int32) ([]interfaces.ProductVariant, error) {
	dbVariants, err := r.db.Queries.GetVariantsByProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variants by product: %w", err)
	}

	variants := make([]interfaces.ProductVariant, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariant(dbVariant)
	}

	return variants, nil
}

func (r *PostgresVariantRepository) GetActiveVariantsByProduct(ctx context.Context, productID int32) ([]interfaces.ProductVariant, error) {
	dbVariants, err := r.db.Queries.GetActiveVariantsByProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active variants by product: %w", err)
	}

	variants := make([]interfaces.ProductVariant, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariant(dbVariant)
	}

	return variants, nil
}

func (r *PostgresVariantRepository) GetVariantsInStock(ctx context.Context, productID int32) ([]interfaces.ProductVariant, error) {
	dbVariants, err := r.db.Queries.GetVariantsInStock(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variants in stock: %w", err)
	}

	variants := make([]interfaces.ProductVariant, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariant(dbVariant)
	}

	return variants, nil
}

func (r *PostgresVariantRepository) GetVariantsWithOptions(ctx context.Context, productID int32) ([]interfaces.ProductVariantWithOptions, error) {
	dbVariants, err := r.db.Queries.GetVariantsWithOptionValues(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variants with options: %w", err)
	}

	variants := make([]interfaces.ProductVariantWithOptions, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariantWithOptionsFromList(dbVariant)
	}

	return variants, nil
}

// =============================================================================
// Stock management
// =============================================================================

func (r *PostgresVariantRepository) UpdateStock(ctx context.Context, id int32, stock int32) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.UpdateVariantStock(ctx, database.UpdateVariantStockParams{
		ID:    id,
		Stock: stock,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update variant stock: %w", err)
	}

	return r.convertToVariant(variant), nil
}

func (r *PostgresVariantRepository) IncrementStock(ctx context.Context, id int32, delta int32) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.IncrementVariantStock(ctx, database.IncrementVariantStockParams{
		ID:    id,
		Stock: delta,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to increment variant stock: %w", err)
	}

	return r.convertToVariant(variant), nil
}

func (r *PostgresVariantRepository) DecrementStock(ctx context.Context, id int32, delta int32) (*interfaces.ProductVariant, error) {
	variant, err := r.db.Queries.DecrementVariantStock(ctx, database.DecrementVariantStockParams{
		ID:    id,
		Stock: delta,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrement variant stock: %w", err)
	}

	return r.convertToVariant(variant), nil
}

// =============================================================================
// Search and filtering
// =============================================================================

func (r *PostgresVariantRepository) SearchVariants(ctx context.Context, query string) ([]interfaces.ProductVariant, error) {
	dbVariants, err := r.db.Queries.SearchVariants(ctx, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search variants: %w", err)
	}

	variants := make([]interfaces.ProductVariant, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariantFromSearch(dbVariant)
	}

	return variants, nil
}

func (r *PostgresVariantRepository) GetVariantsByPriceRange(ctx context.Context, minPrice, maxPrice int32) ([]interfaces.ProductVariant, error) {
	dbVariants, err := r.db.Queries.GetVariantsByPriceRange(ctx, database.GetVariantsByPriceRangeParams{
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get variants by price range: %w", err)
	}

	variants := make([]interfaces.ProductVariant, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariant(dbVariant)
	}

	return variants, nil
}

func (r *PostgresVariantRepository) GetSubscriptionVariants(ctx context.Context) ([]interfaces.ProductVariant, error) {
	dbVariants, err := r.db.Queries.GetSubscriptionVariants(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription variants: %w", err)
	}

	variants := make([]interfaces.ProductVariant, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariant(dbVariant)
	}

	return variants, nil
}

func (r *PostgresVariantRepository) GetLowStockVariants(ctx context.Context, threshold int32) ([]interfaces.ProductVariant, error) {
	dbVariants, err := r.db.Queries.GetLowStockVariants(ctx, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get low stock variants: %w", err)
	}

	variants := make([]interfaces.ProductVariant, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariantFromLowStock(dbVariant)
	}

	return variants, nil
}

// =============================================================================
// Stripe integration
// =============================================================================

func (r *PostgresVariantRepository) UpdateStripeIDs(ctx context.Context, id int32, stripeProductID string, priceIDs map[string]string) (*interfaces.ProductVariant, error) {
	params := database.UpdateVariantStripeIDsParams{
		ID: id,
	}

	if stripeProductID != "" {
		params.StripeProductID = pgtype.Text{String: stripeProductID, Valid: true}
	}

	if onetimeID, exists := priceIDs["onetime"]; exists && onetimeID != "" {
		params.StripePriceOnetimeID = pgtype.Text{String: onetimeID, Valid: true}
	}
	if day14ID, exists := priceIDs["14day"]; exists && day14ID != "" {
		params.StripePrice14dayID = pgtype.Text{String: day14ID, Valid: true}
	}
	if day21ID, exists := priceIDs["21day"]; exists && day21ID != "" {
		params.StripePrice21dayID = pgtype.Text{String: day21ID, Valid: true}
	}
	if day30ID, exists := priceIDs["30day"]; exists && day30ID != "" {
		params.StripePrice30dayID = pgtype.Text{String: day30ID, Valid: true}
	}
	if day60ID, exists := priceIDs["60day"]; exists && day60ID != "" {
		params.StripePrice60dayID = pgtype.Text{String: day60ID, Valid: true}
	}

	variant, err := r.db.Queries.UpdateVariantStripeIDs(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update variant Stripe IDs: %w", err)
	}

	return r.convertToVariant(variant), nil
}

func (r *PostgresVariantRepository) GetVariantsNeedingStripeSync(ctx context.Context, limit, offset int32) ([]interfaces.ProductVariant, error) {
	dbVariants, err := r.db.Queries.GetVariantsNeedingStripeSync(ctx, database.GetVariantsNeedingStripeSyncParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get variants needing Stripe sync: %w", err)
	}

	variants := make([]interfaces.ProductVariant, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariant(dbVariant)
	}

	return variants, nil
}

func (r *PostgresVariantRepository) GetVariantsWithStripeProducts(ctx context.Context, limit, offset int32) ([]interfaces.ProductVariant, error) {
	dbVariants, err := r.db.Queries.GetVariantsWithStripeProducts(ctx, database.GetVariantsWithStripeProductsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get variants with Stripe products: %w", err)
	}

	variants := make([]interfaces.ProductVariant, len(dbVariants))
	for i, dbVariant := range dbVariants {
		variants[i] = *r.convertToVariant(dbVariant)
	}

	return variants, nil
}

// =============================================================================
// Analytics (stub implementations - implement as needed)
// =============================================================================

func (r *PostgresVariantRepository) GetVariantSalesStats(ctx context.Context, dateFrom, dateTo *time.Time, limit, offset int32) ([]interfaces.VariantSalesStats, error) {
	// Implementation would use GetVariantSalesStats query
	return nil, fmt.Errorf("not implemented yet")
}

func (r *PostgresVariantRepository) GetTopSellingVariants(ctx context.Context, dateFrom, dateTo *time.Time, limit, offset int32) ([]interfaces.TopSellingVariant, error) {
	// Implementation would use GetTopSellingVariants query
	return nil, fmt.Errorf("not implemented yet")
}

// =============================================================================
// Helper conversion methods
// =============================================================================

func (r *PostgresVariantRepository) convertToVariant(dbVariant database.ProductVariants) *interfaces.ProductVariant {
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

func (r *PostgresVariantRepository) convertToVariantWithOptions(dbVariant database.GetVariantWithOptionsRow) *interfaces.ProductVariantWithOptions {
	variant := &interfaces.ProductVariantWithOptions{
		ProductVariant: interfaces.ProductVariant{
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
		},
	}

	// Parse options from JSON (this would need proper JSON parsing)
	// For now, this is a placeholder
	variant.Options = []interfaces.VariantOption{}

	return variant
}

// Helper methods for different query result types
func (r *PostgresVariantRepository) convertToVariantFromSearch(dbVariant database.SearchVariantsRow) *interfaces.ProductVariant {
	return &interfaces.ProductVariant{
		ID:             dbVariant.ID,
		ProductID:      dbVariant.ProductID,
		Name:           dbVariant.Name,
		Price:          dbVariant.Price,
		Stock:          dbVariant.Stock,
		Active:         dbVariant.Active,
		IsSubscription: dbVariant.IsSubscription,
		ArchivedAt:     dbVariant.ArchivedAt,
		CreatedAt:      dbVariant.CreatedAt,
		UpdatedAt:      dbVariant.UpdatedAt,
		StripeProductID:      dbVariant.StripeProductID,
		StripePriceOnetimeID: dbVariant.StripePriceOnetimeID,
		StripePrice14dayID:   dbVariant.StripePrice14dayID,
		StripePrice21dayID:   dbVariant.StripePrice21dayID,
		StripePrice30dayID:   dbVariant.StripePrice30dayID,
		StripePrice60dayID:   dbVariant.StripePrice60dayID,
		OptionsDisplay:       dbVariant.OptionsDisplay,
	}
}

func (r *PostgresVariantRepository) convertToVariantFromLowStock(dbVariant database.GetLowStockVariantsRow) *interfaces.ProductVariant {
	return &interfaces.ProductVariant{
		ID:             dbVariant.ID,
		ProductID:      dbVariant.ProductID,
		Name:           dbVariant.Name,
		Price:          dbVariant.Price,
		Stock:          dbVariant.Stock,
		Active:         dbVariant.Active,
		IsSubscription: dbVariant.IsSubscription,
		ArchivedAt:     dbVariant.ArchivedAt,
		CreatedAt:      dbVariant.CreatedAt,
		UpdatedAt:      dbVariant.UpdatedAt,
		StripeProductID:      dbVariant.StripeProductID,
		StripePriceOnetimeID: dbVariant.StripePriceOnetimeID,
		StripePrice14dayID:   dbVariant.StripePrice14dayID,
		StripePrice21dayID:   dbVariant.StripePrice21dayID,
		StripePrice30dayID:   dbVariant.StripePrice30dayID,
		StripePrice60dayID:   dbVariant.StripePrice60dayID,
		OptionsDisplay:       dbVariant.OptionsDisplay,
	}
}

func (r *PostgresVariantRepository) convertToVariantWithOptionsFromList(dbVariant database.GetVariantsWithOptionValuesRow) *interfaces.ProductVariantWithOptions {
	variant := &interfaces.ProductVariantWithOptions{
		ProductVariant: interfaces.ProductVariant{
			ID:             dbVariant.VariantID,
			Name:           dbVariant.VariantName,
			Price:          dbVariant.Price,
			Stock:          dbVariant.Stock,
			Active:         dbVariant.Active,
			// Note: This query result may not have all fields, adjust as needed
		},
	}

	// Parse options from JSON (placeholder implementation)
	variant.Options = []interfaces.VariantOption{}

	return variant
}

func stringToPgText(s string) pgtype.Text {
    return pgtype.Text{
        String: s,
        Valid:  s != "",
    }
}