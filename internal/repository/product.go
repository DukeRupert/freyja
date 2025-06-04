// internal/repository/product.go
package repository

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/jackc/pgx/v5"
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

func (r *PostgresProductRepository) Create(ctx context.Context, product *interfaces.Product) error {

	created, err := r.db.Queries.CreateProduct(ctx, database.CreateProductParams{
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		Active:      product.Active,
	})
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	// Update the product with the generated ID
	product.ID = created.ID
	return nil
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
