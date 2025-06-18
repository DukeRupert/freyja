// internal/server/handler/cart.go
package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/labstack/echo/v4"
)

type CartHandler struct {
	cartService interfaces.CartService
}

func NewCartHandler(cartService interfaces.CartService) *CartHandler {
	return &CartHandler{
		cartService: cartService,
	}
}

// GetCart handles GET /api/v1/cart
func (h *CartHandler) GetCart(c echo.Context) error {
	ctx := c.Request().Context()

	// Get cart based on authentication
	cartID, err := h.resolveCartID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Unable to identify cart",
			"code":  "CART_IDENTIFICATION_FAILED",
		})
	}

	// Get cart with items (includes variant information)
	cartWithItems, err := h.cartService.GetCart(ctx, cartID)
	if err != nil {
		c.Logger().Errorf("Failed to get cart %d: %v", cartID, err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve cart",
			"code":  "CART_RETRIEVAL_FAILED",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    h.cartWithItemsToResponse(cartWithItems),
	})
}

// AddItem handles POST /api/v1/cart/items
func (h *CartHandler) AddItem(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse request body
	var req interfaces.AddCartItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	// Get cart based on authentication
	cartID, err := h.resolveCartID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Unable to identify cart",
			"code":  "CART_IDENTIFICATION_FAILED",
		})
	}

	// Add item to cart (now using product_variant_id)
	cartItem, err := h.cartService.AddItem(
		ctx,
		cartID,
		req.ProductVariantID, // Changed from ProductID to ProductVariantID
		req.Quantity,
		req.PurchaseType,
		req.SubscriptionInterval,
	)
	if err != nil {
		c.Logger().Errorf("Failed to add item to cart %d: %v", cartID, err)

		// Handle specific business logic errors
		switch {
		case strings.Contains(err.Error(), "variant not found"):
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Product variant not found",
				"code":  "VARIANT_NOT_FOUND",
			})
		case strings.Contains(err.Error(), "variant is not available"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Product variant is not available",
				"code":  "VARIANT_UNAVAILABLE",
			})
		case strings.Contains(err.Error(), "insufficient stock"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INSUFFICIENT_STOCK",
			})
		case strings.Contains(err.Error(), "quantity"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INVALID_QUANTITY",
			})
		case strings.Contains(err.Error(), "purchase type"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INVALID_PURCHASE_TYPE",
			})
		case strings.Contains(err.Error(), "subscription interval"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INVALID_SUBSCRIPTION_INTERVAL",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to add item to cart",
				"code":  "CART_ADD_FAILED",
			})
		}
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    h.cartItemToResponse(cartItem),
		"message": "Item added to cart successfully",
	})
}

// UpdateItem handles PUT /api/v1/cart/items/:id
func (h *CartHandler) UpdateItem(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse item ID
	itemIDParam := c.Param("id")
	itemID, err := strconv.Atoi(itemIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid cart item ID",
			"code":  "INVALID_ITEM_ID",
		})
	}

	// Parse request body
	var req interfaces.UpdateCartItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	// Update item quantity
	cartItem, err := h.cartService.UpdateItemQuantity(ctx, int32(itemID), req.Quantity)
	if err != nil {
		c.Logger().Errorf("Failed to update cart item %d: %v", itemID, err)

		// Handle specific business logic errors
		switch {
		case strings.Contains(err.Error(), "cart item not found"):
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Cart item not found",
				"code":  "CART_ITEM_NOT_FOUND",
			})
		case strings.Contains(err.Error(), "variant not found"):
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Product variant not found",
				"code":  "VARIANT_NOT_FOUND",
			})
		case strings.Contains(err.Error(), "variant is no longer available"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Product variant is no longer available",
				"code":  "VARIANT_UNAVAILABLE",
			})
		case strings.Contains(err.Error(), "insufficient stock"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INSUFFICIENT_STOCK",
			})
		case strings.Contains(err.Error(), "quantity"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INVALID_QUANTITY",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to update cart item",
				"code":  "CART_UPDATE_FAILED",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    h.cartItemToResponse(cartItem),
		"message": "Cart item updated successfully",
	})
}

// RemoveItem handles DELETE /api/v1/cart/items/:id
func (h *CartHandler) RemoveItem(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse item ID
	itemIDParam := c.Param("id")
	itemID, err := strconv.Atoi(itemIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid cart item ID",
			"code":  "INVALID_ITEM_ID",
		})
	}

	// Remove item from cart
	err = h.cartService.RemoveItem(ctx, int32(itemID))
	if err != nil {
		c.Logger().Errorf("Failed to remove cart item %d: %v", itemID, err)

		if strings.Contains(err.Error(), "cart item not found") {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Cart item not found",
				"code":  "CART_ITEM_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to remove cart item",
			"code":  "CART_REMOVE_FAILED",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Item removed from cart successfully",
	})
}

// ClearCart handles DELETE /api/v1/cart
func (h *CartHandler) ClearCart(c echo.Context) error {
	ctx := c.Request().Context()

	// Get cart based on authentication
	cartID, err := h.resolveCartID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Unable to identify cart",
			"code":  "CART_IDENTIFICATION_FAILED",
		})
	}

	// Clear cart
	err = h.cartService.Clear(ctx, cartID)
	if err != nil {
		c.Logger().Errorf("Failed to clear cart %d: %v", cartID, err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to clear cart",
			"code":  "CART_CLEAR_FAILED",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Cart cleared successfully",
	})
}

// GetCartSummary handles GET /api/v1/cart/summary
func (h *CartHandler) GetCartSummary(c echo.Context) error {
	ctx := c.Request().Context()

	// Get cart based on authentication
	cartID, err := h.resolveCartID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Unable to identify cart",
			"code":  "CART_IDENTIFICATION_FAILED",
		})
	}

	// Get cart summary
	summary, err := h.cartService.GetCartSummary(ctx, cartID)
	if err != nil {
		c.Logger().Errorf("Failed to get cart summary %d: %v", cartID, err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve cart summary",
			"code":  "CART_SUMMARY_FAILED",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    summary,
	})
}

// =============================================================================
// Helper Methods
// =============================================================================

// resolveCartID determines the cart ID based on customer ID or session ID
func (h *CartHandler) resolveCartID(c echo.Context) (int32, error) {
	ctx := c.Request().Context()

	// Check for customer ID (authenticated users)
	if customerIDHeader := c.Request().Header.Get("X-Customer-ID"); customerIDHeader != "" {
		customerID, err := strconv.Atoi(customerIDHeader)
		if err != nil {
			return 0, err
		}

		customerID32 := int32(customerID)
		cart, err := h.cartService.GetOrCreateCart(ctx, &customerID32, nil)
		if err != nil {
			return 0, err
		}
		return cart.ID, nil
	}

	// Check for session ID (guest users)
	if sessionID := c.Request().Header.Get("X-Session-ID"); sessionID != "" {
		cart, err := h.cartService.GetOrCreateCart(ctx, nil, &sessionID)
		if err != nil {
			return 0, err
		}
		return cart.ID, nil
	}

	return 0, fmt.Errorf("no customer ID or session ID provided")
}

// cartWithItemsToResponse converts CartWithItems to API response format
func (h *CartHandler) cartWithItemsToResponse(cart *interfaces.CartWithItems) interfaces.CartResponse {
	response := interfaces.CartResponse{
		ID:        cart.ID,
		Items:     make([]interfaces.CartItemResponse, len(cart.Items)),
		Total:     cart.Total,
		ItemCount: cart.ItemCount,
		CreatedAt: cart.CreatedAt,
		UpdatedAt: cart.UpdatedAt,
	}

	// Handle optional customer ID (already a pointer)
	if cart.CustomerID != nil {
		response.CustomerID = cart.CustomerID
	}

	// Handle optional session ID (already a pointer)
	if cart.SessionID != nil {
		response.SessionID = cart.SessionID
	}

	// Convert cart items
	for i, item := range cart.Items {
		response.Items[i] = h.cartItemWithVariantToResponse(&item)
	}

	return response
}

// cartItemToResponse converts CartItem to API response format
func (h *CartHandler) cartItemToResponse(item *interfaces.CartItem) interfaces.CartItemResponse {
	response := interfaces.CartItemResponse{
		ID:               item.ID,
		ProductVariantID: item.ProductVariantID,
		Quantity:         item.Quantity,
		Price:            item.Price,
		PurchaseType:     item.PurchaseType,
		StripePriceID:    item.StripePriceID,
		CreatedAt:        item.CreatedAt,
	}

	// Handle optional subscription interval
	if item.SubscriptionInterval.Valid {
		interval := item.SubscriptionInterval.String
		response.SubscriptionInterval = &interval
	}

	return response
}

// cartItemWithVariantToResponse converts CartItemWithVariant to API response format
func (h *CartHandler) cartItemWithVariantToResponse(item *interfaces.CartItemWithVariant) interfaces.CartItemResponse {
	response := interfaces.CartItemResponse{
		ID:               item.ID,
		ProductVariantID: item.ProductVariantID,
		Quantity:         item.Quantity,
		Price:            item.Price,
		PurchaseType:     item.PurchaseType,
		StripePriceID:    item.StripePriceID,
		CreatedAt:        item.CreatedAt,
	}

	// Handle optional subscription interval
	if item.SubscriptionInterval.Valid {
		interval := item.SubscriptionInterval.String
		response.SubscriptionInterval = &interval
	}

	// Add variant information to response
	response.VariantName = item.VariantName
	response.ProductName = item.ProductName
	response.ProductID = item.ProductID

	// Handle optional options display
	if item.OptionsDisplay.Valid {
		optionsDisplay := item.OptionsDisplay.String
		response.OptionsDisplay = &optionsDisplay
	}

	return response
}