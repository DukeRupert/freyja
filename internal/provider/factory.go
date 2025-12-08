package provider

import (
	"fmt"
	"strconv"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/email"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/shipping"
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

	// CreateShippingProvider creates a shipping provider from configuration.
	// Returns an error if the provider name is unknown or configuration is invalid.
	CreateShippingProvider(config *TenantProviderConfig) (shipping.Provider, error)

	// CreateEmailSender creates an email sender from configuration.
	// Returns an error if the provider name is unknown or configuration is invalid.
	CreateEmailSender(config *TenantProviderConfig) (email.Sender, error)
}

// DefaultFactory implements ProviderFactory using constructor functions for each provider.
type DefaultFactory struct {
	validator ProviderValidator
}

// ErrNilValidator is defined in errors.go

// NewDefaultFactory creates a provider factory with configuration validation.
// Returns an error if validator is nil.
func NewDefaultFactory(validator ProviderValidator) (*DefaultFactory, error) {
	if validator == nil {
		return nil, ErrNilValidator
	}
	return &DefaultFactory{
		validator: validator,
	}, nil
}

// MustNewDefaultFactory creates a provider factory with configuration validation.
// Panics if validator is nil. Use only during application initialization.
func MustNewDefaultFactory(validator ProviderValidator) *DefaultFactory {
	factory, err := NewDefaultFactory(validator)
	if err != nil {
		panic(err)
	}
	return factory
}

// CreateTaxCalculator creates a tax calculator based on the provider name in config.
func (f *DefaultFactory) CreateTaxCalculator(config *TenantProviderConfig) (tax.Calculator, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	if config.Type != ProviderTypeTax {
		return nil, ErrProviderTypeMismatch(ProviderTypeTax, config.Type)
	}

	result := f.validator.ValidateTaxConfig(config)
	if !result.Valid {
		return nil, ErrValidationFailed("tax", result.Errors)
	}

	switch config.Name {
	case ProviderNameNoTax:
		return tax.NewNoTaxCalculator(), nil

	case ProviderNamePercentage:
		// Database percentage calculator - requires repository and tenant ID
		// Config should contain repository reference
		repo, ok := config.Config["repository"].(repository.Querier)
		if !ok {
			return nil, ErrMissingRepository
		}
		return tax.NewDatabasePercentageCalculator(repo, config.TenantID), nil

	case ProviderNameStripeTax:
		// Stripe Tax uses estimate rate for preview (optional)
		estimateRate := 0.0
		if rateStr, err := extractString(config.Config, "estimate_rate"); err == nil {
			if rate, err := strconv.ParseFloat(rateStr, 64); err == nil {
				estimateRate = rate
			}
		}
		return billing.NewStripeTaxCalculator(estimateRate), nil

	default:
		return nil, ErrUnknownProvider("tax", config.Name)
	}
}

// CreateBillingProvider creates a billing provider based on the provider name in config.
func (f *DefaultFactory) CreateBillingProvider(config *TenantProviderConfig) (billing.Provider, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	if config.Type != ProviderTypeBilling {
		return nil, ErrProviderTypeMismatch(ProviderTypeBilling, config.Type)
	}

	result := f.validator.ValidateBillingConfig(config)
	if !result.Valid {
		return nil, ErrValidationFailed("billing", result.Errors)
	}

	switch config.Name {
	case ProviderNameStripe:
		apiKey, err := extractString(config.Config, "stripe_api_key")
		if err != nil {
			return nil, fmt.Errorf("failed to extract stripe_api_key: %w", err)
		}

		webhookSecret, err := extractString(config.Config, "stripe_webhook_secret")
		if err != nil {
			return nil, fmt.Errorf("failed to extract stripe_webhook_secret: %w", err)
		}

		stripeConfig := billing.StripeConfig{
			APIKey:        apiKey,
			WebhookSecret: webhookSecret,
		}

		return billing.NewStripeProvider(stripeConfig)

	default:
		return nil, ErrUnknownProvider("billing", config.Name)
	}
}

// CreateShippingProvider creates a shipping provider based on the provider name in config.
func (f *DefaultFactory) CreateShippingProvider(config *TenantProviderConfig) (shipping.Provider, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	if config.Type != ProviderTypeShipping {
		return nil, ErrProviderTypeMismatch(ProviderTypeShipping, config.Type)
	}

	result := f.validator.ValidateShippingConfig(config)
	if !result.Valid {
		return nil, ErrValidationFailed("shipping", result.Errors)
	}

	switch config.Name {
	case ProviderNameEasyPost:
		apiKey, err := extractString(config.Config, "easypost_api_key")
		if err != nil {
			return nil, fmt.Errorf("failed to extract easypost_api_key: %w", err)
		}

		return shipping.NewEasyPostProvider(shipping.EasyPostConfig{
			APIKey: apiKey,
		})

	case ProviderNameManual:
		// Flat rate provider requires implementation
		return nil, ErrFlatRateNotImplemented

	default:
		return nil, ErrUnknownProvider("shipping", config.Name)
	}
}

// CreateEmailSender creates an email sender based on the provider name in config.
func (f *DefaultFactory) CreateEmailSender(config *TenantProviderConfig) (email.Sender, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	if config.Type != ProviderTypeEmail {
		return nil, ErrProviderTypeMismatch(ProviderTypeEmail, config.Type)
	}

	result := f.validator.ValidateEmailConfig(config)
	if !result.Valid {
		return nil, ErrValidationFailed("email", result.Errors)
	}

	switch config.Name {
	case ProviderNameSMTP:
		host, err := extractString(config.Config, "smtp_host")
		if err != nil {
			return nil, fmt.Errorf("failed to extract smtp_host: %w", err)
		}

		port, err := extractInt(config.Config, "smtp_port")
		if err != nil {
			return nil, fmt.Errorf("failed to extract smtp_port: %w", err)
		}

		// Username and password are optional for SMTP
		username, _ := extractString(config.Config, "smtp_username")
		password, _ := extractString(config.Config, "smtp_password")

		from, err := extractString(config.Config, "smtp_from")
		if err != nil {
			return nil, fmt.Errorf("failed to extract smtp_from: %w", err)
		}

		fromName, _ := extractString(config.Config, "from_name")

		return email.NewSMTPSender(host, port, username, password, from, fromName), nil

	case ProviderNamePostmark:
		apiKey, err := extractString(config.Config, "postmark_api_key")
		if err != nil {
			return nil, fmt.Errorf("failed to extract postmark_api_key: %w", err)
		}

		return email.NewPostmarkSender(apiKey, nil), nil

	default:
		return nil, ErrUnknownProvider("email", config.Name)
	}
}

// extractString safely extracts a string value from config map.
func extractString(config map[string]interface{}, key string) (string, error) {
	value, exists := config[key]
	if !exists {
		return "", ErrConfigKeyNotFound(key)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", ErrConfigKeyWrongType(key, "string", value)
	}

	return strValue, nil
}


// extractInt safely extracts an int value from config map.
func extractInt(config map[string]interface{}, key string) (int, error) {
	value, exists := config[key]
	if !exists {
		return 0, ErrConfigKeyNotFound(key)
	}

	// JSON numbers are typically float64
	if floatValue, ok := value.(float64); ok {
		return int(floatValue), nil
	}

	// Try int as fallback
	if intValue, ok := value.(int); ok {
		return intValue, nil
	}

	// Try int64 as fallback
	if int64Value, ok := value.(int64); ok {
		return int(int64Value), nil
	}

	return 0, ErrConfigKeyWrongType(key, "numeric", value)
}
