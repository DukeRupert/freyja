package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/auth"
	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserService implements domain.UserService using PostgreSQL.
type UserService struct {
	repo     repository.Querier
	tenantID pgtype.UUID
}

// Compile-time check to ensure UserService implements domain.UserService.
var _ domain.UserService = (*UserService)(nil)

// NewUserService creates a new UserService instance.
func NewUserService(repo repository.Querier, tenantID string) (*UserService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	return &UserService{
		repo:     repo,
		tenantID: tenantUUID,
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// generateSessionID generates a cryptographically secure session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// mapRepoUserToDomain converts a repository User to a domain Customer.
func mapRepoUserToDomain(u repository.User) *domain.Customer {
	return &domain.Customer{
		ID:                         u.ID,
		TenantID:                   u.TenantID,
		Email:                      u.Email,
		PasswordHash:               u.PasswordHash,
		EmailVerified:              u.EmailVerified,
		AccountType:                domain.UserAccountType(u.AccountType),
		FirstName:                  u.FirstName,
		LastName:                   u.LastName,
		Phone:                      u.Phone,
		CompanyName:                u.CompanyName,
		TaxID:                      u.TaxID,
		BusinessType:               u.BusinessType,
		Status:                     domain.UserStatus(u.Status),
		WholesaleApplicationStatus: u.WholesaleApplicationStatus,
		WholesaleApplicationNotes:  u.WholesaleApplicationNotes,
		WholesaleApprovedAt:        u.WholesaleApprovedAt,
		WholesaleApprovedBy:        u.WholesaleApprovedBy,
		PaymentTerms:               u.PaymentTerms,
		Metadata:                   u.Metadata,
		CreatedAt:                  u.CreatedAt,
		UpdatedAt:                  u.UpdatedAt,
		InternalNote:               u.InternalNote,
		MinimumSpendCents:          u.MinimumSpendCents,
		EmailOrders:                u.EmailOrders,
		EmailDispatches:            u.EmailDispatches,
		EmailInvoices:              u.EmailInvoices,
		PaymentTermsID:             u.PaymentTermsID,
		BillingCycle:               u.BillingCycle,
		BillingCycleDay:            u.BillingCycleDay,
		CustomerReference:          u.CustomerReference,
	}
}

// mapRepoUserToListItem converts a repository User to a UserListItem.
func mapRepoUserToListItem(u repository.User) domain.UserListItem {
	fullName := ""
	if u.FirstName.Valid && u.LastName.Valid {
		fullName = u.FirstName.String + " " + u.LastName.String
	} else if u.FirstName.Valid {
		fullName = u.FirstName.String
	} else if u.LastName.Valid {
		fullName = u.LastName.String
	}

	return domain.UserListItem{
		ID:              u.ID,
		Email:           u.Email,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		FullName:        fullName,
		AccountType:     domain.UserAccountType(u.AccountType),
		Status:          domain.UserStatus(u.Status),
		CreatedAt:       u.CreatedAt,
		CompanyName:     u.CompanyName,
		WholesaleStatus: u.WholesaleApplicationStatus,
	}
}

// =============================================================================
// Authentication Operations
// =============================================================================

// Register creates a new user account.
func (s *UserService) Register(ctx context.Context, email, password, firstName, lastName string) (*domain.Customer, error) {
	// Validate email (basic check)
	if email == "" || len(email) < 3 {
		return nil, domain.ErrInvalidEmail
	}

	// Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, repository.GetUserByEmailParams{
		TenantID: s.tenantID,
		Email:    email,
	})
	if err == nil && existingUser.ID.Valid {
		return nil, domain.ErrUserExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Hash password
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	var firstNameText, lastNameText, passwordHashText pgtype.Text
	if firstName != "" {
		firstNameText = pgtype.Text{String: firstName, Valid: true}
	}
	if lastName != "" {
		lastNameText = pgtype.Text{String: lastName, Valid: true}
	}
	passwordHashText = pgtype.Text{String: passwordHash, Valid: true}

	user, err := s.repo.CreateUser(ctx, repository.CreateUserParams{
		TenantID:     s.tenantID,
		Email:        email,
		PasswordHash: passwordHashText,
		FirstName:    firstNameText,
		LastName:     lastNameText,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return mapRepoUserToDomain(user), nil
}

// Authenticate verifies email/password and returns the user if valid.
func (s *UserService) Authenticate(ctx context.Context, email, password string) (*domain.Customer, error) {
	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, repository.GetUserByEmailParams{
		TenantID: s.tenantID,
		Email:    email,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check account status
	switch user.Status {
	case "suspended":
		return nil, domain.ErrAccountSuspended
	case "pending":
		return nil, domain.ErrAccountPending
	case "closed":
		return nil, domain.ErrUserNotFound
	}

	// Verify password
	if !user.PasswordHash.Valid {
		return nil, domain.ErrInvalidPassword
	}

	if err := auth.VerifyPassword(password, user.PasswordHash.String); err != nil {
		if errors.Is(err, auth.ErrPasswordMismatch) {
			return nil, domain.ErrInvalidPassword
		}
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	// Check if email is verified
	if !user.EmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	return mapRepoUserToDomain(user), nil
}

// CreateSession creates a new session for a user.
func (s *UserService) CreateSession(ctx context.Context, userID string) (string, error) {
	// Generate session token
	token, err := generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}

	// Store user ID in session data
	sessionData := domain.SessionData{UserID: userID}
	dataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Create session (expires in 30 days)
	expiresAt := pgtype.Timestamptz{}
	expiresAt.Scan(time.Now().Add(30 * 24 * time.Hour))

	_, err = s.repo.CreateSession(ctx, repository.CreateSessionParams{
		Token:     token,
		Data:      dataJSON,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

// GetUserBySessionToken retrieves a user from a session token.
func (s *UserService) GetUserBySessionToken(ctx context.Context, token string) (*domain.Customer, error) {
	// Get session
	session, err := s.repo.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check session expiration
	if session.ExpiresAt.Valid && session.ExpiresAt.Time.Before(time.Now()) {
		return nil, domain.ErrSessionExpired
	}

	// Parse session data
	var sessionData domain.SessionData
	if err := json.Unmarshal(session.Data, &sessionData); err != nil {
		return nil, fmt.Errorf("failed to parse session data: %w", err)
	}

	// Get user
	var userUUID pgtype.UUID
	if err := userUUID.Scan(sessionData.UserID); err != nil {
		return nil, fmt.Errorf("invalid user ID in session: %w", err)
	}

	// Use tenant-scoped query to prevent cross-tenant access
	user, err := s.repo.GetUserByIDAndTenant(ctx, repository.GetUserByIDAndTenantParams{
		ID:       userUUID,
		TenantID: s.tenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return mapRepoUserToDomain(user), nil
}

// DeleteSession logs out a user by deleting their session.
func (s *UserService) DeleteSession(ctx context.Context, token string) error {
	if err := s.repo.DeleteSession(ctx, token); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// =============================================================================
// User Retrieval Operations
// =============================================================================

// GetUserByID retrieves a user by ID.
func (s *UserService) GetUserByID(ctx context.Context, userID string) (*domain.Customer, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := s.repo.GetUserByIDAndTenant(ctx, repository.GetUserByIDAndTenantParams{
		ID:       userUUID,
		TenantID: s.tenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return mapRepoUserToDomain(user), nil
}

// GetUserByEmail retrieves a user by email.
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	user, err := s.repo.GetUserByEmail(ctx, repository.GetUserByEmailParams{
		TenantID: s.tenantID,
		Email:    email,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return mapRepoUserToDomain(user), nil
}

// ListUsers returns all users with pagination.
func (s *UserService) ListUsers(ctx context.Context, limit, offset int32) ([]domain.UserListItem, error) {
	users, err := s.repo.ListUsers(ctx, repository.ListUsersParams{
		TenantID: s.tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	result := make([]domain.UserListItem, len(users))
	for i, u := range users {
		result[i] = mapRepoUserToListItem(u)
	}

	return result, nil
}

// ListUsersByAccountType returns users filtered by account type.
func (s *UserService) ListUsersByAccountType(ctx context.Context, accountType domain.UserAccountType) ([]domain.UserListItem, error) {
	users, err := s.repo.ListUsersByAccountType(ctx, repository.ListUsersByAccountTypeParams{
		TenantID:    s.tenantID,
		AccountType: string(accountType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users by account type: %w", err)
	}

	result := make([]domain.UserListItem, len(users))
	for i, u := range users {
		result[i] = mapRepoUserToListItem(u)
	}

	return result, nil
}

// CountUsers returns the total count of users.
func (s *UserService) CountUsers(ctx context.Context) (int64, error) {
	count, err := s.repo.CountUsers(ctx, s.tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// =============================================================================
// User Update Operations
// =============================================================================

// UpdateUserProfile updates basic user profile information.
func (s *UserService) UpdateUserProfile(ctx context.Context, userID string, params domain.UpdateUserProfileParams) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	repoParams := repository.UpdateUserProfileParams{
		ID:       userUUID,
		TenantID: s.tenantID,
	}

	if params.FirstName != nil {
		repoParams.FirstName = pgtype.Text{String: *params.FirstName, Valid: true}
	}
	if params.LastName != nil {
		repoParams.LastName = pgtype.Text{String: *params.LastName, Valid: true}
	}
	if params.Phone != nil {
		repoParams.Phone = pgtype.Text{String: *params.Phone, Valid: true}
	}

	if err := s.repo.UpdateUserProfile(ctx, repoParams); err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	return nil
}

// UpdateUserPassword updates a user's password.
func (s *UserService) UpdateUserPassword(ctx context.Context, userID, newPassword string) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		ID:           userUUID,
		TenantID:     s.tenantID,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// UpdateUserStatus updates a user's status.
func (s *UserService) UpdateUserStatus(ctx context.Context, userID string, status domain.UserStatus) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if err := s.repo.UpdateUserStatus(ctx, repository.UpdateUserStatusParams{
		ID:     userUUID,
		Status: string(status),
	}); err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	return nil
}

// AdminUpdateCustomer updates customer details (admin only).
func (s *UserService) AdminUpdateCustomer(ctx context.Context, userID string, params domain.AdminUpdateCustomerParams) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if err := s.repo.AdminUpdateCustomer(ctx, repository.AdminUpdateCustomerParams{
		ID:           userUUID,
		TenantID:     s.tenantID,
		FirstName:    params.FirstName,
		LastName:     params.LastName,
		Phone:        params.Phone,
		CompanyName:  params.CompanyName,
		BusinessType: params.BusinessType,
		TaxID:        params.TaxID,
		Status:       params.Status,
		InternalNote: params.InternalNote,
	}); err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	return nil
}

// VerifyUserEmail marks a user's email as verified.
func (s *UserService) VerifyUserEmail(ctx context.Context, userID string) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if err := s.repo.VerifyUserEmail(ctx, userUUID); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	return nil
}

// =============================================================================
// Wholesale Operations
// =============================================================================

// SubmitWholesaleApplication submits a wholesale application.
func (s *UserService) SubmitWholesaleApplication(ctx context.Context, userID string, params domain.WholesaleApplicationParams) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if err := s.repo.SubmitWholesaleApplication(ctx, repository.SubmitWholesaleApplicationParams{
		ID:           userUUID,
		CompanyName:  pgtype.Text{String: params.CompanyName, Valid: params.CompanyName != ""},
		BusinessType: pgtype.Text{String: params.BusinessType, Valid: params.BusinessType != ""},
		TaxID:        pgtype.Text{String: params.TaxID, Valid: params.TaxID != ""},
		WholesaleApplicationNotes: pgtype.Text{String: params.Notes, Valid: params.Notes != ""},
	}); err != nil {
		return fmt.Errorf("failed to submit wholesale application: %w", err)
	}

	return nil
}

// UpdateWholesaleApplication approves or rejects a wholesale application.
func (s *UserService) UpdateWholesaleApplication(ctx context.Context, userID string, params domain.UpdateWholesaleApplicationParams) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if err := s.repo.UpdateWholesaleApplication(ctx, repository.UpdateWholesaleApplicationParams{
		ID:                         userUUID,
		WholesaleApplicationStatus: pgtype.Text{String: string(params.Status), Valid: true},
		WholesaleApplicationNotes:  pgtype.Text{String: params.Notes, Valid: params.Notes != ""},
		WholesaleApprovedBy:        params.ApprovedBy,
		PaymentTerms:               pgtype.Text{String: params.PaymentTerms, Valid: params.PaymentTerms != ""},
	}); err != nil {
		return fmt.Errorf("failed to update wholesale application: %w", err)
	}

	return nil
}

// UpdateWholesaleCustomer updates wholesale-specific customer settings.
func (s *UserService) UpdateWholesaleCustomer(ctx context.Context, userID string, params domain.UpdateWholesaleCustomerParams) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if err := s.repo.UpdateWholesaleCustomer(ctx, repository.UpdateWholesaleCustomerParams{
		ID:                userUUID,
		CompanyName:       pgtype.Text{String: params.CompanyName, Valid: params.CompanyName != ""},
		PaymentTermsID:    params.PaymentTermsID,
		BillingCycle:      pgtype.Text{String: string(params.BillingCycle), Valid: params.BillingCycle != ""},
		BillingCycleDay:   pgtype.Int4{Int32: params.BillingCycleDay, Valid: params.BillingCycleDay > 0},
		MinimumSpendCents: pgtype.Int4{Int32: params.MinimumSpendCents, Valid: params.MinimumSpendCents > 0},
		CustomerReference: pgtype.Text{String: params.CustomerReference, Valid: params.CustomerReference != ""},
		InternalNote:      pgtype.Text{String: params.InternalNote, Valid: params.InternalNote != ""},
		EmailOrders:       pgtype.Text{String: params.EmailOrders, Valid: params.EmailOrders != ""},
		EmailDispatches:   pgtype.Text{String: params.EmailDispatches, Valid: params.EmailDispatches != ""},
		EmailInvoices:     pgtype.Text{String: params.EmailInvoices, Valid: params.EmailInvoices != ""},
	}); err != nil {
		return fmt.Errorf("failed to update wholesale customer: %w", err)
	}

	return nil
}

// GetCustomersForBillingCycle returns customers due for billing.
func (s *UserService) GetCustomersForBillingCycle(ctx context.Context, billingCycle domain.BillingCycle, day int32) ([]domain.Customer, error) {
	rows, err := s.repo.GetCustomersForBillingCycle(ctx, repository.GetCustomersForBillingCycleParams{
		TenantID:     s.tenantID,
		BillingCycle: pgtype.Text{String: string(billingCycle), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get customers for billing cycle: %w", err)
	}

	// Filter by day if applicable (the query may not filter by day)
	result := make([]domain.Customer, 0, len(rows))
	for _, row := range rows {
		// If day filtering is needed, check it here
		if day > 0 && row.BillingCycleDay.Valid && row.BillingCycleDay.Int32 != day {
			continue
		}
		result = append(result, domain.Customer{
			ID:              row.ID,
			TenantID:        row.TenantID,
			Email:           row.Email,
			CompanyName:     row.CompanyName,
			PaymentTermsID:  row.PaymentTermsID,
			BillingCycle:    row.BillingCycle,
			BillingCycleDay: row.BillingCycleDay,
		})
	}

	return result, nil
}

// =============================================================================
// Account Operations (Addresses & Payment Methods)
// =============================================================================

// ListAddresses returns all saved addresses for a user.
func (s *UserService) ListAddresses(ctx context.Context, userID string) ([]domain.UserAddress, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	rows, err := s.repo.ListAddressesForUser(ctx, repository.ListAddressesForUserParams{
		TenantID: s.tenantID,
		UserID:   userUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list addresses: %w", err)
	}

	addresses := make([]domain.UserAddress, len(rows))
	for i, row := range rows {
		addresses[i] = domain.UserAddress{
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
func (s *UserService) ListPaymentMethods(ctx context.Context, userID string) ([]domain.UserPaymentMethod, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	rows, err := s.repo.ListPaymentMethodsForUser(ctx, repository.ListPaymentMethodsForUserParams{
		TenantID: s.tenantID,
		UserID:   userUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}

	methods := make([]domain.UserPaymentMethod, len(rows))
	for i, row := range rows {
		methods[i] = domain.UserPaymentMethod{
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
func (s *UserService) GetAccountSummary(ctx context.Context, userID string) (domain.AccountSummary, error) {
	var summary domain.AccountSummary

	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return summary, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get address counts
	addressCounts, err := s.repo.CountAddressesForUser(ctx, repository.CountAddressesForUserParams{
		TenantID: s.tenantID,
		UserID:   userUUID,
	})
	if err != nil {
		return summary, fmt.Errorf("failed to count addresses: %w", err)
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
		TenantID: s.tenantID,
		UserID:   userUUID,
	})
	if err != nil {
		return summary, fmt.Errorf("failed to count payment methods: %w", err)
	}
	summary.PaymentMethodCount = int(paymentCounts.PaymentMethodCount)
	if v, ok := paymentCounts.HasDefaultPayment.(bool); ok {
		summary.HasDefaultPayment = v
	}

	// Get order count
	orderCount, err := s.repo.CountOrdersForUser(ctx, repository.CountOrdersForUserParams{
		TenantID: s.tenantID,
		UserID:   userUUID,
	})
	if err != nil {
		return summary, fmt.Errorf("failed to count orders: %w", err)
	}
	summary.OrderCount = int(orderCount)

	return summary, nil
}
