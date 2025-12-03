package shipping

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotImplemented is returned when a method is not yet implemented.
	ErrNotImplemented = errors.New("not implemented")
	// ErrMultiPackageNotSupported is returned when multiple packages are provided.
	ErrMultiPackageNotSupported = errors.New("multi-package shipments not yet supported")
	// ErrNoPackages is returned when no packages are provided.
	ErrNoPackages = errors.New("at least one package is required")
	// ErrOriginRequired is returned when origin address is missing.
	ErrOriginRequired = errors.New("origin address is required")
	// ErrTenantRequired is returned when tenant ID is missing.
	ErrTenantRequired = errors.New("tenant_id is required")
)

// Provider defines the interface for shipping operations.
// Implementations can integrate with carriers like FedEx, UPS, USPS, etc.
type Provider interface {
	// GetRates returns available shipping options for a shipment.
	GetRates(ctx context.Context, params RateParams) ([]Rate, error)

	// CreateLabel generates a shipping label.
	CreateLabel(ctx context.Context, params LabelParams) (*Label, error)

	// VoidLabel cancels a shipping label.
	VoidLabel(ctx context.Context, params VoidLabelParams) error

	// TrackShipment gets tracking information.
	TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error)

	// ValidateAddress validates and optionally corrects a shipping address.
	ValidateAddress(ctx context.Context, params ValidateAddressParams) (*AddressValidation, error)
}

// RateParams contains parameters for calculating shipping rates.
type RateParams struct {
	TenantID           string          // Required: Tenant identifier for multi-tenancy
	OriginAddress      ShippingAddress // Required: Sender's address
	DestinationAddress ShippingAddress // Required: Recipient's address
	Packages           []Package       // Required: At least one package (MVP: single package only)
	ServiceTypes       []string        // Optional: Filter for specific service types
}

// ShippingAddress represents a complete shipping address.
// Required fields: Name, Line1, City, State, PostalCode, Country
// Optional fields: Company, Line2, Phone, Email
//
// Country should be ISO 3166-1 alpha-2 code (e.g., "US", "CA").
// State/Province codes should match carrier requirements (e.g., "CA" for California).
type ShippingAddress struct {
	Name       string // Required: Recipient name
	Company    string // Optional: Company name
	Line1      string // Required: Street address
	Line2      string // Optional: Apartment, suite, etc.
	City       string // Required: City name
	State      string // Required: State/province code (e.g., "CA")
	PostalCode string // Required: Postal/ZIP code
	Country    string // Required: ISO 3166-1 alpha-2 (e.g., "US")
	Phone      string // Optional but recommended: Contact phone
	Email      string // Optional: Contact email
}

// Package represents a physical package to be shipped.
// Dimensions are stored in metric units.
type Package struct {
	WeightGrams int32
	LengthCm    int32
	WidthCm     int32
	HeightCm    int32
}

// Rate represents a shipping rate option.
type Rate struct {
	RateID                string
	Carrier               string
	ServiceName           string
	ServiceCode           string
	CostCents             int64 // Cost in cents (int64 for large shipments)
	EstimatedDaysMin      int
	EstimatedDaysMax      int
	EstimatedDeliveryDate time.Time
	ExpiresAt             *time.Time // When this rate becomes invalid (typically 24 hours)
}

// Label represents a purchased shipping label.
type Label struct {
	LabelID        string
	TrackingNumber string
	LabelURL       string
	CreatedAt      time.Time
}

// LabelParams contains parameters for creating a shipping label.
type LabelParams struct {
	TenantID           string          // Required: Tenant identifier for security validation
	RateID             string          // Required: Rate ID from GetRates
	OriginAddress      ShippingAddress // Required: Sender's address
	DestinationAddress ShippingAddress // Required: Recipient's address
	Package            Package         // Required: Package dimensions
	IdempotencyKey     string          // Optional: Prevents duplicate purchases
}

// VoidLabelParams contains parameters for voiding a shipping label.
type VoidLabelParams struct {
	TenantID string // Required: Tenant identifier for security validation
	LabelID  string // Required: Label ID to void
}

// ValidateAddressParams contains parameters for address validation.
type ValidateAddressParams struct {
	TenantID string          // Required: Tenant identifier
	Address  ShippingAddress // Required: Address to validate
}

// TrackingInfo contains shipment tracking information.
type TrackingInfo struct {
	TrackingNumber        string
	Status                string
	Events                []TrackingEvent
	EstimatedDeliveryDate time.Time
}

// TrackingEvent represents a single tracking event.
type TrackingEvent struct {
	Timestamp   time.Time
	Status      string
	Location    string
	Description string
}

// AddressValidationStatus represents the outcome of address validation.
type AddressValidationStatus string

const (
	// AddressValid means the address is valid and can be used as-is.
	AddressValid AddressValidationStatus = "valid"
	// AddressValidWithChanges means the address is valid but suggestions are available.
	AddressValidWithChanges AddressValidationStatus = "valid_with_changes"
	// AddressInvalid means the address cannot be validated or corrected.
	AddressInvalid AddressValidationStatus = "invalid"
)

// AddressValidation contains the result of address validation.
type AddressValidation struct {
	Status           AddressValidationStatus
	OriginalAddress  ShippingAddress
	SuggestedAddress *ShippingAddress // nil if no suggestion available
	Messages         []string         // Validation messages or errors
}
