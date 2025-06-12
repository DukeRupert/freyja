// internal/repository/product.go
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

func (r *PostgresProductRepository) GetByID(ctx context.Context, id int32) (*interfaces.Product, error) {
	product, err := r.db.Queries.GetProduct(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &product, nil
}

func (r *PostgresProductRepository) GetByName(ctx context.Context, name string) (*interfaces.Product, error) {
	product, err := r.db.Queries.GetProductByName(ctx, name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &product, nil
}

func (r *PostgresProductRepository) GetByStripeProductID(ctx context.Context, stripeProductID string) (*interfaces.Product, error) {
	product, err := r.db.Queries.GetProductByStripeProductID(ctx, pgtype.Text{
		String: stripeProductID,
		Valid:  true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product by Stripe product ID: %w", err)
	}

	return &product, nil
}

func (r *PostgresProductRepository) GetProductsWithoutStripeSync(ctx context.Context, limit, offset int) ([]interfaces.Product, error) {
	products, err := r.db.Queries.GetProductsWithoutStripeSync(ctx, database.GetProductsWithoutStripeSyncParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get products without Stripe sync: %w", err)
	}

	return products, nil
}

func (r *PostgresProductRepository) GetAll(ctx context.Context, filters interfaces.ProductFilters) ([]interfaces.Product, error) {
	var dbProducts []database.Products
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

		dbProducts, err = r.db.Queries.ListProductsByStatus(ctx, database.ListProductsByStatusParams{
			Active: *filters.Active,
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

		dbProducts, err = r.db.Queries.ListAllProducts(ctx, database.ListAllProductsParams{
			Limit:  limit,
			Offset: offset,
		})
	} else {
		// Default to active products only
		dbProducts, err = r.db.Queries.ListProducts(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	return dbProducts, nil
}

func (r *PostgresProductRepository) SearchProducts(ctx context.Context, query string) ([]interfaces.Product, error) {
	searchTerm := "%" + query + "%"
	dbProducts, err := r.db.Queries.SearchProducts(ctx, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	return dbProducts, nil
}

func (r *PostgresProductRepository) GetInStock(ctx context.Context) ([]interfaces.Product, error) {
	dbProducts, err := r.db.Queries.GetProductsInStock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get in-stock products: %w", err)
	}

	return dbProducts, nil
}

func (r *PostgresProductRepository) GetLowStock(ctx context.Context, threshold int32) ([]interfaces.Product, error) {
	dbProducts, err := r.db.Queries.GetLowStockProducts(ctx, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get low stock products: %w", err)
	}

	return dbProducts, nil
}

func (r *PostgresProductRepository) Create(ctx context.Context, req interfaces.CreateProductRequest) (*interfaces.Product, error) {
	created, err := r.db.Queries.CreateProduct(ctx, database.CreateProductParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Price:       req.Price,
		Stock:       req.Stock,
		Active:      req.Active,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Update the product with the generated ID
	return &created, nil
}

func (r *PostgresProductRepository) Update(ctx context.Context, product *interfaces.Product) error {
	updated, err := r.db.Queries.UpdateProduct(ctx, database.UpdateProductParams{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		Active:      product.Active,
	})
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	// Update the product with fresh data
	product.Name = updated.Name
	product.Description = updated.Description
	product.Price = updated.Price
	product.Stock = updated.Stock
	product.Active = updated.Active

	return nil
}

func (r *PostgresProductRepository) UpdateStock(ctx context.Context, id int32, stock int32) error {
	_, err := r.db.Queries.UpdateProductStock(ctx, database.UpdateProductStockParams{
		ID:    id,
		Stock: stock,
	})
	if err != nil {
		return fmt.Errorf("failed to update product stock: %w", err)
	}
	return nil
}

func (r *PostgresProductRepository) UpdateStripeProductID(ctx context.Context, id int32, stripeProductID string) error {
	_, err := r.db.Queries.UpdateProductStripeProductID(ctx, database.UpdateProductStripeProductIDParams{
		ID: id,
		StripeProductID: pgtype.Text{
			String: stripeProductID,
			Valid:  true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update product Stripe product ID: %w", err)
	}
	return nil
}

func (r *PostgresProductRepository) UpdateStripePriceIDs(ctx context.Context, id int32, priceIDs map[string]string) error {
	// Convert map to individual pgtype.Text fields
	var onetimeID, day14ID, day21ID, day30ID, day60ID pgtype.Text

	if priceID, exists := priceIDs["onetime"]; exists && priceID != "" {
		onetimeID = pgtype.Text{String: priceID, Valid: true}
	}
	if priceID, exists := priceIDs["14day"]; exists && priceID != "" {
		day14ID = pgtype.Text{String: priceID, Valid: true}
	}
	if priceID, exists := priceIDs["21day"]; exists && priceID != "" {
		day21ID = pgtype.Text{String: priceID, Valid: true}
	}
	if priceID, exists := priceIDs["30day"]; exists && priceID != "" {
		day30ID = pgtype.Text{String: priceID, Valid: true}
	}
	if priceID, exists := priceIDs["60day"]; exists && priceID != "" {
		day60ID = pgtype.Text{String: priceID, Valid: true}
	}

	_, err := r.db.Queries.UpdateProductStripePrices(ctx, database.UpdateProductStripePricesParams{
		ID:                   id,
		StripePriceOnetimeID: onetimeID,
		StripePrice14dayID:   day14ID,
		StripePrice21dayID:   day21ID,
		StripePrice30dayID:   day30ID,
		StripePrice60dayID:   day60ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update product Stripe price IDs: %w", err)
	}
	return nil
}

func (r *PostgresProductRepository) Delete(ctx context.Context, id int32) error {
	err := r.db.Queries.DeleteProduct(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	return nil
}

func (r *PostgresProductRepository) GetCount(ctx context.Context, activeOnly bool) (int64, error) {
	count, err := r.db.Queries.GetProductCount(ctx, activeOnly)
	if err != nil {
		return 0, fmt.Errorf("failed to get product count: %w", err)
	}
	return count, nil
}

func (r *PostgresProductRepository) GetTotalValue(ctx context.Context) (int32, error) {
	value, err := r.db.Queries.GetTotalProductValue(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get total product value: %w", err)
	}
	return value, nil
}
