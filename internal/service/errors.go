package service

import "errors"

var (
	// ErrProductNotFound is returned when a product cannot be found by slug
	ErrProductNotFound = errors.New("product not found")

	// ErrSKUNotFound is returned when a SKU cannot be found by ID
	ErrSKUNotFound = errors.New("SKU not found")

	// ErrPriceNotFound is returned when a price cannot be found for a SKU
	ErrPriceNotFound = errors.New("price not found")

	// ErrCartNotFound is returned when a cart cannot be found by ID
	ErrCartNotFound = errors.New("cart not found")

	// ErrCartItemNotFound is returned when a cart item cannot be found
	ErrCartItemNotFound = errors.New("cart item not found")

	// ErrInvalidQuantity is returned when a quantity is invalid (less than 1)
	ErrInvalidQuantity = errors.New("invalid quantity: must be greater than 0")

	// ErrSessionNotFound is returned when a session cannot be found
	ErrSessionNotFound = errors.New("session not found")

	// Order-related errors
	ErrOrderNotFound         = errors.New("order not found")
	ErrPaymentNotSucceeded   = errors.New("payment has not succeeded")
	ErrTenantMismatch        = errors.New("tenant ID mismatch")
	ErrCartAlreadyConverted  = errors.New("cart already converted to order")
	ErrInsufficientStock     = errors.New("insufficient stock for one or more items")
	ErrMissingCartID         = errors.New("cart_id missing from payment metadata")
	ErrPaymentAlreadyProcessed = errors.New("payment intent already processed")
)
