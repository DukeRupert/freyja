package billing

import (
	"fmt"
)

// ============================================================================
// BILLING ERROR CODES
// ============================================================================
// These constants mirror domain error codes to avoid circular imports.
// The handler layer maps these to HTTP status codes.

const (
	codeConflict     = "conflict"
	codeInternal     = "internal"
	codeInvalid      = "invalid"
	codeNotFound     = "not_found"
	codeUnauthorized = "unauthorized"
	codeForbidden    = "forbidden"
	codeNotImpl      = "not_implemented"
	codeRateLimit    = "rate_limit"
	codePayment      = "payment_required"
)

// ============================================================================
// BILLING ERROR TYPE
// ============================================================================

// BillingError represents a billing-specific error with a code and message.
// It implements the domain.Error interface pattern for consistent HTTP status mapping.
type BillingError struct {
	Code    string
	Message string
}

func (e *BillingError) Error() string {
	return e.Message
}

// ErrorCode returns the error code for HTTP status mapping.
func (e *BillingError) ErrorCode() string {
	return e.Code
}

// ErrorMessage returns the user-facing message.
func (e *BillingError) ErrorMessage() string {
	return e.Message
}

// newBillingError creates a new billing error.
func newBillingError(code, message string) *BillingError {
	return &BillingError{Code: code, Message: message}
}

// ============================================================================
// BILLING DOMAIN ERRORS
// ============================================================================
// These errors use domain error codes for consistent HTTP status mapping.
// The handler layer will automatically convert these to appropriate responses.

var (
	// ErrNotImplemented is returned when a provider method is not yet implemented.
	// Used for subscription and advanced features not needed in MVP.
	ErrNotImplemented = newBillingError(codeNotImpl, "Billing method not implemented")

	// ErrInvalidAPIKey is returned when Stripe API key is invalid or missing.
	ErrInvalidAPIKey = newBillingError(codeInternal, "Invalid or missing billing API key")

	// ErrPaymentIntentNotFound is returned when payment intent does not exist.
	ErrPaymentIntentNotFound = newBillingError(codeNotFound, "Payment intent not found")

	// ErrPaymentFailed is returned when payment fails (card declined, etc.)
	ErrPaymentFailed = newBillingError(codePayment, "Payment failed")

	// ErrInvalidWebhookSignature is returned when webhook signature verification fails.
	ErrInvalidWebhookSignature = newBillingError(codeUnauthorized, "Invalid webhook signature")

	// ErrIdempotencyConflict is returned when idempotency key matches a different request.
	ErrIdempotencyConflict = newBillingError(codeConflict, "Idempotency key conflict")

	// ErrAmountTooSmall is returned when payment amount is below Stripe's minimum.
	ErrAmountTooSmall = newBillingError(codeInvalid, "Amount too small (minimum $0.50 USD)")

	// ErrSubscriptionNotFound is returned when subscription does not exist or tenant mismatch.
	ErrSubscriptionNotFound = newBillingError(codeNotFound, "Subscription not found")

	// ErrInvoiceNotFound is returned when invoice does not exist.
	ErrInvoiceNotFound = newBillingError(codeNotFound, "Invoice not found")

	// ErrCustomerNotFound is returned when customer does not exist.
	ErrCustomerNotFound = newBillingError(codeNotFound, "Customer not found")

	// ErrTenantMismatch is returned when resource doesn't belong to specified tenant.
	ErrTenantMismatch = newBillingError(codeForbidden, "Resource does not belong to tenant")

	// ErrMissingTenantID is returned when tenant_id is required but not provided.
	ErrMissingTenantID = newBillingError(codeInvalid, "Tenant ID is required for multi-tenant isolation")

	// ErrMissingCustomerID is returned when customer_id is required but not provided.
	ErrMissingCustomerID = newBillingError(codeInvalid, "Customer ID is required")

	// ErrMissingEmail is returned when email is required but not provided.
	ErrMissingEmail = newBillingError(codeInvalid, "Email is required")

	// ErrMissingPaymentIntentID is returned when payment_intent_id is required but not provided.
	ErrMissingPaymentIntentID = newBillingError(codeInvalid, "Payment intent ID is required")

	// ErrMissingSubscriptionID is returned when subscription_id is required but not provided.
	ErrMissingSubscriptionID = newBillingError(codeInvalid, "Subscription ID is required")

	// ErrMissingInvoiceID is returned when invoice_id is required but not provided.
	ErrMissingInvoiceID = newBillingError(codeInvalid, "Invoice ID is required")

	// ErrMissingProductName is returned when product name is required but not provided.
	ErrMissingProductName = newBillingError(codeInvalid, "Product name is required")

	// ErrMissingPriceID is returned when price_id is required but not provided.
	ErrMissingPriceID = newBillingError(codeInvalid, "Price ID is required")

	// ErrMissingProductID is returned when product_id is required but not provided.
	ErrMissingProductID = newBillingError(codeInvalid, "Product ID is required")

	// ErrMissingCurrency is returned when currency is required but not provided.
	ErrMissingCurrency = newBillingError(codeInvalid, "Currency is required")

	// ErrMissingBillingInterval is returned when billing interval is required but not provided.
	ErrMissingBillingInterval = newBillingError(codeInvalid, "Billing interval is required")

	// ErrMissingReturnURL is returned when return URL is required but not provided.
	ErrMissingReturnURL = newBillingError(codeInvalid, "Return URL is required")

	// ErrInvalidQuantity is returned when quantity is invalid.
	ErrInvalidQuantity = newBillingError(codeInvalid, "Quantity must be greater than 0")

	// ErrInvalidUnitAmount is returned when unit amount is invalid.
	ErrInvalidUnitAmount = newBillingError(codeInvalid, "Unit amount must be greater than 0")

	// ErrMissingShippingAddress is returned when shipping address is required for tax calculation.
	ErrMissingShippingAddress = newBillingError(codeInvalid, "Shipping address is required for tax calculation")

	// ErrMissingCountry is returned when country is required but not provided.
	ErrMissingCountry = newBillingError(codeInvalid, "Shipping address country is required for tax calculation")

	// ErrMissingPostalCode is returned when postal code is required but not provided.
	ErrMissingPostalCode = newBillingError(codeInvalid, "Shipping address postal code is required for tax calculation")

	// ErrMissingSubscriptionMetadata is returned when subscription metadata is missing tenant_id.
	ErrMissingSubscriptionMetadata = newBillingError(codeInvalid, "Subscription missing tenant_id metadata")
)

// ============================================================================
// STRIPE ERROR WRAPPER
// ============================================================================

// StripeError wraps a Stripe API error with additional context.
// It implements the domain.Error interface pattern for consistent error handling.
type StripeError struct {
	DomainCode    string // Domain error code (codePayment, codeInternal, etc.)
	Message       string // Human-readable error message
	Code          string // Stripe error code (e.g., "card_declined")
	DeclineCode   string // Card decline reason (if applicable)
	StripeCode    string // HTTP status code from Stripe
	RequestID     string // Stripe request ID for debugging
	OriginalError error  // Original error from Stripe SDK
}

func (e *StripeError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("stripe: %s (code: %s)", e.Message, e.Code)
	}
	return fmt.Sprintf("stripe: %s", e.Message)
}

func (e *StripeError) Unwrap() error {
	return e.OriginalError
}

// ErrorCode returns the domain error code for HTTP status mapping.
func (e *StripeError) ErrorCode() string {
	return e.DomainCode
}

// ErrorMessage returns the user-facing message.
func (e *StripeError) ErrorMessage() string {
	return e.Message
}

// IsDeclined returns true if error is due to card decline.
func (e *StripeError) IsDeclined() bool {
	return e.Code == "card_declined" || e.DeclineCode != ""
}

// IsTemporary returns true if error is likely transient and retryable.
func (e *StripeError) IsTemporary() bool {
	return e.Code == "rate_limit" || e.Code == "api_connection_error"
}

// NewStripeError creates a new StripeError with appropriate domain code.
func NewStripeError(message, code, declineCode, requestID string, original error) *StripeError {
	domainCode := codeInternal // Default

	// Map Stripe error codes to domain codes
	switch code {
	case "card_declined", "expired_card", "incorrect_cvc", "processing_error":
		domainCode = codePayment
	case "rate_limit":
		domainCode = codeRateLimit
	case "invalid_request_error":
		domainCode = codeInvalid
	case "authentication_error":
		domainCode = codeUnauthorized
	case "resource_missing":
		domainCode = codeNotFound
	case "idempotency_error":
		domainCode = codeConflict
	}

	return &StripeError{
		DomainCode:    domainCode,
		Message:       message,
		Code:          code,
		DeclineCode:   declineCode,
		RequestID:     requestID,
		OriginalError: original,
	}
}
