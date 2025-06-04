// internal/service/cart.go
package service

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/interfaces"
)

type CartService struct {
	cartRepo    interfaces.CartRepository
	productRepo interfaces.ProductRepository
	// Note: cache and events will be added later
}

func NewCartService(cartRepo interfaces.CartRepository, productRepo interfaces.ProductRepository) *CartService {
	return &CartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
	}
}

// GetOrCreateCart gets existing cart or creates a new one
func (s *CartService) GetOrCreateCart(ctx context.Context, customerID *int32, sessionID *string) (*interfaces.CartWithItems, error) {
	var cart *interfaces.Cart
	var err error

	// Try to find existing cart
	if customerID != nil {
		cart, err = s.cartRepo.GetByCustomerID(ctx, *customerID)
	} else if sessionID != nil {
		cart, err = s.cartRepo.GetBySessionID(ctx, *sessionID)
	} else {
		return nil, fmt.Errorf("either customer ID or session ID must be provided")
	}

	// If cart doesn't exist, create a new one
	if err != nil && err.Error() == "cart not found" {
		cart, err = s.cartRepo.Create(ctx, customerID, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to create cart: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	return s.buildCartWithItems(ctx, cart)
}

// GetCart retrieves cart with items and totals
func (s *CartService) GetCart(ctx context.Context, cartID int32) (*interfaces.CartWithItems, error) {
	cart, err := s.cartRepo.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	return s.buildCartWithItems(ctx, cart)
}

// AddItem adds an item to the cart or updates quantity if item already exists
func (s *CartService) AddItem(ctx context.Context, cartID int32, productID int32, quantity int32) (*interfaces.CartItem, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

	if quantity > 100 {
		return nil, fmt.Errorf("quantity cannot exceed 100 items per product")
	}

	// Validate product exists and is active
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	if !product.Active {
		return nil, fmt.Errorf("product is not available")
	}

	// Check if item already exists in cart
	existingItem, err := s.cartRepo.GetCartItemByProductID(ctx, cartID, productID)

	if err != nil && err.Error() != "cart item not found" {
		return nil, fmt.Errorf("failed to check existing cart item: %w", err)
	}

	// If item exists, update quantity instead of adding new item
	if existingItem != nil {
		newQuantity := existingItem.Quantity + quantity

		// Check stock availability for new total quantity
		if product.Stock < newQuantity {
			return nil, fmt.Errorf("insufficient stock: requested %d, available %d", newQuantity, product.Stock)
		}

		if newQuantity > 100 {
			return nil, fmt.Errorf("total quantity cannot exceed 100 items per product")
		}

		return s.cartRepo.UpdateItemQuantity(ctx, existingItem.ID, newQuantity)
	}

	// Check stock availability
	if product.Stock < quantity {
		return nil, fmt.Errorf("insufficient stock: requested %d, available %d", quantity, product.Stock)
	}

	// Add new item to cart
	return s.cartRepo.AddItem(ctx, cartID, productID, quantity, product.Price)
}

// UpdateItemQuantity updates the quantity of a cart item
func (s *CartService) UpdateItemQuantity(ctx context.Context, itemID int32, quantity int32) (*interfaces.CartItem, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

	if quantity > 100 {
		return nil, fmt.Errorf("quantity cannot exceed 100 items per product")
	}

	// Get existing item to validate product
	existingItem, err := s.cartRepo.GetCartItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("cart item not found: %w", err)
	}

	// Validate product stock
	product, err := s.productRepo.GetByID(ctx, existingItem.ProductID)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	if !product.Active {
		return nil, fmt.Errorf("product is no longer available")
	}

	if product.Stock < quantity {
		return nil, fmt.Errorf("insufficient stock: requested %d, available %d", quantity, product.Stock)
	}

	// Update quantity (and potentially price if it changed)
	return s.cartRepo.UpdateItem(ctx, itemID, quantity, product.Price)
}

// RemoveItem removes an item from the cart
func (s *CartService) RemoveItem(ctx context.Context, itemID int32) error {
	return s.cartRepo.RemoveItem(ctx, itemID)
}

// RemoveItemByProductID removes an item from the cart by product ID
func (s *CartService) RemoveItemByProductID(ctx context.Context, cartID int32, productID int32) error {
	return s.cartRepo.RemoveItemByProductID(ctx, cartID, productID)
}

// ClearCart removes all items from the cart
func (s *CartService) ClearCart(ctx context.Context, cartID int32) error {
	return s.cartRepo.Clear(ctx, cartID)
}

// ValidateCartForCheckout validates cart contents before checkout
func (s *CartService) ValidateCartForCheckout(ctx context.Context, cartID int32) (*interfaces.CartWithItems, error) {
	cartWithItems, err := s.GetCart(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if len(cartWithItems.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Validate each item
	for _, item := range cartWithItems.Items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %d not found", item.ProductID)
		}

		if !product.Active {
			return nil, fmt.Errorf("product '%s' is no longer available", product.Name)
		}

		if product.Stock < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for '%s': requested %d, available %d",
				product.Name, item.Quantity, product.Stock)
		}

		// Update price if it has changed
		if product.Price != item.Price {
			_, err := s.cartRepo.UpdateItem(ctx, item.ID, item.Quantity, product.Price)
			if err != nil {
				return nil, fmt.Errorf("failed to update item price: %w", err)
			}
		}
	}

	// Return updated cart
	return s.GetCart(ctx, cartID)
}

// GetCartSummary returns a summary of cart contents
func (s *CartService) GetCartSummary(ctx context.Context, cartID int32) (map[string]interface{}, error) {
	total, err := s.cartRepo.GetCartTotal(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart total: %w", err)
	}

	itemCount, err := s.cartRepo.GetCartItemCount(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart item count: %w", err)
	}

	items, err := s.cartRepo.GetCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	return map[string]interface{}{
		"cart_id":    cartID,
		"total":      total,
		"item_count": itemCount,
		"items":      len(items),
		"subtotal":   total,
		// Add tax, shipping, etc. calculations here later
	}, nil
}

// Helper method to build cart with items and totals
func (s *CartService) buildCartWithItems(ctx context.Context, cart *interfaces.Cart) (*interfaces.CartWithItems, error) {
	items, err := s.cartRepo.GetCartItems(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	total, err := s.cartRepo.GetCartTotal(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart total: %w", err)
	}

	itemCount, err := s.cartRepo.GetCartItemCount(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart item count: %w", err)
	}

	var customerID *int32
	var sessionID *string

	if cart.CustomerID.Valid {
		customerID = &cart.CustomerID.Int32
	}

	if cart.SessionID.Valid {
		sessionID = &cart.SessionID.String
	}

	return &interfaces.CartWithItems{
		ID:         cart.ID,
		CustomerID: customerID,
		SessionID:  sessionID,
		Items:      items,
		Total:      total,
		ItemCount:  itemCount,
		CreatedAt:  cart.CreatedAt,
		UpdatedAt:  cart.UpdatedAt,
	}, nil
}
