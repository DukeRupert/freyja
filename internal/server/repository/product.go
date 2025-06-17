// internal/server/repository/product.go
package repository

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresProductRepository struct {
	db *database.DB
}

func NewPostgresProductRepository(db *database.DB) interfaces.ProductRepository {
	return &PostgresProductRepository{
		db: db,
	}
}

// =============================================================================
// Basic Product Operations (for admin/management)
// =============================================================================

func (r *PostgresProductRepository) GetByID(ctx context.Context, id int32) (*interfaces.Product, error) {
	product, err := r.db.Queries.GetProduct(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &interfaces.Product{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Active:      product.Active,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}, nil
}

func (r *PostgresProductRepository) GetByName(ctx context.Context, name string) (*interfaces.Product, error) {
	product, err := r.db.Queries.GetProductByName(ctx, name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &interfaces.Product{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Active:      product.Active,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}, nil
}

// =============================================================================
// Product Summary Operations (using materialized view)
// =============================================================================

func (r *PostgresProductRepository) GetProductWithSummary(ctx context.Context, id int32) (*interfaces.ProductSummary, error) {
	summary, err := r.db.Queries.GetProductWithSummary(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product summary: %w", err)
	}

	return r.convertToProductSummary(summary), nil
}

func (r *PostgresProductRepository) GetAllWithSummary(ctx context.Context, filters interfaces.ProductFilters) ([]interfaces.ProductSummary, error) {
	var dbSummaries []database.ProductStockSummary
	var err error

	// Handle different filter combinations
	if filters.Active != nil {
		limit := int32(50) // default limit
		offset := int32(0)

		if filters.Limit > 0 {
			limit = int32(filters.Limit)
		}
		if filters.Offset > 0 {
			offset = int32(filters.Offset)
		}

		dbSummaries, err = r.db.Queries.ListProductsByStatus(ctx, database.ListProductsByStatusParams{
			ProductActive: *filters.Active,
			Limit:  limit,
			Offset: offset,
		})
	} else if filters.Limit > 0 || filters.Offset > 0 {
		limit := int32(50)
		offset := int32(0)

		if filters.Limit > 0 {
			limit = int32(filters.Limit)
		}
		if filters.Offset > 0 {
			offset = int32(filters.Offset)
		}

		dbSummaries, err = r.db.Queries.ListAllProducts(ctx, database.ListAllProductsParams{
			Limit:  limit,
			Offset: offset,
		})
	} else {
		// Default to active products only
		dbSummaries, err = r.db.Queries.ListProducts(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	// Convert to interface type
	var summaries []interfaces.ProductSummary
	for _, dbSummary := range dbSummaries {
		summaries = append(summaries, *r.convertToProductSummary(dbSummary))
	}

	return summaries, nil
}

func (r *PostgresProductRepository) GetProductsInStock(ctx context.Context) ([]interfaces.ProductSummary, error) {
	dbSummaries, err := r.db.Queries.GetProductsInStock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get in-stock products: %w", err)
	}

	var summaries []interfaces.ProductSummary
	for _, dbSummary := range dbSummaries {
		summaries = append(summaries, *r.convertToProductSummary(dbSummary))
	}

	return summaries, nil
}

func (r *PostgresProductRepository) SearchProductsWithOptions(ctx context.Context, query string) ([]interfaces.ProductSummary, error) {
	dbSummaries, err := r.db.Queries.SearchProductsWithOptions(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	var summaries []interfaces.ProductSummary
	for _, dbSummary := range dbSummaries {
		summaries = append(summaries, *r.convertToProductSummary(dbSummary))
	}

	return summaries, nil
}

// =============================================================================
// Product Management Operations
// =============================================================================

func (r *PostgresProductRepository) Create(ctx context.Context, req interfaces.CreateProductRequest) (*interfaces.Product, error) {
	created, err := r.db.Queries.CreateProduct(ctx, database.CreateProductParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Active:      req.Active,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return &interfaces.Product{
		ID:          created.ID,
		Name:        created.Name,
		Description: created.Description,
		Active:      created.Active,
		CreatedAt:   created.CreatedAt,
		UpdatedAt:   created.UpdatedAt,
	}, nil
}

func (r *PostgresProductRepository) Update(ctx context.Context, product *interfaces.Product) error {
	updated, err := r.db.Queries.UpdateProduct(ctx, database.UpdateProductParams{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Active:      product.Active,
	})
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	// Update the product with fresh data
	product.Name = updated.Name
	product.Description = updated.Description
	product.Active = updated.Active
	product.UpdatedAt = updated.UpdatedAt

	return nil
}

func (r *PostgresProductRepository) Activate(ctx context.Context, id int32) (*interfaces.Product, error) {
	updated, err := r.db.Queries.ActivateProduct(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to activate product: %w", err)
	}

	return &interfaces.Product{
		ID:          updated.ID,
		Name:        updated.Name,
		Description: updated.Description,
		Active:      updated.Active,
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
	}, nil
}

func (r *PostgresProductRepository) Deactivate(ctx context.Context, id int32) (*interfaces.Product, error) {
	updated, err := r.db.Queries.DeactivateProduct(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate product: %w", err)
	}

	return &interfaces.Product{
		ID:          updated.ID,
		Name:        updated.Name,
		Description: updated.Description,
		Active:      updated.Active,
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
	}, nil
}

func (r *PostgresProductRepository) Delete(ctx context.Context, id int32) error {
	err := r.db.Queries.DeleteProduct(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	return nil
}

// =============================================================================
// Admin Utilities
// =============================================================================

func (r *PostgresProductRepository) GetProductsWithoutVariants(ctx context.Context, limit, offset int32) ([]interfaces.Product, error) {
	dbProducts, err := r.db.Queries.GetProductsWithoutVariants(ctx, database.GetProductsWithoutVariantsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get products without variants: %w", err)
	}

	var products []interfaces.Product
	for _, dbProduct := range dbProducts {
		products = append(products, interfaces.Product{
			ID:          dbProduct.ID,
			Name:        dbProduct.Name,
			Description: dbProduct.Description,
			Active:      dbProduct.Active,
			CreatedAt:   dbProduct.CreatedAt,
			UpdatedAt:   dbProduct.UpdatedAt,
		})
	}

	return products, nil
}

func (r *PostgresProductRepository) RefreshProductStockSummary(ctx context.Context) error {
	err := r.db.Queries.RefreshProductStockSummary(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh product stock summary: %w", err)
	}
	return nil
}

// =============================================================================
// Helper Methods
// =============================================================================

func (r *PostgresProductRepository) convertToProductSummary(dbSummary database.ProductStockSummary) *interfaces.ProductSummary {
	summary := &interfaces.ProductSummary{
		ProductID:        dbSummary.ProductID,
		Name:             dbSummary.Name,
		Description:      dbSummary.Description,
		ProductActive:    dbSummary.ProductActive,
		HasStock:         dbSummary.HasStock,
		StockStatus:      dbSummary.StockStatus,
		AvailableOptions: dbSummary.AvailableOptions,
	}

	// Handle nullable interface{} fields safely
	if dbSummary.TotalStock != nil {
		if val, ok := dbSummary.TotalStock.(int64); ok {
			summary.TotalStock = int32(val)
		}
	}

	if dbSummary.VariantsInStock != nil {
		if val, ok := dbSummary.VariantsInStock.(int64); ok {
			summary.VariantsInStock = int32(val)
		}
	}

	if dbSummary.TotalVariants != nil {
		if val, ok := dbSummary.TotalVariants.(int64); ok {
			summary.TotalVariants = int32(val)
		}
	}

	if dbSummary.MinPrice != nil {
		if val, ok := dbSummary.MinPrice.(int64); ok {
			summary.MinPrice = int32(val)
		}
	}

	if dbSummary.MaxPrice != nil {
		if val, ok := dbSummary.MaxPrice.(int64); ok {
			summary.MaxPrice = int32(val)
		}
	}

	return summary
}