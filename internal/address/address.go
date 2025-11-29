package address

import "context"

// Validator defines the interface for address validation.
// Implementations can use external APIs like Google, USPS, Lob, SmartyStreets, etc.
// For MVP, use MockValidator for basic validation.
type Validator interface {
	// Validate checks if an address is valid and deliverable.
	// Returns normalized address if validation succeeds.
	// Even if IsValid is false, NormalizedAddress may contain corrections.
	Validate(ctx context.Context, addr Address) (*ValidationResult, error)
}

// Address represents a physical address for shipping or billing.
type Address struct {
	Type         string // "shipping" or "billing"
	FullName     string
	Company      string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Country      string
	Phone        string
}

// ValidationResult contains the outcome of address validation.
type ValidationResult struct {
	IsValid           bool
	NormalizedAddress *Address
	Errors            []ValidationError
	Warnings          []string
}

// ValidationError represents a specific validation error.
type ValidationError struct {
	Field   string
	Message string
}
