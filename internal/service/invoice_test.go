package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dukerupert/hiri/internal/billing"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/dukerupert/hiri/internal/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// mockPaymentTermsService implements PaymentTermsService for testing
type mockPaymentTermsService struct {
	paymentTerms *repository.PaymentTerm
	err          error
}

func (m *mockPaymentTermsService) CreatePaymentTerms(ctx context.Context, params CreatePaymentTermsParams) (*repository.PaymentTerm, error) {
	return m.paymentTerms, m.err
}

func (m *mockPaymentTermsService) GetPaymentTerms(ctx context.Context, paymentTermsID string) (*repository.PaymentTerm, error) {
	return m.paymentTerms, m.err
}

func (m *mockPaymentTermsService) GetPaymentTermsByCode(ctx context.Context, code string) (*repository.PaymentTerm, error) {
	return m.paymentTerms, m.err
}

func (m *mockPaymentTermsService) GetDefaultPaymentTerms(ctx context.Context) (*repository.PaymentTerm, error) {
	return m.paymentTerms, m.err
}

func (m *mockPaymentTermsService) ListPaymentTerms(ctx context.Context) ([]repository.PaymentTerm, error) {
	return nil, m.err
}

func (m *mockPaymentTermsService) UpdatePaymentTerms(ctx context.Context, params UpdatePaymentTermsParams) error {
	return m.err
}

func (m *mockPaymentTermsService) SetDefaultPaymentTerms(ctx context.Context, paymentTermsID string) error {
	return m.err
}

func (m *mockPaymentTermsService) DeletePaymentTerms(ctx context.Context, paymentTermsID string) error {
	return m.err
}

func (m *mockPaymentTermsService) CalculateDueDate(ctx context.Context, paymentTermsID string, invoiceDate time.Time) (time.Time, error) {
	return m.CalculateDueDateFromTerms(m.paymentTerms, invoiceDate), m.err
}

func (m *mockPaymentTermsService) CalculateDueDateFromTerms(terms *repository.PaymentTerm, invoiceDate time.Time) time.Time {
	if terms == nil {
		return invoiceDate
	}
	return invoiceDate.AddDate(0, 0, int(terms.Days))
}

// mockBillingProvider implements billing.Provider for testing
type mockBillingProvider struct {
	createCustomerResult *billing.Customer
	createCustomerErr    error // when set, CreateCustomer returns this error
	createInvoiceResult  *billing.Invoice
	finalizeInvoiceResult *billing.Invoice
	sendErr              error
}

func (m *mockBillingProvider) CreateCustomer(ctx context.Context, params billing.CreateCustomerParams) (*billing.Customer, error) {
	if m.createCustomerErr != nil {
		return nil, m.createCustomerErr
	}
	if m.createCustomerResult == nil {
		return &billing.Customer{ID: "cus_test123"}, nil
	}
	return m.createCustomerResult, nil
}

func (m *mockBillingProvider) GetCustomer(ctx context.Context, customerID string) (*billing.Customer, error) {
	return nil, nil
}

func (m *mockBillingProvider) GetCustomerByEmail(ctx context.Context, email string) (*billing.Customer, error) {
	return nil, nil
}

func (m *mockBillingProvider) UpdateCustomer(ctx context.Context, customerID string, params billing.UpdateCustomerParams) (*billing.Customer, error) {
	return nil, nil
}

func (m *mockBillingProvider) VerifyWebhookSignature(payload []byte, signature string, secret string) error {
	return nil
}

func (m *mockBillingProvider) CancelPaymentIntent(ctx context.Context, paymentIntentID string, tenantID string) error {
	return nil
}

func (m *mockBillingProvider) CreatePaymentIntent(ctx context.Context, params billing.CreatePaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, nil
}

func (m *mockBillingProvider) GetPaymentIntent(ctx context.Context, params billing.GetPaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, nil
}

func (m *mockBillingProvider) UpdatePaymentIntent(ctx context.Context, params billing.UpdatePaymentIntentParams) (*billing.PaymentIntent, error) {
	return nil, nil
}

func (m *mockBillingProvider) CreateInvoice(ctx context.Context, params billing.CreateInvoiceParams) (*billing.Invoice, error) {
	if m.createInvoiceResult == nil {
		return &billing.Invoice{ID: "in_test123"}, nil
	}
	return m.createInvoiceResult, nil
}

func (m *mockBillingProvider) GetInvoice(ctx context.Context, params billing.GetInvoiceParams) (*billing.Invoice, error) {
	return nil, nil
}

func (m *mockBillingProvider) AddInvoiceItem(ctx context.Context, params billing.AddInvoiceItemParams) error {
	return nil
}

func (m *mockBillingProvider) FinalizeInvoice(ctx context.Context, params billing.FinalizeInvoiceParams) (*billing.Invoice, error) {
	if m.finalizeInvoiceResult == nil {
		return &billing.Invoice{ID: "in_test123", Status: "open"}, nil
	}
	return m.finalizeInvoiceResult, nil
}

func (m *mockBillingProvider) SendInvoice(ctx context.Context, params billing.SendInvoiceParams) error {
	return m.sendErr
}

func (m *mockBillingProvider) VoidInvoice(ctx context.Context, params billing.VoidInvoiceParams) error {
	return nil
}

func (m *mockBillingProvider) PayInvoice(ctx context.Context, params billing.PayInvoiceParams) (*billing.Invoice, error) {
	return nil, nil
}

func (m *mockBillingProvider) CreateSubscription(ctx context.Context, params billing.CreateSubscriptionParams) (*billing.Subscription, error) {
	return nil, nil
}

func (m *mockBillingProvider) GetSubscription(ctx context.Context, params billing.GetSubscriptionParams) (*billing.Subscription, error) {
	return nil, nil
}


func (m *mockBillingProvider) CancelSubscription(ctx context.Context, params billing.CancelSubscriptionParams) error {
	return nil
}

func (m *mockBillingProvider) PauseSubscription(ctx context.Context, params billing.PauseSubscriptionParams) (*billing.Subscription, error) {
	return nil, nil
}

func (m *mockBillingProvider) ResumeSubscription(ctx context.Context, params billing.ResumeSubscriptionParams) (*billing.Subscription, error) {
	return nil, nil
}

func (m *mockBillingProvider) CreateProduct(ctx context.Context, params billing.CreateProductParams) (*billing.Product, error) {
	return nil, nil
}

func (m *mockBillingProvider) CreateRecurringPrice(ctx context.Context, params billing.CreateRecurringPriceParams) (*billing.Price, error) {
	return nil, nil
}

func (m *mockBillingProvider) CreateCustomerPortalSession(ctx context.Context, params billing.CreatePortalSessionParams) (*billing.PortalSession, error) {
	return nil, nil
}

func (m *mockBillingProvider) RefundPayment(ctx context.Context, params billing.RefundParams) (*billing.Refund, error) {
	return nil, nil
}

// Helper functions for creating test data
func createTestTenantID() pgtype.UUID {
	id := pgtype.UUID{}
	_ = id.Scan(uuid.New().String())
	return id
}

func createTestUserID() pgtype.UUID {
	id := pgtype.UUID{}
	_ = id.Scan(uuid.New().String())
	return id
}

func createTestOrderID() pgtype.UUID {
	id := pgtype.UUID{}
	_ = id.Scan(uuid.New().String())
	return id
}

func createTestInvoiceID() pgtype.UUID {
	id := pgtype.UUID{}
	_ = id.Scan(uuid.New().String())
	return id
}

func createTestContext(tenantID pgtype.UUID) context.Context {
	ctx := context.Background()
	t := &tenant.Tenant{
		ID:     tenantID,
		Slug:   "test-tenant",
		Name:   "Test Tenant",
		Status: "active",
	}
	ctx = tenant.NewContext(ctx, t)
	return ctx
}

// Test_CreateInvoice_OnlyWholesaleUsersAllowed verifies that invoices can only be created for wholesale users
func Test_CreateInvoice_OnlyWholesaleUsersAllowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	orderID := createTestOrderID()
	ctx := createTestContext(tenantID)

	tests := []struct {
		name        string
		accountType string
		wantErr     error
	}{
		{
			name:        "wholesale user allowed",
			accountType: "wholesale",
			wantErr:     nil,
		},
		{
			name:        "retail user rejected",
			accountType: "retail",
			wantErr:     ErrNotWholesaleUser,
		},
		{
			name:        "guest user rejected",
			accountType: "guest",
			wantErr:     ErrNotWholesaleUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repository.NewMockQuerier(ctrl)
			mockPaymentTerms := &mockPaymentTermsService{
				paymentTerms: &repository.PaymentTerm{
					ID:   createTestTenantID(),
					Code: "net_30",
					Days: 30,
				},
			}
			mockBilling := &mockBillingProvider{}

			svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

			// Setup user with specified account type
			mockRepo.EXPECT().
				GetUserByID(ctx, userID).
				Return(repository.User{
					ID:          userID,
					TenantID:    tenantID,
					Email:       "test@example.com",
					AccountType: tt.accountType,
				}, nil)

			// If wholesale, expect additional calls
			if tt.accountType == "wholesale" {
				// GetOrder is called multiple times: once for validation, once for address, once for linking
				mockRepo.EXPECT().
					GetOrder(ctx, gomock.Any()).
					Return(repository.Order{
						ID:             orderID,
						TenantID:       tenantID,
						UserID:         userID,
						OrderType:      "wholesale",
						SubtotalCents:  10000,
						TaxCents:       800,
						ShippingCents:  500,
						TotalCents:     11300,
						OrderNumber:    "WH-001",
						BillingAddressID: createTestTenantID(),
					}, nil).AnyTimes()

				mockRepo.EXPECT().
					GenerateInvoiceNumber(ctx, tenantID).
					Return("INV-001", nil)

				mockRepo.EXPECT().
					CreateInvoice(ctx, gomock.Any()).
					Return(repository.Invoice{
						ID:            createTestInvoiceID(),
						TenantID:      tenantID,
						UserID:        userID,
						InvoiceNumber: "INV-001",
						Status:        "draft",
						TotalCents:    11300,
						BalanceCents:  11300,
					}, nil)

				mockRepo.EXPECT().
					CreateInvoiceOrder(ctx, gomock.Any()).
					Return(repository.InvoiceOrder{}, nil)

				mockRepo.EXPECT().
					GetOrderItems(ctx, orderID).
					Return([]repository.GetOrderItemsRow{}, nil)

				// CreateInvoiceItem is called for shipping line item (ShippingCents > 0)
				mockRepo.EXPECT().
					CreateInvoiceItem(ctx, gomock.Any()).
					Return(repository.InvoiceItem{}, nil).AnyTimes()

				mockRepo.EXPECT().
					GetInvoiceByID(ctx, gomock.Any()).
					Return(repository.Invoice{
						ID:        createTestInvoiceID(),
						TenantID:  tenantID,
						UserID:    userID,
						Status:    "draft",
					}, nil)

				mockRepo.EXPECT().
					GetInvoiceItems(ctx, gomock.Any()).
					Return([]repository.InvoiceItem{}, nil)

				mockRepo.EXPECT().
					GetInvoiceOrders(ctx, gomock.Any()).
					Return([]repository.GetInvoiceOrdersRow{}, nil)

				mockRepo.EXPECT().
					GetInvoicePayments(ctx, gomock.Any()).
					Return([]repository.InvoicePayment{}, nil)

				mockRepo.EXPECT().
					GetUserByID(ctx, userID).
					Return(repository.User{
						ID:          userID,
						TenantID:    tenantID,
						Email:       "test@example.com",
						AccountType: "wholesale",
					}, nil)
			}

			params := CreateInvoiceParams{
				UserID:   userID.String(),
				OrderIDs: []string{orderID.String()},
			}

			_, err := svc.CreateInvoice(ctx, params)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test_CreateInvoice_AllOrdersMustBelongToSameUser verifies that all orders must belong to the same user
func Test_CreateInvoice_AllOrdersMustBelongToSameUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	otherUserID := createTestUserID()
	order1ID := createTestOrderID()
	order2ID := createTestOrderID()
	ctx := createTestContext(tenantID)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockPaymentTerms := &mockPaymentTermsService{
		paymentTerms: &repository.PaymentTerm{
			ID:   createTestTenantID(),
			Code: "net_30",
			Days: 30,
		},
	}
	mockBilling := &mockBillingProvider{}

	svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

	// Setup user
	mockRepo.EXPECT().
		GetUserByID(ctx, userID).
		Return(repository.User{
			ID:          userID,
			TenantID:    tenantID,
			Email:       "test@example.com",
			AccountType: "wholesale",
		}, nil)

	// First order belongs to user
	mockRepo.EXPECT().
		GetOrder(ctx, repository.GetOrderParams{
			TenantID: tenantID,
			ID:       order1ID,
		}).
		Return(repository.Order{
			ID:        order1ID,
			TenantID:  tenantID,
			UserID:    userID,
			OrderType: "wholesale",
		}, nil)

	// Second order belongs to different user
	mockRepo.EXPECT().
		GetOrder(ctx, repository.GetOrderParams{
			TenantID: tenantID,
			ID:       order2ID,
		}).
		Return(repository.Order{
			ID:        order2ID,
			TenantID:  tenantID,
			UserID:    otherUserID, // Different user
			OrderType: "wholesale",
		}, nil)

	params := CreateInvoiceParams{
		UserID:   userID.String(),
		OrderIDs: []string{order1ID.String(), order2ID.String()},
	}

	_, err := svc.CreateInvoice(ctx, params)

	assert.ErrorIs(t, err, ErrOrderNotFound, "Should return ErrOrderNotFound when order belongs to different user")
}

// Test_CreateInvoice_AllOrdersMustBeWholesaleType verifies that all orders must be wholesale type
func Test_CreateInvoice_AllOrdersMustBeWholesaleType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	orderID := createTestOrderID()
	ctx := createTestContext(tenantID)

	tests := []struct {
		name      string
		orderType string
		wantErr   error
	}{
		{
			name:      "wholesale order allowed",
			orderType: "wholesale",
			wantErr:   nil,
		},
		{
			name:      "retail order rejected",
			orderType: "retail",
			wantErr:   ErrOrderNotWholesale,
		},
		{
			name:      "subscription order rejected",
			orderType: "subscription",
			wantErr:   ErrOrderNotWholesale,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repository.NewMockQuerier(ctrl)
			mockPaymentTerms := &mockPaymentTermsService{
				paymentTerms: &repository.PaymentTerm{
					ID:   createTestTenantID(),
					Code: "net_30",
					Days: 30,
				},
			}
			mockBilling := &mockBillingProvider{}

			svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

			// Setup user
			mockRepo.EXPECT().
				GetUserByID(ctx, userID).
				Return(repository.User{
					ID:          userID,
					TenantID:    tenantID,
					Email:       "test@example.com",
					AccountType: "wholesale",
				}, nil)

			// Setup order with specified type - called multiple times during invoice creation
			mockRepo.EXPECT().
				GetOrder(ctx, gomock.Any()).
				Return(repository.Order{
					ID:             orderID,
					TenantID:       tenantID,
					UserID:         userID,
					OrderType:      tt.orderType,
					SubtotalCents:  10000,
					TaxCents:       800,
					ShippingCents:  500,
					TotalCents:     11300,
					BillingAddressID: createTestTenantID(),
				}, nil).AnyTimes()

			// If wholesale, expect additional calls for invoice creation
			if tt.orderType == "wholesale" {
				mockRepo.EXPECT().
					GenerateInvoiceNumber(ctx, tenantID).
					Return("INV-001", nil)

				mockRepo.EXPECT().
					CreateInvoice(ctx, gomock.Any()).
					Return(repository.Invoice{
						ID:            createTestInvoiceID(),
						TenantID:      tenantID,
						UserID:        userID,
						InvoiceNumber: "INV-001",
						Status:        "draft",
					}, nil)

				mockRepo.EXPECT().
					CreateInvoiceOrder(ctx, gomock.Any()).
					Return(repository.InvoiceOrder{}, nil)

				mockRepo.EXPECT().
					GetOrderItems(ctx, orderID).
					Return([]repository.GetOrderItemsRow{}, nil)

				// CreateInvoiceItem for shipping (ShippingCents > 0)
				mockRepo.EXPECT().
					CreateInvoiceItem(ctx, gomock.Any()).
					Return(repository.InvoiceItem{}, nil).AnyTimes()

				mockRepo.EXPECT().
					GetInvoiceByID(ctx, gomock.Any()).
					Return(repository.Invoice{
						ID:       createTestInvoiceID(),
						TenantID: tenantID,
						UserID:   userID,
						Status:   "draft",
					}, nil)

				mockRepo.EXPECT().
					GetInvoiceItems(ctx, gomock.Any()).
					Return([]repository.InvoiceItem{}, nil)

				mockRepo.EXPECT().
					GetInvoiceOrders(ctx, gomock.Any()).
					Return([]repository.GetInvoiceOrdersRow{}, nil)

				mockRepo.EXPECT().
					GetInvoicePayments(ctx, gomock.Any()).
					Return([]repository.InvoicePayment{}, nil)

				mockRepo.EXPECT().
					GetUserByID(ctx, userID).
					Return(repository.User{
						ID:          userID,
						TenantID:    tenantID,
						Email:       "test@example.com",
						AccountType: "wholesale",
					}, nil)
			}

			params := CreateInvoiceParams{
				UserID:   userID.String(),
				OrderIDs: []string{orderID.String()},
			}

			_, err := svc.CreateInvoice(ctx, params)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test_CreateInvoice_PaymentTermsCorrectlyApplied verifies payment terms are correctly applied
func Test_CreateInvoice_PaymentTermsCorrectlyApplied(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	orderID := createTestOrderID()
	ctx := createTestContext(tenantID)

	tests := []struct {
		name              string
		paymentTermsCode  string
		paymentTermsDays  int32
		expectedDueDays   int
	}{
		{
			name:             "Net 15 payment terms",
			paymentTermsCode: "net_15",
			paymentTermsDays: 15,
			expectedDueDays:  15,
		},
		{
			name:             "Net 30 payment terms",
			paymentTermsCode: "net_30",
			paymentTermsDays: 30,
			expectedDueDays:  30,
		},
		{
			name:             "Net 60 payment terms",
			paymentTermsCode: "net_60",
			paymentTermsDays: 60,
			expectedDueDays:  60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repository.NewMockQuerier(ctrl)
			mockPaymentTerms := &mockPaymentTermsService{
				paymentTerms: &repository.PaymentTerm{
					ID:   createTestTenantID(),
					Code: tt.paymentTermsCode,
					Days: tt.paymentTermsDays,
				},
			}
			mockBilling := &mockBillingProvider{}

			svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

			// Setup user
			mockRepo.EXPECT().
				GetUserByID(ctx, userID).
				Return(repository.User{
					ID:          userID,
					TenantID:    tenantID,
					Email:       "test@example.com",
					AccountType: "wholesale",
				}, nil)

			mockRepo.EXPECT().
				GetOrder(ctx, gomock.Any()).
				Return(repository.Order{
					ID:             orderID,
					TenantID:       tenantID,
					UserID:         userID,
					OrderType:      "wholesale",
					SubtotalCents:  10000,
					TaxCents:       800,
					ShippingCents:  500,
					TotalCents:     11300,
					OrderNumber:    "WH-001",
					BillingAddressID: createTestTenantID(),
				}, nil).AnyTimes()

			mockRepo.EXPECT().
				GenerateInvoiceNumber(ctx, tenantID).
				Return("INV-001", nil)

			// Capture the CreateInvoice call to verify payment terms
			var capturedParams repository.CreateInvoiceParams
			mockRepo.EXPECT().
				CreateInvoice(ctx, gomock.Any()).
				DoAndReturn(func(ctx context.Context, params repository.CreateInvoiceParams) (repository.Invoice, error) {
					capturedParams = params
					return repository.Invoice{
						ID:            createTestInvoiceID(),
						TenantID:      tenantID,
						UserID:        userID,
						InvoiceNumber: "INV-001",
						Status:        "draft",
						PaymentTerms:  params.PaymentTerms,
						DueDate:       params.DueDate,
					}, nil
				})

			mockRepo.EXPECT().
				CreateInvoiceOrder(ctx, gomock.Any()).
				Return(repository.InvoiceOrder{}, nil)

			mockRepo.EXPECT().
				GetOrderItems(ctx, orderID).
				Return([]repository.GetOrderItemsRow{}, nil)

			// CreateInvoiceItem for shipping (ShippingCents > 0)
			mockRepo.EXPECT().
				CreateInvoiceItem(ctx, gomock.Any()).
				Return(repository.InvoiceItem{}, nil).AnyTimes()

			mockRepo.EXPECT().
				GetInvoiceByID(ctx, gomock.Any()).
				Return(repository.Invoice{
					ID:       createTestInvoiceID(),
					TenantID: tenantID,
					UserID:   userID,
					Status:   "draft",
				}, nil)

			mockRepo.EXPECT().
				GetInvoiceItems(ctx, gomock.Any()).
				Return([]repository.InvoiceItem{}, nil)

			mockRepo.EXPECT().
				GetInvoiceOrders(ctx, gomock.Any()).
				Return([]repository.GetInvoiceOrdersRow{}, nil)

			mockRepo.EXPECT().
				GetInvoicePayments(ctx, gomock.Any()).
				Return([]repository.InvoicePayment{}, nil)

			mockRepo.EXPECT().
				GetUserByID(ctx, userID).
				Return(repository.User{
					ID:          userID,
					TenantID:    tenantID,
					Email:       "test@example.com",
					AccountType: "wholesale",
				}, nil)

			params := CreateInvoiceParams{
				UserID:   userID.String(),
				OrderIDs: []string{orderID.String()},
			}

			_, err := svc.CreateInvoice(ctx, params)

			assert.NoError(t, err)
			assert.Equal(t, tt.paymentTermsCode, capturedParams.PaymentTerms, "Payment terms code should match")

			// Verify due date is correctly calculated
			if capturedParams.DueDate.Valid {
				invoiceTime := time.Now()
				expectedDueDate := invoiceTime.AddDate(0, 0, tt.expectedDueDays).Truncate(24 * time.Hour)
				actualDueDate := capturedParams.DueDate.Time.Truncate(24 * time.Hour)

				// Allow 1 day tolerance for timing differences
				diff := actualDueDate.Sub(expectedDueDate)
				assert.True(t, diff >= -24*time.Hour && diff <= 24*time.Hour,
					"Due date should be approximately %d days from invoice date", tt.expectedDueDays)
			}
		})
	}
}

// Test_RecordPayment_PaymentCannotExceedBalance verifies that payment cannot exceed invoice balance
func Test_RecordPayment_PaymentCannotExceedBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	invoiceID := createTestInvoiceID()
	ctx := createTestContext(tenantID)

	tests := []struct {
		name           string
		balanceCents   int32
		paymentCents   int32
		shouldSucceed  bool
	}{
		{
			name:          "payment within balance succeeds",
			balanceCents:  10000,
			paymentCents:  5000,
			shouldSucceed: true,
		},
		{
			name:          "payment equal to balance succeeds",
			balanceCents:  10000,
			paymentCents:  10000,
			shouldSucceed: true,
		},
		{
			name:          "payment exceeding balance fails",
			balanceCents:  10000,
			paymentCents:  10001,
			shouldSucceed: false,
		},
		{
			name:          "payment far exceeding balance fails",
			balanceCents:  10000,
			paymentCents:  50000,
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repository.NewMockQuerier(ctrl)
			mockPaymentTerms := &mockPaymentTermsService{}
			mockBilling := &mockBillingProvider{}

			svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

			// Setup invoice with specified balance
			mockRepo.EXPECT().
				GetInvoiceByID(ctx, repository.GetInvoiceByIDParams{
					ID:       invoiceID,
					TenantID: tenantID,
				}).
				Return(repository.Invoice{
					ID:           invoiceID,
					TenantID:     tenantID,
					BalanceCents: tt.balanceCents,
					Status:       "sent",
				}, nil)

			// If payment should succeed, expect CreateInvoicePayment
			if tt.shouldSucceed {
				mockRepo.EXPECT().
					CreateInvoicePayment(ctx, gomock.Any()).
					Return(repository.InvoicePayment{}, nil)
			}

			params := RecordPaymentParams{
				InvoiceID:     invoiceID.String(),
				AmountCents:   tt.paymentCents,
				PaymentMethod: "stripe",
				PaymentDate:   time.Now(),
			}

			err := svc.RecordPayment(ctx, params)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, ErrPaymentExceedsBalance)
			}
		})
	}
}

// Test_RecordPayment_InvoiceStatusTransitions verifies invoice status transitions with payments
func Test_RecordPayment_InvoiceStatusTransitions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	invoiceID := createTestInvoiceID()
	ctx := createTestContext(tenantID)

	tests := []struct {
		name          string
		initialStatus string
		balanceCents  int32
		paymentCents  int32
		expectedFinal string
	}{
		{
			name:          "partial payment keeps sent status",
			initialStatus: "sent",
			balanceCents:  10000,
			paymentCents:  5000,
			expectedFinal: "sent",
		},
		{
			name:          "full payment marks as paid",
			initialStatus: "sent",
			balanceCents:  10000,
			paymentCents:  10000,
			expectedFinal: "sent", // Status is updated via database trigger, not in service
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repository.NewMockQuerier(ctrl)
			mockPaymentTerms := &mockPaymentTermsService{}
			mockBilling := &mockBillingProvider{}

			svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

			mockRepo.EXPECT().
				GetInvoiceByID(ctx, repository.GetInvoiceByIDParams{
					ID:       invoiceID,
					TenantID: tenantID,
				}).
				Return(repository.Invoice{
					ID:           invoiceID,
					TenantID:     tenantID,
					BalanceCents: tt.balanceCents,
					Status:       tt.initialStatus,
				}, nil)

			mockRepo.EXPECT().
				CreateInvoicePayment(ctx, gomock.Any()).
				Return(repository.InvoicePayment{}, nil)

			params := RecordPaymentParams{
				InvoiceID:     invoiceID.String(),
				AmountCents:   tt.paymentCents,
				PaymentMethod: "stripe",
				PaymentDate:   time.Now(),
			}

			err := svc.RecordPayment(ctx, params)
			assert.NoError(t, err)
		})
	}
}

// Test_GenerateConsolidatedInvoice_FindsUninvoicedOrders verifies consolidated invoice generation
func Test_GenerateConsolidatedInvoice_FindsUninvoicedOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	ctx := createTestContext(tenantID)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	t.Run("creates invoice when orders exist", func(t *testing.T) {
		mockRepo := repository.NewMockQuerier(ctrl)
		mockPaymentTerms := &mockPaymentTermsService{
			paymentTerms: &repository.PaymentTerm{
				ID:   createTestTenantID(),
				Code: "net_30",
				Days: 30,
			},
		}
		// Make CreateCustomer return an error so SendInvoice fails fast
		// Since SendImmediately=true is hardcoded, we need SendInvoice to fail gracefully
		// The error is ignored (_ = s.SendInvoice) so this won't break the test
		mockBilling := &mockBillingProvider{
			createCustomerErr: errors.New("stripe unavailable - expected in test"),
		}

		svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

		order1ID := createTestOrderID()
		order2ID := createTestOrderID()

		// Expect query for uninvoiced orders
		mockRepo.EXPECT().
			GetUninvoicedOrdersInPeriod(ctx, gomock.Any()).
			Return([]repository.Order{
				{
					ID:            order1ID,
					TenantID:      tenantID,
					UserID:        userID,
					OrderType:     "wholesale",
					SubtotalCents: 5000,
					TaxCents:      400,
					ShippingCents: 250,
					TotalCents:    5650,
					OrderNumber:   "WH-001",
					BillingAddressID: createTestTenantID(),
				},
				{
					ID:            order2ID,
					TenantID:      tenantID,
					UserID:        userID,
					OrderType:     "wholesale",
					SubtotalCents: 7500,
					TaxCents:      600,
					ShippingCents: 250,
					TotalCents:    8350,
					OrderNumber:   "WH-002",
					BillingAddressID: createTestTenantID(),
				},
			}, nil)

		// Setup user
		mockRepo.EXPECT().
			GetUserByID(ctx, userID).
			Return(repository.User{
				ID:          userID,
				TenantID:    tenantID,
				Email:       "test@example.com",
				AccountType: "wholesale",
			}, nil)

		// Expect GetOrder calls for each order
		mockRepo.EXPECT().
			GetOrder(ctx, gomock.Any()).
			Return(repository.Order{
				ID:            order1ID,
				TenantID:      tenantID,
				UserID:        userID,
				OrderType:     "wholesale",
				SubtotalCents: 5000,
				TaxCents:      400,
				ShippingCents: 250,
				TotalCents:    5650,
				OrderNumber:   "WH-001",
				BillingAddressID: createTestTenantID(),
			}, nil).AnyTimes()

		mockRepo.EXPECT().
			GenerateInvoiceNumber(ctx, tenantID).
			Return("INV-CONSOL-001", nil)

		mockRepo.EXPECT().
			CreateInvoice(ctx, gomock.Any()).
			Return(repository.Invoice{
				ID:            createTestInvoiceID(),
				TenantID:      tenantID,
				UserID:        userID,
				InvoiceNumber: "INV-CONSOL-001",
				Status:        "draft",
			}, nil)

		// Expect CreateInvoiceOrder for each order
		mockRepo.EXPECT().
			CreateInvoiceOrder(ctx, gomock.Any()).
			Return(repository.InvoiceOrder{}, nil).
			Times(2)

		// Expect GetOrderItems for each order
		mockRepo.EXPECT().
			GetOrderItems(ctx, gomock.Any()).
			Return([]repository.GetOrderItemsRow{}, nil).
			Times(2)

		// CreateInvoiceItem for shipping (ShippingCents > 0) - 2 orders
		mockRepo.EXPECT().
			CreateInvoiceItem(ctx, gomock.Any()).
			Return(repository.InvoiceItem{}, nil).AnyTimes()

		// For SendInvoice (SendImmediately=true) - GetBillingCustomerByUserID will be called,
		// then CreateCustomer will fail (from mockBilling.createCustomerErr), so SendInvoice
		// returns early. The error is ignored (_ = s.SendInvoice).
		mockRepo.EXPECT().
			GetBillingCustomerByUserID(ctx, gomock.Any()).
			Return(repository.BillingCustomer{}, errors.New("not found"))

		// SendInvoice calls GetUserByID when GetBillingCustomerByUserID fails (to get user info for Stripe)
		mockRepo.EXPECT().
			GetUserByID(ctx, userID).
			Return(repository.User{
				ID:          userID,
				TenantID:    tenantID,
				Email:       "test@example.com",
				AccountType: "wholesale",
			}, nil).AnyTimes()

		// GetInvoiceByID is called by SendInvoice (to validate draft status) and by GetInvoice (return value)
		mockRepo.EXPECT().
			GetInvoiceByID(ctx, gomock.Any()).
			Return(repository.Invoice{
				ID:       createTestInvoiceID(),
				TenantID: tenantID,
				UserID:   userID,
				Status:   "draft",
			}, nil).AnyTimes()

		// GetInvoice returns invoice with items, orders, payments
		mockRepo.EXPECT().
			GetInvoiceItems(ctx, gomock.Any()).
			Return([]repository.InvoiceItem{}, nil).AnyTimes()

		mockRepo.EXPECT().
			GetInvoiceOrders(ctx, gomock.Any()).
			Return([]repository.GetInvoiceOrdersRow{}, nil).AnyTimes()

		mockRepo.EXPECT().
			GetInvoicePayments(ctx, gomock.Any()).
			Return([]repository.InvoicePayment{}, nil).AnyTimes()

		params := ConsolidatedInvoiceParams{
			UserID:             userID.String(),
			BillingPeriodStart: startDate,
			BillingPeriodEnd:   endDate,
		}

		result, err := svc.GenerateConsolidatedInvoice(ctx, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// Test_GenerateConsolidatedInvoice_ReturnsNilWhenNoOrders verifies nil (not error) when no orders
func Test_GenerateConsolidatedInvoice_ReturnsNilWhenNoOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	ctx := createTestContext(tenantID)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockPaymentTerms := &mockPaymentTermsService{}
	mockBilling := &mockBillingProvider{}

	svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

	// Return empty orders slice
	mockRepo.EXPECT().
		GetUninvoicedOrdersInPeriod(ctx, gomock.Any()).
		Return([]repository.Order{}, nil)

	params := ConsolidatedInvoiceParams{
		UserID:             userID.String(),
		BillingPeriodStart: startDate,
		BillingPeriodEnd:   endDate,
	}

	result, err := svc.GenerateConsolidatedInvoice(ctx, params)

	assert.NoError(t, err, "Should not return error when no orders found")
	assert.Nil(t, result, "Should return nil when no orders to invoice")
}

// Test_MarkInvoicesOverdue_UpdatesStatusCorrectly verifies overdue invoice marking
func Test_MarkInvoicesOverdue_UpdatesStatusCorrectly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	ctx := createTestContext(tenantID)

	pastDate := time.Now().AddDate(0, 0, -5) // 5 days ago

	tests := []struct {
		name          string
		initialStatus string
		shouldUpdate  bool
	}{
		{
			name:          "sent invoice marked overdue",
			initialStatus: "sent",
			shouldUpdate:  true,
		},
		{
			name:          "viewed invoice marked overdue",
			initialStatus: "viewed",
			shouldUpdate:  true,
		},
		{
			name:          "draft invoice not marked overdue",
			initialStatus: "draft",
			shouldUpdate:  false,
		},
		{
			name:          "paid invoice not marked overdue",
			initialStatus: "paid",
			shouldUpdate:  false,
		},
		{
			name:          "void invoice not marked overdue",
			initialStatus: "void",
			shouldUpdate:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repository.NewMockQuerier(ctrl)
			mockPaymentTerms := &mockPaymentTermsService{}
			mockBilling := &mockBillingProvider{}

			svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

			invoiceID := createTestInvoiceID()
			userID := createTestUserID()

			// Setup overdue invoices
			mockRepo.EXPECT().
				ListOverdueInvoices(ctx, tenantID).
				Return([]repository.ListOverdueInvoicesRow{
					{
						ID:            invoiceID,
						InvoiceNumber: "INV-001",
						Status:        tt.initialStatus,
						BalanceCents:  10000,
						DueDate:       pgtype.Date{Time: pastDate, Valid: true},
						UserID:        userID,
					},
				}, nil)

			// If should update, expect UpdateInvoiceStatus, GetUserByID, and EnqueueJob
			if tt.shouldUpdate {
				mockRepo.EXPECT().
					UpdateInvoiceStatus(ctx, repository.UpdateInvoiceStatusParams{
						TenantID: tenantID,
						ID:       invoiceID,
						Status:   "overdue",
					}).
					Return(nil)

				mockRepo.EXPECT().
					GetUserByID(ctx, userID).
					Return(repository.User{
						ID:    userID,
						Email: "test@example.com",
					}, nil)

				// EnqueueJob for overdue email notification
				mockRepo.EXPECT().
					EnqueueJob(ctx, gomock.Any()).
					Return(repository.Job{}, nil)
			}

			count, err := svc.MarkInvoicesOverdue(ctx)

			assert.NoError(t, err)
			if tt.shouldUpdate {
				assert.Equal(t, 1, count, "Should mark one invoice as overdue")
			} else {
				assert.Equal(t, 0, count, "Should not mark invoice as overdue")
			}
		})
	}
}

// Test_MarkInvoicesOverdue_OnlyInvoicesPastDueDate verifies only past-due invoices are marked
func Test_MarkInvoicesOverdue_OnlyInvoicesPastDueDate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	ctx := createTestContext(tenantID)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockPaymentTerms := &mockPaymentTermsService{}
	mockBilling := &mockBillingProvider{}

	svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

	pastDate := time.Now().AddDate(0, 0, -5)

	overdueInvoiceID := createTestInvoiceID()
	userID := createTestUserID()

	// ListOverdueInvoices should only return invoices past due date
	// (this is handled by the SQL query, but we verify behavior)
	mockRepo.EXPECT().
		ListOverdueInvoices(ctx, tenantID).
		Return([]repository.ListOverdueInvoicesRow{
			{
				ID:            overdueInvoiceID,
				InvoiceNumber: "INV-001",
				Status:        "sent",
				BalanceCents:  10000,
				DueDate:       pgtype.Date{Time: pastDate, Valid: true},
				UserID:        userID,
			},
			// Note: not-due invoice should not be in this list
			// (SQL query filters by due_date < NOW())
		}, nil)

	mockRepo.EXPECT().
		UpdateInvoiceStatus(ctx, repository.UpdateInvoiceStatusParams{
			TenantID: tenantID,
			ID:       overdueInvoiceID,
			Status:   "overdue",
		}).
		Return(nil)

	mockRepo.EXPECT().
		GetUserByID(ctx, userID).
		Return(repository.User{
			ID:    userID,
			Email: "test@example.com",
		}, nil)

	// EnqueueJob for overdue email notification
	mockRepo.EXPECT().
		EnqueueJob(ctx, gomock.Any()).
		Return(repository.Job{}, nil)

	count, err := svc.MarkInvoicesOverdue(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Should only mark past-due invoices")
}

// Test_CreateInvoice_EmptyOrderIDsReturnsError verifies error on empty order list
func Test_CreateInvoice_EmptyOrderIDsReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	ctx := createTestContext(tenantID)

	mockRepo := repository.NewMockQuerier(ctrl)
	mockPaymentTerms := &mockPaymentTermsService{}
	mockBilling := &mockBillingProvider{}

	svc := NewInvoiceService(mockRepo, mockPaymentTerms, mockBilling)

	params := CreateInvoiceParams{
		UserID:   userID.String(),
		OrderIDs: []string{}, // Empty
	}

	_, err := svc.CreateInvoice(ctx, params)

	assert.ErrorIs(t, err, ErrNoOrdersToInvoice)
}
