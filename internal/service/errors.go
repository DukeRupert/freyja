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
	ErrOrderNotFound           = errors.New("order not found")
	ErrPaymentNotSucceeded     = errors.New("payment has not succeeded")
	ErrTenantMismatch          = errors.New("tenant ID mismatch")
	ErrCartAlreadyConverted    = errors.New("cart already converted to order")
	ErrInsufficientStock       = errors.New("insufficient stock for one or more items")
	ErrMissingCartID           = errors.New("cart_id missing from payment metadata")
	ErrPaymentAlreadyProcessed = errors.New("payment intent already processed")

	// Subscription-related errors
	ErrSubscriptionNotFound    = errors.New("subscription not found")
	ErrNoPaymentMethod         = errors.New("no payment method on file")
	ErrInvalidBillingInterval  = errors.New("invalid billing interval")
	ErrSubscriptionNotActive   = errors.New("subscription is not active")
	ErrSubscriptionNotPaused   = errors.New("subscription is not paused")
	ErrInvoiceNotFound         = errors.New("invoice not found")
	ErrInvoiceAlreadyProcessed = errors.New("invoice already processed")

	// Payment terms errors
	ErrPaymentTermsNotFound      = errors.New("payment terms not found")
	ErrPaymentTermsInUse         = errors.New("payment terms in use by customers or invoices")
	ErrDuplicatePaymentTermsCode = errors.New("payment terms code already exists")

	// Wholesale invoice errors
	ErrInvoiceAlreadyFinalized = errors.New("invoice already finalized")
	ErrInvoiceNotDraft         = errors.New("invoice must be in draft status")
	ErrInvoiceAlreadyPaid      = errors.New("invoice already paid in full")
	ErrPaymentExceedsBalance   = errors.New("payment amount exceeds invoice balance")
	ErrNoOrdersToInvoice       = errors.New("no uninvoiced orders found for period")
	ErrOrderNotWholesale       = errors.New("order is not a wholesale order")
	ErrOrderAlreadyInvoiced    = errors.New("order already invoiced")

	// Fulfillment errors
	ErrShipmentNotFound       = errors.New("shipment not found")
	ErrExceedsOrderedQuantity = errors.New("shipment quantity exceeds ordered quantity")
	ErrItemAlreadyFulfilled   = errors.New("order item already fully fulfilled")
	ErrNoItemsToShip          = errors.New("no items to ship")

	// User/customer errors
	ErrNotWholesaleUser   = errors.New("user is not a wholesale customer")
	ErrMinimumSpendNotMet = errors.New("order does not meet minimum spend requirement")
)
