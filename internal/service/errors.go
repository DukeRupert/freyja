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

// Payment terms errors - re-exported from domain
var (
	ErrPaymentTermsNotFound      = domain.ErrPaymentTermsNotFound
	ErrPaymentTermsInUse         = domain.ErrPaymentTermsInUse
	ErrDuplicatePaymentTermsCode = domain.ErrDuplicatePaymentTermsCode
)

// Wholesale invoice errors - re-exported from domain
var (
	ErrInvoiceAlreadyFinalized = domain.ErrInvoiceAlreadyFinalized
	ErrInvoiceNotDraft         = domain.ErrInvoiceNotDraft
	ErrInvoiceAlreadyPaid      = domain.ErrInvoiceAlreadyPaid
	ErrPaymentExceedsBalance   = domain.ErrPaymentExceedsBalance
	ErrNoOrdersToInvoice       = domain.ErrNoOrdersToInvoice
	ErrOrderNotWholesale       = domain.ErrOrderNotWholesale
	ErrOrderAlreadyInvoiced    = domain.ErrOrderAlreadyInvoiced
	ErrInvoiceNumberGeneration = domain.ErrInvoiceNumberGeneration
	ErrNoPaymentTermsAvailable = domain.ErrNoPaymentTermsAvailable
)

// Fulfillment errors - re-exported from domain
var (
	ErrShipmentNotFound       = domain.ErrShipmentNotFound
	ErrExceedsOrderedQuantity = domain.ErrExceedsOrderedQuantity
	ErrItemAlreadyFulfilled   = domain.ErrItemAlreadyFulfilled
	ErrNoItemsToShip          = domain.ErrNoItemsToShip
)

// User/customer errors - re-exported from domain
var (
	ErrNotWholesaleUser   = domain.ErrNotWholesaleUser
	ErrMinimumSpendNotMet = domain.ErrMinimumSpendNotMet
)
