package domain

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// USER/CUSTOMER DOMAIN TYPES
// =============================================================================

// UserAccountType represents the type of user account.
type UserAccountType string

const (
	UserAccountTypeRetail    UserAccountType = "retail"
	UserAccountTypeWholesale UserAccountType = "wholesale"
	UserAccountTypeAdmin     UserAccountType = "admin"
)

// UserStatus represents the status of a user account.
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusClosed    UserStatus = "closed"
	UserStatusPending   UserStatus = "pending"
)

// WholesaleApplicationStatus represents the status of a wholesale application.
type WholesaleApplicationStatus string

const (
	WholesaleStatusPending  WholesaleApplicationStatus = "pending"
	WholesaleStatusApproved WholesaleApplicationStatus = "approved"
	WholesaleStatusRejected WholesaleApplicationStatus = "rejected"
)

// BillingCycle represents how often a wholesale customer is billed.
type BillingCycle string

const (
	BillingCycleWeekly   BillingCycle = "weekly"
	BillingCycleBiweekly BillingCycle = "biweekly"
	BillingCycleMonthly  BillingCycle = "monthly"
	BillingCycleOnOrder  BillingCycle = "on_order"
)

// Customer represents a full customer/user record in the system.
// This is distinct from domain.User which is a minimal context type.
type Customer struct {
	ID                         pgtype.UUID
	TenantID                   pgtype.UUID
	Email                      string
	PasswordHash               pgtype.Text
	EmailVerified              bool
	AccountType                UserAccountType
	FirstName                  pgtype.Text
	LastName                   pgtype.Text
	Phone                      pgtype.Text
	CompanyName                pgtype.Text
	TaxID                      pgtype.Text
	BusinessType               pgtype.Text
	Status                     UserStatus
	WholesaleApplicationStatus pgtype.Text
	WholesaleApplicationNotes  pgtype.Text
	WholesaleApprovedAt        pgtype.Timestamptz
	WholesaleApprovedBy        pgtype.UUID
	PaymentTerms               pgtype.Text // Legacy field
	Metadata                   []byte
	CreatedAt                  pgtype.Timestamptz
	UpdatedAt                  pgtype.Timestamptz
	InternalNote               pgtype.Text
	MinimumSpendCents          pgtype.Int4
	EmailOrders                pgtype.Text
	EmailDispatches            pgtype.Text
	EmailInvoices              pgtype.Text
	PaymentTermsID             pgtype.UUID
	BillingCycle               pgtype.Text
	BillingCycleDay            pgtype.Int4
	CustomerReference          pgtype.Text
}

// FullName returns the customer's full name.
func (c *Customer) FullName() string {
	if c.FirstName.Valid && c.LastName.Valid {
		return c.FirstName.String + " " + c.LastName.String
	} else if c.FirstName.Valid {
		return c.FirstName.String
	} else if c.LastName.Valid {
		return c.LastName.String
	}
	return ""
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

// AccountSummary contains aggregate counts for the account dashboard.
type AccountSummary struct {
	AddressCount       int
	PaymentMethodCount int
	OrderCount         int
	HasDefaultShipping bool
	HasDefaultBilling  bool
	HasDefaultPayment  bool
}

// SessionData represents the data stored in a session.
type SessionData struct {
	UserID string `json:"user_id"`
}

// UserListItem represents a user in a listing with display fields.
type UserListItem struct {
	ID                 pgtype.UUID
	Email              string
	FirstName          pgtype.Text
	LastName           pgtype.Text
	FullName           string
	AccountType        UserAccountType
	Status             UserStatus
	CreatedAt          pgtype.Timestamptz
	CompanyName        pgtype.Text
	WholesaleStatus    pgtype.Text
}

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// UserService provides business logic for user/customer operations.
// Implementations should be tenant-scoped at construction time.
type UserService interface {
	// -------------------------------------------------------------------------
	// Authentication Operations
	// -------------------------------------------------------------------------

	// Register creates a new user account.
	Register(ctx context.Context, email, password, firstName, lastName string) (*Customer, error)

	// Authenticate verifies email/password and returns the user if valid.
	Authenticate(ctx context.Context, email, password string) (*Customer, error)

	// CreateSession creates a new session for a user.
	CreateSession(ctx context.Context, userID string) (string, error)

	// GetUserBySessionToken retrieves a user from a session token.
	GetUserBySessionToken(ctx context.Context, token string) (*Customer, error)

	// DeleteSession logs out a user by deleting their session.
	DeleteSession(ctx context.Context, token string) error

	// -------------------------------------------------------------------------
	// User Retrieval Operations
	// -------------------------------------------------------------------------

	// GetUserByID retrieves a user by ID.
	GetUserByID(ctx context.Context, userID string) (*Customer, error)

	// GetUserByEmail retrieves a user by email.
	GetUserByEmail(ctx context.Context, email string) (*Customer, error)

	// ListUsers returns all users with pagination.
	ListUsers(ctx context.Context, limit, offset int32) ([]UserListItem, error)

	// ListUsersByAccountType returns users filtered by account type.
	ListUsersByAccountType(ctx context.Context, accountType UserAccountType) ([]UserListItem, error)

	// CountUsers returns the total count of users.
	CountUsers(ctx context.Context) (int64, error)

	// -------------------------------------------------------------------------
	// User Update Operations
	// -------------------------------------------------------------------------

	// UpdateUserProfile updates basic user profile information.
	UpdateUserProfile(ctx context.Context, userID string, params UpdateUserProfileParams) error

	// UpdateUserPassword updates a user's password.
	UpdateUserPassword(ctx context.Context, userID, newPassword string) error

	// UpdateUserStatus updates a user's status.
	UpdateUserStatus(ctx context.Context, userID string, status UserStatus) error

	// AdminUpdateCustomer updates customer details (admin only).
	AdminUpdateCustomer(ctx context.Context, userID string, params AdminUpdateCustomerParams) error

	// VerifyUserEmail marks a user's email as verified.
	VerifyUserEmail(ctx context.Context, userID string) error

	// -------------------------------------------------------------------------
	// Wholesale Operations
	// -------------------------------------------------------------------------

	// SubmitWholesaleApplication submits a wholesale application.
	SubmitWholesaleApplication(ctx context.Context, userID string, params WholesaleApplicationParams) error

	// UpdateWholesaleApplication approves or rejects a wholesale application.
	UpdateWholesaleApplication(ctx context.Context, userID string, params UpdateWholesaleApplicationParams) error

	// UpdateWholesaleCustomer updates wholesale-specific customer settings.
	UpdateWholesaleCustomer(ctx context.Context, userID string, params UpdateWholesaleCustomerParams) error

	// GetCustomersForBillingCycle returns customers due for billing.
	GetCustomersForBillingCycle(ctx context.Context, billingCycle BillingCycle, day int32) ([]Customer, error)

	// -------------------------------------------------------------------------
	// Account Operations (Addresses & Payment Methods)
	// -------------------------------------------------------------------------

	// ListAddresses returns all saved addresses for a user.
	ListAddresses(ctx context.Context, userID string) ([]UserAddress, error)

	// ListPaymentMethods returns all saved payment methods for a user.
	ListPaymentMethods(ctx context.Context, userID string) ([]UserPaymentMethod, error)

	// GetAccountSummary returns aggregate counts for the account dashboard.
	GetAccountSummary(ctx context.Context, userID string) (AccountSummary, error)
}

// =============================================================================
// PARAMETER TYPES
// =============================================================================

// RegisterUserParams contains parameters for user registration.
type RegisterUserParams struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// UpdateUserProfileParams contains parameters for updating a user profile.
type UpdateUserProfileParams struct {
	FirstName *string
	LastName  *string
	Phone     *string
}

// AdminUpdateCustomerParams contains parameters for admin customer updates.
type AdminUpdateCustomerParams struct {
	FirstName    pgtype.Text
	LastName     pgtype.Text
	Phone        pgtype.Text
	CompanyName  pgtype.Text
	BusinessType pgtype.Text
	TaxID        pgtype.Text
	Status       pgtype.Text
	InternalNote pgtype.Text
}

// WholesaleApplicationParams contains parameters for submitting a wholesale application.
type WholesaleApplicationParams struct {
	CompanyName  string
	BusinessType string
	TaxID        string
	Notes        string
}

// UpdateWholesaleApplicationParams contains parameters for updating a wholesale application.
type UpdateWholesaleApplicationParams struct {
	Status         WholesaleApplicationStatus
	Notes          string
	ApprovedBy     pgtype.UUID
	PaymentTerms   string // Legacy field
	PaymentTermsID pgtype.UUID
	BillingCycle   BillingCycle
}

// UpdateWholesaleCustomerParams contains parameters for updating wholesale settings.
type UpdateWholesaleCustomerParams struct {
	CompanyName       string
	PaymentTermsID    pgtype.UUID
	BillingCycle      BillingCycle
	BillingCycleDay   int32
	MinimumSpendCents int32
	CustomerReference string
	InternalNote      string
	EmailOrders       string
	EmailDispatches   string
	EmailInvoices     string
}

// =============================================================================
// DOMAIN ERRORS
// =============================================================================

// User-specific errors.
var (
	ErrUserNotFound     = &Error{Code: ENOTFOUND, Message: "User not found"}
	ErrUserExists       = &Error{Code: ECONFLICT, Message: "User with this email already exists"}
	ErrInvalidEmail     = &Error{Code: EINVALID, Message: "Invalid email address"}
	ErrInvalidPassword  = &Error{Code: EUNAUTHORIZED, Message: "Invalid email or password"}
	ErrSessionExpired   = &Error{Code: EUNAUTHORIZED, Message: "Session has expired"}
	ErrAccountSuspended = &Error{Code: EFORBIDDEN, Message: "Account is suspended"}
	ErrAccountPending   = &Error{Code: EFORBIDDEN, Message: "Account is pending approval"}
	ErrEmailNotVerified = &Error{Code: EFORBIDDEN, Message: "Email has not been verified"}

	ErrNoWholesaleApplication = &Error{Code: EINVALID, Message: "No pending wholesale application"}
)
