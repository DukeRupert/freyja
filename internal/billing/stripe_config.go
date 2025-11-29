package billing

import (
	"errors"
)

// StripeConfig contains configuration for Stripe provider.
type StripeConfig struct {
	// APIKey is the Stripe secret key (sk_test_... or sk_live_...)
	APIKey string

	// WebhookSecret is the webhook signing secret (whsec_...)
	// Used to verify webhook signatures from Stripe
	WebhookSecret string

	// EnableStripeTax determines if Stripe Tax should calculate tax
	// If false, tax is calculated by application-level tax calculator
	EnableStripeTax bool

	// MaxRetries is the maximum number of retries for transient failures
	// Default: 3
	MaxRetries int

	// TimeoutSeconds is the HTTP timeout for Stripe API calls in seconds
	// Default: 30
	TimeoutSeconds int
}

// Validate checks that required configuration is present.
func (c *StripeConfig) Validate() error {
	if c.APIKey == "" {
		return errors.New("stripe: API key is required")
	}
	if c.WebhookSecret == "" {
		return errors.New("stripe: webhook secret is required")
	}
	return nil
}

// IsTestMode returns true if using test mode API keys.
func (c *StripeConfig) IsTestMode() bool {
	return len(c.APIKey) > 7 && c.APIKey[:8] == "sk_test_"
}
