// internal/provider/stripe.go
package provider

import (
	"context"
	"fmt"
	"log"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
)

type StripeProvider struct {
	apiKey string
}

func NewStripeProvider(apiKey string) (*StripeProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Stripe API key is required")
	}

	provider := &StripeProvider{
		apiKey: apiKey,
	}

	// Set the global Stripe API key
	stripe.Key = apiKey

	// Perform health check
	if err := provider.HealthCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("Stripe health check failed: %w", err)
	}

	log.Println("✅ Stripe provider initialized and health check passed")
	return provider, nil
}

// HealthCheck verifies the API key and Stripe service connectivity
func (s *StripeProvider) HealthCheck(ctx context.Context) error {
	// Attempt to retrieve account information to verify API key and connectivity
	_, err := account.Get()
	if err != nil {
		return fmt.Errorf("failed to connect to Stripe: %w", err)
	}

	log.Printf("Stripe health check passed - API key valid and service reachable")
	return nil
}