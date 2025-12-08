package provider

import "fmt"

// ============================================================================
// PROVIDER ERROR CODES
// ============================================================================
// These constants mirror domain error codes to avoid circular imports.
// The handler layer maps these to HTTP status codes.

const (
	codeInternal = "internal"
	codeInvalid  = "invalid"
	codeNotFound = "not_found"
	codeNotImpl  = "not_implemented"
)

// ============================================================================
// PROVIDER ERROR TYPE
// ============================================================================

// ProviderError represents a provider-specific error with a code and message.
// It implements the domain.Error interface pattern for consistent HTTP status mapping.
type ProviderError struct {
	Code    string
	Message string
}

func (e *ProviderError) Error() string {
	return e.Message
}

// ErrorCode returns the error code for HTTP status mapping.
func (e *ProviderError) ErrorCode() string {
	return e.Code
}

// ErrorMessage returns the user-facing message.
func (e *ProviderError) ErrorMessage() string {
	return e.Message
}

// newProviderError creates a new provider error.
func newProviderError(code, message string) *ProviderError {
	return &ProviderError{Code: code, Message: message}
}

// ============================================================================
// PROVIDER DOMAIN ERRORS
// ============================================================================

var (
	// ErrNilValidator is returned when a nil validator is passed to NewDefaultFactory.
	ErrNilValidator = newProviderError(codeInvalid, "validator cannot be nil")

	// ErrNilConfig is returned when a nil config is passed to factory methods.
	ErrNilConfig = newProviderError(codeInvalid, "config cannot be nil")

	// ErrMissingRepository is returned when repository is missing from tax config.
	ErrMissingRepository = newProviderError(codeInvalid, "missing or invalid repository in config")

	// ErrFlatRateNotImplemented is returned when flat rate shipping is requested but not implemented.
	ErrFlatRateNotImplemented = newProviderError(codeNotImpl, "flat rate shipping provider not yet implemented")
)

// ErrProviderTypeMismatch creates an error for provider type mismatches.
func ErrProviderTypeMismatch(expected, got ProviderType) error {
	return &ProviderError{
		Code:    codeInvalid,
		Message: fmt.Sprintf("expected provider type %s, got %s", expected, got),
	}
}

// ErrValidationFailed creates an error for config validation failures.
func ErrValidationFailed(providerType string, errors []string) error {
	return &ProviderError{
		Code:    codeInvalid,
		Message: fmt.Sprintf("%s config validation failed: %v", providerType, errors),
	}
}

// ErrUnknownProvider creates an error for unknown provider names.
func ErrUnknownProvider(providerType string, name ProviderName) error {
	return &ProviderError{
		Code:    codeInvalid,
		Message: fmt.Sprintf("unknown %s provider: %s", providerType, name),
	}
}

// ErrNoProviderConfigured creates an error when no provider is configured for tenant.
func ErrNoProviderConfigured(providerType string) error {
	return &ProviderError{
		Code:    codeNotFound,
		Message: fmt.Sprintf("no %s provider configured for tenant", providerType),
	}
}

// ErrConfigKeyNotFound creates an error for missing config keys.
func ErrConfigKeyNotFound(key string) error {
	return &ProviderError{
		Code:    codeInvalid,
		Message: fmt.Sprintf("config key %q not found", key),
	}
}

// ErrConfigKeyWrongType creates an error for config keys with wrong type.
func ErrConfigKeyWrongType(key string, expectedType string, gotType interface{}) error {
	return &ProviderError{
		Code:    codeInvalid,
		Message: fmt.Sprintf("config key %q must be %s, got %T", key, expectedType, gotType),
	}
}
