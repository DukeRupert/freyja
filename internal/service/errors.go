package service

import (
	"github.com/dukerupert/freyja/internal/domain"
)

// Product/SKU/Cart errors - re-exported from domain
var (
	ErrProductNotFound  = domain.ErrProductNotFound
	ErrSKUNotFound      = domain.ErrSKUNotFound
	ErrPriceNotFound    = domain.ErrPriceNotFound
	ErrCartNotFound     = domain.ErrCartNotFound
	ErrCartItemNotFound = domain.ErrCartItemNotFound
	ErrSessionNotFound  = domain.ErrSessionNotFound
	ErrInvalidQuantity  = domain.ErrInvalidQuantity
)

// Order-related errors
var (
	ErrOrderNotFound            = domain.Errorf(domain.ENOTFOUND, "", "Order not found")
	ErrPaymentNotSucceeded      = domain.Errorf(domain.EPAYMENT, "", "Payment has not succeeded")
	ErrTenantMismatch           = domain.ErrTenantMismatch
	ErrCartAlreadyConverted     = domain.Errorf(domain.ECONFLICT, "", "Cart already converted to order")
	ErrInsufficientStock        = domain.Errorf(domain.ECONFLICT, "", "Insufficient stock for one or more items")
	ErrMissingCartID            = domain.Errorf(domain.EINVALID, "", "Cart ID missing from payment metadata")
	ErrPaymentAlreadyProcessed  = domain.Errorf(domain.ECONFLICT, "", "Payment intent already processed")
	ErrEmptyCart                = domain.Errorf(domain.EINVALID, "", "Cart is empty")
	ErrMissingShippingAddress   = domain.Errorf(domain.EINVALID, "", "Shipping address missing from payment metadata")
	ErrMissingBillingAddress    = domain.Errorf(domain.EINVALID, "", "Billing address missing from payment metadata")
	ErrMissingCustomerEmail     = domain.Errorf(domain.EINVALID, "", "Customer email required for guest checkout")
	ErrInvalidAddressJSON       = domain.Errorf(domain.EINVALID, "", "Address JSON is empty or invalid")
)

// Subscription-related errors
var (
	ErrSubscriptionNotFound       = domain.Errorf(domain.ENOTFOUND, "", "Subscription not found")
	ErrNoPaymentMethod            = domain.Errorf(domain.EPAYMENT, "", "No payment method on file")
	ErrInvalidBillingInterval     = domain.Errorf(domain.EINVALID, "", "Invalid billing interval")
	ErrSubscriptionNotActive      = domain.Errorf(domain.EINVALID, "", "Subscription is not active")
	ErrSubscriptionNotPaused      = domain.Errorf(domain.EINVALID, "", "Subscription is not paused")
	ErrInvoiceNotFound            = domain.Errorf(domain.ENOTFOUND, "", "Invoice not found")
	ErrInvoiceAlreadyProcessed    = domain.Errorf(domain.ECONFLICT, "", "Invoice already processed")
	ErrPaymentMethodOwnership     = domain.Errorf(domain.EINVALID, "", "Payment method does not belong to user")
	ErrInvoiceNotSubscription     = domain.Errorf(domain.EINVALID, "", "Invoice is not for a subscription")
	ErrSubscriptionHasNoItems     = domain.Errorf(domain.EINVALID, "", "Subscription has no items")
)

// Payment terms errors
var (
	ErrPaymentTermsNotFound      = domain.Errorf(domain.ENOTFOUND, "", "Payment terms not found")
	ErrPaymentTermsInUse         = domain.Errorf(domain.ECONFLICT, "", "Payment terms in use by customers or invoices")
	ErrDuplicatePaymentTermsCode = domain.Errorf(domain.ECONFLICT, "", "Payment terms code already exists")
)

// Wholesale invoice errors
var (
	ErrInvoiceAlreadyFinalized  = domain.Errorf(domain.ECONFLICT, "", "Invoice already finalized")
	ErrInvoiceNotDraft          = domain.Errorf(domain.EINVALID, "", "Invoice must be in draft status")
	ErrInvoiceAlreadyPaid       = domain.Errorf(domain.ECONFLICT, "", "Invoice already paid in full")
	ErrPaymentExceedsBalance    = domain.Errorf(domain.EINVALID, "", "Payment amount exceeds invoice balance")
	ErrNoOrdersToInvoice        = domain.Errorf(domain.ENOTFOUND, "", "No uninvoiced orders found for period")
	ErrOrderNotWholesale        = domain.Errorf(domain.EINVALID, "", "Order is not a wholesale order")
	ErrOrderAlreadyInvoiced     = domain.Errorf(domain.ECONFLICT, "", "Order already invoiced")
	ErrInvoiceNumberGeneration  = domain.Errorf(domain.EINTERNAL, "", "Failed to generate invoice number")
	ErrNoPaymentTermsAvailable  = domain.Errorf(domain.ENOTFOUND, "", "No payment terms available")
)

// Fulfillment errors
var (
	ErrShipmentNotFound       = domain.Errorf(domain.ENOTFOUND, "", "Shipment not found")
	ErrExceedsOrderedQuantity = domain.Errorf(domain.EINVALID, "", "Shipment quantity exceeds ordered quantity")
	ErrItemAlreadyFulfilled   = domain.Errorf(domain.ECONFLICT, "", "Order item already fully fulfilled")
	ErrNoItemsToShip          = domain.Errorf(domain.EINVALID, "", "No items to ship")
)

// User/customer errors
var (
	ErrNotWholesaleUser   = domain.Errorf(domain.EFORBIDDEN, "", "User is not a wholesale customer")
	ErrMinimumSpendNotMet = domain.Errorf(domain.EINVALID, "", "Order does not meet minimum spend requirement")
)
