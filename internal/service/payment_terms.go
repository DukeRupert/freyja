package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// PaymentTermsService manages reusable payment terms for wholesale invoicing.
// Payment terms define when invoices are due (Net 15, Net 30, etc.).
type PaymentTermsService interface {
	// CreatePaymentTerms creates a new payment terms record.
	CreatePaymentTerms(ctx context.Context, params CreatePaymentTermsParams) (*repository.PaymentTerm, error)

	// GetPaymentTerms retrieves payment terms by ID.
	GetPaymentTerms(ctx context.Context, paymentTermsID string) (*repository.PaymentTerm, error)

	// GetPaymentTermsByCode retrieves payment terms by code (e.g., "net_30").
	GetPaymentTermsByCode(ctx context.Context, code string) (*repository.PaymentTerm, error)

	// GetDefaultPaymentTerms retrieves the default payment terms for the tenant.
	GetDefaultPaymentTerms(ctx context.Context) (*repository.PaymentTerm, error)

	// ListPaymentTerms lists all active payment terms for the tenant.
	ListPaymentTerms(ctx context.Context) ([]repository.PaymentTerm, error)

	// UpdatePaymentTerms updates an existing payment terms record.
	UpdatePaymentTerms(ctx context.Context, params UpdatePaymentTermsParams) error

	// SetDefaultPaymentTerms sets a payment terms record as the tenant default.
	SetDefaultPaymentTerms(ctx context.Context, paymentTermsID string) error

	// DeletePaymentTerms soft-deletes payment terms (deactivates).
	// Returns ErrPaymentTermsInUse if any customers reference it.
	DeletePaymentTerms(ctx context.Context, paymentTermsID string) error

	// CalculateDueDate calculates invoice due date from payment terms.
	// invoiceDate + payment_terms.days = due_date
	CalculateDueDate(ctx context.Context, paymentTermsID string, invoiceDate time.Time) (time.Time, error)

	// CalculateDueDateFromTerms calculates due date directly from a PaymentTerm.
	// Useful when you already have the payment terms loaded.
	CalculateDueDateFromTerms(terms *repository.PaymentTerm, invoiceDate time.Time) time.Time
}

// CreatePaymentTermsParams contains parameters for creating payment terms.
type CreatePaymentTermsParams struct {
	Name        string
	Code        string
	Days        int32
	IsDefault   bool
	SortOrder   int32
	Description string
}

// UpdatePaymentTermsParams contains parameters for updating payment terms.
type UpdatePaymentTermsParams struct {
	PaymentTermsID string
	Name           string
	Code           string
	Days           int32
	IsDefault      bool
	IsActive       bool
	SortOrder      int32
	Description    string
}

type paymentTermsService struct {
	repo     repository.Querier
	tenantID pgtype.UUID
}

// NewPaymentTermsService creates a new PaymentTermsService instance.
func NewPaymentTermsService(repo repository.Querier, tenantID string) (PaymentTermsService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	return &paymentTermsService{
		repo:     repo,
		tenantID: tenantUUID,
	}, nil
}

// CreatePaymentTerms creates a new payment terms record.
func (s *paymentTermsService) CreatePaymentTerms(ctx context.Context, params CreatePaymentTermsParams) (*repository.PaymentTerm, error) {
	// Check for duplicate code
	existing, err := s.repo.GetPaymentTermsByCode(ctx, repository.GetPaymentTermsByCodeParams{
		TenantID: s.tenantID,
		Code:     params.Code,
	})
	if err == nil && existing.ID.Valid {
		return nil, ErrDuplicatePaymentTermsCode
	}

	description := pgtype.Text{}
	if params.Description != "" {
		description.String = params.Description
		description.Valid = true
	}

	pt, err := s.repo.CreatePaymentTerms(ctx, repository.CreatePaymentTermsParams{
		TenantID:    s.tenantID,
		Name:        params.Name,
		Code:        params.Code,
		Days:        params.Days,
		IsDefault:   params.IsDefault,
		IsActive:    true,
		SortOrder:   params.SortOrder,
		Description: description,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment terms: %w", err)
	}

	return &pt, nil
}

// GetPaymentTerms retrieves payment terms by ID.
func (s *paymentTermsService) GetPaymentTerms(ctx context.Context, paymentTermsID string) (*repository.PaymentTerm, error) {
	var ptID pgtype.UUID
	if err := ptID.Scan(paymentTermsID); err != nil {
		return nil, fmt.Errorf("invalid payment terms ID: %w", err)
	}

	pt, err := s.repo.GetPaymentTermsByID(ctx, repository.GetPaymentTermsByIDParams{
		ID:       ptID,
		TenantID: s.tenantID,
	})
	if err != nil {
		return nil, ErrPaymentTermsNotFound
	}

	return &pt, nil
}

// GetPaymentTermsByCode retrieves payment terms by code.
func (s *paymentTermsService) GetPaymentTermsByCode(ctx context.Context, code string) (*repository.PaymentTerm, error) {
	pt, err := s.repo.GetPaymentTermsByCode(ctx, repository.GetPaymentTermsByCodeParams{
		TenantID: s.tenantID,
		Code:     code,
	})
	if err != nil {
		return nil, ErrPaymentTermsNotFound
	}

	return &pt, nil
}

// GetDefaultPaymentTerms retrieves the default payment terms for the tenant.
func (s *paymentTermsService) GetDefaultPaymentTerms(ctx context.Context) (*repository.PaymentTerm, error) {
	pt, err := s.repo.GetDefaultPaymentTerms(ctx, s.tenantID)
	if err != nil {
		return nil, ErrPaymentTermsNotFound
	}

	return &pt, nil
}

// ListPaymentTerms lists all active payment terms for the tenant.
func (s *paymentTermsService) ListPaymentTerms(ctx context.Context) ([]repository.PaymentTerm, error) {
	terms, err := s.repo.ListPaymentTerms(ctx, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment terms: %w", err)
	}

	return terms, nil
}

// UpdatePaymentTerms updates an existing payment terms record.
func (s *paymentTermsService) UpdatePaymentTerms(ctx context.Context, params UpdatePaymentTermsParams) error {
	var ptID pgtype.UUID
	if err := ptID.Scan(params.PaymentTermsID); err != nil {
		return fmt.Errorf("invalid payment terms ID: %w", err)
	}

	// Check if payment terms exists
	_, err := s.repo.GetPaymentTermsByID(ctx, repository.GetPaymentTermsByIDParams{
		ID:       ptID,
		TenantID: s.tenantID,
	})
	if err != nil {
		return ErrPaymentTermsNotFound
	}

	description := pgtype.Text{}
	if params.Description != "" {
		description.String = params.Description
		description.Valid = true
	}

	err = s.repo.UpdatePaymentTerms(ctx, repository.UpdatePaymentTermsParams{
		TenantID:    s.tenantID,
		ID:          ptID,
		Name:        params.Name,
		Code:        params.Code,
		Days:        params.Days,
		IsDefault:   params.IsDefault,
		IsActive:    params.IsActive,
		SortOrder:   params.SortOrder,
		Description: description,
	})
	if err != nil {
		return fmt.Errorf("failed to update payment terms: %w", err)
	}

	return nil
}

// SetDefaultPaymentTerms sets a payment terms record as the tenant default.
func (s *paymentTermsService) SetDefaultPaymentTerms(ctx context.Context, paymentTermsID string) error {
	var ptID pgtype.UUID
	if err := ptID.Scan(paymentTermsID); err != nil {
		return fmt.Errorf("invalid payment terms ID: %w", err)
	}

	// Verify it exists
	_, err := s.repo.GetPaymentTermsByID(ctx, repository.GetPaymentTermsByIDParams{
		ID:       ptID,
		TenantID: s.tenantID,
	})
	if err != nil {
		return ErrPaymentTermsNotFound
	}

	err = s.repo.SetDefaultPaymentTerms(ctx, repository.SetDefaultPaymentTermsParams{
		TenantID: s.tenantID,
		ID:       ptID,
	})
	if err != nil {
		return fmt.Errorf("failed to set default payment terms: %w", err)
	}

	return nil
}

// DeletePaymentTerms soft-deletes payment terms by deactivating.
func (s *paymentTermsService) DeletePaymentTerms(ctx context.Context, paymentTermsID string) error {
	var ptID pgtype.UUID
	if err := ptID.Scan(paymentTermsID); err != nil {
		return fmt.Errorf("invalid payment terms ID: %w", err)
	}

	// Check if any users are using this payment terms
	count, err := s.repo.CountUsersWithPaymentTerms(ctx, ptID)
	if err != nil {
		return fmt.Errorf("failed to check payment terms usage: %w", err)
	}

	if count > 0 {
		return ErrPaymentTermsInUse
	}

	err = s.repo.DeletePaymentTerms(ctx, repository.DeletePaymentTermsParams{
		TenantID: s.tenantID,
		ID:       ptID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete payment terms: %w", err)
	}

	return nil
}

// CalculateDueDate calculates invoice due date from payment terms.
func (s *paymentTermsService) CalculateDueDate(ctx context.Context, paymentTermsID string, invoiceDate time.Time) (time.Time, error) {
	pt, err := s.GetPaymentTerms(ctx, paymentTermsID)
	if err != nil {
		return time.Time{}, err
	}

	return s.CalculateDueDateFromTerms(pt, invoiceDate), nil
}

// CalculateDueDateFromTerms calculates due date directly from a PaymentTerm.
func (s *paymentTermsService) CalculateDueDateFromTerms(terms *repository.PaymentTerm, invoiceDate time.Time) time.Time {
	return invoiceDate.AddDate(0, 0, int(terms.Days))
}
