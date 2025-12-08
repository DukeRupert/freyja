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

// Order-related errors - re-exported from domain
var (
	ErrOrderNotFound           = domain.ErrOrderNotFound
	ErrPaymentNotSucceeded     = domain.ErrPaymentNotSucceeded
	ErrTenantMismatch          = domain.ErrTenantMismatch
	ErrCartAlreadyConverted    = domain.ErrCartAlreadyConverted
	ErrInsufficientStock       = domain.ErrInsufficientStock
	ErrMissingCartID           = domain.ErrMissingCartID
	ErrPaymentAlreadyProcessed = domain.ErrPaymentAlreadyProcessed
	ErrEmptyCart               = domain.ErrEmptyCart
	ErrMissingShippingAddress  = domain.ErrMissingShippingAddress
	ErrMissingBillingAddress   = domain.ErrMissingBillingAddress
	ErrMissingCustomerEmail    = domain.ErrMissingCustomerEmail
	ErrInvalidAddressJSON      = domain.ErrInvalidAddressJSON
)

// Subscription-related errors - re-exported from domain
var (
	ErrSubscriptionNotFound    = domain.ErrSubscriptionNotFound
	ErrNoPaymentMethod         = domain.ErrNoPaymentMethod
	ErrInvalidBillingInterval  = domain.ErrInvalidBillingInterval
	ErrSubscriptionNotActive   = domain.ErrSubscriptionNotActive
	ErrSubscriptionNotPaused   = domain.ErrSubscriptionNotPaused
	ErrInvoiceNotFound         = domain.ErrInvoiceNotFound
	ErrInvoiceAlreadyProcessed = domain.ErrInvoiceAlreadyProcessed
	ErrPaymentMethodOwnership  = domain.ErrPaymentMethodOwnership
	ErrInvoiceNotSubscription  = domain.ErrInvoiceNotSubscription
	ErrSubscriptionHasNoItems  = domain.ErrSubscriptionHasNoItems
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
