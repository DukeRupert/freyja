package email

import "fmt"

// ============================================================================
// EMAIL ERROR CODES
// ============================================================================
// These constants mirror domain error codes to avoid circular imports.
// The handler layer maps these to HTTP status codes.

const (
	codeInternal = "internal"
	codeNotFound = "not_found"
	codeNotImpl  = "not_implemented"
	codeInvalid  = "invalid"
)

// ============================================================================
// EMAIL ERROR TYPE
// ============================================================================

// EmailError represents an email-specific error with a code and message.
// It implements the domain.Error interface pattern for consistent HTTP status mapping.
type EmailError struct {
	Code    string
	Message string
}

func (e *EmailError) Error() string {
	return e.Message
}

// ErrorCode returns the error code for HTTP status mapping.
func (e *EmailError) ErrorCode() string {
	return e.Code
}

// ErrorMessage returns the user-facing message.
func (e *EmailError) ErrorMessage() string {
	return e.Message
}

// newEmailError creates a new email error.
func newEmailError(code, message string) *EmailError {
	return &EmailError{Code: code, Message: message}
}

// ============================================================================
// EMAIL DOMAIN ERRORS
// ============================================================================

var (
	// ErrNotImplemented is returned when an email method is not yet implemented.
	ErrNotImplemented = newEmailError(codeNotImpl, "Email method not implemented")

	// ErrInvalidFromAddress is returned when the from address is invalid.
	ErrInvalidFromAddress = newEmailError(codeInvalid, "Invalid from email address")

	// ErrInvalidToAddress is returned when the to address is invalid.
	ErrInvalidToAddress = newEmailError(codeInvalid, "Invalid to email address")
)

// ErrTemplateNotFound creates a template not found error.
func ErrTemplateNotFound(templateName string) error {
	return &EmailError{
		Code:    codeNotFound,
		Message: fmt.Sprintf("Email template %s not found", templateName),
	}
}
