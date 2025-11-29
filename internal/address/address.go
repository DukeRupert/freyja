package address

import "context"

// Validator defines the interface for address validation.
// Implementations can use external APIs like Google, USPS, Lob, SmartyStreets, etc.
type Validator interface {
	// Validate checks if an address is valid and deliverable.
	// Returns normalized address if validation succeeds.
	Validate(ctx context.Context, addr Address) (*ValidationResult, error)
}

// Address represents a physical address to be validated.
type Address struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

// ValidationResult contains the outcome of address validation.
type ValidationResult struct {
	IsValid           bool
	IsNormalized      bool
	NormalizedAddress Address
	Messages          []ValidationMessage
}

// ValidationMessage represents a single validation error, warning, or info message.
type ValidationMessage struct {
	Severity string // "error", "warning", "info"
	Code     string // Machine-readable error code
	Message  string // Human-readable message
	Field    string // Affected field name
}
