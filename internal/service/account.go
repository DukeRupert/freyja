package service

import (
	"context"

	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// AccountService provides business logic for customer account operations.
type AccountService interface {
	// ListAddresses returns all saved addresses for a user.
	ListAddresses(ctx context.Context, tenantID, userID pgtype.UUID) ([]UserAddress, error)

	// ListPaymentMethods returns all saved payment methods for a user.
	ListPaymentMethods(ctx context.Context, tenantID, userID pgtype.UUID) ([]UserPaymentMethod, error)

	// GetAccountSummary returns aggregate counts for the account dashboard.
	GetAccountSummary(ctx context.Context, tenantID, userID pgtype.UUID) (AccountSummary, error)
}

// AccountSummary contains aggregate counts for the account dashboard.
type AccountSummary struct {
	AddressCount       int
	PaymentMethodCount int
	OrderCount         int
	HasDefaultShipping bool
	HasDefaultBilling  bool
	HasDefaultPayment  bool
}

// UserAddress represents a user's saved address with metadata.
type UserAddress struct {
	ID                pgtype.UUID
	FullName          string
	Company           string
	AddressLine1      string
	AddressLine2      string
	City              string
	State             string
	PostalCode        string
	Country           string
	Phone             string
	IsDefaultShipping bool
	IsDefaultBilling  bool
	Label             string
}

// UserPaymentMethod represents a user's saved payment method.
type UserPaymentMethod struct {
	ID              pgtype.UUID
	MethodType      string
	DisplayBrand    string
	DisplayLast4    string
	DisplayExpMonth int32
	DisplayExpYear  int32
	IsDefault       bool
}

// accountServiceImpl implements AccountService.
type accountServiceImpl struct {
	repo *repository.Queries
}

// NewAccountService creates a new account service.
func NewAccountService(repo *repository.Queries) AccountService {
	return &accountServiceImpl{repo: repo}
}

// ListAddresses returns all saved addresses for a user.
func (s *accountServiceImpl) ListAddresses(ctx context.Context, tenantID, userID pgtype.UUID) ([]UserAddress, error) {
	rows, err := s.repo.ListAddressesForUser(ctx, repository.ListAddressesForUserParams{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		return nil, err
	}

	addresses := make([]UserAddress, len(rows))
	for i, row := range rows {
		addresses[i] = UserAddress{
			ID:                row.ID,
			FullName:          row.FullName.String,
			Company:           row.Company.String,
			AddressLine1:      row.AddressLine1,
			AddressLine2:      row.AddressLine2.String,
			City:              row.City,
			State:             row.State,
			PostalCode:        row.PostalCode,
			Country:           row.Country,
			Phone:             row.Phone.String,
			IsDefaultShipping: row.IsDefaultShipping,
			IsDefaultBilling:  row.IsDefaultBilling,
			Label:             row.Label.String,
		}
	}

	return addresses, nil
}

// ListPaymentMethods returns all saved payment methods for a user.
func (s *accountServiceImpl) ListPaymentMethods(ctx context.Context, tenantID, userID pgtype.UUID) ([]UserPaymentMethod, error) {
	rows, err := s.repo.ListPaymentMethodsForUser(ctx, repository.ListPaymentMethodsForUserParams{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		return nil, err
	}

	methods := make([]UserPaymentMethod, len(rows))
	for i, row := range rows {
		methods[i] = UserPaymentMethod{
			ID:              row.ID,
			MethodType:      row.MethodType,
			DisplayBrand:    row.DisplayBrand.String,
			DisplayLast4:    row.DisplayLast4.String,
			DisplayExpMonth: row.DisplayExpMonth.Int32,
			DisplayExpYear:  row.DisplayExpYear.Int32,
			IsDefault:       row.IsDefault,
		}
	}

	return methods, nil
}

// GetAccountSummary returns aggregate counts for the account dashboard.
func (s *accountServiceImpl) GetAccountSummary(ctx context.Context, tenantID, userID pgtype.UUID) (AccountSummary, error) {
	var summary AccountSummary

	// Get address counts
	addressCounts, err := s.repo.CountAddressesForUser(ctx, repository.CountAddressesForUserParams{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		return summary, err
	}
	summary.AddressCount = int(addressCounts.AddressCount)
	if v, ok := addressCounts.HasDefaultShipping.(bool); ok {
		summary.HasDefaultShipping = v
	}
	if v, ok := addressCounts.HasDefaultBilling.(bool); ok {
		summary.HasDefaultBilling = v
	}

	// Get payment method counts
	paymentCounts, err := s.repo.CountPaymentMethodsForUser(ctx, repository.CountPaymentMethodsForUserParams{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		return summary, err
	}
	summary.PaymentMethodCount = int(paymentCounts.PaymentMethodCount)
	if v, ok := paymentCounts.HasDefaultPayment.(bool); ok {
		summary.HasDefaultPayment = v
	}

	// Get order count
	orderCount, err := s.repo.CountOrdersForUser(ctx, repository.CountOrdersForUserParams{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		return summary, err
	}
	summary.OrderCount = int(orderCount)

	return summary, nil
}
