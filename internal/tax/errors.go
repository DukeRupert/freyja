package tax

// ============================================================================
// TAX ERROR CODES
// ============================================================================
// These constants mirror domain error codes to avoid circular imports.
// The handler layer maps these to HTTP status codes.

const (
	codeInternal = "internal"
	codeInvalid  = "invalid"
)

// ============================================================================
// TAX ERROR TYPE
// ============================================================================

// TaxError represents a tax-specific error with a code and message.
// It implements the domain.Error interface pattern for consistent HTTP status mapping.
type TaxError struct {
	Code    string
	Message string
}

func (e *TaxError) Error() string {
	return e.Message
}

// ErrorCode returns the error code for HTTP status mapping.
func (e *TaxError) ErrorCode() string {
	return e.Code
}

// ErrorMessage returns the user-facing message.
func (e *TaxError) ErrorMessage() string {
	return e.Message
}

// newTaxError creates a new tax error.
func newTaxError(code, message string) *TaxError {
	return &TaxError{Code: code, Message: message}
}

// ============================================================================
// TAX DOMAIN ERRORS
// ============================================================================
// Currently, tax calculation errors are infrastructure-level (database/conversion)
// and are correctly wrapped with fmt.Errorf. Domain-level errors can be added
// here as needed.

// Example typed errors for future use:
// var (
// 	ErrInvalidTaxRate = newTaxError(codeInvalid, "Tax rate must be between 0 and 1")
// 	ErrUnsupportedJurisdiction = newTaxError(codeInvalid, "Tax calculation not supported for this jurisdiction")
// )
