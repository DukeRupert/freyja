//go:build integration
// +build integration

package billing

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadTestConfig loads Stripe test credentials from .env.test
func loadTestConfig(t *testing.T) StripeConfig {
	t.Helper()

	// Load .env.test from project root
	err := godotenv.Load("../../.env.test")
	if err != nil {
		t.Skipf("Skipping integration test: .env.test not found (%v)", err)
	}

	// Try both naming conventions for API key
	apiKey := os.Getenv("STRIPE_TEST_API_SECRET_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("STRIPE_SECRET_KEY")
	}

	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	if apiKey == "" || apiKey == "sk_test_your_key_here" {
		t.Skip("Skipping integration test: STRIPE_TEST_API_SECRET_KEY or STRIPE_SECRET_KEY not set in .env.test")
	}

	// Webhook secret is optional for now (only needed for webhook tests)
	// Will be provided by Stripe CLI when running: stripe listen --forward-to localhost:3000/webhooks/stripe
	if webhookSecret == "" || webhookSecret == "whsec_your_webhook_secret_here" {
		webhookSecret = "placeholder_for_cli" // Will skip webhook-specific tests
	}

	config := StripeConfig{
		APIKey:          apiKey,
		WebhookSecret:   webhookSecret,
		EnableStripeTax: false, // Set to true if you enable Stripe Tax in test dashboard
		MaxRetries:      3,
		TimeoutSeconds:  30,
	}

	// Verify it's a test key, not a live key
	if !config.IsTestMode() {
		t.Fatal("DANGER: Live Stripe key detected! Integration tests must use test mode keys (sk_test_...)")
	}

	return config
}

// TestStripeIntegration_CreatePaymentIntent tests creating a real payment intent via Stripe API
func TestStripeIntegration_CreatePaymentIntent(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewStripeProvider(config)
	require.NoError(t, err, "Failed to create Stripe provider")

	ctx := context.Background()

	tests := []struct {
		name    string
		params  CreatePaymentIntentParams
		wantErr bool
		verify  func(*testing.T, *PaymentIntent)
	}{
		{
			name: "creates payment intent with minimum required fields",
			params: CreatePaymentIntentParams{
				AmountCents: 5000, // $50.00
				Currency:    "usd",
				Description: "Integration test payment",
				Metadata: map[string]string{
					"tenant_id":  "tenant_integration_test",
					"cart_id":    "cart_test_123",
					"order_type": "retail",
				},
				IdempotencyKey: "integration_test_" + time.Now().Format("20060102_150405"),
			},
			wantErr: false,
			verify: func(t *testing.T, pi *PaymentIntent) {
				assert.NotEmpty(t, pi.ID, "Payment intent ID should be set")
				assert.NotEmpty(t, pi.ClientSecret, "Client secret should be set")
				assert.Equal(t, int32(5000), pi.AmountCents, "Amount should match")
				assert.Equal(t, "usd", pi.Currency, "Currency should match")
				assert.Equal(t, "requires_payment_method", pi.Status, "Initial status should be requires_payment_method")
				assert.Equal(t, "tenant_integration_test", pi.Metadata["tenant_id"], "tenant_id should be preserved")
				assert.Equal(t, "cart_test_123", pi.Metadata["cart_id"], "cart_id should be preserved")
				assert.NotZero(t, pi.CreatedAt, "CreatedAt should be set")
			},
		},
		{
			name: "validates minimum amount (below 50 cents)",
			params: CreatePaymentIntentParams{
				AmountCents: 25, // $0.25 - below Stripe minimum
				Currency:    "usd",
				Metadata: map[string]string{
					"tenant_id": "tenant_integration_test",
				},
			},
			wantErr: true, // Should fail validation before reaching Stripe
		},
		{
			name: "creates payment intent with customer email",
			params: CreatePaymentIntentParams{
				AmountCents:   10000, // $100.00
				Currency:      "usd",
				Description:   "Test order with receipt email",
				CustomerEmail: "test@example.com",
				Metadata: map[string]string{
					"tenant_id": "tenant_integration_test",
					"cart_id":   "cart_test_456",
				},
				IdempotencyKey: "integration_test_email_" + time.Now().Format("20060102_150405"),
			},
			wantErr: false,
			verify: func(t *testing.T, pi *PaymentIntent) {
				assert.NotEmpty(t, pi.ID)
				assert.NotEmpty(t, pi.ClientSecret)
				assert.Equal(t, int32(10000), pi.AmountCents)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pi, err := provider.CreatePaymentIntent(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pi)

			if tt.verify != nil {
				tt.verify(t, pi)
			}

			// Log payment intent ID for manual verification in Stripe Dashboard
			t.Logf("Created payment intent: %s (view at https://dashboard.stripe.com/test/payments/%s)", pi.ID, pi.ID)
		})
	}
}

// TestStripeIntegration_GetPaymentIntent tests retrieving a payment intent
func TestStripeIntegration_GetPaymentIntent(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewStripeProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	// First, create a payment intent
	createParams := CreatePaymentIntentParams{
		AmountCents: 7500, // $75.00
		Currency:    "usd",
		Description: "Test payment for retrieval",
		Metadata: map[string]string{
			"tenant_id": "tenant_integration_test",
			"cart_id":   "cart_retrieval_test",
		},
		IdempotencyKey: "integration_test_get_" + time.Now().Format("20060102_150405"),
	}

	createdPI, err := provider.CreatePaymentIntent(ctx, createParams)
	require.NoError(t, err)
	require.NotNil(t, createdPI)

	t.Logf("Created payment intent for retrieval test: %s", createdPI.ID)

	// Now retrieve it
	retrievedPI, err := provider.GetPaymentIntent(ctx, GetPaymentIntentParams{
		PaymentIntentID: createdPI.ID,
		TenantID:        "tenant_integration_test",
	})

	require.NoError(t, err)
	require.NotNil(t, retrievedPI)

	// Verify retrieved payment intent matches created one
	assert.Equal(t, createdPI.ID, retrievedPI.ID, "Payment intent ID should match")
	assert.Equal(t, createdPI.AmountCents, retrievedPI.AmountCents, "Amount should match")
	assert.Equal(t, createdPI.Currency, retrievedPI.Currency, "Currency should match")
	assert.Equal(t, createdPI.Metadata["tenant_id"], retrievedPI.Metadata["tenant_id"], "tenant_id should match")
	assert.Equal(t, createdPI.Metadata["cart_id"], retrievedPI.Metadata["cart_id"], "cart_id should match")
}

// TestStripeIntegration_TenantIsolation tests that tenant isolation is enforced
func TestStripeIntegration_TenantIsolation(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewStripeProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create payment intent for tenant A
	createParams := CreatePaymentIntentParams{
		AmountCents: 5000,
		Currency:    "usd",
		Metadata: map[string]string{
			"tenant_id": "tenant_a",
			"cart_id":   "cart_123",
		},
		IdempotencyKey: "integration_test_isolation_" + time.Now().Format("20060102_150405"),
	}

	pi, err := provider.CreatePaymentIntent(ctx, createParams)
	require.NoError(t, err)
	require.NotNil(t, pi)

	t.Logf("Created payment intent for tenant_a: %s", pi.ID)

	// Try to retrieve with correct tenant_id - should succeed
	retrievedPI, err := provider.GetPaymentIntent(ctx, GetPaymentIntentParams{
		PaymentIntentID: pi.ID,
		TenantID:        "tenant_a",
	})
	require.NoError(t, err)
	require.NotNil(t, retrievedPI)
	assert.Equal(t, pi.ID, retrievedPI.ID)

	// Try to retrieve with different tenant_id - should fail
	_, err = provider.GetPaymentIntent(ctx, GetPaymentIntentParams{
		PaymentIntentID: pi.ID,
		TenantID:        "tenant_b", // Wrong tenant!
	})
	assert.Error(t, err, "Should not allow cross-tenant access")
	assert.Equal(t, ErrPaymentIntentNotFound, err, "Should return ErrPaymentIntentNotFound to avoid leaking existence")
}

// TestStripeIntegration_UpdatePaymentIntent tests updating a payment intent
func TestStripeIntegration_UpdatePaymentIntent(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewStripeProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create initial payment intent
	createParams := CreatePaymentIntentParams{
		AmountCents: 5000, // $50.00
		Currency:    "usd",
		Description: "Initial order",
		Metadata: map[string]string{
			"tenant_id": "tenant_integration_test",
			"cart_id":   "cart_update_test",
		},
		IdempotencyKey: "integration_test_update_" + time.Now().Format("20060102_150405"),
	}

	pi, err := provider.CreatePaymentIntent(ctx, createParams)
	require.NoError(t, err)
	require.NotNil(t, pi)

	t.Logf("Created payment intent for update test: %s", pi.ID)

	// Update amount (customer added items to cart)
	updateParams := UpdatePaymentIntentParams{
		PaymentIntentID: pi.ID,
		TenantID:        "tenant_integration_test",
		AmountCents:     7500, // $75.00 - increased amount
		Description:     "Updated order with additional items",
		Metadata: map[string]string{
			"updated_at": time.Now().Format(time.RFC3339),
		},
	}

	updatedPI, err := provider.UpdatePaymentIntent(ctx, updateParams)
	require.NoError(t, err)
	require.NotNil(t, updatedPI)

	// Verify update was applied
	assert.Equal(t, pi.ID, updatedPI.ID, "ID should remain the same")
	assert.Equal(t, int32(7500), updatedPI.AmountCents, "Amount should be updated")
	assert.Equal(t, "tenant_integration_test", updatedPI.Metadata["tenant_id"], "tenant_id should be preserved")
	assert.NotEmpty(t, updatedPI.Metadata["updated_at"], "New metadata should be added")
}

// TestStripeIntegration_CancelPaymentIntent tests canceling a payment intent
func TestStripeIntegration_CancelPaymentIntent(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewStripeProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create payment intent to cancel
	createParams := CreatePaymentIntentParams{
		AmountCents: 5000,
		Currency:    "usd",
		Description: "Order to be canceled",
		Metadata: map[string]string{
			"tenant_id": "tenant_integration_test",
			"cart_id":   "cart_cancel_test",
		},
		IdempotencyKey: "integration_test_cancel_" + time.Now().Format("20060102_150405"),
	}

	pi, err := provider.CreatePaymentIntent(ctx, createParams)
	require.NoError(t, err)
	require.NotNil(t, pi)

	t.Logf("Created payment intent for cancellation test: %s", pi.ID)

	// Cancel the payment intent
	err = provider.CancelPaymentIntent(ctx, pi.ID, "tenant_integration_test")
	require.NoError(t, err)

	// Verify it was canceled by retrieving it
	canceledPI, err := provider.GetPaymentIntent(ctx, GetPaymentIntentParams{
		PaymentIntentID: pi.ID,
		TenantID:        "tenant_integration_test",
	})
	require.NoError(t, err)
	assert.Equal(t, "canceled", canceledPI.Status, "Status should be canceled")

	// Test idempotency - canceling again should succeed
	err = provider.CancelPaymentIntent(ctx, pi.ID, "tenant_integration_test")
	assert.NoError(t, err, "Canceling already-canceled payment intent should be idempotent")
}

// TestStripeIntegration_IdempotencyKey tests idempotency key behavior
func TestStripeIntegration_IdempotencyKey(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewStripeProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	idempotencyKey := "integration_test_idempotency_" + time.Now().Format("20060102_150405")

	params := CreatePaymentIntentParams{
		AmountCents: 5000,
		Currency:    "usd",
		Metadata: map[string]string{
			"tenant_id": "tenant_integration_test",
			"cart_id":   "cart_idempotency_test",
		},
		IdempotencyKey: idempotencyKey,
	}

	// Create payment intent first time
	pi1, err := provider.CreatePaymentIntent(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, pi1)

	t.Logf("Created payment intent with idempotency key: %s", pi1.ID)

	// Create again with same idempotency key - should return same payment intent
	pi2, err := provider.CreatePaymentIntent(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, pi2)

	// Should be the exact same payment intent
	assert.Equal(t, pi1.ID, pi2.ID, "Idempotency key should return same payment intent")
	assert.Equal(t, pi1.ClientSecret, pi2.ClientSecret, "Client secret should match")
}

// TestStripeIntegration_ErrorHandling tests error scenarios
func TestStripeIntegration_ErrorHandling(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewStripeProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("get non-existent payment intent", func(t *testing.T) {
		_, err := provider.GetPaymentIntent(ctx, GetPaymentIntentParams{
			PaymentIntentID: "pi_nonexistent_12345",
			TenantID:        "tenant_integration_test",
		})
		assert.Error(t, err)
		assert.Equal(t, ErrPaymentIntentNotFound, err)
	})

	t.Run("update non-existent payment intent", func(t *testing.T) {
		_, err := provider.UpdatePaymentIntent(ctx, UpdatePaymentIntentParams{
			PaymentIntentID: "pi_nonexistent_12345",
			TenantID:        "tenant_integration_test",
			AmountCents:     5000,
		})
		assert.Error(t, err)
		assert.Equal(t, ErrPaymentIntentNotFound, err)
	})

	t.Run("cancel non-existent payment intent", func(t *testing.T) {
		err := provider.CancelPaymentIntent(ctx, "pi_nonexistent_12345", "tenant_integration_test")
		assert.Error(t, err)
		assert.Equal(t, ErrPaymentIntentNotFound, err)
	})

	t.Run("missing tenant_id in metadata", func(t *testing.T) {
		_, err := provider.CreatePaymentIntent(ctx, CreatePaymentIntentParams{
			AmountCents: 5000,
			Currency:    "usd",
			Metadata: map[string]string{
				"cart_id": "cart_123",
				// Missing tenant_id!
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_id")
	})
}
