package storage

import "fmt"

// ============================================================================
// STORAGE ERROR CODES
// ============================================================================
// These constants mirror domain error codes to avoid circular imports.
// The handler layer maps these to HTTP status codes.

const (
	codeInternal = "internal"
	codeInvalid  = "invalid"
	codeNotFound = "not_found"
)

// ============================================================================
// STORAGE ERROR TYPE
// ============================================================================

// StorageError represents a storage-specific error with a code and message.
// It implements the domain.Error interface pattern for consistent HTTP status mapping.
type StorageError struct {
	Code    string
	Message string
}

func (e *StorageError) Error() string {
	return e.Message
}

// ErrorCode returns the error code for HTTP status mapping.
func (e *StorageError) ErrorCode() string {
	return e.Code
}

// ErrorMessage returns the user-facing message.
func (e *StorageError) ErrorMessage() string {
	return e.Message
}

// newStorageError creates a new storage error.
func newStorageError(code, message string) *StorageError {
	return &StorageError{Code: code, Message: message}
}

// ============================================================================
// STORAGE DOMAIN ERRORS
// ============================================================================

var (
	// ErrR2AccountIDRequired is returned when R2 account ID is missing.
	ErrR2AccountIDRequired = newStorageError(codeInvalid, "R2 account ID is required")

	// ErrR2CredentialsRequired is returned when R2 credentials are missing.
	ErrR2CredentialsRequired = newStorageError(codeInvalid, "R2 credentials are required")

	// ErrR2BucketRequired is returned when R2 bucket name is missing.
	ErrR2BucketRequired = newStorageError(codeInvalid, "R2 bucket name is required")
)

// ErrFileNotFound creates an error for when a file is not found.
func ErrFileNotFound(key string) error {
	return &StorageError{
		Code:    codeNotFound,
		Message: fmt.Sprintf("file not found: %s", key),
	}
}

// ErrUnknownProvider creates an error for unknown storage providers.
func ErrUnknownProvider(provider string) error {
	return &StorageError{
		Code:    codeInvalid,
		Message: fmt.Sprintf("unknown storage provider: %s", provider),
	}
}
