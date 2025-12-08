package tax

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
