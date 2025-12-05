package provider

import (
	"fmt"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/tax"
)

// ProviderFactory creates provider instances from configuration.
// The factory pattern allows us to instantiate different provider implementations
// based on the ProviderName in the configuration without the registry needing to
// know about all the concrete implementations.
type ProviderFactory interface {
	// CreateTaxCalculator creates a tax calculator from configuration.
	// Returns an error if the provider name is unknown or configuration is invalid.
	CreateTaxCalculator(config *TenantProviderConfig) (tax.Calculator, error)

	// CreateBillingProvider creates a billing provider from configuration.
	// Returns an error if the provider name is unknown or configuration is invalid.
	CreateBillingProvider(config *TenantProviderConfig) (billing.Provider, error)

	// TODO: Add CreateShippingProvider(config *TenantProviderConfig) (shipping.Provider, error)
	// TODO: Add CreateEmailProvider(config *TenantProviderConfig) (email.Provider, error)
}

// DefaultFactory implements ProviderFactory using constructor functions for each provider.
type DefaultFactory struct {
	validator ProviderValidator
}

// NewDefaultFactory creates a provider factory with configuration validation.
func NewDefaultFactory(validator ProviderValidator) *DefaultFactory {
	// TODO: Initialize DefaultFactory with provided validator
	// TODO: Validate that validator is not nil
	// TODO: Return initialized factory
	return nil
}

// CreateTaxCalculator creates a tax calculator based on the provider name in config.
func (f *DefaultFactory) CreateTaxCalculator(config *TenantProviderConfig) (tax.Calculator, error) {
	// TODO: Validate that config is not nil
	// TODO: Validate that config.Type == ProviderTypeTax
	// TODO: Call f.validator.ValidateTaxConfig(config) to ensure config is valid
	// TODO: If validation fails, return nil and validation errors
	// TODO: Switch on config.Name:
	//   - ProviderNameStripeTax:
	//       * Extract required config values: api_key
	//       * Call tax.NewStripeTaxCalculator(apiKey) (when implemented)
	//   - ProviderNameTaxJar:
	//       * Extract required config values: api_key
	//       * Call tax.NewTaxJarCalculator(apiKey) (when implemented)
	//   - ProviderNameAvalara:
	//       * Extract required config values: account_id, license_key
	//       * Call tax.NewAvalaraCalculator(accountID, licenseKey) (when implemented)
	//   - ProviderNamePercentage:
	//       * Extract required config values: rate (as float64)
	//       * Call tax.NewPercentageCalculator(rate) (when implemented)
	//   - ProviderNameNoTax:
	//       * Call tax.NewNoTaxCalculator() (when implemented)
	//   - default:
	//       * Return error: "unknown tax provider: {config.Name}"
	// TODO: Return created instance or error
	return nil, fmt.Errorf("tax calculator factory not implemented")
}

// CreateBillingProvider creates a billing provider based on the provider name in config.
func (f *DefaultFactory) CreateBillingProvider(config *TenantProviderConfig) (billing.Provider, error) {
	// TODO: Validate that config is not nil
	// TODO: Validate that config.Type == ProviderTypeBilling
	// TODO: Call f.validator.ValidateBillingConfig(config) to ensure config is valid
	// TODO: If validation fails, return nil and validation errors
	// TODO: Switch on config.Name:
	//   - ProviderNameStripe:
	//       * Extract required config values: api_key, webhook_secret
	//       * Call billing.NewStripeProvider(apiKey, webhookSecret) (when implemented)
	//   - default:
	//       * Return error: "unknown billing provider: {config.Name}"
	// TODO: Return created instance or error
	return nil, fmt.Errorf("billing provider factory not implemented")
}

// extractString safely extracts a string value from config map.
func extractString(config map[string]interface{}, key string) (string, error) {
	// TODO: Check if key exists in config map
	// TODO: Type assert value to string
	// TODO: If key missing or wrong type, return error: "missing or invalid config key: {key}"
	// TODO: Return string value
	return "", fmt.Errorf("not implemented")
}

// extractFloat64 safely extracts a float64 value from config map.
func extractFloat64(config map[string]interface{}, key string) (float64, error) {
	// TODO: Check if key exists in config map
	// TODO: Type assert value to float64 (note: JSON numbers might be float64 or int)
	// TODO: Handle both float64 and int types by converting int to float64 if needed
	// TODO: If key missing or wrong type, return error: "missing or invalid config key: {key}"
	// TODO: Return float64 value
	return 0, fmt.Errorf("not implemented")
}

// extractInt safely extracts an int value from config map.
func extractInt(config map[string]interface{}, key string) (int, error) {
	// TODO: Check if key exists in config map
	// TODO: Type assert value to float64 (JSON numbers are typically float64)
	// TODO: Convert float64 to int
	// TODO: If key missing or wrong type, return error: "missing or invalid config key: {key}"
	// TODO: Return int value
	return 0, fmt.Errorf("not implemented")
}
