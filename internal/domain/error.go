package domain

import (
	"errors"
	"fmt"
)

// Application error codes.
// These map to HTTP status codes and determine user-facing messages.
const (
	ECONFLICT     = "conflict"        // 409 - Resource conflict (duplicate email, etc.)
	EINTERNAL     = "internal"        // 500 - Internal server error (hide details)
	EINVALID      = "invalid"         // 400 - Validation error (bad input)
	ENOTFOUND     = "not_found"       // 404 - Resource not found
	EUNAUTHORIZED = "unauthorized"    // 401 - Authentication required
	EFORBIDDEN    = "forbidden"       // 403 - Authenticated but not permitted
	ENOTIMPL      = "not_implemented" // 501 - Feature not implemented
	ERATELIMIT    = "rate_limit"      // 429 - Too many requests
	EPAYMENT      = "payment_required" // 402 - Payment failed or required
	EGONE         = "gone"            // 410 - Resource permanently deleted
)

// Error represents an application error with a code and message.
// It implements the error interface and supports error wrapping.
type Error struct {
	// Code is a machine-readable error code (e.g., EINVALID, ENOTFOUND).
	Code string

	// Message is a human-readable error message safe to show to users.
	Message string

	// Op is the operation where the error occurred (e.g., "product.create").
	// Used for debugging and logging, not shown to users.
	Op string

	// Err is the underlying error, if any. Used for error wrapping.
	Err error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err != nil {
		if e.Op != "" {
			return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
		}
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	if e.Op != "" {
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	}
	return e.Message
}

// Unwrap implements error unwrapping for errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.Err
}

// ErrorCode extracts the error code from an error.
// Returns EINTERNAL for nil or non-domain errors.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}

	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}

	return EINTERNAL
}

// ErrorMessage extracts a user-facing message from an error.
// For internal errors, returns a generic message to avoid leaking details.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	var e *Error
	if errors.As(err, &e) {
		// For internal errors, hide details from users
		if e.Code == EINTERNAL {
			return "An internal error occurred. Please try again later."
		}
		return e.Message
	}

	// Unknown error type - hide details
	return "An internal error occurred. Please try again later."
}

// ErrorOp extracts the operation from an error (for logging).
func ErrorOp(err error) string {
	if err == nil {
		return ""
	}

	var e *Error
	if errors.As(err, &e) {
		return e.Op
	}

	return ""
}

// Errorf creates a new domain error with formatted message.
// Example: domain.Errorf(domain.EINVALID, "product.validate", "invalid roast level: %s", level)
func Errorf(code, op, format string, args ...interface{}) error {
	return &Error{
		Code:    code,
		Op:      op,
		Message: fmt.Sprintf(format, args...),
	}
}

// WrapError wraps an existing error with a domain error code and operation.
// Preserves the underlying error for logging while providing structure.
// Returns nil if err is nil.
// Example: domain.WrapError(err, domain.EINTERNAL, "product.create", "failed to save product")
func WrapError(err error, code, op, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Code:    code,
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// IsCode returns true if err has the given error code.
func IsCode(err error, code string) bool {
	return ErrorCode(err) == code
}

// =============================================================================
// Validation Errors (field-level errors for forms)
// =============================================================================

// ValidationError represents one or more field validation failures.
// Used for form validation where multiple fields may have errors.
type ValidationError struct {
	// Fields maps field names to error messages.
	Fields map[string]string

	// Op is the operation where validation failed.
	Op string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if len(e.Fields) == 1 {
		for field, msg := range e.Fields {
			if e.Op != "" {
				return fmt.Sprintf("%s: %s: %s", e.Op, field, msg)
			}
			return fmt.Sprintf("%s: %s", field, msg)
		}
	}
	if e.Op != "" {
		return fmt.Sprintf("%s: validation failed for %d fields", e.Op, len(e.Fields))
	}
	return fmt.Sprintf("validation failed for %d fields", len(e.Fields))
}

// NewValidationError creates a validation error for a single field.
func NewValidationError(op, field, message string) error {
	return &ValidationError{
		Op:     op,
		Fields: map[string]string{field: message},
	}
}

// AddFieldError adds a field error to an existing ValidationError.
// If err is nil, creates a new ValidationError.
// If err is not a ValidationError, creates a new one with the field.
func AddFieldError(err error, field, message string) error {
	var ve *ValidationError
	if err != nil && errors.As(err, &ve) {
		ve.Fields[field] = message
		return ve
	}

	return &ValidationError{
		Fields: map[string]string{field: message},
	}
}

// IsValidationError returns true if err is a ValidationError.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// GetValidationFields extracts field errors from a ValidationError.
// Returns nil if err is not a ValidationError.
func GetValidationFields(err error) map[string]string {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve.Fields
	}
	return nil
}

// =============================================================================
// Multi-tenant errors (security-critical)
// =============================================================================

// Common multi-tenant errors as pre-defined instances for consistency.
var (
	// ErrTenantMismatch indicates an attempt to access a resource from another tenant.
	ErrTenantMismatch = &Error{
		Code:    EFORBIDDEN,
		Message: "Access denied: resource belongs to a different tenant",
	}

	// ErrTenantRequired indicates tenant context was expected but not found.
	ErrTenantRequired = &Error{
		Code:    EINTERNAL,
		Message: "Tenant context required but not found",
	}
)

// =============================================================================
// Common errors (convenience)
// =============================================================================

// NotFound creates a not found error for a resource.
// Example: domain.NotFound("product.get", "product", productID.String())
func NotFound(op, resource, identifier string) error {
	return &Error{
		Code:    ENOTFOUND,
		Op:      op,
		Message: fmt.Sprintf("%s not found: %s", resource, identifier),
	}
}

// Unauthorized creates an unauthorized error.
// Example: domain.Unauthorized("auth.check", "invalid credentials")
func Unauthorized(op, message string) error {
	return &Error{
		Code:    EUNAUTHORIZED,
		Op:      op,
		Message: message,
	}
}

// Forbidden creates a forbidden error.
// Example: domain.Forbidden("product.delete", "only product owner can delete")
func Forbidden(op, message string) error {
	return &Error{
		Code:    EFORBIDDEN,
		Op:      op,
		Message: message,
	}
}

// Invalid creates a validation error for a single issue.
// Example: domain.Invalid("product.create", "price must be positive")
func Invalid(op, message string) error {
	return &Error{
		Code:    EINVALID,
		Op:      op,
		Message: message,
	}
}

// Conflict creates a conflict error.
// Example: domain.Conflict("product.create", "product slug already exists")
func Conflict(op, message string) error {
	return &Error{
		Code:    ECONFLICT,
		Op:      op,
		Message: message,
	}
}

// Internal creates an internal error (wraps underlying error).
// The message shown to users will be generic; the underlying error is for logging.
// Example: domain.Internal(err, "product.create", "failed to save product")
func Internal(err error, op, message string) error {
	return &Error{
		Code:    EINTERNAL,
		Op:      op,
		Message: message,
		Err:     err,
	}
}
