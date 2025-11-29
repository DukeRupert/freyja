package address

import (
	"context"
)

// BasicValidator performs basic format validation without external API calls.
// Checks for required fields and basic format rules (e.g., ZIP code format).
type BasicValidator struct{}

// NewBasicValidator creates a new basic address validator.
func NewBasicValidator() Validator {
	panic("not implemented")
}

// Validate performs basic validation checks on the address.
// TODO: Implement required field checks and format validation (ZIP code, etc.)
func (v *BasicValidator) Validate(ctx context.Context, addr Address) (*ValidationResult, error) {
	panic("not implemented")
}
