package shipping

import "fmt"

// ============================================================================
// SHIPPING ERROR CODES
// ============================================================================
// These constants mirror domain error codes to avoid circular imports.
// The handler layer maps these to HTTP status codes.

const (
	codeConflict     = "conflict"
	codeInternal     = "internal"
	codeInvalid      = "invalid"
	codeNotFound     = "not_found"
	codeForbidden    = "forbidden"
	codeNotImpl      = "not_implemented"
	codeUnavailable  = "unavailable" // For service-level errors like no rates
)

// ============================================================================
// SHIPPING ERROR TYPE
// ============================================================================

// ShippingError represents a shipping-specific error with a code and message.
// It implements the domain.Error interface pattern for consistent HTTP status mapping.
type ShippingError struct {
	Code    string
	Message string
}

func (e *ShippingError) Error() string {
	return e.Message
}

// ErrorCode returns the error code for HTTP status mapping.
func (e *ShippingError) ErrorCode() string {
	return e.Code
}

// ErrorMessage returns the user-facing message.
func (e *ShippingError) ErrorMessage() string {
	return e.Message
}

// newShippingError creates a new shipping error.
func newShippingError(code, message string) *ShippingError {
	return &ShippingError{Code: code, Message: message}
}

// ============================================================================
// SHIPPING DOMAIN ERRORS
// ============================================================================

var (
	// ErrNotImplemented is returned when a method is not yet implemented.
	ErrNotImplemented = newShippingError(codeNotImpl, "Shipping method not implemented")

	// ErrMultiPackageNotSupported is returned when multiple packages are provided.
	ErrMultiPackageNotSupported = newShippingError(codeNotImpl, "Multi-package shipments not yet supported")

	// ErrNoPackages is returned when no packages are provided.
	ErrNoPackages = newShippingError(codeInvalid, "At least one package is required")

	// ErrOriginRequired is returned when origin address is missing.
	ErrOriginRequired = newShippingError(codeInvalid, "Origin address is required")

	// ErrTenantRequired is returned when tenant ID is missing.
	ErrTenantRequired = newShippingError(codeInvalid, "Tenant ID is required")

	// ErrNoRates is returned when no shipping rates are available.
	ErrNoRates = newShippingError(codeUnavailable, "No shipping rates available")

	// ErrInvalidRate is returned when a rate ID is invalid or expired.
	ErrInvalidRate = newShippingError(codeInvalid, "Invalid or expired rate")

	// ErrLabelNotFound is returned when a label cannot be found.
	ErrLabelNotFound = newShippingError(codeNotFound, "Label not found")

	// ErrAddressInvalid is returned when address validation fails.
	ErrAddressInvalid = newShippingError(codeInvalid, "Address validation failed")

	// ErrTenantMismatch is returned when tenant validation fails.
	ErrTenantMismatch = newShippingError(codeForbidden, "Resource belongs to different tenant")

	// ErrLabelAlreadyPurchased is returned when attempting to purchase a label that already exists.
	ErrLabelAlreadyPurchased = newShippingError(codeConflict, "Label already purchased for this shipment")

	// ErrMissingAPIKey is returned when the shipping provider API key is missing.
	ErrMissingAPIKey = newShippingError(codeInternal, "Shipping provider API key is required")

	// ErrInvalidRateIDFormat is returned when the rate ID format is invalid.
	ErrInvalidRateIDFormat = newShippingError(codeInvalid, "Invalid rate ID format")
)

// ErrInvalidAmount creates an error for invalid amount parsing.
func ErrInvalidAmount(amount string, err error) error {
	return &ShippingError{
		Code:    codeInvalid,
		Message: fmt.Sprintf("Invalid dollar amount %q: %v", amount, err),
	}
}
