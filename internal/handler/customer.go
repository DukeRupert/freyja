// internal/handler/customer.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type CustomerHandler struct {
	customerService *service.CustomerService
}

func NewCustomerHandler(customerService *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
	}
}

// CreateCustomer handles POST /api/v1/customers
func (h *CustomerHandler) CreateCustomer(c echo.Context) error {
	ctx := c.Request().Context()

	var req service.CreateCustomerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// Hash the password (in a real app, this might be done in middleware or service)
	if password := c.FormValue("password"); password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to process password",
				"code":  "PASSWORD_HASH_ERROR",
			})
		}
		req.PasswordHash = string(hashedPassword)
	}

	// Validate required fields
	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Email is required",
			"code":  "MISSING_EMAIL",
		})
	}

	if req.PasswordHash == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Password is required",
			"code":  "MISSING_PASSWORD",
		})
	}

	// Create customer
	customer, err := h.customerService.CreateCustomer(ctx, req)
	if err != nil {
		c.Logger().Errorf("Failed to create customer: %v", err)

		if containsString(err.Error(), "already exists") {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "Customer with this email already exists",
				"code":  "CUSTOMER_EXISTS",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create customer",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"customer": h.customerToAPI(*customer),
		"message":  "Customer created successfully",
	})
}

// GetCustomer handles GET /api/v1/customers/:id
func (h *CustomerHandler) GetCustomer(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse customer ID
	customerIDParam := c.Param("id")
	customerID, err := strconv.Atoi(customerIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid customer ID",
			"code":  "INVALID_CUSTOMER_ID",
		})
	}

	// Get customer
	customer, err := h.customerService.GetCustomerByID(ctx, customerID)
	if err != nil {
		c.Logger().Errorf("Failed to get customer %d: %v", customerID, err)

		if err.Error() == "customer not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Customer not found",
				"code":  "CUSTOMER_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve customer",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, h.customerToAPI(*customer))
}

// GetCustomerByEmail handles GET /api/v1/customers/by-email/:email
func (h *CustomerHandler) GetCustomerByEmail(c echo.Context) error {
	ctx := c.Request().Context()

	email := c.Param("email")
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Email is required",
			"code":  "MISSING_EMAIL",
		})
	}

	customer, err := h.customerService.GetCustomerByEmail(ctx, email)
	if err != nil {
		c.Logger().Errorf("Failed to get customer by email %s: %v", email, err)

		if err.Error() == "customer not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Customer not found",
				"code":  "CUSTOMER_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve customer",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, h.customerToAPI(*customer))
}

// UpdateCustomer handles PUT /api/v1/customers/:id
func (h *CustomerHandler) UpdateCustomer(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse customer ID
	customerIDParam := c.Param("id")
	customerID, err := strconv.Atoi(customerIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid customer ID",
			"code":  "INVALID_CUSTOMER_ID",
		})
	}

	// Parse request body
	var req service.UpdateCustomerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// Update customer
	customer, err := h.customerService.UpdateCustomer(ctx, customerID, req)
	if err != nil {
		c.Logger().Errorf("Failed to update customer %d: %v", customerID, err)

		if err.Error() == "customer not found" || containsString(err.Error(), "failed to get customer") {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Customer not found",
				"code":  "CUSTOMER_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update customer",
			"code":  "INTERNAL_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"customer": h.customerToAPI(*customer),
		"message":  "Customer updated successfully",
	})
}

// EnsureStripeCustomer handles POST /api/v1/customers/:id/stripe
func (h *CustomerHandler) EnsureStripeCustomer(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse customer ID
	customerIDParam := c.Param("id")
	customerID, err := strconv.Atoi(customerIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid customer ID",
			"code":  "INVALID_CUSTOMER_ID",
		})
	}

	// Ensure Stripe customer exists
	stripeCustomerID, err := h.customerService.EnsureStripeCustomer(ctx, customerID)
	if err != nil {
		c.Logger().Errorf("Failed to ensure Stripe customer for %d: %v", customerID, err)

		if containsString(err.Error(), "failed to get customer") {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Customer not found",
				"code":  "CUSTOMER_NOT_FOUND",
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create Stripe customer",
			"code":  "STRIPE_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"stripe_customer_id": stripeCustomerID,
		"message":            "Stripe customer ensured successfully",
	})
}

// Helper method to convert Customer to API format
func (h *CustomerHandler) customerToAPI(customer interfaces.Customer) map[string]interface{} {
	apiCustomer := map[string]interface{}{
		"id":         customer.ID,
		"email":      customer.Email,
		"created_at": customer.CreatedAt,
		"updated_at": customer.UpdatedAt,
	}

	if customer.FirstName.Valid {
		apiCustomer["first_name"] = customer.FirstName.String
	}

	if customer.LastName.Valid {
		apiCustomer["last_name"] = customer.LastName.String
	}

	if customer.StripeCustomerID.Valid {
		apiCustomer["stripe_customer_id"] = customer.StripeCustomerID.String
	}

	return apiCustomer
}