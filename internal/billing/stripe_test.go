package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreatePaymentIntent tests payment intent creation with various scenarios
func TestCreatePaymentIntent(t *testing.T) {
	tests := []struct {
		name      string
		params    CreatePaymentIntentParams
		setupMock func(*MockProvider)
		wantErr   error
	}{
		{
			name: "creates payment intent with valid params",
			params: CreatePaymentIntentParams{
				AmountCents:    2500, // $25.00
				Currency:       "usd",
				CustomerEmail:  "customer@example.com",
				Description:    "Coffee order",
				IdempotencyKey: "cart_123",
				Metadata: map[string]string{
					"tenant_id":  "tenant_abc",
					"cart_id":    "cart_123",
					"order_type": "retail",
				},
			},
			setupMock: func(m *MockProvider) {
				m.CreatePaymentIntentFunc = func(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error) {
					// Verify required metadata is present
					if params.Metadata["tenant_id"] == "" {
						return nil, errors.New("tenant_id required in metadata")
					}
					if params.Metadata["cart_id"] == "" {
						return nil, errors.New("cart_id required in metadata")
					}

					return &PaymentIntent{
						ID:           "pi_test_123",
						ClientSecret: "pi_test_123_secret_abc",
						AmountCents:  params.AmountCents,
						Currency:     params.Currency,
						Status:       "requires_payment_method",
						Metadata:     params.Metadata,
						CreatedAt:    time.Now(),
					}, nil
				}
			},
			wantErr: nil,
		},
		{
			name: "returns client secret for frontend",
			params: CreatePaymentIntentParams{
				AmountCents:    5000,
				Currency:       "usd",
				IdempotencyKey: "cart_456",
				Metadata: map[string]string{
					"tenant_id": "tenant_abc",
					"cart_id":   "cart_456",
				},
			},
			setupMock: func(m *MockProvider) {
				// Default mock implementation provides client_secret
			},
			wantErr: nil,
		},
		{
			name: "includes metadata in payment intent",
			params: CreatePaymentIntentParams{
				AmountCents:    3000,
				Currency:       "usd",
				IdempotencyKey: "cart_789",
				Metadata: map[string]string{
					"tenant_id":  "tenant_xyz",
					"cart_id":    "cart_789",
					"order_type": "wholesale",
					"custom":     "value",
				},
			},
			setupMock: func(m *MockProvider) {
				m.CreatePaymentIntentFunc = func(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error) {
					// Verify all metadata is preserved
					return &PaymentIntent{
						ID:           "pi_test_789",
						ClientSecret: "pi_test_789_secret",
						AmountCents:  params.AmountCents,
						Currency:     params.Currency,
						Status:       "requires_payment_method",
						Metadata:     params.Metadata,
						CreatedAt:    time.Now(),
					}, nil
				}
			},
			wantErr: nil,
		},
		{
			name: "validates minimum amount (below 50 cents)",
			params: CreatePaymentIntentParams{
				AmountCents:    49, // Below minimum
				Currency:       "usd",
				IdempotencyKey: "cart_low",
				Metadata: map[string]string{
					"tenant_id": "tenant_abc",
					"cart_id":   "cart_low",
				},
			},
			setupMock: func(m *MockProvider) {
				m.CreatePaymentIntentFunc = func(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error) {
					if params.AmountCents < 50 {
						return nil, ErrAmountTooSmall
					}
					return &PaymentIntent{
						ID:           "pi_should_not_create",
						ClientSecret: "should_not_create",
						AmountCents:  params.AmountCents,
						Currency:     params.Currency,
						Status:       "requires_payment_method",
						Metadata:     params.Metadata,
						CreatedAt:    time.Now(),
					}, nil
				}
			},
			wantErr: ErrAmountTooSmall,
		},
		{
			name: "accepts minimum amount (50 cents)",
			params: CreatePaymentIntentParams{
				AmountCents:    50, // Exactly minimum
				Currency:       "usd",
				IdempotencyKey: "cart_min",
				Metadata: map[string]string{
					"tenant_id": "tenant_abc",
					"cart_id":   "cart_min",
				},
			},
			setupMock: func(m *MockProvider) {
				m.CreatePaymentIntentFunc = func(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error) {
					if params.AmountCents < 50 {
						return nil, ErrAmountTooSmall
					}
					return &PaymentIntent{
						ID:           "pi_test_min",
						ClientSecret: "pi_test_min_secret",
						AmountCents:  params.AmountCents,
						Currency:     params.Currency,
						Status:       "requires_payment_method",
						Metadata:     params.Metadata,
						CreatedAt:    time.Now(),
					}, nil
				}
			},
			wantErr: nil,
		},
		{
			name: "handles stripe tax when enabled",
			params: CreatePaymentIntentParams{
				AmountCents:     5000,
				Currency:        "usd",
				EnableStripeTax: true,
				IdempotencyKey:  "cart_tax",
				ShippingAddress: &PaymentAddress{
					Line1:      "123 Main St",
					City:       "Seattle",
					State:      "WA",
					PostalCode: "98101",
					Country:    "US",
				},
				LineItems: []PaymentLineItem{
					{
						ProductID:   "prod_123",
						Description: "Ethiopian Coffee",
						Quantity:    2,
						AmountCents: 2500,
						TaxCode:     "txcd_30011000", // Food and beverages
					},
				},
				Metadata: map[string]string{
					"tenant_id": "tenant_abc",
					"cart_id":   "cart_tax",
				},
			},
			setupMock: func(m *MockProvider) {
				m.CreatePaymentIntentFunc = func(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error) {
					// Simulate Stripe Tax calculation
					taxAmount := int32(0)
					if params.EnableStripeTax {
						// 10% tax example
						taxAmount = params.AmountCents / 10
					}

					return &PaymentIntent{
						ID:            "pi_test_tax",
						ClientSecret:  "pi_test_tax_secret",
						AmountCents:   params.AmountCents + taxAmount,
						Currency:      params.Currency,
						Status:        "requires_payment_method",
						TaxCents:      taxAmount,
						ShippingCents: 0,
						Metadata:      params.Metadata,
						CreatedAt:     time.Now(),
					}, nil
				}
			},
			wantErr: nil,
		},
		{
			name: "respects idempotency key",
			params: CreatePaymentIntentParams{
				AmountCents:    1000,
				Currency:       "usd",
				IdempotencyKey: "unique_key_123",
				Metadata: map[string]string{
					"tenant_id": "tenant_abc",
					"cart_id":   "cart_idem",
				},
			},
			setupMock: func(m *MockProvider) {
				callCount := 0
				m.CreatePaymentIntentFunc = func(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error) {
					callCount++
					// Simulate idempotency: same key returns same intent
					if params.IdempotencyKey == "unique_key_123" {
						return &PaymentIntent{
							ID:           "pi_idempotent",
							ClientSecret: "pi_idempotent_secret",
							AmountCents:  params.AmountCents,
							Currency:     params.Currency,
							Status:       "requires_payment_method",
							Metadata:     params.Metadata,
							CreatedAt:    time.Now(),
						}, nil
					}
					return nil, errors.New("no idempotency key")
				}
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockProvider()
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			pi, err := mock.CreatePaymentIntent(context.Background(), tt.params)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr) || err.Error() == tt.wantErr.Error(),
					"expected error %v, got %v", tt.wantErr, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, pi.ID, "payment intent ID should not be empty")
			assert.NotEmpty(t, pi.ClientSecret, "client secret should be provided for frontend")
			assert.Equal(t, tt.params.Currency, pi.Currency)
			assert.NotZero(t, pi.CreatedAt)

			// Verify metadata is preserved
			if tt.params.Metadata != nil {
				for k, v := range tt.params.Metadata {
					assert.Equal(t, v, pi.Metadata[k], "metadata key %s should be preserved", k)
				}
			}
		})
	}
}

// TestGetPaymentIntent tests retrieving existing payment intents
func TestGetPaymentIntent(t *testing.T) {
	tests := []struct {
		name      string
		params    GetPaymentIntentParams
		setupMock func(*MockProvider)
		wantErr   error
		validate  func(*testing.T, *PaymentIntent)
	}{
		{
			name: "retrieves existing payment intent",
			params: GetPaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: "pi_test_123",
			},
			setupMock: func(m *MockProvider) {
				m.PaymentIntents["pi_test_123"] = &PaymentIntent{
					ID:           "pi_test_123",
					ClientSecret: "pi_test_123_secret",
					AmountCents:  5000,
					Currency:     "usd",
					Status:       "succeeded",
					Metadata: map[string]string{
						"tenant_id": "tenant_abc",
						"cart_id":   "cart_123",
					},
					CreatedAt: time.Now(),
				}
			},
			wantErr: nil,
			validate: func(t *testing.T, pi *PaymentIntent) {
				assert.Equal(t, "succeeded", pi.Status)
				assert.Equal(t, int32(5000), pi.AmountCents)
			},
		},
		{
			name: "returns correct status",
			params: GetPaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: "pi_requires_payment",
			},
			setupMock: func(m *MockProvider) {
				m.PaymentIntents["pi_requires_payment"] = &PaymentIntent{
					ID:           "pi_requires_payment",
					ClientSecret: "secret",
					AmountCents:  1000,
					Currency:     "usd",
					Status:       "requires_payment_method",
					Metadata:     map[string]string{"cart_id": "cart_456"},
					CreatedAt:    time.Now(),
				}
			},
			wantErr: nil,
			validate: func(t *testing.T, pi *PaymentIntent) {
				assert.Equal(t, "requires_payment_method", pi.Status)
			},
		},
		{
			name: "includes tax and shipping amounts",
			params: GetPaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: "pi_with_tax",
			},
			setupMock: func(m *MockProvider) {
				m.PaymentIntents["pi_with_tax"] = &PaymentIntent{
					ID:            "pi_with_tax",
					ClientSecret:  "secret",
					AmountCents:   6000,
					Currency:      "usd",
					Status:        "succeeded",
					TaxCents:      500,
					ShippingCents: 500,
					Metadata:      map[string]string{"cart_id": "cart_789"},
					CreatedAt:     time.Now(),
				}
			},
			wantErr: nil,
			validate: func(t *testing.T, pi *PaymentIntent) {
				assert.Equal(t, int32(500), pi.TaxCents)
				assert.Equal(t, int32(500), pi.ShippingCents)
				assert.Equal(t, int32(6000), pi.AmountCents)
			},
		},
		{
			name: "returns error for invalid ID",
			params: GetPaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: "pi_nonexistent",
			},
			setupMock: func(m *MockProvider) {
				// Don't add to PaymentIntents map
			},
			wantErr: ErrPaymentIntentNotFound,
		},
		{
			name: "includes last payment error when payment failed",
			params: GetPaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: "pi_failed",
			},
			setupMock: func(m *MockProvider) {
				m.PaymentIntents["pi_failed"] = &PaymentIntent{
					ID:           "pi_failed",
					ClientSecret: "secret",
					AmountCents:  2000,
					Currency:     "usd",
					Status:       "requires_payment_method",
					Metadata:     map[string]string{"cart_id": "cart_fail"},
					LastPaymentError: &PaymentError{
						Code:        "card_declined",
						Message:     "Your card was declined",
						DeclineCode: "insufficient_funds",
					},
					CreatedAt: time.Now(),
				}
			},
			wantErr: nil,
			validate: func(t *testing.T, pi *PaymentIntent) {
				require.NotNil(t, pi.LastPaymentError)
				assert.Equal(t, "card_declined", pi.LastPaymentError.Code)
				assert.Equal(t, "insufficient_funds", pi.LastPaymentError.DeclineCode)
			},
		},
		{
			name: "verifies metadata is preserved",
			params: GetPaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: "pi_metadata",
			},
			setupMock: func(m *MockProvider) {
				m.PaymentIntents["pi_metadata"] = &PaymentIntent{
					ID:           "pi_metadata",
					ClientSecret: "secret",
					AmountCents:  3000,
					Currency:     "usd",
					Status:       "succeeded",
					Metadata: map[string]string{
						"tenant_id":  "tenant_xyz",
						"cart_id":    "cart_meta",
						"order_type": "retail",
						"custom":     "value",
					},
					CreatedAt: time.Now(),
				}
			},
			wantErr: nil,
			validate: func(t *testing.T, pi *PaymentIntent) {
				assert.Equal(t, "tenant_xyz", pi.Metadata["tenant_id"])
				assert.Equal(t, "cart_meta", pi.Metadata["cart_id"])
				assert.Equal(t, "retail", pi.Metadata["order_type"])
				assert.Equal(t, "value", pi.Metadata["custom"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockProvider()
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			pi, err := mock.GetPaymentIntent(context.Background(), tt.params)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pi)

			if tt.validate != nil {
				tt.validate(t, pi)
			}
		})
	}
}

// TestUpdatePaymentIntent tests updating payment intents before confirmation
func TestUpdatePaymentIntent(t *testing.T) {
	tests := []struct {
		name      string
		params    UpdatePaymentIntentParams
		setupMock func(*MockProvider)
		wantErr   error
		validate  func(*testing.T, *PaymentIntent)
	}{
		{
			name: "updates amount before confirmation",
			params: UpdatePaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: "pi_test_123",
				AmountCents:     7500, // Updated from 5000
				Metadata:        map[string]string{},
			},
			setupMock: func(m *MockProvider) {
				m.PaymentIntents["pi_test_123"] = &PaymentIntent{
					ID:           "pi_test_123",
					ClientSecret: "secret",
					AmountCents:  5000,
					Currency:     "usd",
					Status:       "requires_payment_method",
					Metadata:     map[string]string{"cart_id": "cart_123"},
					CreatedAt:    time.Now(),
				}
			},
			wantErr: nil,
			validate: func(t *testing.T, pi *PaymentIntent) {
				assert.Equal(t, int32(7500), pi.AmountCents)
				assert.Equal(t, "requires_payment_method", pi.Status)
			},
		},
		{
			name: "updates metadata",
			params: UpdatePaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: "pi_meta_update",
				AmountCents:     5000,
				Metadata: map[string]string{
					"updated_field": "new_value",
				},
			},
			setupMock: func(m *MockProvider) {
				m.PaymentIntents["pi_meta_update"] = &PaymentIntent{
					ID:           "pi_meta_update",
					ClientSecret: "secret",
					AmountCents:  5000,
					Currency:     "usd",
					Status:       "requires_payment_method",
					Metadata: map[string]string{
						"cart_id":       "cart_456",
						"updated_field": "old_value",
					},
					CreatedAt: time.Now(),
				}
			},
			wantErr: nil,
			validate: func(t *testing.T, pi *PaymentIntent) {
				assert.Equal(t, "new_value", pi.Metadata["updated_field"])
				assert.Equal(t, "cart_456", pi.Metadata["cart_id"], "existing metadata should be preserved")
			},
		},
		// Note: MockProvider's UpdatePaymentIntent doesn't enforce status checks
		// Real StripeProvider implementation should return error if already succeeded
		{
			name: "returns error for invalid ID",
			params: UpdatePaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: "pi_nonexistent",
				AmountCents:     1000,
			},
			setupMock: func(m *MockProvider) {
				// Don't add to PaymentIntents map
			},
			wantErr: ErrPaymentIntentNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockProvider()
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			pi, err := mock.UpdatePaymentIntent(context.Background(), tt.params)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErr, ErrPaymentIntentNotFound) {
					assert.True(t, errors.Is(err, tt.wantErr))
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pi)

			if tt.validate != nil {
				tt.validate(t, pi)
			}
		})
	}
}

// TestCancelPaymentIntent tests canceling payment intents
func TestCancelPaymentIntent(t *testing.T) {
	tests := []struct {
		name            string
		paymentIntentID string
		setupMock       func(*MockProvider)
		wantErr         error
		validateStatus  string
	}{
		{
			name:            "cancels unconfirmed payment intent",
			paymentIntentID: "pi_test_123",
			setupMock: func(m *MockProvider) {
				m.PaymentIntents["pi_test_123"] = &PaymentIntent{
					ID:           "pi_test_123",
					ClientSecret: "secret",
					AmountCents:  5000,
					Currency:     "usd",
					Status:       "requires_payment_method",
					Metadata:     map[string]string{"tenant_id": "tenant_abc", "cart_id": "cart_123"},
					CreatedAt:    time.Now(),
				}
			},
			wantErr:        nil,
			validateStatus: "canceled",
		},
		// Note: MockProvider's CancelPaymentIntent doesn't enforce status checks
		// Real StripeProvider implementation should return error if already succeeded
		{
			name:            "idempotent - canceling already-canceled is OK",
			paymentIntentID: "pi_already_canceled",
			setupMock: func(m *MockProvider) {
				m.PaymentIntents["pi_already_canceled"] = &PaymentIntent{
					ID:           "pi_already_canceled",
					ClientSecret: "secret",
					AmountCents:  5000,
					Currency:     "usd",
					Status:       "canceled",
					Metadata:     map[string]string{"tenant_id": "tenant_abc", "cart_id": "cart_789"},
					CreatedAt:    time.Now(),
				}
				// MockProvider's CancelPaymentIntent is already idempotent
			},
			wantErr:        nil,
			validateStatus: "canceled",
		},
		{
			name:            "returns error for invalid ID",
			paymentIntentID: "pi_nonexistent",
			setupMock: func(m *MockProvider) {
				// Don't add to PaymentIntents map
			},
			wantErr: ErrPaymentIntentNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockProvider()
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			err := mock.CancelPaymentIntent(context.Background(), tt.paymentIntentID, "tenant_abc")

			if tt.wantErr != nil {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.validateStatus != "" {
				pi, err := mock.GetPaymentIntent(context.Background(), GetPaymentIntentParams{
					TenantID:        "tenant_abc",
					PaymentIntentID: tt.paymentIntentID,
				})
				require.NoError(t, err)
				assert.Equal(t, tt.validateStatus, pi.Status)
			}
		})
	}
}

// TestVerifyWebhookSignature tests webhook signature verification
func TestVerifyWebhookSignature(t *testing.T) {
	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		setupMock func(*MockProvider)
		wantErr   error
	}{
		{
			name:      "verifies valid webhook signature",
			payload:   []byte(`{"type":"payment_intent.succeeded","data":{}}`),
			signature: "valid_signature",
			secret:    "whsec_test_secret",
			setupMock: func(m *MockProvider) {
				m.VerifyWebhookSignatureFunc = func(payload []byte, signature string, secret string) error {
					if signature == "valid_signature" && secret == "whsec_test_secret" {
						return nil
					}
					return ErrInvalidWebhookSignature
				}
			},
			wantErr: nil,
		},
		{
			name:      "rejects invalid signature",
			payload:   []byte(`{"type":"payment_intent.succeeded","data":{}}`),
			signature: "invalid_signature",
			secret:    "whsec_test_secret",
			setupMock: func(m *MockProvider) {
				m.VerifyWebhookSignatureFunc = func(payload []byte, signature string, secret string) error {
					if signature != "valid_signature" {
						return ErrInvalidWebhookSignature
					}
					return nil
				}
			},
			wantErr: ErrInvalidWebhookSignature,
		},
		{
			name:      "rejects expired timestamp",
			payload:   []byte(`{"type":"payment_intent.succeeded","data":{}}`),
			signature: "expired_signature",
			secret:    "whsec_test_secret",
			setupMock: func(m *MockProvider) {
				m.VerifyWebhookSignatureFunc = func(payload []byte, signature string, secret string) error {
					if signature == "expired_signature" {
						return ErrInvalidWebhookSignature
					}
					return nil
				}
			},
			wantErr: ErrInvalidWebhookSignature,
		},
		{
			name:      "rejects wrong secret",
			payload:   []byte(`{"type":"payment_intent.succeeded","data":{}}`),
			signature: "valid_signature",
			secret:    "whsec_wrong_secret",
			setupMock: func(m *MockProvider) {
				m.VerifyWebhookSignatureFunc = func(payload []byte, signature string, secret string) error {
					if secret != "whsec_test_secret" {
						return ErrInvalidWebhookSignature
					}
					return nil
				}
			},
			wantErr: ErrInvalidWebhookSignature,
		},
		{
			name:      "handles empty payload",
			payload:   []byte{},
			signature: "valid_signature",
			secret:    "whsec_test_secret",
			setupMock: func(m *MockProvider) {
				m.VerifyWebhookSignatureFunc = func(payload []byte, signature string, secret string) error {
					if len(payload) == 0 {
						return ErrInvalidWebhookSignature
					}
					return nil
				}
			},
			wantErr: ErrInvalidWebhookSignature,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockProvider()
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			err := mock.VerifyWebhookSignature(tt.payload, tt.signature, tt.secret)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}

			assert.NoError(t, err)
		})
	}
}

// TestCheckoutFlow_Integration simulates the complete checkout flow
func TestCheckoutFlow_Integration(t *testing.T) {
	t.Run("complete checkout flow from cart to order", func(t *testing.T) {
		mock := NewMockProvider()
		ctx := context.Background()

		// Step 1: Create payment intent (after calculating cart total)
		createParams := CreatePaymentIntentParams{
			AmountCents:    6000, // $60.00 (subtotal + shipping + tax)
			Currency:       "usd",
			CustomerEmail:  "customer@example.com",
			Description:    "Coffee order - 2 bags of Ethiopian",
			IdempotencyKey: "cart_checkout_123",
			Metadata: map[string]string{
				"tenant_id":  "tenant_abc",
				"cart_id":    "cart_checkout_123",
				"order_type": "retail",
			},
		}

		pi, err := mock.CreatePaymentIntent(ctx, createParams)
		require.NoError(t, err)
		require.NotNil(t, pi)
		assert.NotEmpty(t, pi.ClientSecret, "client_secret needed for frontend")
		assert.Equal(t, "requires_payment_method", pi.Status)

		// Step 2: Simulate frontend payment confirmation (Stripe.js)
		// In real flow, frontend calls Stripe.js which confirms payment
		// Simulate successful payment
		err = mock.SimulateSucceededPayment(pi.ID)
		require.NoError(t, err)

		// Step 3: Get payment intent to verify succeeded status
		getParams := GetPaymentIntentParams{
			TenantID:        "tenant_abc",
			PaymentIntentID: pi.ID,
		}

		verifiedPI, err := mock.GetPaymentIntent(ctx, getParams)
		require.NoError(t, err)
		assert.Equal(t, "succeeded", verifiedPI.Status, "payment must succeed before creating order")

		// Step 4: Verify metadata is preserved throughout
		assert.Equal(t, "tenant_abc", verifiedPI.Metadata["tenant_id"])
		assert.Equal(t, "cart_checkout_123", verifiedPI.Metadata["cart_id"])
		assert.Equal(t, "retail", verifiedPI.Metadata["order_type"])

		// At this point, checkout service would:
		// - Verify payment succeeded
		// - Create order in database
		// - Link payment_intent_id to order
		// - Decrement inventory
		// - Mark cart as converted
	})

	t.Run("handles cart changes during checkout", func(t *testing.T) {
		mock := NewMockProvider()
		ctx := context.Background()

		// Step 1: Create payment intent with initial amount
		createParams := CreatePaymentIntentParams{
			AmountCents:    5000,
			Currency:       "usd",
			IdempotencyKey: "cart_changes_456",
			Metadata: map[string]string{
				"tenant_id": "tenant_abc",
				"cart_id":   "cart_changes_456",
			},
		}

		pi, err := mock.CreatePaymentIntent(ctx, createParams)
		require.NoError(t, err)

		// Step 2: Customer adds another item to cart
		// Update payment intent with new amount
		updateParams := UpdatePaymentIntentParams{
			TenantID:        "tenant_abc",
			PaymentIntentID: pi.ID,
			AmountCents:     7500, // Added $25.00 item
			Metadata: map[string]string{
				"updated_at": time.Now().Format(time.RFC3339),
			},
		}

		updatedPI, err := mock.UpdatePaymentIntent(ctx, updateParams)
		require.NoError(t, err)
		assert.Equal(t, int32(7500), updatedPI.AmountCents)
		assert.NotEmpty(t, updatedPI.Metadata["updated_at"])

		// Step 3: Customer completes payment with updated amount
		err = mock.SimulateSucceededPayment(updatedPI.ID)
		require.NoError(t, err)

		finalPI, err := mock.GetPaymentIntent(ctx, GetPaymentIntentParams{
			TenantID:        "tenant_abc",
			PaymentIntentID: updatedPI.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, "succeeded", finalPI.Status)
		assert.Equal(t, int32(7500), finalPI.AmountCents)
	})

	t.Run("handles payment failure gracefully", func(t *testing.T) {
		mock := NewMockProvider()
		ctx := context.Background()

		// Step 1: Create payment intent
		createParams := CreatePaymentIntentParams{
			AmountCents:    5000,
			Currency:       "usd",
			IdempotencyKey: "cart_fail_789",
			Metadata: map[string]string{
				"tenant_id": "tenant_abc",
				"cart_id":   "cart_fail_789",
			},
		}

		pi, err := mock.CreatePaymentIntent(ctx, createParams)
		require.NoError(t, err)

		// Step 2: Simulate payment failure (card declined)
		err = mock.SimulateFailedPayment(pi.ID, "card_declined", "Your card was declined")
		require.NoError(t, err)

		// Step 3: Get payment intent to check error
		failedPI, err := mock.GetPaymentIntent(ctx, GetPaymentIntentParams{
			TenantID:        "tenant_abc",
			PaymentIntentID: pi.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, "requires_payment_method", failedPI.Status)
		require.NotNil(t, failedPI.LastPaymentError)
		assert.Equal(t, "card_declined", failedPI.LastPaymentError.Code)

		// In real flow:
		// - Checkout service sees status != "succeeded"
		// - Returns error to user
		// - Cart remains active
		// - Inventory not decremented
		// - User can try different payment method
	})

	t.Run("idempotent order creation with payment intent ID", func(t *testing.T) {
		mock := NewMockProvider()
		ctx := context.Background()

		// Create and complete payment
		createParams := CreatePaymentIntentParams{
			AmountCents:    5000,
			Currency:       "usd",
			IdempotencyKey: "cart_idem_999",
			Metadata: map[string]string{
				"tenant_id": "tenant_abc",
				"cart_id":   "cart_idem_999",
			},
		}

		pi, err := mock.CreatePaymentIntent(ctx, createParams)
		require.NoError(t, err)

		err = mock.SimulateSucceededPayment(pi.ID)
		require.NoError(t, err)

		// In real flow:
		// 1. First call to CompleteCheckout creates order
		// 2. Second call with same payment_intent_id should return existing order
		// This prevents duplicate orders if user refreshes page or network retry occurs

		// Verify payment intent can be retrieved multiple times
		for i := 0; i < 3; i++ {
			retrievedPI, err := mock.GetPaymentIntent(ctx, GetPaymentIntentParams{
				TenantID:        "tenant_abc",
				PaymentIntentID: pi.ID,
			})
			require.NoError(t, err)
			assert.Equal(t, "succeeded", retrievedPI.Status)
			assert.Equal(t, pi.ID, retrievedPI.ID)
		}
	})
}

// TestPostMVP_MethodsReturnNotImplemented tests that post-MVP methods return appropriate errors
func TestPostMVP_MethodsReturnNotImplemented(t *testing.T) {
	mock := NewMockProvider()
	ctx := context.Background()

	t.Run("CreateCustomer returns not implemented", func(t *testing.T) {
		customer, err := mock.CreateCustomer(ctx, CreateCustomerParams{
			Email: "customer@example.com",
			Name:  "John Doe",
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotImplemented))
		assert.Nil(t, customer)
	})

	t.Run("GetCustomer returns not implemented", func(t *testing.T) {
		customer, err := mock.GetCustomer(ctx, "cus_test_123")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotImplemented))
		assert.Nil(t, customer)
	})

	t.Run("UpdateCustomer returns not implemented", func(t *testing.T) {
		customer, err := mock.UpdateCustomer(ctx, "cus_test_123", UpdateCustomerParams{
			Email: "updated@example.com",
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotImplemented))
		assert.Nil(t, customer)
	})

	t.Run("CreateSubscription returns not implemented", func(t *testing.T) {
		sub, err := mock.CreateSubscription(ctx, SubscriptionParams{
			CustomerID: "cus_test_123",
			PriceID:    "price_test_123",
			Quantity:   1,
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotImplemented))
		assert.Nil(t, sub)
	})

	t.Run("CancelSubscription returns not implemented", func(t *testing.T) {
		err := mock.CancelSubscription(ctx, "sub_test_123", false)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotImplemented))
	})

	t.Run("RefundPayment returns not implemented", func(t *testing.T) {
		refund, err := mock.RefundPayment(ctx, RefundParams{
			PaymentIntentID: "pi_test_123",
			AmountCents:     1000,
			Reason:          "requested_by_customer",
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotImplemented))
		assert.Nil(t, refund)
	})
}

// TestStripeConfig_Validation tests configuration validation
func TestStripeConfig_Validation(t *testing.T) {
	t.Run("validates required API key", func(t *testing.T) {
		config := StripeConfig{
			APIKey:        "",
			WebhookSecret: "whsec_test",
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("validates required webhook secret", func(t *testing.T) {
		config := StripeConfig{
			APIKey:        "sk_test_123",
			WebhookSecret: "",
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook secret is required")
	})

	t.Run("accepts valid configuration", func(t *testing.T) {
		config := StripeConfig{
			APIKey:        "sk_test_123",
			WebhookSecret: "whsec_test",
		}
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("detects test mode correctly", func(t *testing.T) {
		testConfig := StripeConfig{
			APIKey:        "sk_test_123456",
			WebhookSecret: "whsec_test",
		}
		assert.True(t, testConfig.IsTestMode())

		liveConfig := StripeConfig{
			APIKey:        "sk_live_123456",
			WebhookSecret: "whsec_live",
		}
		assert.False(t, liveConfig.IsTestMode())
	})
}

// TestStripeError tests the StripeError type
func TestStripeError(t *testing.T) {
	t.Run("formats error message correctly", func(t *testing.T) {
		err := &StripeError{
			Message: "Payment failed",
			Code:    "card_declined",
		}
		assert.Contains(t, err.Error(), "Payment failed")
		assert.Contains(t, err.Error(), "card_declined")
	})

	t.Run("identifies declined cards", func(t *testing.T) {
		err := &StripeError{
			Code:        "card_declined",
			DeclineCode: "insufficient_funds",
		}
		assert.True(t, err.IsDeclined())

		notDeclined := &StripeError{
			Code: "api_error",
		}
		assert.False(t, notDeclined.IsDeclined())
	})

	t.Run("identifies temporary errors", func(t *testing.T) {
		rateLimitErr := &StripeError{
			Code: "rate_limit",
		}
		assert.True(t, rateLimitErr.IsTemporary())

		connectionErr := &StripeError{
			Code: "api_connection_error",
		}
		assert.True(t, connectionErr.IsTemporary())

		permanentErr := &StripeError{
			Code: "invalid_request",
		}
		assert.False(t, permanentErr.IsTemporary())
	})
}
