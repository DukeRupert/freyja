// internal/handler/checkout.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/labstack/echo/v4"
)

type CheckoutHandler struct {
	checkoutService interfaces.CheckoutService
}

func NewCheckoutHandler(checkoutService interfaces.CheckoutService) *CheckoutHandler {
	return &CheckoutHandler{
		checkoutService: checkoutService,
	}
}

type CreateCheckoutSessionRequest struct {
	SuccessURL string `json:"success_url" validate:"required,url"`
	CancelURL  string `json:"cancel_url" validate:"required,url"`
}

type CreateCheckoutSessionResponse struct {
	CheckoutSessionID string `json:"checkout_session_id"`
	CheckoutURL       string `json:"checkout_url"`
}

// CreateCheckoutSession creates a Stripe checkout session from the customer's cart
func (h *CheckoutHandler) CreateCheckoutSession(c echo.Context) error {
	// Parse request
	var req CreateCheckoutSessionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Validation failed: " + err.Error(),
		})
	}

	// Get customer ID from auth or header (for testing)
	var customerID *int32
	if customerIDHeader := c.Request().Header.Get("X-Customer-ID"); customerIDHeader != "" {
		if id, err := strconv.ParseInt(customerIDHeader, 10, 32); err == nil {
			customerID32 := int32(id)
			customerID = &customerID32
		}
	}

	// Get session ID for guest carts
	var sessionID *string
	if sessionIDHeader := c.Request().Header.Get("X-Session-ID"); sessionIDHeader != "" {
		sessionID = &sessionIDHeader
	}

	// Ensure we have either customer ID or session ID
	if customerID == nil && sessionID == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Either X-Customer-ID or X-Session-ID header is required",
		})
	}

	// Create checkout session
	checkoutResponse, err := h.checkoutService.CreateCheckoutSession(
		c.Request().Context(),
		customerID,
		sessionID,
		req.SuccessURL,
		req.CancelURL,
	)
	if err != nil {
		// Handle specific error types
		switch err.Error() {
		case "cart is empty":
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Cannot checkout with an empty cart",
			})
		case "failed to get cart: cart not found":
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Cart not found",
			})
		default:
			if contains(err.Error(), "cart validation failed") {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Cart validation failed: " + err.Error(),
				})
			}
			
			// Log the error for debugging but return generic message
			c.Logger().Errorf("Failed to create checkout session: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create checkout session",
			})
		}
	}

	// Return successful response
	response := CreateCheckoutSessionResponse{
		CheckoutSessionID: checkoutResponse.SessionID,
		CheckoutURL:       checkoutResponse.CheckoutURL,
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}