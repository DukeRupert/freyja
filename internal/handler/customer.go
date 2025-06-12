// internal/handler/customer.go
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/labstack/echo/v4"
)

type CustomerHandler struct {
	service interfaces.CustomerService
}

func NewCustomerHandler(service interfaces.CustomerService) *CustomerHandler {
	return &CustomerHandler{
		service: service,
	}
}

// CreateCustomer creates a new customer
func (h *CustomerHandler) CreateCustomer(c echo.Context) error {
	var req interfaces.CreateCustomerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid JSON",
			"code":  "INVALID_REQUEST",
		})
	}

	ctx := c.Request().Context()

	customer, err := h.service.CreateCustomer(ctx, req)
	if err != nil {
		if err.Error() == "validation failed" ||
			err.Error() == "customer with email already exists" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"code":  "VALIDATION_ERROR",
			})
		}
		c.Logger().Errorf("Failed to create customer: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"customer": customer,
		"message":  "Customer created successfully",
	})
}

// GetCustomers retrieves a list of customers with pagination
func (h *CustomerHandler) GetCustomers(c echo.Context) error {
	// Parse query parameters
	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")

	limit := 50 // default
	offset := 0 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	ctx := c.Request().Context()

	// For now, we'll use the search functionality with empty query to get all customers
	customers, err := h.service.SearchCustomers(ctx, "", limit, offset)
	if err != nil {
		c.Logger().Errorf("Failed to retrieve customers: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve customers",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Get total count for pagination metadata
	totalCount, err := h.service.GetCustomerCount(ctx)
	if err != nil {
		c.Logger().Errorf("Failed to get customer count: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get customer count",
			"code":  "INTERNAL_ERROR",
		})
	}

	response := map[string]interface{}{
		"customers": customers,
		"pagination": map[string]interface{}{
			"total":  totalCount,
			"limit":  limit,
			"offset": offset,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// GetCustomerByID retrieves a specific customer by ID
func (h *CustomerHandler) GetCustomerByID(c echo.Context) error {
	customerID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid customer ID",
			"code":  "INVALID_ID",
		})
	}

	ctx := c.Request().Context()

	customer, err := h.service.GetCustomerByID(ctx, int32(customerID))
	if err != nil {
		if err.Error() == "customer not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Customer not found",
				"code":  "NOT_FOUND",
			})
		}
		c.Logger().Errorf("Failed to get customer: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, customer)
}

// UpdateCustomer updates an existing customer
func (h *CustomerHandler) UpdateCustomer(c echo.Context) error {
	customerID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid customer ID",
			"code":  "INVALID_ID",
		})
	}

	var req interfaces.UpdateCustomerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid JSON",
			"code":  "INVALID_REQUEST",
		})
	}

	ctx := c.Request().Context()

	customer, err := h.service.UpdateCustomer(ctx, int32(customerID), req)
	if err != nil {
		if err.Error() == "customer not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Customer not found",
				"code":  "NOT_FOUND",
			})
		}
		c.Logger().Errorf("Failed to update customer: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"customer": customer,
		"message":  "Customer updated successfully",
	})
}

// DeleteCustomer soft deletes a customer
func (h *CustomerHandler) DeleteCustomer(c echo.Context) error {
	customerID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid customer ID",
			"code":  "INVALID_ID",
		})
	}

	ctx := c.Request().Context()

	if err := h.service.DeleteCustomer(ctx, int32(customerID)); err != nil {
		if err.Error() == "customer not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Customer not found",
				"code":  "NOT_FOUND",
			})
		}
		c.Logger().Errorf("Failed to delete customer: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// SearchCustomers searches customers by query
func (h *CustomerHandler) SearchCustomers(c echo.Context) error {
	query := c.QueryParam("q")
	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")

	limit := 50 // default
	offset := 0 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	ctx := c.Request().Context()

	customers, err := h.service.SearchCustomers(ctx, query, limit, offset)
	if err != nil {
		c.Logger().Errorf("Failed to search customers: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to search customers",
			"code":  "INTERNAL_ERROR",
		})
	}

	response := map[string]interface{}{
		"customers": customers,
		"query":     query,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// GetCustomerStats returns customer analytics (admin only)
func (h *CustomerHandler) GetCustomerStats(c echo.Context) error {
	ctx := c.Request().Context()

	stats, err := h.service.GetCustomerStats(ctx)
	if err != nil {
		c.Logger().Errorf("Failed to get customer statistics: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get customer statistics",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, stats)
}

// GetCustomersWithoutStripe returns customers missing Stripe IDs (admin only)
func (h *CustomerHandler) GetCustomersWithoutStripe(c echo.Context) error {
	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")

	limit := 50 // default
	offset := 0 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	ctx := c.Request().Context()

	customers, err := h.service.GetCustomersWithoutStripeIDs(ctx, limit, offset)
	if err != nil {
		c.Logger().Errorf("Failed to get customers without Stripe IDs: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get customers without Stripe IDs",
			"code":  "INTERNAL_ERROR",
		})
	}

	// Get counts for metadata
	totalCustomers, _ := h.service.GetCustomerCount(ctx)
	customersWithStripe, _ := h.service.GetCustomersWithStripeCount(ctx)

	response := map[string]interface{}{
		"customers": customers,
		"metadata": map[string]interface{}{
			"total_customers":          totalCustomers,
			"customers_with_stripe":    customersWithStripe,
			"customers_without_stripe": totalCustomers - customersWithStripe,
		},
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// GetCustomerByEmail retrieves a specific customer by email
func (h *CustomerHandler) GetCustomerByEmail(c echo.Context) error {
	email := c.Param("email")
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Email is required",
			"code":  "MISSING_EMAIL",
		})
	}

	ctx := c.Request().Context()

	customer, err := h.service.GetCustomerByEmail(ctx, email)
	if err != nil {
		if err.Error() == "customer not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Customer not found",
				"code":  "NOT_FOUND",
			})
		}
		c.Logger().Errorf("Failed to get customer by email: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, customer)
}

// EnsureStripeCustomer creates or retrieves a Stripe customer ID for the customer
func (h *CustomerHandler) EnsureStripeCustomer(c echo.Context) error {
	customerID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid customer ID",
			"code":  "INVALID_ID",
		})
	}

	ctx := c.Request().Context()

	stripeCustomerID, err := h.service.EnsureStripeCustomer(ctx, int32(customerID))
	if err != nil {
		if err.Error() == "customer not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Customer not found",
				"code":  "NOT_FOUND",
			})
		}
		c.Logger().Errorf("Failed to ensure Stripe customer: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create/retrieve Stripe customer",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"stripe_customer_id": stripeCustomerID,
		"message":            "Stripe customer ensured successfully",
	})
}

// SyncStripeCustomers ensures all customers have Stripe customer IDs (admin only)
func (h *CustomerHandler) SyncStripeCustomers(c echo.Context) error {
	ctx := c.Request().Context()

	// Get customers without Stripe IDs
	customers, err := h.service.GetCustomersWithoutStripeIDs(ctx, 100, 0) // Process in batches
	if err != nil {
		c.Logger().Errorf("Failed to get customers for sync: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get customers for sync",
			"code":  "INTERNAL_ERROR",
		})
	}

	syncResults := map[string]interface{}{
		"processed":  0,
		"successful": 0,
		"failed":     0,
		"errors":     []string{},
	}

	for _, customer := range customers {
		syncResults["processed"] = syncResults["processed"].(int) + 1

		_, err := h.service.EnsureStripeCustomer(ctx, customer.ID)
		if err != nil {
			syncResults["failed"] = syncResults["failed"].(int) + 1
			errorMsg := fmt.Sprintf("Customer ID %d: %v", customer.ID, err)
			syncResults["errors"] = append(syncResults["errors"].([]string), errorMsg)
		} else {
			syncResults["successful"] = syncResults["successful"].(int) + 1
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Stripe customer sync completed",
		"results": syncResults,
	})
}
