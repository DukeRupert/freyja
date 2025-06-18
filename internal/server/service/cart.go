// internal/server/service/cart.go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type CartService struct {
	cartRepo    interfaces.CartRepository
	variantRepo interfaces.VariantRepository // Will need this for variant operations
	events      interfaces.EventPublisher
}

func NewCartService(cartRepo interfaces.CartRepository, variantRepo interfaces.VariantRepository, events interfaces.EventPublisher) interfaces.CartService {
	return &CartService{
		cartRepo:    cartRepo,
		variantRepo: variantRepo,
		events:      events,
	}
}

// =============================================================================
// Cart Retrieval
// =============================================================================

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

		// Publish cart created event
		if err := s.publishCartEvent(ctx, "cart.created", cart.ID, map[string]interface{}{
			"customer_id": customerID,
			"session_id":  sessionID,
		}); err != nil {
			log.Printf("Failed to publish cart created event: %v", err)
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

// GetCustomerCart retrieves the cart for a specific customer
func (s *CartService) GetCustomerCart(ctx context.Context, customerID int32) (*interfaces.CartWithItems, error) {
	// Get cart by customer ID
	cart, err := s.cartRepo.GetByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart for customer %d: %w", customerID, err)
	}

	if cart == nil {
		return nil, nil // No cart found
	}

	// Get cart items with variant details
	items, err := s.cartRepo.GetCartItems(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	// Calculate total
	var total int32
	for _, item := range items {
		total += item.Price * item.Quantity
	}

	// Convert pgtype.Text to *string  
	var sessionID *string
	if cart.SessionID.Valid {
		sessionID = &cart.SessionID.String
	}

	return &interfaces.CartWithItems{
		ID:         cart.ID,
		CustomerID: &customerID,
		SessionID:  sessionID,
		Items:      items,
		Total:      total,
		CreatedAt:  cart.CreatedAt,
		UpdatedAt:  cart.UpdatedAt,
	}, nil
}

// GetCartItems retrieves all items in a cart with variant details
func (s *CartService) GetCartItems(ctx context.Context, cartID int32) ([]interfaces.CartItemWithVariant, error) {
	if cartID <= 0 {
		return nil, fmt.Errorf("invalid cart ID: %d", cartID)
	}

	// Verify cart exists
	_, err := s.cartRepo.GetByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart not found: %w", err)
	}

	items, err := s.cartRepo.GetCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	return items, nil
}

// GetCartItemsWithOptions retrieves cart items with detailed option information
func (s *CartService) GetCartItemsWithOptions(ctx context.Context, cartID int32) ([]interfaces.CartItemWithOptions, error) {
	if cartID <= 0 {
		return nil, fmt.Errorf("invalid cart ID: %d", cartID)
	}

	items, err := s.cartRepo.GetCartItemsWithOptions(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items with options: %w", err)
	}

	return items, nil
}

// =============================================================================
// Cart Item Management (now using variants)
// =============================================================================

// AddItem adds a product variant to the cart
func (s *CartService) AddItem(ctx context.Context, cartID int32, productVariantID int32, quantity int32, purchaseType string, subscriptionInterval *string) (*interfaces.CartItem, error) {
	if cartID <= 0 {
		return nil, fmt.Errorf("invalid cart ID: %d", cartID)
	}
	if productVariantID <= 0 {
		return nil, fmt.Errorf("invalid product variant ID: %d", productVariantID)
	}
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}
	if quantity > 100 {
		return nil, fmt.Errorf("quantity cannot exceed 100 items per variant")
	}

	// Validate purchase type
	if purchaseType != "one_time" && purchaseType != "subscription" {
		return nil, fmt.Errorf("invalid purchase type: %s", purchaseType)
	}

	// Validate subscription interval if it's a subscription
	if purchaseType == "subscription" {
		if subscriptionInterval == nil {
			return nil, fmt.Errorf("subscription interval is required for subscription purchases")
		}
		if !isValidSubscriptionInterval(*subscriptionInterval) {
			return nil, fmt.Errorf("invalid subscription interval: %s", *subscriptionInterval)
		}
	}

	// Check variant availability
	availability, err := s.cartRepo.CheckVariantAvailability(ctx, productVariantID, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to check variant availability: %w", err)
	}

	if !availability.IsAvailable {
		if !availability.ProductActive {
			return nil, fmt.Errorf("product is no longer available")
		}
		if !availability.Active {
			return nil, fmt.Errorf("variant is no longer available")
		}
		if availability.Stock < quantity {
			return nil, fmt.Errorf("insufficient stock: requested %d, available %d", quantity, availability.Stock)
		}
		return nil, fmt.Errorf("variant is not available")
	}

	// Get variant details for pricing
	variant, err := s.variantRepo.GetByID(ctx, productVariantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variant details: %w", err)
	}

	// Get appropriate Stripe price ID
	stripePriceID, err := s.getStripePriceID(variant, purchaseType, subscriptionInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe price ID: %w", err)
	}

	// Check if item with same variant and purchase type already exists
	existingItems, err := s.cartRepo.GetCartItemsByVariant(ctx, cartID, productVariantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing items: %w", err)
	}

	for _, item := range existingItems {
		if item.PurchaseType == purchaseType {
			// Check subscription interval match for subscriptions
			if purchaseType == "subscription" {
				if (item.SubscriptionInterval.Valid && subscriptionInterval != nil &&
					item.SubscriptionInterval.String == *subscriptionInterval) ||
					(!item.SubscriptionInterval.Valid && subscriptionInterval == nil) {
					// Update existing item quantity
					newQuantity := item.Quantity + quantity
					if newQuantity > 100 {
						return nil, fmt.Errorf("total quantity would exceed 100 items per variant")
					}

					// Check stock for new total quantity
					availability, err := s.cartRepo.CheckVariantAvailability(ctx, productVariantID, newQuantity)
					if err != nil || !availability.IsAvailable {
						return nil, fmt.Errorf("insufficient stock for total quantity: %d", newQuantity)
					}

					return s.cartRepo.UpdateItemQuantity(ctx, item.ID, newQuantity)
				}
			} else {
				// For one-time purchases, just increment quantity
				newQuantity := item.Quantity + quantity
				if newQuantity > 100 {
					return nil, fmt.Errorf("total quantity would exceed 100 items per variant")
				}

				// Check stock for new total quantity
				availability, err := s.cartRepo.CheckVariantAvailability(ctx, productVariantID, newQuantity)
				if err != nil || !availability.IsAvailable {
					return nil, fmt.Errorf("insufficient stock for total quantity: %d", newQuantity)
				}

				return s.cartRepo.UpdateItemQuantity(ctx, item.ID, newQuantity)
			}
		}
	}

	// Add new item to cart
	cartItem, err := s.cartRepo.AddItem(ctx, cartID, productVariantID, quantity, variant.Price, purchaseType, subscriptionInterval, stripePriceID)
	if err != nil {
		return nil, fmt.Errorf("failed to add item to cart: %w", err)
	}

	// Publish item added event
	if err := s.publishCartEvent(ctx, "cart.item_added", cartID, map[string]interface{}{
		"product_variant_id":    productVariantID,
		"quantity":              quantity,
		"price":                 variant.Price,
		"purchase_type":         purchaseType,
		"subscription_interval": subscriptionInterval,
	}); err != nil {
		log.Printf("Failed to publish cart item added event: %v", err)
	}

	return cartItem, nil
}

// UpdateItemQuantity updates the quantity of a cart item
func (s *CartService) UpdateItemQuantity(ctx context.Context, itemID int32, quantity int32) (*interfaces.CartItem, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

	if quantity > 100 {
		return nil, fmt.Errorf("quantity cannot exceed 100 items per variant")
	}

	// Get existing item to validate variant
	existingItem, err := s.cartRepo.GetCartItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("cart item not found: %w", err)
	}

	// Check variant availability for new quantity
	availability, err := s.cartRepo.CheckVariantAvailability(ctx, existingItem.ProductVariantID, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to check variant availability: %w", err)
	}

	if !availability.IsAvailable {
		if !availability.ProductActive {
			return nil, fmt.Errorf("product is no longer available")
		}
		if !availability.Active {
			return nil, fmt.Errorf("variant is no longer available")
		}
		if availability.Stock < quantity {
			return nil, fmt.Errorf("insufficient stock: requested %d, available %d", quantity, availability.Stock)
		}
		return nil, fmt.Errorf("variant is not available")
	}

	// Update quantity
	updatedItem, err := s.cartRepo.UpdateItemQuantity(ctx, itemID, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to update item quantity: %w", err)
	}

	// Publish item updated event
	if err := s.publishCartEvent(ctx, "cart.item_updated", updatedItem.CartID, map[string]interface{}{
		"item_id":            itemID,
		"product_variant_id": updatedItem.ProductVariantID,
		"old_quantity":       existingItem.Quantity,
		"new_quantity":       quantity,
	}); err != nil {
		log.Printf("Failed to publish cart item updated event: %v", err)
	}

	return updatedItem, nil
}

// RemoveItem removes an item from the cart
func (s *CartService) RemoveItem(ctx context.Context, itemID int32) error {
	// Get item details before removal for event
	item, err := s.cartRepo.GetCartItem(ctx, itemID)
	if err != nil {
		return fmt.Errorf("cart item not found: %w", err)
	}

	err = s.cartRepo.RemoveItem(ctx, itemID)
	if err != nil {
		return fmt.Errorf("failed to remove item: %w", err)
	}

	// Publish item removed event
	if err := s.publishCartEvent(ctx, "cart.item_removed", item.CartID, map[string]interface{}{
		"item_id":            itemID,
		"product_variant_id": item.ProductVariantID,
		"quantity":           item.Quantity,
	}); err != nil {
		log.Printf("Failed to publish cart item removed event: %v", err)
	}

	return nil
}

// RemoveItemByVariantID removes an item from the cart by variant ID
func (s *CartService) RemoveItemByVariantID(ctx context.Context, cartID int32, productVariantID int32) error {
	err := s.cartRepo.RemoveItemByVariantID(ctx, cartID, productVariantID)
	if err != nil {
		return fmt.Errorf("failed to remove item by variant ID: %w", err)
	}

	// Publish item removed event
	if err := s.publishCartEvent(ctx, "cart.item_removed_by_variant", cartID, map[string]interface{}{
		"product_variant_id": productVariantID,
	}); err != nil {
		log.Printf("Failed to publish cart item removed event: %v", err)
	}

	return nil
}

// Clear removes all items from the cart
func (s *CartService) Clear(ctx context.Context, cartID int32) error {
	err := s.cartRepo.Clear(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to clear cart: %w", err)
	}

	// Publish cart cleared event
	if err := s.publishCartEvent(ctx, "cart.cleared", cartID, map[string]interface{}{}); err != nil {
		log.Printf("Failed to publish cart cleared event: %v", err)
	}

	return nil
}

// =============================================================================
// Cart Validation and Maintenance
// =============================================================================

// ValidateCartForCheckout validates cart contents before checkout
func (s *CartService) ValidateCartForCheckout(ctx context.Context, cartID int32) (*interfaces.CartWithItems, error) {
	cartWithItems, err := s.GetCart(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if len(cartWithItems.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Run validation on all items
	validations, err := s.cartRepo.ValidateCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate cart items: %w", err)
	}

	var invalidItems []string
	for _, validation := range validations {
		if validation.ValidationStatus != "valid" {
			switch validation.ValidationStatus {
			case "product_inactive":
				invalidItems = append(invalidItems, fmt.Sprintf("Product no longer available (item %d)", validation.CartItemID))
			case "variant_inactive":
				invalidItems = append(invalidItems, fmt.Sprintf("Variant no longer available (item %d)", validation.CartItemID))
			case "variant_archived":
				invalidItems = append(invalidItems, fmt.Sprintf("Variant has been discontinued (item %d)", validation.CartItemID))
			case "insufficient_stock":
				invalidItems = append(invalidItems, fmt.Sprintf("Insufficient stock: requested %d, available %d (item %d)",
					validation.RequestedQuantity, validation.AvailableStock, validation.CartItemID))
			}
		}
	}

	if len(invalidItems) > 0 {
		return nil, fmt.Errorf("cart validation failed: %s", invalidItems[0])
	}

	// Update prices if they've changed
	err = s.cartRepo.UpdateCartItemPrices(ctx, cartID)
	if err != nil {
		log.Printf("Failed to update cart item prices: %v", err)
	}

	// Return updated cart
	return s.GetCart(ctx, cartID)
}

// CleanupInvalidItems removes invalid items from cart
func (s *CartService) CleanupInvalidItems(ctx context.Context, cartID int32) ([]interfaces.CartItemWithVariant, error) {
	// Get invalid items before removing them
	invalidItems, err := s.cartRepo.GetInvalidCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invalid cart items: %w", err)
	}

	if len(invalidItems) > 0 {
		// Remove invalid items
		err = s.cartRepo.RemoveUnavailableItems(ctx, cartID)
		if err != nil {
			return nil, fmt.Errorf("failed to remove invalid cart items: %w", err)
		}

		// Publish cleanup event
		if err := s.publishCartEvent(ctx, "cart.cleaned_up", cartID, map[string]interface{}{
			"removed_items_count": len(invalidItems),
		}); err != nil {
			log.Printf("Failed to publish cart cleanup event: %v", err)
		}
	}

	return invalidItems, nil
}

// =============================================================================
// Cart Summary and Analytics
// =============================================================================

// GetCartSummary returns a summary of cart contents
func (s *CartService) GetCartSummary(ctx context.Context, cartID int32) (*interfaces.CartSummary, error) {
	total, err := s.cartRepo.GetCartTotal(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart total: %w", err)
	}

	itemCount, err := s.cartRepo.GetCartItemCount(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart item count: %w", err)
	}

	onetimeTotal, err := s.cartRepo.GetCartTotalByPurchaseType(ctx, cartID, "one_time")
	if err != nil {
		return nil, fmt.Errorf("failed to get one-time total: %w", err)
	}

	subscriptionSummaries, err := s.cartRepo.GetCartSubscriptionSummary(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription summary: %w", err)
	}

	return &interfaces.CartSummary{
		CartID:                cartID,
		Total:                 total,
		ItemCount:             itemCount,
		OneTimeTotal:          onetimeTotal,
		SubscriptionSummaries: subscriptionSummaries,
	}, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

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

// getStripePriceID returns the appropriate Stripe price ID for the purchase type
func (s *CartService) getStripePriceID(variant *interfaces.ProductVariant, purchaseType string, subscriptionInterval *string) (string, error) {
	switch purchaseType {
	case "one_time":
		if !variant.StripePriceOnetimeID.Valid {
			return "", fmt.Errorf("one-time purchase not available for this variant")
		}
		return variant.StripePriceOnetimeID.String, nil
	case "subscription":
		if subscriptionInterval == nil {
			return "", fmt.Errorf("subscription interval required for subscription purchases")
		}
		switch *subscriptionInterval {
		case "14_day":
			if !variant.StripePrice14dayID.Valid {
				return "", fmt.Errorf("14-day subscription not available for this variant")
			}
			return variant.StripePrice14dayID.String, nil
		case "21_day":
			if !variant.StripePrice21dayID.Valid {
				return "", fmt.Errorf("21-day subscription not available for this variant")
			}
			return variant.StripePrice21dayID.String, nil
		case "30_day":
			if !variant.StripePrice30dayID.Valid {
				return "", fmt.Errorf("30-day subscription not available for this variant")
			}
			return variant.StripePrice30dayID.String, nil
		case "60_day":
			if !variant.StripePrice60dayID.Valid {
				return "", fmt.Errorf("60-day subscription not available for this variant")
			}
			return variant.StripePrice60dayID.String, nil
		default:
			return "", fmt.Errorf("invalid subscription interval: %s", *subscriptionInterval)
		}
	default:
		return "", fmt.Errorf("invalid purchase type: %s", purchaseType)
	}
}

// isValidSubscriptionInterval validates subscription interval values
func isValidSubscriptionInterval(interval string) bool {
	validIntervals := []string{"14_day", "21_day", "30_day", "60_day"}
	for _, valid := range validIntervals {
		if interval == valid {
			return true
		}
	}
	return false
}

// publishCartEvent publishes cart-related events
func (s *CartService) publishCartEvent(ctx context.Context, eventType string, cartID int32, data map[string]interface{}) error {
	if s.events == nil {
		return nil // Events are optional
	}

	event := interfaces.Event{
		ID:          generateEventID(),
		Type:        eventType,
		AggregateID: fmt.Sprintf("cart:%d", cartID),
		Data:        data,
		Timestamp:   time.Now(),
		Version:     1,
	}

	return s.events.PublishEvent(ctx, event)
}
