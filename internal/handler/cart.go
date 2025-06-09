// internal/handler/cart.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/labstack/echo/v4"
)

type CartHandler struct {
	cartService *service.CartService
}

func NewCartHandler(cartService *service.CartService) *CartHandler {
	return &CartHandler{
		cartService: cartService,
	}
}

// GetCart handles GET /api/v1/cart
func (h *CartHandler) GetCart(c echo.Context) error {
	ctx := c.Request().Context()

	// Get customer ID from authentication (JWT) or session ID from header
	customerID := h.getCustomerIDFromContext(c)
	sessionID := h.getSessionIDFromHeader(c)

	if customerID == nil && sessionID == nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Authentication required or session ID must be provided",
			"code":  "MISSING_AUTH_OR_SESSION",
		})
	}

	cart, err := h.cartService.GetOrCreateCart(ctx, customerID, sessionID)
	if err != nil {
		c.Logger().Errorf("Failed to get cart: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve cart",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"cart": h.cartToAPI(cart),
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
	if req.ProductID <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Product ID must be greater than 0",
			"code":  "INVALID_PRODUCT_ID",
		})
	}

	if req.Quantity <= 0 || req.Quantity > 100 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Quantity must be between 1 and 100",
			"code":  "INVALID_QUANTITY",
		})
	}

	// Get customer ID from authentication or session ID from header
	customerID := h.getCustomerIDFromContext(c)
	sessionID := h.getSessionIDFromHeader(c)

	if customerID == nil && sessionID == nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Authentication required or session ID must be provided",
			"code":  "MISSING_AUTH_OR_SESSION",
		})
	}

	// Get or create cart
	cart, err := h.cartService.GetOrCreateCart(ctx, customerID, sessionID)
	if err != nil {
		c.Logger().Errorf("Failed to get cart: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve cart",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Add item to cart
	cartItem, err := h.cartService.AddItem(ctx, cart.ID, req.ProductID, req.Quantity)
	if err != nil {
		c.Logger().Errorf("Failed to add item to cart: %v", err)

		// Handle specific business logic errors
		switch {
		case err.Error() == "product not found":
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Product not found",
				"code":  "PRODUCT_NOT_FOUND",
			})
		case err.Error() == "product is not available":
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Product is not available",
				"code":  "PRODUCT_UNAVAILABLE",
			})
		case containsString(err.Error(), "insufficient stock"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INSUFFICIENT_STOCK",
			})
		case containsString(err.Error(), "quantity"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INVALID_QUANTITY",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to add item to cart",
				"code":  "INTERNAL_ERROR",
			})
		}
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"cart_item": h.cartItemToAPI(cartItem),
		"message":   "Item added to cart successfully",
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
	if req.Quantity <= 0 || req.Quantity > 100 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Quantity must be between 1 and 100",
			"code":  "INVALID_QUANTITY",
		})
	}

	// Update item quantity
	cartItem, err := h.cartService.UpdateItemQuantity(ctx, int32(itemID), req.Quantity)
	if err != nil {
		c.Logger().Errorf("Failed to update cart item %d: %v", itemID, err)

		// Handle specific business logic errors
		switch {
		case err.Error() == "cart item not found":
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Cart item not found",
				"code":  "CART_ITEM_NOT_FOUND",
			})
		case err.Error() == "product not found":
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Product not found",
				"code":  "PRODUCT_NOT_FOUND",
			})
		case err.Error() == "product is no longer available":
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Product is no longer available",
				"code":  "PRODUCT_UNAVAILABLE",
			})
		case containsString(err.Error(), "insufficient stock"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INSUFFICIENT_STOCK",
			})
		case containsString(err.Error(), "quantity"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INVALID_QUANTITY",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to update cart item",
				"code":  "INTERNAL_ERROR",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"cart_item": h.cartItemToAPI(cartItem),
		"message":   "Cart item updated successfully",
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

		if err.Error() == "cart item not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Cart item not found",
				"code":  "CART_ITEM_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to remove cart item",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// ClearCart handles DELETE /api/v1/cart
func (h *CartHandler) ClearCart(c echo.Context) error {
	ctx := c.Request().Context()

	// Get customer ID from authentication or session ID from header
	customerID := h.getCustomerIDFromContext(c)
	sessionID := h.getSessionIDFromHeader(c)

	if customerID == nil && sessionID == nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Authentication required or session ID must be provided",
			"code":  "MISSING_AUTH_OR_SESSION",
		})
	}

	// Get cart
	cart, err := h.cartService.GetOrCreateCart(ctx, customerID, sessionID)
	if err != nil {
		c.Logger().Errorf("Failed to get cart: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve cart",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Clear cart
	err = h.cartService.ClearCart(ctx, cart.ID)
	if err != nil {
		c.Logger().Errorf("Failed to clear cart %d: %v", cart.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to clear cart",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Cart cleared successfully",
	})
}

// GetCartSummary handles GET /api/v1/cart/summary
func (h *CartHandler) GetCartSummary(c echo.Context) error {
	ctx := c.Request().Context()

	// Get customer ID from authentication or session ID from header
	customerID := h.getCustomerIDFromContext(c)
	sessionID := h.getSessionIDFromHeader(c)

	if customerID == nil && sessionID == nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Authentication required or session ID must be provided",
			"code":  "MISSING_AUTH_OR_SESSION",
		})
	}

	// Get cart
	cart, err := h.cartService.GetOrCreateCart(ctx, customerID, sessionID)
	if err != nil {
		c.Logger().Errorf("Failed to get cart: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve cart",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Get cart summary
	summary, err := h.cartService.GetCartSummary(ctx, cart.ID)
	if err != nil {
		c.Logger().Errorf("Failed to get cart summary: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve cart summary",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, summary)
}

// Helper methods

func (h *CartHandler) getCustomerIDFromContext(c echo.Context) *int32 {
	// Extract from JWT token or X-Customer-ID header for testing
	if customerIDHeader := c.Request().Header.Get("X-Customer-ID"); customerIDHeader != "" {
		if id, err := strconv.Atoi(customerIDHeader); err == nil {
			customerID := int32(id)
			return &customerID
		}
	}
	// TODO: Extract from JWT token in production
	return nil
}

func (h *CartHandler) getSessionIDFromHeader(c echo.Context) *string {
	sessionID := c.Request().Header.Get("X-Session-ID")
	if sessionID == "" {
		return nil
	}
	return &sessionID
}

func (h *CartHandler) cartToAPI(cart *interfaces.CartWithItems) map[string]interface{} {
	items := make([]map[string]interface{}, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = map[string]interface{}{
			"id":              item.ID,
			"product_id":      item.ProductID,
			"name":            item.ProductName,
			"description":     item.ProductDescription,
			"quantity":        item.Quantity,
			"price":           item.Price,
			"subtotal":        item.Quantity * item.Price,
			"stock":           item.ProductStock,
			"price_formatted": formatPrice(int32(item.Price)),
		}
	}

	return map[string]interface{}{
		"id":              cart.ID,
		"items":           items,
		"total":           cart.Total,
		"item_count":      cart.ItemCount,
		"total_formatted": formatPrice(cart.Total),
		"created_at":      cart.CreatedAt,
		"updated_at":      cart.UpdatedAt,
	}
}

func (h *CartHandler) cartItemToAPI(item *interfaces.CartItem) map[string]interface{} {
	return map[string]interface{}{
		"id":              item.ID,
		"cart_id":         item.CartID,
		"product_id":      item.ProductID,
		"quantity":        item.Quantity,
		"price":           item.Price,
		"subtotal":        item.Quantity * item.Price,
		"price_formatted": formatPrice(item.Price),
		"created_at":      item.CreatedAt,
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
