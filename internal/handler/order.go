// internal/handler/order.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/labstack/echo/v4"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// GetOrders handles GET /api/v1/orders
func (h *OrderHandler) GetOrders(c echo.Context) error {
	ctx := c.Request().Context()

	// Check if user is authenticated
	customerID := getCustomerIDFromContext(c)
	if customerID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
			"code":  "UNAUTHORIZED",
		})
	}

	// Parse query parameters for filtering
	filters := interfaces.OrderFilters{}

	// Parse pagination
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if limit, err := strconv.Atoi(limitParam); err == nil && limit > 0 && limit <= 100 {
			filters.Limit = limit
		}
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if offset, err := strconv.Atoi(offsetParam); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Parse status filter
	if status := c.QueryParam("status"); status != "" {
		if interfaces.IsValidOrderStatus(status) {
			filters.Status = &status
		} else {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid order status",
				"code":  "INVALID_STATUS",
			})
		}
	}

	// Get customer's orders
	orders, err := h.orderService.GetByCustomer(ctx, *customerID, filters)
	if err != nil {
		c.Logger().Errorf("Failed to get orders for customer %d: %v", *customerID, err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve orders",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Convert to API format
	apiOrders := make([]map[string]interface{}, len(orders))
	for i, order := range orders {
		apiOrders[i] = h.orderToAPI(order)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"orders": apiOrders,
		"total":  len(apiOrders),
		"filters": map[string]interface{}{
			"limit":  filters.Limit,
			"offset": filters.Offset,
			"status": filters.Status,
		},
	})
}

// GetOrder handles GET /api/v1/orders/:id
func (h *OrderHandler) GetOrder(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse order ID
	orderIDParam := c.Param("id")
	orderID, err := strconv.Atoi(orderIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid order ID",
			"code":  "INVALID_ORDER_ID",
		})
	}

	// Check authentication
	customerID := getCustomerIDFromContext(c)
	if customerID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
			"code":  "UNAUTHORIZED",
		})
	}

	// Get order with items
	order, err := h.orderService.GetByID(ctx, int32(orderID))
	if err != nil {
		c.Logger().Errorf("Failed to get order %d: %v", orderID, err)

		if err.Error() == "order not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Order not found",
				"code":  "ORDER_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve order",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Verify order belongs to customer
	if order.CustomerID != *customerID {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Access denied to this order",
			"code":  "ACCESS_DENIED",
		})
	}

	return c.JSON(http.StatusOK, h.orderToAPI(*order))
}

// =============================================================================
// Admin Handlers (for staff/admin users)
// =============================================================================

// GetAllOrders handles GET /api/v1/admin/orders
func (h *OrderHandler) GetAllOrders(c echo.Context) error {
	ctx := c.Request().Context()

	// TODO: Add admin authentication check
	// For MVP, we'll assume this is protected by middleware

	// Parse query parameters
	filters := interfaces.OrderFilters{}

	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if limit, err := strconv.Atoi(limitParam); err == nil && limit > 0 && limit <= 100 {
			filters.Limit = limit
		}
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if offset, err := strconv.Atoi(offsetParam); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	if status := c.QueryParam("status"); status != "" {
		if interfaces.IsValidOrderStatus(status) {
			filters.Status = &status
		}
	}

	if customerIDParam := c.QueryParam("customer_id"); customerIDParam != "" {
		if customerID, err := strconv.Atoi(customerIDParam); err == nil && customerID > 0 {
			customerID32 := int32(customerID)
			filters.CustomerID = &customerID32
		}
	}

	// Get all orders
	orders, err := h.orderService.GetAll(ctx, filters)
	if err != nil {
		c.Logger().Errorf("Failed to get all orders: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve orders",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Convert to API format
	apiOrders := make([]map[string]interface{}, len(orders))
	for i, order := range orders {
		apiOrders[i] = h.orderToAPI(order)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"orders": apiOrders,
		"total":  len(apiOrders),
		"filters": filters,
	})
}

// UpdateOrderStatus handles PUT /api/v1/admin/orders/:id/status
func (h *OrderHandler) UpdateOrderStatus(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse order ID
	orderIDParam := c.Param("id")
	orderID, err := strconv.Atoi(orderIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid order ID",
			"code":  "INVALID_ORDER_ID",
		})
	}

	// Parse request body
	var req interfaces.UpdateOrderStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate request
	if err := interfaces.ValidateUpdateStatusRequest(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
	}

	// Update order status
	if err := h.orderService.UpdateOrderStatus(ctx, int32(orderID), req); err != nil {
		c.Logger().Errorf("Failed to update order %d status: %v", orderID, err)

		switch {
		case err.Error() == "order not found":
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Order not found",
				"code":  "ORDER_NOT_FOUND",
			})
		case containsString(err.Error(), "transition"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "INVALID_TRANSITION",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to update order status",
				"code":  "INTERNAL_ERROR",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Order status updated successfully",
	})
}

// CancelOrder handles POST /api/v1/orders/:id/cancel
func (h *OrderHandler) CancelOrder(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse order ID
	orderIDParam := c.Param("id")
	orderID, err := strconv.Atoi(orderIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid order ID",
			"code":  "INVALID_ORDER_ID",
		})
	}

	// Check authentication
	customerID := getCustomerIDFromContext(c)
	if customerID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
			"code":  "UNAUTHORIZED",
		})
	}

	// Parse request body for cancellation reason
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&req); err != nil {
		// Reason is optional, so ignore bind errors
		req.Reason = "Cancelled by customer"
	}

	// Cancel order
	if err := h.orderService.CancelOrder(ctx, int32(orderID), req.Reason); err != nil {
		c.Logger().Errorf("Failed to cancel order %d: %v", orderID, err)

		switch {
		case err.Error() == "order not found":
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Order not found",
				"code":  "ORDER_NOT_FOUND",
			})
		case containsString(err.Error(), "cannot be cancelled"):
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "CANNOT_CANCEL",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to cancel order",
				"code":  "INTERNAL_ERROR",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Order cancelled successfully",
	})
}

// GetOrderStats handles GET /api/v1/admin/orders/stats
func (h *OrderHandler) GetOrderStats(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse optional filters
	filters := interfaces.OrderFilters{}

	// Get order statistics
	stats, err := h.orderService.GetOrderStats(ctx, filters)
	if err != nil {
		c.Logger().Errorf("Failed to get order stats: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve order statistics",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"stats": stats,
	})
}

// =============================================================================
// Helper Methods
// =============================================================================

func (h *OrderHandler) orderToAPI(order interfaces.OrderWithItems) map[string]interface{} {
	apiOrder := map[string]interface{}{
		"id":          order.ID,
		"customer_id": order.CustomerID,
		"status":      order.Status,
		"total":       order.Total,
		"created_at":  order.CreatedAt,
		"updated_at":  order.UpdatedAt,
		"items":       h.orderItemsToAPI(order.Items),
		"item_count":  len(order.Items),
		"total_formatted": formatPrice(order.Total),
	}

	// Add optional Stripe fields
	if order.StripeSessionID != nil {
		apiOrder["stripe_session_id"] = *order.StripeSessionID
	}

	if order.StripePaymentIntentID != nil {
		apiOrder["stripe_payment_intent_id"] = *order.StripePaymentIntentID
	}

	return apiOrder
}

func (h *OrderHandler) orderItemsToAPI(items []interfaces.OrderItem) []map[string]interface{} {
	apiItems := make([]map[string]interface{}, len(items))
	for i, item := range items {
		apiItems[i] = map[string]interface{}{
			"id":          item.ID,
			"product_id":  item.ProductID,
			"name":        item.Name,
			"quantity":    item.Quantity,
			"price":       item.Price,
			"subtotal":    item.Quantity * item.Price,
			"price_formatted": formatPrice(item.Price),
			"subtotal_formatted": formatPrice(item.Quantity * item.Price),
			"created_at":  item.CreatedAt,
		}
	}
	return apiItems
}

// Helper functions (assuming these exist from other handlers)
func getCustomerIDFromContext(c echo.Context) *int32 {
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