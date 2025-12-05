package provider

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ProviderType represents the category of provider service.
type ProviderType string

const (
	// ProviderTypeTax represents tax calculation providers (TaxJar, Avalara, Stripe Tax).
	ProviderTypeTax ProviderType = "tax"

	// ProviderTypeShipping represents shipping providers (ShipStation, EasyPost, Shippo).
	ProviderTypeShipping ProviderType = "shipping"

	// ProviderTypeBilling represents billing/payment providers (Stripe).
	ProviderTypeBilling ProviderType = "billing"

	// ProviderTypeEmail represents email service providers (Postmark, Resend, SES).
	ProviderTypeEmail ProviderType = "email"
)

// ProviderName represents specific provider implementations.
type ProviderName string

const (
	// Tax providers
	ProviderNameStripeTax  ProviderName = "stripe_tax"
	ProviderNameTaxJar     ProviderName = "taxjar"
	ProviderNameAvalara    ProviderName = "avalara"
	ProviderNamePercentage ProviderName = "percentage" // Simple percentage-based tax
	ProviderNameNoTax      ProviderName = "no_tax"     // No tax calculation

	// Shipping providers
	ProviderNameShipStation ProviderName = "shipstation"
	ProviderNameEasyPost    ProviderName = "easypost"
	ProviderNameShippo      ProviderName = "shippo"
	ProviderNameManual      ProviderName = "manual" // Manual shipping (flat rates)

	// Billing providers
	ProviderNameStripe ProviderName = "stripe"

	// Email providers
	ProviderNamePostmark ProviderName = "postmark"
	ProviderNameResend   ProviderName = "resend"
	ProviderNameSES      ProviderName = "ses"
	ProviderNameSMTP     ProviderName = "smtp" // Generic SMTP
)

// TenantProviderConfig represents a tenant's configuration for a specific provider.
// This is the domain model corresponding to the tenant_provider_configs database table.
type TenantProviderConfig struct {
	ID         pgtype.UUID
	TenantID   pgtype.UUID
	Type       ProviderType
	Name       ProviderName
	IsActive   bool
	IsDefault  bool
	Priority   int32
	Config     map[string]interface{} // Decrypted configuration
	ConfigJSON []byte                 // Raw encrypted JSON from database
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TenantShippingRate represents a cached or manual shipping rate for a tenant.
// This is the domain model corresponding to the tenant_shipping_rates database table.
type TenantShippingRate struct {
	ID                    pgtype.UUID
	TenantID              pgtype.UUID
	ProviderConfigID      pgtype.UUID
	ServiceCode           string
	ServiceName           string
	OriginPostalCode      string
	DestinationPostalCode string
	WeightGrams           int32
	RateCents             int32
	Currency              string
	ValidUntil            time.Time
	Metadata              map[string]interface{}
	CreatedAt             time.Time
}

// ValidationResult represents the outcome of validating provider configuration.
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// AddError adds an error message to the validation result.
func (v *ValidationResult) AddError(err string) {
	v.Valid = false
	v.Errors = append(v.Errors, err)
}

// IsValidProviderNameForType checks if a provider name is valid for the given provider type.
// This prevents mismatched configurations (e.g., setting a tax provider as billing type).
func IsValidProviderNameForType(name ProviderName, providerType ProviderType) bool {
	switch providerType {
	case ProviderTypeTax:
		switch name {
		case ProviderNameStripeTax, ProviderNameTaxJar, ProviderNameAvalara,
			ProviderNamePercentage, ProviderNameNoTax:
			return true
		}
	case ProviderTypeShipping:
		switch name {
		case ProviderNameShipStation, ProviderNameEasyPost, ProviderNameShippo, ProviderNameManual:
			return true
		}
	case ProviderTypeBilling:
		switch name {
		case ProviderNameStripe:
			return true
		}
	case ProviderTypeEmail:
		switch name {
		case ProviderNamePostmark, ProviderNameResend, ProviderNameSES, ProviderNameSMTP:
			return true
		}
	}
	return false
}
