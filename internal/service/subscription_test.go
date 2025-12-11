package service

import (
	"context"
	"database/sql"
	"math/big"
	"testing"
	"time"

	"github.com/dukerupert/hiri/internal/billing"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// =============================================================================
// MOCK BILLING PROVIDER FOR SUBSCRIPTIONS
// =============================================================================

// mockSubscriptionBillingProvider implements billing.Provider with subscription-specific methods
type mockSubscriptionBillingProvider struct {
	// Product creation
	createProductResult *billing.Product
	createProductErr    error

	// Price creation
	createPriceResult *billing.Price
	createPriceErr    error

	// Subscription operations
	createSubscriptionResult *billing.Subscription
	createSubscriptionErr    error
	getSubscriptionResult    *billing.Subscription
	getSubscriptionErr       error
	pauseSubscriptionResult  *billing.Subscription
	pauseSubscriptionErr     error
	resumeSubscriptionResult *billing.Subscription
	resumeSubscriptionErr    error
	cancelSubscriptionErr    error

	// Invoice operations
	getInvoiceResult *billing.Invoice
	getInvoiceErr    error

	// Tracking calls
	pauseCalls   []billing.PauseSubscriptionParams
	resumeCalls  []billing.ResumeSubscriptionParams
	cancelCalls  []billing.CancelSubscriptionParams
}

func (m *mockSubscriptionBillingProvider) CreateCustomer(ctx context.Context, params billing.CreateCustomerParams) (*billing.Customer, error) {
	return &billing.Customer{ID: "cus_test123"}, nil
}

func (m *mockSubscriptionBillingProvider) GetCustomer(ctx context.Context, customerID string) (*billing.Customer, error) {
	return nil, nil
}

func (m *mockSubscriptionBillingProvider) GetCustomerByEmail(ctx context.Context, email string) (*billing.Customer, error) {
	return nil, nil
}

func (m *mockSubscriptionBillingProvider) UpdateCustomer(ctx context.Context, customerID string, params billing.UpdateCustomerParams) (*billing.Customer, error) {
	return nil, nil
}

func (m *mockSubscriptionBillingProvider) VerifyWebhookSignature(payload []byte, signature string, secret string) error {
	return nil
}

func (m *mockSubscriptionBillingProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string, tenantID string) error {
	return nil
}

func (m *mockSubscriptionBillingProvider) CreatePaymentIntent(ctx context.Context, params billing.CreatePaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, nil
}

func (m *mockSubscriptionBillingProvider) GetPaymentIntent(ctx context.Context, params billing.GetPaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, nil
}

func (m *mockSubscriptionBillingProvider) UpdatePaymentIntent(ctx context.Context, params billing.UpdatePaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, nil
}

func (m *mockSubscriptionBillingProvider) CreateInvoice(ctx context.Context, params billing.CreateInvoiceParams) (*billing.Invoice, error) {
	return &billing.Invoice{ID: "in_test123"}, nil
}

func (m *mockSubscriptionBillingProvider) GetInvoice(ctx context.Context, params billing.GetInvoiceParams) (*billing.Invoice, error) {
	if m.getInvoiceErr != nil {
		return nil, m.getInvoiceErr
	}
	return m.getInvoiceResult, nil
}

func (m *mockSubscriptionBillingProvider) AddInvoiceItem(ctx context.Context, params billing.AddInvoiceItemParams) error {
	return nil
}

func (m *mockSubscriptionBillingProvider) FinalizeInvoice(ctx context.Context, params billing.FinalizeInvoiceParams) (*billing.Invoice, error) {
	return &billing.Invoice{ID: "in_test123", Status: "open"}, nil
}

func (m *mockSubscriptionBillingProvider) SendInvoice(ctx context.Context, params billing.SendInvoiceParams) error {
	return nil
}

func (m *mockSubscriptionBillingProvider) VoidInvoice(ctx context.Context, params billing.VoidInvoiceParams) error {
	return nil
}

func (m *mockSubscriptionBillingProvider) PayInvoice(ctx context.Context, params billing.PayInvoiceParams) (*billing.Invoice, error) {
	return nil, nil
}

func (m *mockSubscriptionBillingProvider) CreateProduct(ctx context.Context, params billing.CreateProductParams) (*billing.Product, error) {
	if m.createProductErr != nil {
		return nil, m.createProductErr
	}
	if m.createProductResult != nil {
		return m.createProductResult, nil
	}
	return &billing.Product{
		ID:          "prod_test123",
		Name:        params.Name,
		Description: params.Description,
		Active:      params.Active,
		Metadata:    params.Metadata,
		CreatedAt:   time.Now(),
	}, nil
}

func (m *mockSubscriptionBillingProvider) CreateRecurringPrice(ctx context.Context, params billing.CreateRecurringPriceParams) (*billing.Price, error) {
	if m.createPriceErr != nil {
		return nil, m.createPriceErr
	}
	if m.createPriceResult != nil {
		return m.createPriceResult, nil
	}
	return &billing.Price{
		ID:              "price_test123",
		ProductID:       params.ProductID,
		Currency:        params.Currency,
		UnitAmountCents: params.UnitAmountCents,
		Type:            "recurring",
		Recurring: &billing.PriceRecurring{
			Interval:      params.BillingInterval,
			IntervalCount: params.IntervalCount,
		},
		Active:    true,
		Metadata:  params.Metadata,
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockSubscriptionBillingProvider) CreateSubscription(ctx context.Context, params billing.CreateSubscriptionParams) (*billing.Subscription, error) {
	if m.createSubscriptionErr != nil {
		return nil, m.createSubscriptionErr
	}
	if m.createSubscriptionResult != nil {
		return m.createSubscriptionResult, nil
	}
	now := time.Now()
	return &billing.Subscription{
		ID:                     "sub_test123",
		CustomerID:             params.CustomerID,
		Status:                 "active",
		DefaultPaymentMethodID: params.DefaultPaymentMethodID,
		CurrentPeriodStart:     now,
		CurrentPeriodEnd:       now.AddDate(0, 1, 0),
		CancelAtPeriodEnd:      false,
		Metadata:               params.Metadata,
		CreatedAt:              now,
	}, nil
}

func (m *mockSubscriptionBillingProvider) GetSubscription(ctx context.Context, params billing.GetSubscriptionParams) (*billing.Subscription, error) {
	if m.getSubscriptionErr != nil {
		return nil, m.getSubscriptionErr
	}
	return m.getSubscriptionResult, nil
}

func (m *mockSubscriptionBillingProvider) PauseSubscription(ctx context.Context, params billing.PauseSubscriptionParams) (*billing.Subscription, error) {
	m.pauseCalls = append(m.pauseCalls, params)
	if m.pauseSubscriptionErr != nil {
		return nil, m.pauseSubscriptionErr
	}
	if m.pauseSubscriptionResult != nil {
		return m.pauseSubscriptionResult, nil
	}
	now := time.Now()
	return &billing.Subscription{
		ID:                 params.SubscriptionID,
		Status:             "paused",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		CreatedAt:          now,
	}, nil
}

func (m *mockSubscriptionBillingProvider) ResumeSubscription(ctx context.Context, params billing.ResumeSubscriptionParams) (*billing.Subscription, error) {
	m.resumeCalls = append(m.resumeCalls, params)
	if m.resumeSubscriptionErr != nil {
		return nil, m.resumeSubscriptionErr
	}
	if m.resumeSubscriptionResult != nil {
		return m.resumeSubscriptionResult, nil
	}
	now := time.Now()
	return &billing.Subscription{
		ID:                 params.SubscriptionID,
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		CreatedAt:          now,
	}, nil
}

func (m *mockSubscriptionBillingProvider) CancelSubscription(ctx context.Context, params billing.CancelSubscriptionParams) error {
	m.cancelCalls = append(m.cancelCalls, params)
	return m.cancelSubscriptionErr
}

func (m *mockSubscriptionBillingProvider) CreateCustomerPortalSession(ctx context.Context, params billing.CreatePortalSessionParams) (*billing.PortalSession, error) {
	return &billing.PortalSession{
		ID:        "cs_test123",
		URL:       "https://billing.stripe.com/session/test",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}, nil
}

func (m *mockSubscriptionBillingProvider) RefundPayment(ctx context.Context, params billing.RefundParams) (*billing.Refund, error) {
	return nil, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// createTestSubscription creates a test subscription with standard fields
func createTestSubscription(tenantID, userID pgtype.UUID, status string) repository.Subscription {
	now := time.Now()
	return repository.Subscription{
		ID:                     newUUID(),
		TenantID:               tenantID,
		UserID:                 userID,
		Status:                 status,
		BillingInterval:        "monthly",
		SubtotalCents:          1800,
		TaxCents:               140,
		ShippingCents:          500,
		TotalCents:             2440,
		Currency:               "usd",
		Provider:               "stripe",
		ProviderSubscriptionID: pgtype.Text{String: "sub_test123", Valid: true},
		BillingCustomerID:      newUUID(),
		ShippingAddressID:      newUUID(),
		ShippingMethodID:       newUUID(),
		PaymentMethodID:        newUUID(),
		CurrentPeriodStart:     pgtype.Timestamptz{Time: now, Valid: true},
		CurrentPeriodEnd:       pgtype.Timestamptz{Time: now.AddDate(0, 1, 0), Valid: true},
		NextBillingDate:        pgtype.Timestamptz{Time: now.AddDate(0, 1, 0), Valid: true},
		CreatedAt:              pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:              pgtype.Timestamptz{Time: now, Valid: true},
	}
}

// =============================================================================
// TEST: CreateSubscription
// =============================================================================

func Test_CreateSubscription_ValidatesBillingInterval(t *testing.T) {
	tests := []struct {
		name            string
		billingInterval string
		wantErr         error
	}{
		{
			name:            "valid weekly interval",
			billingInterval: "weekly",
			wantErr:         nil,
		},
		{
			name:            "valid biweekly interval",
			billingInterval: "biweekly",
			wantErr:         nil,
		},
		{
			name:            "valid monthly interval",
			billingInterval: "monthly",
			wantErr:         nil,
		},
		{
			name:            "valid every_6_weeks interval",
			billingInterval: "every_6_weeks",
			wantErr:         nil,
		},
		{
			name:            "valid every_2_months interval",
			billingInterval: "every_2_months",
			wantErr:         nil,
		},
		{
			name:            "invalid yearly interval",
			billingInterval: "yearly",
			wantErr:         ErrInvalidBillingInterval,
		},
		{
			name:            "invalid daily interval",
			billingInterval: "daily",
			wantErr:         ErrInvalidBillingInterval,
		},
		{
			name:            "invalid empty interval",
			billingInterval: "",
			wantErr:         ErrInvalidBillingInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tenantID := newUUID()
			userID := newUUID()
			ctx := contextWithTenant(tenantID)

			mockRepo := repository.NewMockQuerier(ctrl)
			mockBilling := &mockSubscriptionBillingProvider{}

			svc := NewSubscriptionService(mockRepo, mockBilling)

			// Only expect SKU lookup if interval is valid
			if tt.wantErr == nil {
				mockRepo.EXPECT().GetSKUByID(gomock.Any(), gomock.Any()).Return(repository.ProductSku{}, sql.ErrNoRows)
			}

			params := CreateSubscriptionParams{
				UserID:          userID,
				ProductSKUID:    newUUID(),
				Quantity:        1,
				BillingInterval: tt.billingInterval,
			}

			_, err := svc.CreateSubscription(ctx, params)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				// Expect failure but not invalid billing interval
				assert.Error(t, err)
				assert.NotErrorIs(t, err, ErrInvalidBillingInterval)
			}
		})
	}
}

func Test_CreateSubscription_CreatesStripeProductPriceSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	productID := newUUID()
	skuID := newUUID()
	priceListID := newUUID()
	billingCustomerID := newUUID()
	paymentMethodID := newUUID()
	shippingAddressID := newUUID()
	shippingMethodID := newUUID()

	ctx := contextWithTenant(tenantID)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	// Setup mocks
	mockRepo.EXPECT().GetSKUByID(gomock.Any(), skuID).Return(repository.ProductSku{
		ID:          skuID,
		ProductID:   productID,
		Sku:         "ETH-YIRG-12OZ-WB",
		WeightValue: pgtype.Numeric{Int: bigIntFromInt64(12), Valid: true},
		WeightUnit:  "oz",
		Grind:       "whole_bean",
	}, nil)

	mockRepo.EXPECT().GetDefaultPriceList(gomock.Any(), tenantID).Return(repository.PriceList{
		ID:       priceListID,
		TenantID: tenantID,
		Name:     "Retail",
	}, nil)

	mockRepo.EXPECT().GetPriceForSKU(gomock.Any(), repository.GetPriceForSKUParams{
		PriceListID:  priceListID,
		ProductSkuID: skuID,
	}).Return(repository.PriceListEntry{
		PriceCents: 1800,
	}, nil)

	mockRepo.EXPECT().GetBillingCustomerForUser(gomock.Any(), repository.GetBillingCustomerForUserParams{
		UserID:   userID,
		TenantID: tenantID,
		Provider: "stripe",
	}).Return(repository.BillingCustomer{
		ID:                 billingCustomerID,
		TenantID:           tenantID,
		UserID:             userID,
		ProviderCustomerID: "cus_test123",
	}, nil)

	mockRepo.EXPECT().GetPaymentMethodByID(gomock.Any(), repository.GetPaymentMethodByIDParams{
		ID:       paymentMethodID,
		TenantID: tenantID,
	}).Return(repository.GetPaymentMethodByIDRow{
		ID:                       paymentMethodID,
		BillingCustomerID:        billingCustomerID,
		ProviderPaymentMethodID:  "pm_test123",
		MethodType:               "card",
		DisplayBrand:             pgtype.Text{String: "Visa", Valid: true},
		DisplayLast4:             pgtype.Text{String: "4242", Valid: true},
	}, nil)

	// Subscription creation
	subscriptionID := newUUID()
	mockRepo.EXPECT().CreateSubscription(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.CreateSubscriptionParams) (repository.Subscription, error) {
			return repository.Subscription{
				ID:                subscriptionID,
				TenantID:          params.TenantID,
				UserID:            params.UserID,
				BillingInterval:   params.BillingInterval,
				Status:            params.Status,
				SubtotalCents:     params.SubtotalCents,
				TaxCents:          params.TaxCents,
				ShippingCents:     params.ShippingCents,
				TotalCents:        params.TotalCents,
				Currency:          params.Currency,
				Provider:          params.Provider,
				BillingCustomerID: params.BillingCustomerID,
			}, nil
		})

	mockRepo.EXPECT().GetProductByID(gomock.Any(), repository.GetProductByIDParams{
		ID:       productID,
		TenantID: tenantID,
	}).Return(repository.Product{
		ID:          productID,
		Name:        "Ethiopian Yirgacheffe",
		Description: pgtype.Text{String: "Floral notes", Valid: true},
	}, nil)

	// Subscription updates
	mockRepo.EXPECT().UpdateSubscriptionProviderID(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.UpdateSubscriptionProviderIDParams) (repository.Subscription, error) {
			return repository.Subscription{
				ID:                     subscriptionID,
				TenantID:               tenantID,
				ProviderSubscriptionID: params.ProviderSubscriptionID,
				Status:                 params.Status,
			}, nil
		})

	mockRepo.EXPECT().UpdateSubscriptionStatus(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.UpdateSubscriptionStatusParams) (repository.Subscription, error) {
			return repository.Subscription{
				ID:                 subscriptionID,
				TenantID:           tenantID,
				Status:             params.Status,
				CurrentPeriodStart: params.CurrentPeriodStart,
				CurrentPeriodEnd:   params.CurrentPeriodEnd,
				NextBillingDate:    params.NextBillingDate,
			}, nil
		})

	mockRepo.EXPECT().CreateSubscriptionItem(gomock.Any(), gomock.Any()).Return(repository.SubscriptionItem{
		ID:             newUUID(),
		SubscriptionID: subscriptionID,
		ProductSkuID:   skuID,
		Quantity:       2,
		UnitPriceCents: 1800,
	}, nil)

	mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{
		ID: newUUID(),
	}, nil)

	// For GetSubscription return
	mockRepo.EXPECT().GetSubscriptionWithDetails(gomock.Any(), gomock.Any()).Return(repository.GetSubscriptionWithDetailsRow{
		ID:              subscriptionID,
		TenantID:        tenantID,
		UserID:          userID,
		Status:          "active",
		BillingInterval: "monthly",
	}, nil)

	mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{}, nil)

	params := CreateSubscriptionParams{
		UserID:            userID,
		ProductSKUID:      skuID,
		Quantity:          2,
		BillingInterval:   "monthly",
		PaymentMethodID:   paymentMethodID,
		ShippingAddressID: shippingAddressID,
		ShippingMethodID:  shippingMethodID,
	}

	result, err := svc.CreateSubscription(ctx, params)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, subscriptionID, result.ID)
}

func Test_CreateSubscription_LinksToStripeViaProviderSubscriptionID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	ctx := contextWithTenant(tenantID)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{
		createSubscriptionResult: &billing.Subscription{
			ID:                 "sub_stripe_abc123",
			Status:             "active",
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
		},
	}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	// Setup basic mocks (abbreviated for focus)
	mockRepo.EXPECT().GetSKUByID(gomock.Any(), gomock.Any()).Return(repository.ProductSku{
		ID:        newUUID(),
		ProductID: newUUID(),
		Sku:       "TEST-SKU",
	}, nil)
	mockRepo.EXPECT().GetDefaultPriceList(gomock.Any(), gomock.Any()).Return(repository.PriceList{ID: newUUID()}, nil)
	mockRepo.EXPECT().GetPriceForSKU(gomock.Any(), gomock.Any()).Return(repository.PriceListEntry{PriceCents: 1800}, nil)

	billingCustomerID := newUUID()
	mockRepo.EXPECT().GetBillingCustomerForUser(gomock.Any(), gomock.Any()).Return(repository.BillingCustomer{
		ID:                 billingCustomerID,
		ProviderCustomerID: "cus_test",
	}, nil)
	mockRepo.EXPECT().GetPaymentMethodByID(gomock.Any(), gomock.Any()).Return(repository.GetPaymentMethodByIDRow{
		ID:                      newUUID(),
		BillingCustomerID:       billingCustomerID,
		ProviderPaymentMethodID: "pm_test",
	}, nil)
	mockRepo.EXPECT().CreateSubscription(gomock.Any(), gomock.Any()).Return(repository.Subscription{ID: newUUID()}, nil)
	mockRepo.EXPECT().GetProductByID(gomock.Any(), gomock.Any()).Return(repository.Product{Name: "Test"}, nil)

	// Verify provider_subscription_id is set
	var capturedProviderID string
	mockRepo.EXPECT().UpdateSubscriptionProviderID(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.UpdateSubscriptionProviderIDParams) (repository.Subscription, error) {
			capturedProviderID = params.ProviderSubscriptionID.String
			return repository.Subscription{
				ID:                     params.ID,
				ProviderSubscriptionID: params.ProviderSubscriptionID,
			}, nil
		})

	mockRepo.EXPECT().UpdateSubscriptionStatus(gomock.Any(), gomock.Any()).Return(repository.Subscription{}, nil)
	mockRepo.EXPECT().CreateSubscriptionItem(gomock.Any(), gomock.Any()).Return(repository.SubscriptionItem{}, nil)
	mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
	mockRepo.EXPECT().GetSubscriptionWithDetails(gomock.Any(), gomock.Any()).Return(repository.GetSubscriptionWithDetailsRow{}, nil)
	mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{}, nil)

	params := CreateSubscriptionParams{
		UserID:            userID,
		ProductSKUID:      newUUID(),
		Quantity:          1,
		BillingInterval:   "monthly",
		PaymentMethodID:   newUUID(),
		ShippingAddressID: newUUID(),
		ShippingMethodID:  newUUID(),
	}

	_, err := svc.CreateSubscription(ctx, params)

	require.NoError(t, err)
	assert.Equal(t, "sub_stripe_abc123", capturedProviderID, "Should link to Stripe via provider_subscription_id")
}

// =============================================================================
// TEST: PauseSubscription
// =============================================================================

func Test_PauseSubscription_OnlyActiveCanBePaused(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  string
		shouldSucceed  bool
		expectedError  error
	}{
		{
			name:          "active subscription can be paused",
			initialStatus: "active",
			shouldSucceed: true,
		},
		{
			name:          "paused subscription cannot be paused again",
			initialStatus: "paused",
			shouldSucceed: false,
			expectedError: ErrSubscriptionNotActive,
		},
		{
			name:          "canceled subscription cannot be paused",
			initialStatus: "canceled",
			shouldSucceed: false,
			expectedError: ErrSubscriptionNotActive,
		},
		{
			name:          "past_due subscription cannot be paused",
			initialStatus: "past_due",
			shouldSucceed: false,
			expectedError: ErrSubscriptionNotActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tenantID := newUUID()
			userID := newUUID()
			subscriptionID := newUUID()
			ctx := contextWithTenant(tenantID)

			mockRepo := repository.NewMockQuerier(ctrl)
			mockBilling := &mockSubscriptionBillingProvider{}

			svc := NewSubscriptionService(mockRepo, mockBilling)

			subscription := createTestSubscription(tenantID, userID, tt.initialStatus)
			subscription.ID = subscriptionID

			mockRepo.EXPECT().GetSubscriptionByID(gomock.Any(), repository.GetSubscriptionByIDParams{
				ID:       subscriptionID,
				TenantID: tenantID,
			}).Return(subscription, nil)

			if tt.shouldSucceed {
				mockRepo.EXPECT().UpdateSubscriptionPauseResume(gomock.Any(), gomock.Any()).Return(repository.Subscription{}, nil)
				mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
				mockRepo.EXPECT().GetSubscriptionWithDetails(gomock.Any(), gomock.Any()).Return(repository.GetSubscriptionWithDetailsRow{
					ID:       subscriptionID,
					TenantID: tenantID,
					Status:   "paused",
				}, nil)
				mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{}, nil)
			}

			params := PauseSubscriptionParams{
				TenantID:       tenantID,
				SubscriptionID: subscriptionID,
			}

			_, err := svc.PauseSubscription(ctx, params)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.expectedError)
			}
		})
	}
}

func Test_PauseSubscription_CallsStripeWithVoidBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	subscriptionID := newUUID()
	ctx := contextWithTenant(tenantID)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	subscription := createTestSubscription(tenantID, userID, "active")
	subscription.ID = subscriptionID
	subscription.ProviderSubscriptionID = pgtype.Text{String: "sub_stripe123", Valid: true}

	mockRepo.EXPECT().GetSubscriptionByID(gomock.Any(), gomock.Any()).Return(subscription, nil)
	mockRepo.EXPECT().UpdateSubscriptionPauseResume(gomock.Any(), gomock.Any()).Return(repository.Subscription{}, nil)
	mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
	mockRepo.EXPECT().GetSubscriptionWithDetails(gomock.Any(), gomock.Any()).Return(repository.GetSubscriptionWithDetailsRow{
		ID:       subscriptionID,
		TenantID: tenantID,
		Status:   "paused",
	}, nil)
	mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{}, nil)

	params := PauseSubscriptionParams{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
	}

	_, err := svc.PauseSubscription(ctx, params)

	require.NoError(t, err)
	require.Len(t, mockBilling.pauseCalls, 1, "Should call Stripe PauseSubscription once")
	assert.Equal(t, "sub_stripe123", mockBilling.pauseCalls[0].SubscriptionID)
	assert.Equal(t, "void", mockBilling.pauseCalls[0].Behavior, "Should void pending invoices")
}

func Test_PauseSubscription_UpdatesLocalStatusToPaused(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	subscriptionID := newUUID()
	ctx := contextWithTenant(tenantID)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	subscription := createTestSubscription(tenantID, userID, "active")
	subscription.ID = subscriptionID

	mockRepo.EXPECT().GetSubscriptionByID(gomock.Any(), gomock.Any()).Return(subscription, nil)

	var capturedStatus string
	mockRepo.EXPECT().UpdateSubscriptionPauseResume(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.UpdateSubscriptionPauseResumeParams) (repository.Subscription, error) {
			capturedStatus = params.Status
			return repository.Subscription{ID: subscriptionID, Status: params.Status}, nil
		})

	mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
	mockRepo.EXPECT().GetSubscriptionWithDetails(gomock.Any(), gomock.Any()).Return(repository.GetSubscriptionWithDetailsRow{
		ID:       subscriptionID,
		TenantID: tenantID,
		Status:   "paused",
	}, nil)
	mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{}, nil)

	params := PauseSubscriptionParams{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
	}

	_, err := svc.PauseSubscription(ctx, params)

	require.NoError(t, err)
	assert.Equal(t, "paused", capturedStatus, "Local status should be updated to 'paused'")
}

// =============================================================================
// TEST: ResumeSubscription
// =============================================================================

func Test_ResumeSubscription_OnlyPausedCanBeResumed(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus string
		shouldSucceed bool
		expectedError error
	}{
		{
			name:          "paused subscription can be resumed",
			initialStatus: "paused",
			shouldSucceed: true,
		},
		{
			name:          "active subscription cannot be resumed",
			initialStatus: "active",
			shouldSucceed: false,
			expectedError: ErrSubscriptionNotPaused,
		},
		{
			name:          "canceled subscription cannot be resumed",
			initialStatus: "canceled",
			shouldSucceed: false,
			expectedError: ErrSubscriptionNotPaused,
		},
		{
			name:          "past_due subscription cannot be resumed",
			initialStatus: "past_due",
			shouldSucceed: false,
			expectedError: ErrSubscriptionNotPaused,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tenantID := newUUID()
			userID := newUUID()
			subscriptionID := newUUID()
			ctx := contextWithTenant(tenantID)

			mockRepo := repository.NewMockQuerier(ctrl)
			mockBilling := &mockSubscriptionBillingProvider{}

			svc := NewSubscriptionService(mockRepo, mockBilling)

			subscription := createTestSubscription(tenantID, userID, tt.initialStatus)
			subscription.ID = subscriptionID

			mockRepo.EXPECT().GetSubscriptionByID(gomock.Any(), repository.GetSubscriptionByIDParams{
				ID:       subscriptionID,
				TenantID: tenantID,
			}).Return(subscription, nil)

			if tt.shouldSucceed {
				mockRepo.EXPECT().UpdateSubscriptionStatus(gomock.Any(), gomock.Any()).Return(repository.Subscription{}, nil)
				mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
				mockRepo.EXPECT().GetSubscriptionWithDetails(gomock.Any(), gomock.Any()).Return(repository.GetSubscriptionWithDetailsRow{
					ID:       subscriptionID,
					TenantID: tenantID,
					Status:   "active",
				}, nil)
				mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{}, nil)
			}

			params := ResumeSubscriptionParams{
				TenantID:       tenantID,
				SubscriptionID: subscriptionID,
			}

			_, err := svc.ResumeSubscription(ctx, params)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.expectedError)
			}
		})
	}
}

func Test_ResumeSubscription_UpdatesPeriodDatesFromStripe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	subscriptionID := newUUID()
	ctx := contextWithTenant(tenantID)

	stripeStart := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	stripeEnd := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{
		resumeSubscriptionResult: &billing.Subscription{
			ID:                 "sub_stripe123",
			Status:             "active",
			CurrentPeriodStart: stripeStart,
			CurrentPeriodEnd:   stripeEnd,
		},
	}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	subscription := createTestSubscription(tenantID, userID, "paused")
	subscription.ID = subscriptionID
	subscription.ProviderSubscriptionID = pgtype.Text{String: "sub_stripe123", Valid: true}

	mockRepo.EXPECT().GetSubscriptionByID(gomock.Any(), gomock.Any()).Return(subscription, nil)

	var capturedPeriodStart, capturedPeriodEnd time.Time
	mockRepo.EXPECT().UpdateSubscriptionStatus(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.UpdateSubscriptionStatusParams) (repository.Subscription, error) {
			capturedPeriodStart = params.CurrentPeriodStart.Time
			capturedPeriodEnd = params.CurrentPeriodEnd.Time
			return repository.Subscription{}, nil
		})

	mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
	mockRepo.EXPECT().GetSubscriptionWithDetails(gomock.Any(), gomock.Any()).Return(repository.GetSubscriptionWithDetailsRow{
		ID:       subscriptionID,
		TenantID: tenantID,
		Status:   "active",
	}, nil)
	mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{}, nil)

	params := ResumeSubscriptionParams{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
	}

	_, err := svc.ResumeSubscription(ctx, params)

	require.NoError(t, err)
	assert.Equal(t, stripeStart, capturedPeriodStart, "Should update period start from Stripe")
	assert.Equal(t, stripeEnd, capturedPeriodEnd, "Should update period end from Stripe")
}

// =============================================================================
// TEST: CancelSubscription
// =============================================================================

func Test_CancelSubscription_CanCancelAtPeriodEndOrImmediately(t *testing.T) {
	tests := []struct {
		name              string
		cancelAtPeriodEnd bool
	}{
		{
			name:              "cancel at period end",
			cancelAtPeriodEnd: true,
		},
		{
			name:              "cancel immediately",
			cancelAtPeriodEnd: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tenantID := newUUID()
			userID := newUUID()
			subscriptionID := newUUID()
			ctx := contextWithTenant(tenantID)

			mockRepo := repository.NewMockQuerier(ctrl)
			mockBilling := &mockSubscriptionBillingProvider{}

			svc := NewSubscriptionService(mockRepo, mockBilling)

			subscription := createTestSubscription(tenantID, userID, "active")
			subscription.ID = subscriptionID
			subscription.ProviderSubscriptionID = pgtype.Text{String: "sub_stripe123", Valid: true}

			mockRepo.EXPECT().GetSubscriptionByID(gomock.Any(), gomock.Any()).Return(subscription, nil)
			mockRepo.EXPECT().UpdateSubscriptionCancellation(gomock.Any(), gomock.Any()).Return(repository.Subscription{}, nil)
			mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
			mockRepo.EXPECT().GetSubscriptionWithDetails(gomock.Any(), gomock.Any()).Return(repository.GetSubscriptionWithDetailsRow{
				ID:       subscriptionID,
				TenantID: tenantID,
			}, nil)
			mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{}, nil)

			params := CancelSubscriptionParams{
				TenantID:          tenantID,
				SubscriptionID:    subscriptionID,
				CancelAtPeriodEnd: tt.cancelAtPeriodEnd,
			}

			_, err := svc.CancelSubscription(ctx, params)

			require.NoError(t, err)
			require.Len(t, mockBilling.cancelCalls, 1)
			assert.Equal(t, tt.cancelAtPeriodEnd, mockBilling.cancelCalls[0].CancelAtPeriodEnd)
		})
	}
}

func Test_CancelSubscription_StoresCancellationReason(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	subscriptionID := newUUID()
	ctx := contextWithTenant(tenantID)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	subscription := createTestSubscription(tenantID, userID, "active")
	subscription.ID = subscriptionID

	mockRepo.EXPECT().GetSubscriptionByID(gomock.Any(), gomock.Any()).Return(subscription, nil)

	var capturedReason string
	mockRepo.EXPECT().UpdateSubscriptionCancellation(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.UpdateSubscriptionCancellationParams) (repository.Subscription, error) {
			if params.CancellationReason.Valid {
				capturedReason = params.CancellationReason.String
			}
			return repository.Subscription{}, nil
		})

	mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
	mockRepo.EXPECT().GetSubscriptionWithDetails(gomock.Any(), gomock.Any()).Return(repository.GetSubscriptionWithDetailsRow{}, nil)
	mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{}, nil)

	params := CancelSubscriptionParams{
		TenantID:           tenantID,
		SubscriptionID:     subscriptionID,
		CancelAtPeriodEnd:  true,
		CancellationReason: "customer_request",
	}

	_, err := svc.CancelSubscription(ctx, params)

	require.NoError(t, err)
	assert.Equal(t, "customer_request", capturedReason, "Should store cancellation reason")
}

// =============================================================================
// TEST: CreateOrderFromSubscriptionInvoice
// =============================================================================

func Test_CreateOrderFromSubscriptionInvoice_IdempotentSameInvoiceReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	subscriptionID := newUUID()
	ctx := contextWithTenant(tenantID)

	invoiceID := "in_test123"

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{
		getInvoiceResult: &billing.Invoice{
			ID:             invoiceID,
			SubscriptionID: "sub_stripe123",
			Status:         "paid",
			Currency:       "usd",
		},
	}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	mockRepo.EXPECT().GetSubscriptionByProviderID(gomock.Any(), gomock.Any()).Return(repository.Subscription{
		ID:                     subscriptionID,
		TenantID:               tenantID,
		UserID:                 userID,
		ProviderSubscriptionID: pgtype.Text{String: "sub_stripe123", Valid: true},
	}, nil)

	// Simulate invoice already processed - event with this invoice_id exists
	mockRepo.EXPECT().GetSubscriptionScheduleEventByInvoiceID(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{
		ID: pgtype.UUID{Valid: true}, // Existing event
	}, nil)

	_, err := svc.CreateOrderFromSubscriptionInvoice(ctx, invoiceID, tenantID)

	assert.ErrorIs(t, err, ErrInvoiceAlreadyProcessed, "Should return ErrInvoiceAlreadyProcessed on duplicate")
}

func Test_CreateOrderFromSubscriptionInvoice_CreatesOrderLinkedToSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	subscriptionID := newUUID()
	shippingAddressID := newUUID()
	ctx := contextWithTenant(tenantID)

	invoiceID := "in_test123"

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{
		getInvoiceResult: &billing.Invoice{
			ID:              invoiceID,
			SubscriptionID:  "sub_stripe123",
			Status:          "paid",
			Currency:        "usd",
			AmountPaidCents: 2440,
			PaymentIntentID: "pi_test123",
		},
	}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	mockRepo.EXPECT().GetSubscriptionByProviderID(gomock.Any(), gomock.Any()).Return(repository.Subscription{
		ID:                     subscriptionID,
		TenantID:               tenantID,
		UserID:                 userID,
		ProviderSubscriptionID: pgtype.Text{String: "sub_stripe123", Valid: true},
		BillingCustomerID:      newUUID(),
		PaymentMethodID:        newUUID(),
		SubtotalCents:          1800,
		ShippingCents:          500,
		TaxCents:               140,
		TotalCents:             2440,
		Currency:               "usd",
		ShippingAddressID:      shippingAddressID,
	}, nil)

	// Not processed yet
	mockRepo.EXPECT().GetSubscriptionScheduleEventByInvoiceID(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, sql.ErrNoRows)

	mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{
		{
			ProductSkuID:   newUUID(),
			ProductName:    "Ethiopian Yirgacheffe",
			Sku:            "ETH-YIRG-12OZ-WB",
			Quantity:       1,
			UnitPriceCents: 1800,
			Grind:          "whole_bean",
		},
	}, nil)

	var capturedOrderParams repository.CreateOrderParams
	mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.CreateOrderParams) (repository.Order, error) {
			capturedOrderParams = params
			return repository.Order{
				ID:             newUUID(),
				TenantID:       params.TenantID,
				UserID:         params.UserID,
				OrderType:      params.OrderType,
				Status:         params.Status,
				SubscriptionID: params.SubscriptionID,
			}, nil
		})

	mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).Return(repository.OrderItem{ID: newUUID()}, nil)
	mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil)
	mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(repository.Payment{ID: newUUID()}, nil)
	mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).Return(nil)
	mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
	mockRepo.EXPECT().GetAddressByID(gomock.Any(), shippingAddressID).Return(repository.Address{ID: shippingAddressID}, nil)

	result, err := svc.CreateOrderFromSubscriptionInvoice(ctx, invoiceID, tenantID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "subscription", capturedOrderParams.OrderType, "Order type should be 'subscription'")
	assert.Equal(t, subscriptionID, capturedOrderParams.SubscriptionID, "Order should be linked to subscription")
	assert.Equal(t, "confirmed", capturedOrderParams.Status, "Subscription orders should be immediately confirmed")
}

func Test_CreateOrderFromSubscriptionInvoice_CreatesOrderItemsFromSubscriptionItems(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	subscriptionID := newUUID()
	ctx := contextWithTenant(tenantID)

	invoiceID := "in_test123"

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{
		getInvoiceResult: &billing.Invoice{
			ID:              invoiceID,
			SubscriptionID:  "sub_stripe123",
			Status:          "paid",
			Currency:        "usd",
			AmountPaidCents: 3600,
			PaymentIntentID: "pi_test123",
		},
	}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	mockRepo.EXPECT().GetSubscriptionByProviderID(gomock.Any(), gomock.Any()).Return(repository.Subscription{
		ID:                     subscriptionID,
		TenantID:               tenantID,
		UserID:                 userID,
		ProviderSubscriptionID: pgtype.Text{String: "sub_stripe123", Valid: true},
		BillingCustomerID:      newUUID(),
		PaymentMethodID:        newUUID(),
		ShippingAddressID:      newUUID(),
	}, nil)

	mockRepo.EXPECT().GetSubscriptionScheduleEventByInvoiceID(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, sql.ErrNoRows)

	subscriptionItems := []repository.ListSubscriptionItemsForSubscriptionRow{
		{
			ProductSkuID:   newUUID(),
			ProductName:    "Ethiopian Yirgacheffe",
			Sku:            "ETH-YIRG-12OZ-WB",
			Quantity:       2,
			UnitPriceCents: 1800,
			WeightValue:    pgtype.Numeric{Int: bigIntFromInt64(12), Valid: true},
			WeightUnit:     "oz",
			Grind:          "whole_bean",
		},
	}

	mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return(subscriptionItems, nil)

	mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(repository.Order{ID: newUUID()}, nil)

	var capturedOrderItems []repository.CreateOrderItemParams
	mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.CreateOrderItemParams) (repository.OrderItem, error) {
			capturedOrderItems = append(capturedOrderItems, params)
			return repository.OrderItem{ID: newUUID()}, nil
		})

	mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil)
	mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(repository.Payment{}, nil)
	mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).Return(nil)
	mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
	mockRepo.EXPECT().GetAddressByID(gomock.Any(), gomock.Any()).Return(repository.Address{}, nil)

	_, err := svc.CreateOrderFromSubscriptionInvoice(ctx, invoiceID, tenantID)

	require.NoError(t, err)
	require.Len(t, capturedOrderItems, 1, "Should create order items from subscription items")
	assert.Equal(t, subscriptionItems[0].ProductSkuID, capturedOrderItems[0].ProductSkuID)
	assert.Equal(t, subscriptionItems[0].Quantity, capturedOrderItems[0].Quantity)
	assert.Equal(t, subscriptionItems[0].UnitPriceCents, capturedOrderItems[0].UnitPriceCents)
}

func Test_CreateOrderFromSubscriptionInvoice_DecrementsInventory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	userID := newUUID()
	subscriptionID := newUUID()
	ctx := contextWithTenant(tenantID)

	invoiceID := "in_test123"
	skuID := newUUID()

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{
		getInvoiceResult: &billing.Invoice{
			ID:              invoiceID,
			SubscriptionID:  "sub_stripe123",
			Status:          "paid",
			Currency:        "usd",
			AmountPaidCents: 3600,
			PaymentIntentID: "pi_test123",
		},
	}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	mockRepo.EXPECT().GetSubscriptionByProviderID(gomock.Any(), gomock.Any()).Return(repository.Subscription{
		ID:                     subscriptionID,
		TenantID:               tenantID,
		UserID:                 userID,
		ProviderSubscriptionID: pgtype.Text{String: "sub_stripe123", Valid: true},
		BillingCustomerID:      newUUID(),
		PaymentMethodID:        newUUID(),
		ShippingAddressID:      newUUID(),
	}, nil)

	mockRepo.EXPECT().GetSubscriptionScheduleEventByInvoiceID(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, sql.ErrNoRows)

	mockRepo.EXPECT().ListSubscriptionItemsForSubscription(gomock.Any(), gomock.Any()).Return([]repository.ListSubscriptionItemsForSubscriptionRow{
		{
			ProductSkuID:   skuID,
			ProductName:    "Ethiopian Yirgacheffe",
			Sku:            "ETH-YIRG-12OZ-WB",
			Quantity:       3,
			UnitPriceCents: 1800,
		},
	}, nil)

	mockRepo.EXPECT().CreateOrder(gomock.Any(), gomock.Any()).Return(repository.Order{ID: newUUID()}, nil)
	mockRepo.EXPECT().CreateOrderItem(gomock.Any(), gomock.Any()).Return(repository.OrderItem{}, nil)
	mockRepo.EXPECT().GetOrderItems(gomock.Any(), gomock.Any()).Return([]repository.GetOrderItemsRow{}, nil)
	mockRepo.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(repository.Payment{}, nil)

	var decrementedSKUID pgtype.UUID
	var decrementedQuantity int32
	mockRepo.EXPECT().DecrementSKUStock(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.DecrementSKUStockParams) error {
			decrementedSKUID = params.ID
			decrementedQuantity = params.InventoryQuantity
			return nil
		})

	mockRepo.EXPECT().CreateSubscriptionScheduleEvent(gomock.Any(), gomock.Any()).Return(repository.SubscriptionSchedule{}, nil)
	mockRepo.EXPECT().GetAddressByID(gomock.Any(), gomock.Any()).Return(repository.Address{}, nil)

	_, err := svc.CreateOrderFromSubscriptionInvoice(ctx, invoiceID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, skuID, decrementedSKUID, "Should decrement correct SKU")
	assert.Equal(t, int32(3), decrementedQuantity, "Should decrement by subscription item quantity")
}

func Test_CreateOrderFromSubscriptionInvoice_RejectsNonSubscriptionInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	ctx := contextWithTenant(tenantID)

	invoiceID := "in_test123"

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{
		getInvoiceResult: &billing.Invoice{
			ID:             invoiceID,
			SubscriptionID: "", // Not a subscription invoice
			Status:         "paid",
		},
	}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	_, err := svc.CreateOrderFromSubscriptionInvoice(ctx, invoiceID, tenantID)

	assert.ErrorIs(t, err, ErrInvoiceNotSubscription, "Should reject non-subscription invoices")
}

// =============================================================================
// TEST: SyncSubscriptionFromWebhook
// =============================================================================

func Test_SyncSubscriptionFromWebhook_IdempotentViaWebhookEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	ctx := contextWithTenant(tenantID)

	eventID := "evt_test123"

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	// Event already processed
	mockRepo.EXPECT().GetWebhookEventByProviderID(gomock.Any(), repository.GetWebhookEventByProviderIDParams{
		ProviderEventID: eventID,
		Provider:        "stripe",
		TenantID:        tenantID,
	}).Return(repository.WebhookEvent{
		ID:              newUUID(),
		ProviderEventID: eventID,
	}, nil)

	params := SyncSubscriptionParams{
		TenantID:               tenantID,
		EventID:                eventID,
		EventType:              "customer.subscription.updated",
		ProviderSubscriptionID: "sub_stripe123",
	}

	err := svc.SyncSubscriptionFromWebhook(ctx, params)

	assert.NoError(t, err, "Should return nil (not error) on duplicate event")
}

func Test_SyncSubscriptionFromWebhook_UpdatesLocalStatusFromStripe(t *testing.T) {
	tests := []struct {
		name          string
		stripeStatus  string
		expectedLocal string
	}{
		{
			name:          "active status synced",
			stripeStatus:  "active",
			expectedLocal: "active",
		},
		{
			name:          "past_due status synced",
			stripeStatus:  "past_due",
			expectedLocal: "past_due",
		},
		{
			name:          "canceled status synced",
			stripeStatus:  "canceled",
			expectedLocal: "canceled",
		},
		{
			name:          "paused status synced",
			stripeStatus:  "paused",
			expectedLocal: "paused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tenantID := newUUID()
			subscriptionID := newUUID()
			ctx := contextWithTenant(tenantID)

			eventID := "evt_test123"

			mockRepo := repository.NewMockQuerier(ctrl)
			mockBilling := &mockSubscriptionBillingProvider{
				getSubscriptionResult: &billing.Subscription{
					ID:                 "sub_stripe123",
					Status:             tt.stripeStatus,
					CurrentPeriodStart: time.Now(),
					CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
				},
			}

			svc := NewSubscriptionService(mockRepo, mockBilling)

			// Event not processed
			mockRepo.EXPECT().GetWebhookEventByProviderID(gomock.Any(), gomock.Any()).Return(repository.WebhookEvent{}, sql.ErrNoRows)
			mockRepo.EXPECT().CreateWebhookEvent(gomock.Any(), gomock.Any()).Return(repository.WebhookEvent{}, nil)

			mockRepo.EXPECT().GetSubscriptionByProviderID(gomock.Any(), gomock.Any()).Return(repository.Subscription{
				ID:                     subscriptionID,
				TenantID:               tenantID,
				ProviderSubscriptionID: pgtype.Text{String: "sub_stripe123", Valid: true},
			}, nil)

			var capturedStatus string
			mockRepo.EXPECT().UpdateSubscriptionStatus(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, params repository.UpdateSubscriptionStatusParams) (repository.Subscription, error) {
					capturedStatus = params.Status
					return repository.Subscription{}, nil
				})

			params := SyncSubscriptionParams{
				TenantID:               tenantID,
				EventID:                eventID,
				EventType:              "customer.subscription.updated",
				ProviderSubscriptionID: "sub_stripe123",
			}

			err := svc.SyncSubscriptionFromWebhook(ctx, params)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedLocal, capturedStatus, "Local status should match Stripe")
		})
	}
}

func Test_SyncSubscriptionFromWebhook_HandlesAllStatusTransitions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := newUUID()
	subscriptionID := newUUID()
	ctx := contextWithTenant(tenantID)

	eventID := "evt_test123"
	canceledAt := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockBilling := &mockSubscriptionBillingProvider{
		getSubscriptionResult: &billing.Subscription{
			ID:                 "sub_stripe123",
			Status:             "canceled",
			CanceledAt:         &canceledAt,
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
			CancelAtPeriodEnd:  true,
		},
	}

	svc := NewSubscriptionService(mockRepo, mockBilling)

	mockRepo.EXPECT().GetWebhookEventByProviderID(gomock.Any(), gomock.Any()).Return(repository.WebhookEvent{}, sql.ErrNoRows)
	mockRepo.EXPECT().CreateWebhookEvent(gomock.Any(), gomock.Any()).Return(repository.WebhookEvent{}, nil)

	mockRepo.EXPECT().GetSubscriptionByProviderID(gomock.Any(), gomock.Any()).Return(repository.Subscription{
		ID:                     subscriptionID,
		TenantID:               tenantID,
		ProviderSubscriptionID: pgtype.Text{String: "sub_stripe123", Valid: true},
	}, nil)

	var capturedCanceledAt pgtype.Timestamptz
	var capturedCancelAtPeriodEnd pgtype.Bool
	mockRepo.EXPECT().UpdateSubscriptionStatus(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params repository.UpdateSubscriptionStatusParams) (repository.Subscription, error) {
			capturedCanceledAt = params.CancelledAt
			capturedCancelAtPeriodEnd = params.CancelAtPeriodEnd
			return repository.Subscription{}, nil
		})

	params := SyncSubscriptionParams{
		TenantID:               tenantID,
		EventID:                eventID,
		EventType:              "customer.subscription.deleted",
		ProviderSubscriptionID: "sub_stripe123",
	}

	err := svc.SyncSubscriptionFromWebhook(ctx, params)

	require.NoError(t, err)
	assert.True(t, capturedCanceledAt.Valid, "Should set canceled_at when subscription canceled")
	assert.Equal(t, canceledAt, capturedCanceledAt.Time)
	assert.True(t, capturedCancelAtPeriodEnd.Bool, "Should sync cancel_at_period_end flag")
}

// =============================================================================
// HELPER: bigIntFromInt64
// =============================================================================

func bigIntFromInt64(n int64) *big.Int {
	return big.NewInt(n)
}
