// internal/repository/cart.go
package repository

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresCartRepository struct {
	db *database.DB
}

func NewPostgresCartRepository(db *database.DB) interfaces.CartRepository {
	return &PostgresCartRepository{
		db: db,
	}
}

// Cart operations

func (r *PostgresCartRepository) GetByID(ctx context.Context, id int32) (*interfaces.Cart, error) {
	cart, err := r.db.Queries.GetCart(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cart not found")
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	return &cart, nil
}

func (r *PostgresCartRepository) GetByCustomerID(ctx context.Context, customerID int32) (*interfaces.Cart, error) {
	cart, err := r.db.Queries.GetCartByCustomerID(ctx, pgtype.Int4{Int32: customerID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cart not found")
		}
		return nil, fmt.Errorf("failed to get cart by customer ID: %w", err)
	}
	return &cart, nil
}

func (r *PostgresCartRepository) GetBySessionID(ctx context.Context, sessionID string) (*interfaces.Cart, error) {
	cart, err := r.db.Queries.GetCartBySessionID(ctx, pgtype.Text{String: sessionID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cart not found")
		}
		return nil, fmt.Errorf("failed to get cart by session ID: %w", err)
	}
	return &cart, nil
}

func (r *PostgresCartRepository) Create(ctx context.Context, customerID *int32, sessionID *string) (*interfaces.Cart, error) {
	var custID pgtype.Int4
	var sessID pgtype.Text

	if customerID != nil {
		custID = pgtype.Int4{Int32: *customerID, Valid: true}
	}

	if sessionID != nil {
		sessID = pgtype.Text{String: *sessionID, Valid: true}
	}

	cart, err := r.db.Queries.CreateCart(ctx, database.CreateCartParams{
		CustomerID: custID,
		SessionID:  sessID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cart: %w", err)
	}

	return &cart, nil
}

func (r *PostgresCartRepository) UpdateTimestamp(ctx context.Context, cartID int32) (*interfaces.Cart, error) {
	cart, err := r.db.Queries.UpdateCartTimestamp(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to update cart timestamp: %w", err)
	}
	return &cart, nil
}

func (r *PostgresCartRepository) Delete(ctx context.Context, id int32) error {
	err := r.db.Queries.DeleteCart(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}
	return nil
}

func (r *PostgresCartRepository) Clear(ctx context.Context, cartID int32) error {
	err := r.db.Queries.ClearCartItems(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to clear cart items: %w", err)
	}
	return nil
}

// Cart item operations

func (r *PostgresCartRepository) GetCartItems(ctx context.Context, cartID int32) ([]interfaces.CartItemWithProduct, error) {
	dbItems, err := r.db.Queries.GetCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	items := make([]interfaces.CartItemWithProduct, len(dbItems))
	for i, item := range dbItems {
		items[i] = interfaces.CartItemWithProduct{
			ID:                 item.ID,
			CartID:             item.CartID,
			ProductID:          item.ProductID,
			Quantity:           item.Quantity,
			Price:              item.Price,
			CreatedAt:          item.CreatedAt,
			ProductName:        item.ProductName,
			ProductDescription: item.ProductDescription.String,
			ProductStock:       item.ProductStock,
		}
	}

	return items, nil
}

func (r *PostgresCartRepository) GetCartItem(ctx context.Context, itemID int32) (*interfaces.CartItem, error) {
	item, err := r.db.Queries.GetCartItem(ctx, itemID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cart item not found")
		}
		return nil, fmt.Errorf("failed to get cart item: %w", err)
	}
	return &item, nil
}

func (r *PostgresCartRepository) GetCartItemByProductID(ctx context.Context, cartID int32, productID int32) (*interfaces.CartItem, error) {
	item, err := r.db.Queries.GetCartItemByProductID(ctx, database.GetCartItemByProductIDParams{
		CartID:    cartID,
		ProductID: productID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cart item not found")
		}
		return nil, fmt.Errorf("failed to get cart item by product ID: %w", err)
	}
	return &item, nil
}

func (r *PostgresCartRepository) AddItem(ctx context.Context, cartID int32, productID int32, quantity int32, price int32) (*interfaces.CartItem, error) {
	item, err := r.db.Queries.CreateCartItem(ctx, database.CreateCartItemParams{
		CartID:    cartID,
		ProductID: productID,
		Quantity:  quantity,
		Price:     price,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add cart item: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, cartID)

	return &item, nil
}

func (r *PostgresCartRepository) UpdateItem(ctx context.Context, itemID int32, quantity int32, price int32) (*interfaces.CartItem, error) {
	item, err := r.db.Queries.UpdateCartItem(ctx, database.UpdateCartItemParams{
		ID:       itemID,
		Quantity: quantity,
		Price:    price,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update cart item: %w", err)
	}

	// Update cart timestamp
	cartItem, _ := r.GetCartItem(ctx, itemID)
	if cartItem != nil {
		_, _ = r.UpdateTimestamp(ctx, cartItem.CartID)
	}

	return &item, nil
}

func (r *PostgresCartRepository) UpdateItemQuantity(ctx context.Context, itemID int32, quantity int32) (*interfaces.CartItem, error) {
	item, err := r.db.Queries.UpdateCartItemQuantity(ctx, database.UpdateCartItemQuantityParams{
		ID:       itemID,
		Quantity: quantity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update cart item quantity: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, item.CartID)

	return &item, nil
}

func (r *PostgresCartRepository) RemoveItem(ctx context.Context, itemID int32) error {
	// Get cart ID before deletion for timestamp update
	cartItem, err := r.GetCartItem(ctx, itemID)
	if err != nil {
		return err
	}

	err = r.db.Queries.DeleteCartItem(ctx, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove cart item: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, cartItem.CartID)

	return nil
}

func (r *PostgresCartRepository) RemoveItemByProductID(ctx context.Context, cartID int32, productID int32) error {
	err := r.db.Queries.DeleteCartItemByProductID(ctx, database.DeleteCartItemByProductIDParams{
		CartID:    cartID,
		ProductID: productID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove cart item by product ID: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, cartID)

	return nil
}

// Cart totals

func (r *PostgresCartRepository) GetCartTotal(ctx context.Context, cartID int32) (int32, error) {
	total, err := r.db.Queries.GetCartTotal(ctx, cartID)
	if err != nil {
		return 0, fmt.Errorf("failed to get cart total: %w", err)
	}
	return total, nil
}

func (r *PostgresCartRepository) GetCartItemCount(ctx context.Context, cartID int32) (int32, error) {
	count, err := r.db.Queries.GetCartItemCount(ctx, cartID)
	if err != nil {
		return 0, fmt.Errorf("failed to get cart item count: %w", err)
	}
	return count, nil
}
