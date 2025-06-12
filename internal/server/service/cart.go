// internal/service/cart.go
package service

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type CartService struct {
	cartRepo    interfaces.CartRepository
	productRepo interfaces.ProductRepository
	// Note: cache and events will be added later
}

func NewCartService(cartRepo interfaces.CartRepository, productRepo interfaces.ProductRepository) interfaces.CartService {
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

// GetCartItems retrieves all items in a cart with product details
func (s *CartService) GetCartItems(ctx context.Context, cartID int32) ([]interfaces.CartItemWithProduct, error) {
	if cartID <= 0 {
		return nil, fmt.Errorf("invalid cart ID: %d", cartID)
	}

	// Verify cart exists
	_, err := s.cartRepo.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart not found: %w", err)
	}

	// Get cart items with product details
	items, err := s.cartRepo.GetCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	// Validate each item's product is still active and in stock
	var validItems []interfaces.CartItemWithProduct
	for _, item := range items {
		// Check if product is still active
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			// Product not found - could log this as a warning but don't fail the request
			continue
		}

		if !product.Active {
			// Product is inactive - could log this as a warning but don't fail the request
			continue
		}

		// Add item to valid items list
		validItems = append(validItems, item)
	}

	return validItems, nil
}

// AddItem adds an item to the cart or updates quantity if item already exists
func (s *CartService) AddItem(ctx context.Context, cartID int32, productID int32, quantity int32, purchaseType string, subscriptionInterval *string) (*interfaces.CartItem, error) {
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

	// Get the appropriate Stripe Price ID based on purchase type and interval
	stripePriceID, err := s.getStripePriceID(product, purchaseType, subscriptionInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe price ID: %w", err)
	}

	// Check if item already exists in cart with same purchase type and interval
	existingItem, err := s.cartRepo.GetCartItemByProductAndType(ctx, cartID, productID, purchaseType, subscriptionInterval)

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
	return s.cartRepo.AddItem(ctx, cartID, productID, quantity, product.Price, purchaseType, subscriptionInterval, stripePriceID)
}

// Helper method to get Stripe Price ID based on purchase type and interval
// Helper method to get Stripe Price ID based on purchase type and interval
func (s *CartService) getStripePriceID(product *interfaces.Product, purchaseType string, subscriptionInterval *string) (string, error) {
	switch purchaseType {
	case "one_time":
		if !product.StripePriceOnetimeID.Valid {
			return "", fmt.Errorf("one-time purchase not available for this product")
		}
		return product.StripePriceOnetimeID.String, nil
	case "subscription":
		if subscriptionInterval == nil {
			return "", fmt.Errorf("subscription interval required for subscription purchases")
		}
		switch *subscriptionInterval {
		case "14_day":
			if !product.StripePrice14dayID.Valid {
				return "", fmt.Errorf("14-day subscription not available for this product")
			}
			return product.StripePrice14dayID.String, nil
		case "21_day":
			if !product.StripePrice21dayID.Valid {
				return "", fmt.Errorf("21-day subscription not available for this product")
			}
			return product.StripePrice21dayID.String, nil
		case "30_day":
			if !product.StripePrice30dayID.Valid {
				return "", fmt.Errorf("30-day subscription not available for this product")
			}
			return product.StripePrice30dayID.String, nil
		case "60_day":
			if !product.StripePrice60dayID.Valid {
				return "", fmt.Errorf("60-day subscription not available for this product")
			}
			return product.StripePrice60dayID.String, nil
		default:
			return "", fmt.Errorf("invalid subscription interval: %s", *subscriptionInterval)
		}
	default:
		return "", fmt.Errorf("invalid purchase type: %s", purchaseType)
	}
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
	return s.cartRepo.UpdateItem(ctx, itemID, quantity, product.Price, existingItem.StripePriceID)
}

// RemoveItem removes an item from the cart
func (s *CartService) RemoveItem(ctx context.Context, itemID int32) error {
	return s.cartRepo.RemoveItem(ctx, itemID)
}

// RemoveItemByProductID removes an item from the cart by product ID
func (s *CartService) RemoveItemByProductID(ctx context.Context, cartID int32, productID int32) error {
	return s.cartRepo.RemoveItemByProductID(ctx, cartID, productID)
}

// Clear removes all items from the cart
func (s *CartService) Clear(ctx context.Context, cartID int32) error {
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
			_, err := s.cartRepo.UpdateItem(ctx, item.ID, item.Quantity, item.Price, item.StripePriceID)
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
