package address

import (
	"context"
	"regexp"
	"strings"
)

// MockValidator performs basic address validation without external API calls.
// Suitable for MVP - validates required fields and basic formats.
type MockValidator struct{}

// NewMockValidator creates a new mock address validator.
func NewMockValidator() Validator {
	return &MockValidator{}
}

// Validate performs basic validation on required fields and formats.
// For MVP, this validates:
// - Required fields are not empty
// - US postal code format (5 digits or 5+4)
// - US state is valid 2-letter code
// Returns normalized address with trimmed/uppercased values.
func (m *MockValidator) Validate(ctx context.Context, addr Address) (*ValidationResult, error) {
	var errors []ValidationError
	var warnings []string

	// Create normalized copy
	normalized := Address{
		Type:         addr.Type,
		FullName:     strings.TrimSpace(addr.FullName),
		Company:      strings.TrimSpace(addr.Company),
		AddressLine1: strings.TrimSpace(addr.AddressLine1),
		AddressLine2: strings.TrimSpace(addr.AddressLine2),
		City:         strings.TrimSpace(addr.City),
		State:        strings.ToUpper(strings.TrimSpace(addr.State)),
		PostalCode:   strings.TrimSpace(addr.PostalCode),
		Country:      strings.ToUpper(strings.TrimSpace(addr.Country)),
		Phone:        strings.TrimSpace(addr.Phone),
	}

	// Validate required fields
	if normalized.AddressLine1 == "" {
		errors = append(errors, ValidationError{
			Field:   "AddressLine1",
			Message: "Address line 1 is required",
		})
	}

	if normalized.City == "" {
		errors = append(errors, ValidationError{
			Field:   "City",
			Message: "City is required",
		})
	}

	if normalized.State == "" {
		errors = append(errors, ValidationError{
			Field:   "State",
			Message: "State is required",
		})
	}

	if normalized.PostalCode == "" {
		errors = append(errors, ValidationError{
			Field:   "PostalCode",
			Message: "Postal code is required",
		})
	}

	if normalized.Country == "" {
		errors = append(errors, ValidationError{
			Field:   "Country",
			Message: "Country is required",
		})
	}

	// Validate US postal code format if country is US
	if normalized.Country == "US" && normalized.PostalCode != "" {
		// Accept 5 digits or 5+4 format
		postalRegex := regexp.MustCompile(`^\d{5}(-\d{4})?$`)
		if !postalRegex.MatchString(normalized.PostalCode) {
			errors = append(errors, ValidationError{
				Field:   "PostalCode",
				Message: "Invalid US postal code format (use 12345 or 12345-6789)",
			})
		}
	}

	// Validate US state code
	if normalized.Country == "US" && normalized.State != "" {
		if !isValidUSState(normalized.State) {
			errors = append(errors, ValidationError{
				Field:   "State",
				Message: "Invalid US state code (use 2-letter abbreviation)",
			})
		}
	}

	// Add warnings for optional but recommended fields
	if normalized.FullName == "" {
		warnings = append(warnings, "Recipient name is recommended for delivery")
	}

	if normalized.Phone == "" {
		warnings = append(warnings, "Phone number is recommended for delivery issues")
	}

	isValid := len(errors) == 0

	return &ValidationResult{
		IsValid:           isValid,
		NormalizedAddress: &normalized,
		Errors:            errors,
		Warnings:          warnings,
	}, nil
}

// isValidUSState checks if a 2-letter code is a valid US state abbreviation.
func isValidUSState(code string) bool {
	validStates := map[string]bool{
		"AL": true, "AK": true, "AZ": true, "AR": true, "CA": true,
		"CO": true, "CT": true, "DE": true, "FL": true, "GA": true,
		"HI": true, "ID": true, "IL": true, "IN": true, "IA": true,
		"KS": true, "KY": true, "LA": true, "ME": true, "MD": true,
		"MA": true, "MI": true, "MN": true, "MS": true, "MO": true,
		"MT": true, "NE": true, "NV": true, "NH": true, "NJ": true,
		"NM": true, "NY": true, "NC": true, "ND": true, "OH": true,
		"OK": true, "OR": true, "PA": true, "RI": true, "SC": true,
		"SD": true, "TN": true, "TX": true, "UT": true, "VT": true,
		"VA": true, "WA": true, "WV": true, "WI": true, "WY": true,
		"DC": true, // District of Columbia
	}
	return validStates[code]
}
