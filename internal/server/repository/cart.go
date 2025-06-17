// internal/server/repository/cart.go
package repository

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
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

// =============================================================================
// Cart operations
// =============================================================================

func (r *PostgresCartRepository) GetByID(ctx context.Context, id int32) (*interfaces.Cart, error) {
	cart, err := r.db.Queries.GetCart(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cart not found")
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	return &interfaces.Cart{
		ID:         cart.ID,
		CustomerID: cart.CustomerID,
		SessionID:  cart.SessionID,
		CreatedAt:  cart.CreatedAt,
		UpdatedAt:  cart.UpdatedAt,
	}, nil
}

func (r *PostgresCartRepository) GetByCustomerID(ctx context.Context, customerID int32) (*interfaces.Cart, error) {
	cart, err := r.db.Queries.GetCartByCustomerID(ctx, pgtype.Int4{Int32: customerID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cart not found")
		}
		return nil, fmt.Errorf("failed to get cart by customer ID: %w", err)
	}
	return &interfaces.Cart{
		ID:         cart.ID,
		CustomerID: cart.CustomerID,
		SessionID:  cart.SessionID,
		CreatedAt:  cart.CreatedAt,
		UpdatedAt:  cart.UpdatedAt,
	}, nil
}

func (r *PostgresCartRepository) GetBySessionID(ctx context.Context, sessionID string) (*interfaces.Cart, error) {
	cart, err := r.db.Queries.GetCartBySessionID(ctx, pgtype.Text{String: sessionID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cart not found")
		}
		return nil, fmt.Errorf("failed to get cart by session ID: %w", err)
	}
	return &interfaces.Cart{
		ID:         cart.ID,
		CustomerID: cart.CustomerID,
		SessionID:  cart.SessionID,
		CreatedAt:  cart.CreatedAt,
		UpdatedAt:  cart.UpdatedAt,
	}, nil
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

	return &interfaces.Cart{
		ID:         cart.ID,
		CustomerID: cart.CustomerID,
		SessionID:  cart.SessionID,
		CreatedAt:  cart.CreatedAt,
		UpdatedAt:  cart.UpdatedAt,
	}, nil
}

func (r *PostgresCartRepository) UpdateTimestamp(ctx context.Context, cartID int32) (*interfaces.Cart, error) {
	cart, err := r.db.Queries.UpdateCartTimestamp(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to update cart timestamp: %w", err)
	}
	return &interfaces.Cart{
		ID:         cart.ID,
		CustomerID: cart.CustomerID,
		SessionID:  cart.SessionID,
		CreatedAt:  cart.CreatedAt,
		UpdatedAt:  cart.UpdatedAt,
	}, nil
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

// =============================================================================
// Cart item operations (now using product_variant_id)
// =============================================================================

func (r *PostgresCartRepository) GetCartItems(ctx context.Context, cartID int32) ([]interfaces.CartItemWithVariant, error) {
	dbItems, err := r.db.Queries.GetCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	items := make([]interfaces.CartItemWithVariant, len(dbItems))
	for i, item := range dbItems {
		items[i] = interfaces.CartItemWithVariant{
			ID:                   item.ID,
			CartID:               item.CartID,
			ProductVariantID:     item.ProductVariantID,
			Quantity:             item.Quantity,
			Price:                item.Price,
			PurchaseType:         item.PurchaseType,
			SubscriptionInterval: item.SubscriptionInterval,
			StripePriceID:        item.StripePriceID,
			CreatedAt:            item.CreatedAt,
			// Variant information
			VariantName:    item.VariantName,
			VariantStock:   item.VariantStock,
			VariantActive:  item.VariantActive,
			OptionsDisplay: item.OptionsDisplay,
			// Product information
			ProductID:          item.ProductID,
			ProductName:        item.ProductName,
			ProductDescription: item.ProductDescription,
			ProductActive:      item.ProductActive,
		}
	}

	return items, nil
}

func (r *PostgresCartRepository) GetCartItemsByVariant(ctx context.Context, cartID int32, productVariantID int32) ([]interfaces.CartItem, error) {
	dbItems, err := r.db.Queries.GetCartItemsByVariant(ctx, database.GetCartItemsByVariantParams{
		CartID:           cartID,
		ProductVariantID: productVariantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items by variant: %w", err)
	}

	var items []interfaces.CartItem
	for _, dbItem := range dbItems {
		items = append(items, interfaces.CartItem{
			ID:                   dbItem.ID,
			CartID:               dbItem.CartID,
			ProductVariantID:     dbItem.ProductVariantID,
			Quantity:             dbItem.Quantity,
			Price:                dbItem.Price,
			PurchaseType:         dbItem.PurchaseType,
			SubscriptionInterval: dbItem.SubscriptionInterval,
			StripePriceID:        dbItem.StripePriceID,
			CreatedAt:            dbItem.CreatedAt,
		})
	}

	return items, nil
}

func (r *PostgresCartRepository) GetCartItemsWithOptions(ctx context.Context, cartID int32) ([]interfaces.CartItemWithOptions, error) {
	dbItems, err := r.db.Queries.GetCartItemsWithOptions(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items with options: %w", err)
	}

	items := make([]interfaces.CartItemWithOptions, len(dbItems))
	for i, item := range dbItems {
		items[i] = interfaces.CartItemWithOptions{
			CartItemWithVariant: interfaces.CartItemWithVariant{
				ID:                   item.ID,
				CartID:               item.CartID,
				ProductVariantID:     item.ProductVariantID,
				Quantity:             item.Quantity,
				Price:                item.Price,
				PurchaseType:         item.PurchaseType,
				SubscriptionInterval: item.SubscriptionInterval,
				StripePriceID:        item.StripePriceID,
				CreatedAt:            item.CreatedAt,
				// Variant information
				VariantName:    item.VariantName,
				VariantStock:   item.VariantStock,
				VariantActive:  item.VariantActive,
				OptionsDisplay: item.OptionsDisplay,
				// Product information
				ProductID:          item.ProductID,
				ProductName:        item.ProductName,
				ProductDescription: item.ProductDescription,
				ProductActive:      item.ProductActive,
			},
			// Detailed options
			VariantOptions: item.VariantOptions,
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

	cartItem := &interfaces.CartItem{
		ID:                   item.ID,
		CartID:               item.CartID,
		ProductVariantID:     item.ProductVariantID,
		Quantity:             item.Quantity,
		Price:                item.Price,
		PurchaseType:         item.PurchaseType,
		SubscriptionInterval: item.SubscriptionInterval,
		StripePriceID:        item.StripePriceID,
		CreatedAt:            item.CreatedAt,
	}

	return cartItem, nil
}

func (r *PostgresCartRepository) AddItem(ctx context.Context, cartID int32, productVariantID int32, quantity int32, price int32, purchaseType string, subscriptionInterval *string, stripePriceID string) (*interfaces.CartItem, error) {
	var interval pgtype.Text
	if subscriptionInterval != nil {
		interval = pgtype.Text{String: *subscriptionInterval, Valid: true}
	}

	item, err := r.db.Queries.CreateCartItem(ctx, database.CreateCartItemParams{
		CartID:               cartID,
		ProductVariantID:     productVariantID,
		Quantity:             quantity,
		Price:                price,
		PurchaseType:         purchaseType,
		SubscriptionInterval: interval,
		StripePriceID:        stripePriceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add cart item: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, cartID)

	cartItem := &interfaces.CartItem{
		ID:                   item.ID,
		CartID:               item.CartID,
		ProductVariantID:     item.ProductVariantID,
		Quantity:             item.Quantity,
		Price:                item.Price,
		PurchaseType:         item.PurchaseType,
		SubscriptionInterval: item.SubscriptionInterval,
		StripePriceID:        item.StripePriceID,
		CreatedAt:            item.CreatedAt,
	}

	return cartItem, nil
}

func (r *PostgresCartRepository) UpdateItem(ctx context.Context, itemID int32, quantity int32, price int32, stripePriceID string) (*interfaces.CartItem, error) {
	item, err := r.db.Queries.UpdateCartItem(ctx, database.UpdateCartItemParams{
		ID:            itemID,
		Quantity:      quantity,
		Price:         price,
		StripePriceID: stripePriceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update cart item: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, item.CartID)

	cartItem := &interfaces.CartItem{
		ID:                   item.ID,
		CartID:               item.CartID,
		ProductVariantID:     item.ProductVariantID,
		Quantity:             item.Quantity,
		Price:                item.Price,
		PurchaseType:         item.PurchaseType,
		SubscriptionInterval: item.SubscriptionInterval,
		StripePriceID:        item.StripePriceID,
		CreatedAt:            item.CreatedAt,
	}

	return cartItem, nil
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

	cartItem := &interfaces.CartItem{
		ID:                   item.ID,
		CartID:               item.CartID,
		ProductVariantID:     item.ProductVariantID,
		Quantity:             item.Quantity,
		Price:                item.Price,
		PurchaseType:         item.PurchaseType,
		SubscriptionInterval: item.SubscriptionInterval,
		StripePriceID:        item.StripePriceID,
		CreatedAt:            item.CreatedAt,
	}

	return cartItem, nil
}

func (r *PostgresCartRepository) IncrementItemQuantity(ctx context.Context, itemID int32, delta int32) (*interfaces.CartItem, error) {
	item, err := r.db.Queries.IncrementCartItemQuantity(ctx, database.IncrementCartItemQuantityParams{
		ID:    itemID,
		Quantity: delta,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to increment cart item quantity: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, item.CartID)

	cartItem := &interfaces.CartItem{
		ID:                   item.ID,
		CartID:               item.CartID,
		ProductVariantID:     item.ProductVariantID,
		Quantity:             item.Quantity,
		Price:                item.Price,
		PurchaseType:         item.PurchaseType,
		SubscriptionInterval: item.SubscriptionInterval,
		StripePriceID:        item.StripePriceID,
		CreatedAt:            item.CreatedAt,
	}

	return cartItem, nil
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

func (r *PostgresCartRepository) RemoveItemByVariantID(ctx context.Context, cartID int32, productVariantID int32) error {
	err := r.db.Queries.DeleteCartItemByVariantID(ctx, database.DeleteCartItemByVariantIDParams{
		CartID:           cartID,
		ProductVariantID: productVariantID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove cart item by variant ID: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, cartID)

	return nil
}

func (r *PostgresCartRepository) RemoveItemByVariantAndType(ctx context.Context, cartID int32, productVariantID int32, purchaseType string, subscriptionInterval *string) error {
	var interval pgtype.Text
	if subscriptionInterval != nil {
		interval = pgtype.Text{String: *subscriptionInterval, Valid: true}
	}

	err := r.db.Queries.DeleteCartItemByVariantAndType(ctx, database.DeleteCartItemByVariantAndTypeParams{
		CartID:               cartID,
		ProductVariantID:     productVariantID,
		PurchaseType:         purchaseType,
		SubscriptionInterval: interval,
	})
	if err != nil {
		return fmt.Errorf("failed to remove cart item by variant and type: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, cartID)

	return nil
}

// =============================================================================
// Cart totals and analytics
// =============================================================================

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

func (r *PostgresCartRepository) GetCartTotalByPurchaseType(ctx context.Context, cartID int32, purchaseType string) (int32, error) {
	total, err := r.db.Queries.GetCartTotalByPurchaseType(ctx, database.GetCartTotalByPurchaseTypeParams{
		CartID:       cartID,
		PurchaseType: purchaseType,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get cart total by purchase type: %w", err)
	}
	return total, nil
}

func (r *PostgresCartRepository) GetCartSubscriptionSummary(ctx context.Context, cartID int32) ([]interfaces.CartSubscriptionSummary, error) {
	dbSummaries, err := r.db.Queries.GetCartSubscriptionSummary(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart subscription summary: %w", err)
	}

	var summaries []interfaces.CartSubscriptionSummary
	for _, dbSummary := range dbSummaries {
		summaries = append(summaries, interfaces.CartSubscriptionSummary{
			SubscriptionInterval: dbSummary.SubscriptionInterval,
			ItemCount:            dbSummary.ItemCount,
			TotalQuantity:        dbSummary.TotalQuantity,
			TotalAmount:          dbSummary.TotalAmount,
		})
	}

	return summaries, nil
}

// =============================================================================
// Cart validation and maintenance
// =============================================================================

func (r *PostgresCartRepository) ValidateCartItems(ctx context.Context, cartID int32) ([]interfaces.CartItemValidation, error) {
	dbValidations, err := r.db.Queries.ValidateCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate cart items: %w", err)
	}

	var validations []interfaces.CartItemValidation
	for _, dbValidation := range dbValidations {
		validations = append(validations, interfaces.CartItemValidation{
			CartItemID:        dbValidation.CartItemID,
			ProductVariantID:  dbValidation.ProductVariantID,
			RequestedQuantity: dbValidation.RequestedQuantity,
			AvailableStock:    dbValidation.AvailableStock,
			VariantActive:     dbValidation.VariantActive,
			ProductActive:     dbValidation.ProductActive,
			ValidationStatus:  dbValidation.ValidationStatus,
		})
	}

	return validations, nil
}

func (r *PostgresCartRepository) GetInvalidCartItems(ctx context.Context, cartID int32) ([]interfaces.CartItemWithVariant, error) {
	dbItems, err := r.db.Queries.GetInvalidCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invalid cart items: %w", err)
	}

	var items []interfaces.CartItemWithVariant
	for _, item := range dbItems {
		items = append(items, interfaces.CartItemWithVariant{
			ID:                   item.ID,
			CartID:               item.CartID,
			ProductVariantID:     item.ProductVariantID,
			Quantity:             item.Quantity,
			Price:                item.Price,
			PurchaseType:         item.PurchaseType,
			SubscriptionInterval: item.SubscriptionInterval,
			StripePriceID:        item.StripePriceID,
			CreatedAt:            item.CreatedAt,
			VariantName:          item.VariantName,
			VariantStock:         item.VariantStock,
			VariantActive:        item.VariantActive,
			ProductName:          item.ProductName,
			IssueType:            item.IssueType,
		})
	}

	return items, nil
}

func (r *PostgresCartRepository) CheckVariantAvailability(ctx context.Context, productVariantID int32, requestedQuantity int32) (*interfaces.VariantAvailability, error) {
	availability, err := r.db.Queries.CheckVariantAvailability(ctx, database.CheckVariantAvailabilityParams{
		ID:    productVariantID,
		Stock: requestedQuantity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check variant availability: %w", err)
	}

	return &interfaces.VariantAvailability{
		VariantID:     availability.ID,
		Stock:         availability.Stock,
		Active:        availability.Active,
		ProductActive: availability.ProductActive,
		IsAvailable:   availability.IsAvailable,
	}, nil
}

func (r *PostgresCartRepository) RemoveUnavailableItems(ctx context.Context, cartID int32) error {
	err := r.db.Queries.RemoveUnavailableCartItems(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to remove unavailable cart items: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, cartID)

	return nil
}

func (r *PostgresCartRepository) UpdateCartItemPrices(ctx context.Context, cartID int32) error {
	err := r.db.Queries.UpdateCartItemPrices(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to update cart item prices: %w", err)
	}

	// Update cart timestamp
	_, _ = r.UpdateTimestamp(ctx, cartID)

	return nil
}
