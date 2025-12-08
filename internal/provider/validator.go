package provider

import (
	"fmt"
	"strings"
)

// ProviderValidator validates provider configurations before creating instances.
// Each provider type has specific required configuration fields that must be present
// and valid. Validation happens at two points:
// 1. When tenant saves/updates provider config (via admin UI)
// 2. Before creating provider instance (in factory)
type ProviderValidator interface {
	// ValidateTaxConfig validates tax provider configuration.
	// Returns ValidationResult with any configuration errors.
	ValidateTaxConfig(config *TenantProviderConfig) *ValidationResult

	// ValidateBillingConfig validates billing provider configuration.
	// Returns ValidationResult with any configuration errors.
	ValidateBillingConfig(config *TenantProviderConfig) *ValidationResult

	// ValidateShippingConfig validates shipping provider configuration.
	// Returns ValidationResult with any configuration errors.
	ValidateShippingConfig(config *TenantProviderConfig) *ValidationResult

	// ValidateEmailConfig validates email provider configuration.
	// Returns ValidationResult with any configuration errors.
	ValidateEmailConfig(config *TenantProviderConfig) *ValidationResult
}

// DefaultValidator implements ProviderValidator with provider-specific validation rules.
type DefaultValidator struct {
	// No dependencies needed for basic validation
	// Could add external API validation in the future (e.g., test Stripe API key)
}

// NewDefaultValidator creates a provider configuration validator.
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{}
}

// ValidateTaxConfig validates tax provider configuration.
func (v *DefaultValidator) ValidateTaxConfig(config *TenantProviderConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if config == nil {
		result.AddError("config cannot be nil")
		return result
	}

	if config.Type != ProviderTypeTax {
		result.AddError("config type must be tax")
		return result
	}

	if config.Config == nil {
		result.AddError("config map cannot be nil")
		return result
	}

	switch config.Name {
	case ProviderNameStripeTax:
		requireStringPrefix(config.Config, "stripe_api_key", "sk_", result)
	case ProviderNameTaxJar:
		requireString(config.Config, "api_key", result)
	case ProviderNameAvalara:
		requireString(config.Config, "account_id", result)
		requireString(config.Config, "license_key", result)
	case ProviderNamePercentage:
		// No secrets required - uses database rates
	case ProviderNameNoTax:
		// No config required
	default:
		result.AddError("unknown tax provider: " + string(config.Name))
	}

	return result
}

// ValidateBillingConfig validates billing provider configuration.
func (v *DefaultValidator) ValidateBillingConfig(config *TenantProviderConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if config == nil {
		result.AddError("config cannot be nil")
		return result
	}

	if config.Type != ProviderTypeBilling {
		result.AddError("config type must be billing")
		return result
	}

	if config.Config == nil {
		result.AddError("config map cannot be nil")
		return result
	}

	switch config.Name {
	case ProviderNameStripe:
		requireStringPrefix(config.Config, "stripe_api_key", "sk_", result)
		requireStringPrefix(config.Config, "stripe_webhook_secret", "whsec_", result)
	default:
		result.AddError("unknown billing provider: " + string(config.Name))
	}

	return result
}

// ValidateShippingConfig validates shipping provider configuration.
func (v *DefaultValidator) ValidateShippingConfig(config *TenantProviderConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if config == nil {
		result.AddError("config cannot be nil")
		return result
	}

	if config.Type != ProviderTypeShipping {
		result.AddError("config type must be shipping")
		return result
	}

	if config.Config == nil {
		result.AddError("config map cannot be nil")
		return result
	}

	switch config.Name {
	case ProviderNameShipStation:
		requireString(config.Config, "api_key", result)
		requireString(config.Config, "api_secret", result)
	case ProviderNameEasyPost:
		// EZAK = production key, EZTK = test key
		requireStringPrefixes(config.Config, "easypost_api_key", []string{"EZAK", "EZTK"}, result)
	case ProviderNameShippo:
		requireString(config.Config, "api_key", result)
	case ProviderNameManual:
		// No required fields - uses tenant_shipping_rates table
	default:
		result.AddError("unknown shipping provider: " + string(config.Name))
	}

	return result
}

// ValidateEmailConfig validates email provider configuration.
func (v *DefaultValidator) ValidateEmailConfig(config *TenantProviderConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if config == nil {
		result.AddError("config cannot be nil")
		return result
	}

	if config.Type != ProviderTypeEmail {
		result.AddError("config type must be email")
		return result
	}

	if config.Config == nil {
		result.AddError("config map cannot be nil")
		return result
	}

	switch config.Name {
	case ProviderNamePostmark:
		requireString(config.Config, "postmark_api_key", result)
	case ProviderNameResend:
		requireString(config.Config, "api_key", result)
	case ProviderNameSES:
		requireString(config.Config, "access_key_id", result)
		requireString(config.Config, "secret_access_key", result)
		requireString(config.Config, "region", result)
	case ProviderNameSMTP:
		requireString(config.Config, "smtp_host", result)
		requireIntRange(config.Config, "smtp_port", 1, 65535, result)
		requireString(config.Config, "smtp_from", result)
	default:
		result.AddError("unknown email provider: " + string(config.Name))
	}

	return result
}

// requireString validates that a config field exists and is a non-empty string.
func requireString(config map[string]interface{}, key string, result *ValidationResult) string {
	value, exists := config[key]
	if !exists {
		result.AddError("missing required field: " + key)
		return ""
	}

	strValue, ok := value.(string)
	if !ok {
		result.AddError("field " + key + " must be a string")
		return ""
	}

	if strValue == "" {
		result.AddError("field " + key + " cannot be empty")
		return ""
	}

	return strValue
}

// requireStringPrefix validates that a config field is a string starting with prefix.
func requireStringPrefix(config map[string]interface{}, key string, prefix string, result *ValidationResult) string {
	value := requireString(config, key, result)
	if value == "" {
		return ""
	}

	if len(value) < len(prefix) || value[:len(prefix)] != prefix {
		result.AddError("field " + key + " must start with " + prefix)
		return ""
	}

	return value
}

// requireStringPrefixes validates that a config field is a string starting with one of the prefixes.
func requireStringPrefixes(config map[string]interface{}, key string, prefixes []string, result *ValidationResult) string {
	value := requireString(config, key, result)
	if value == "" {
		return ""
	}

	for _, prefix := range prefixes {
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return value
		}
	}

	result.AddError("field " + key + " must start with one of: " + strings.Join(prefixes, ", "))
	return ""
}

// requireIntRange validates that a config field is an int within range [min, max].
// Safely handles float64 to int conversion with overflow protection.
func requireIntRange(config map[string]interface{}, key string, min, max int, result *ValidationResult) int {
	value, exists := config[key]
	if !exists {
		result.AddError("missing required field: " + key)
		return 0
	}

	var intValue int
	switch v := value.(type) {
	case float64:
		// Check for overflow before conversion
		if v < float64(min) || v > float64(max) || v != float64(int(v)) {
			result.AddError("field " + key + " must be a whole number between " + formatInt(min) + " and " + formatInt(max))
			return 0
		}
		intValue = int(v)
	case int:
		intValue = v
	case int64:
		// Check for int overflow from int64
		if v < int64(min) || v > int64(max) {
			result.AddError("field " + key + " must be between " + formatInt(min) + " and " + formatInt(max))
			return 0
		}
		intValue = int(v)
	default:
		result.AddError("field " + key + " must be a number")
		return 0
	}

	if intValue < min || intValue > max {
		result.AddError("field " + key + " must be between " + formatInt(min) + " and " + formatInt(max))
		return 0
	}

	return intValue
}

// formatInt formats an int for error messages.
func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}
