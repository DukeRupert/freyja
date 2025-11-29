package address

import (
	"context"
)

// MockValidator is a test implementation of Validator.
type MockValidator struct {
	ValidateFunc func(ctx context.Context, addr Address) (*ValidationResult, error)
}

// NewMockValidator creates a new mock address validator for testing.
func NewMockValidator() *MockValidator {
	panic("not implemented")
}

// Validate delegates to the configured function or returns a default result.
func (m *MockValidator) Validate(ctx context.Context, addr Address) (*ValidationResult, error) {
	panic("not implemented")
}
