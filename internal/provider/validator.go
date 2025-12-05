package provider

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
	// TODO: Initialize DefaultValidator
	// TODO: Return initialized validator
	return nil
}

// ValidateTaxConfig validates tax provider configuration.
func (v *DefaultValidator) ValidateTaxConfig(config *TenantProviderConfig) *ValidationResult {
	// TODO: Initialize ValidationResult with Valid: true
	// TODO: Check that config is not nil
	// TODO: Check that config.Type == ProviderTypeTax
	// TODO: Check that config.Config map is not nil
	// TODO: Switch on config.Name:
	//   - ProviderNameStripeTax:
	//       * Require "api_key" field (string, non-empty)
	//       * Require "api_key" starts with "sk_" (Stripe secret key format)
	//   - ProviderNameTaxJar:
	//       * Require "api_key" field (string, non-empty)
	//   - ProviderNameAvalara:
	//       * Require "account_id" field (string, non-empty)
	//       * Require "license_key" field (string, non-empty)
	//   - ProviderNamePercentage:
	//       * Require "rate" field (number between 0.0 and 1.0)
	//       * Example: 0.08 for 8% tax
	//   - ProviderNameNoTax:
	//       * No config required
	//   - default:
	//       * Add error: "unknown tax provider: {config.Name}"
	// TODO: Return validation result
	return nil
}

// ValidateBillingConfig validates billing provider configuration.
func (v *DefaultValidator) ValidateBillingConfig(config *TenantProviderConfig) *ValidationResult {
	// TODO: Initialize ValidationResult with Valid: true
	// TODO: Check that config is not nil
	// TODO: Check that config.Type == ProviderTypeBilling
	// TODO: Check that config.Config map is not nil
	// TODO: Switch on config.Name:
	//   - ProviderNameStripe:
	//       * Require "api_key" field (string, non-empty)
	//       * Require "api_key" starts with "sk_" (Stripe secret key format)
	//       * Require "webhook_secret" field (string, non-empty)
	//       * Require "webhook_secret" starts with "whsec_" (Stripe webhook secret format)
	//   - default:
	//       * Add error: "unknown billing provider: {config.Name}"
	// TODO: Return validation result
	return nil
}

// ValidateShippingConfig validates shipping provider configuration.
func (v *DefaultValidator) ValidateShippingConfig(config *TenantProviderConfig) *ValidationResult {
	// TODO: Initialize ValidationResult with Valid: true
	// TODO: Check that config is not nil
	// TODO: Check that config.Type == ProviderTypeShipping
	// TODO: Check that config.Config map is not nil
	// TODO: Switch on config.Name:
	//   - ProviderNameShipStation:
	//       * Require "api_key" field (string, non-empty)
	//       * Require "api_secret" field (string, non-empty)
	//   - ProviderNameEasyPost:
	//       * Require "api_key" field (string, non-empty)
	//       * Require "api_key" starts with "EZAK" (EasyPost API key format)
	//   - ProviderNameShippo:
	//       * Require "api_key" field (string, non-empty)
	//   - ProviderNameManual:
	//       * No required fields (manual shipping uses tenant_shipping_rates table)
	//   - default:
	//       * Add error: "unknown shipping provider: {config.Name}"
	// TODO: Return validation result
	return nil
}

// ValidateEmailConfig validates email provider configuration.
func (v *DefaultValidator) ValidateEmailConfig(config *TenantProviderConfig) *ValidationResult {
	// TODO: Initialize ValidationResult with Valid: true
	// TODO: Check that config is not nil
	// TODO: Check that config.Type == ProviderTypeEmail
	// TODO: Check that config.Config map is not nil
	// TODO: Switch on config.Name:
	//   - ProviderNamePostmark:
	//       * Require "server_token" field (string, non-empty)
	//       * Require "from_email" field (string, valid email format)
	//   - ProviderNameResend:
	//       * Require "api_key" field (string, non-empty)
	//       * Require "from_email" field (string, valid email format)
	//   - ProviderNameSES:
	//       * Require "access_key_id" field (string, non-empty)
	//       * Require "secret_access_key" field (string, non-empty)
	//       * Require "region" field (string, non-empty, e.g., "us-east-1")
	//       * Require "from_email" field (string, valid email format)
	//   - ProviderNameSMTP:
	//       * Require "host" field (string, non-empty)
	//       * Require "port" field (number, 1-65535)
	//       * Require "username" field (string, can be empty for no auth)
	//       * Require "password" field (string, can be empty for no auth)
	//       * Require "from_email" field (string, valid email format)
	//   - default:
	//       * Add error: "unknown email provider: {config.Name}"
	// TODO: Return validation result
	return nil
}

// requireString validates that a config field exists and is a non-empty string.
func requireString(config map[string]interface{}, key string, result *ValidationResult) string {
	// TODO: Check if key exists in config map
	// TODO: Type assert value to string
	// TODO: If key missing, add error: "missing required field: {key}"
	// TODO: If wrong type, add error: "field {key} must be a string"
	// TODO: If empty string, add error: "field {key} cannot be empty"
	// TODO: Return string value (or empty string if validation failed)
	return ""
}

// requireStringPrefix validates that a config field is a string starting with prefix.
func requireStringPrefix(config map[string]interface{}, key string, prefix string, result *ValidationResult) string {
	// TODO: Call requireString to get the value
	// TODO: Check if value starts with prefix
	// TODO: If not, add error: "field {key} must start with {prefix}"
	// TODO: Return string value
	return ""
}

// requireFloat64Range validates that a config field is a float64 within range [min, max].
func requireFloat64Range(config map[string]interface{}, key string, min, max float64, result *ValidationResult) float64 {
	// TODO: Check if key exists in config map
	// TODO: Type assert value to float64 (handle both float64 and int from JSON)
	// TODO: If key missing, add error: "missing required field: {key}"
	// TODO: If wrong type, add error: "field {key} must be a number"
	// TODO: If value < min or value > max, add error: "field {key} must be between {min} and {max}"
	// TODO: Return float64 value (or 0 if validation failed)
	return 0
}

// requireIntRange validates that a config field is an int within range [min, max].
func requireIntRange(config map[string]interface{}, key string, min, max int, result *ValidationResult) int {
	// TODO: Check if key exists in config map
	// TODO: Type assert value to float64 (JSON numbers are float64)
	// TODO: Convert float64 to int
	// TODO: If key missing, add error: "missing required field: {key}"
	// TODO: If wrong type, add error: "field {key} must be a number"
	// TODO: If value < min or value > max, add error: "field {key} must be between {min} and {max}"
	// TODO: Return int value (or 0 if validation failed)
	return 0
}

// isValidEmail performs basic email validation.
func isValidEmail(email string) bool {
	// TODO: Implement basic email validation
	// TODO: Check that email contains "@"
	// TODO: Check that email has text before and after "@"
	// TODO: Check that email has "." in domain part
	// TODO: For MVP, simple validation is sufficient
	// TODO: Could use regex or email parsing library for production
	return false
}
