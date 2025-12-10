# SaaS Onboarding and Authentication Implementation Specification

This document provides a complete, actionable specification for implementing the SaaS onboarding and authentication system for Freyja. It includes all database migrations, Go structures, sqlc queries, service interfaces, handler signatures, and implementation notes needed for development.

**Last updated:** December 6, 2024

---

## Table of Contents

1. [Database Migrations](#1-database-migrations)
2. [Go Data Structures](#2-go-data-structures)
3. [sqlc Queries](#3-sqlc-queries)
4. [Service Layer Interfaces](#4-service-layer-interfaces)
5. [Handler Signatures](#5-handler-signatures)
6. [Middleware Signatures](#6-middleware-signatures)
7. [Webhook Handler Signatures](#7-webhook-handler-signatures)
8. [Email Job Definitions](#8-email-job-definitions)
9. [Implementation Notes](#9-implementation-notes)

---

## 1. Database Migrations

### Migration: 00026_add_saas_onboarding.sql

```sql
-- +goose Up
-- +goose StatementBegin

-- Step 1: Add Stripe integration columns to tenants table
ALTER TABLE tenants
    ADD COLUMN stripe_customer_id VARCHAR(255) UNIQUE,
    ADD COLUMN stripe_subscription_id VARCHAR(255) UNIQUE,
    ADD COLUMN grace_period_started_at TIMESTAMPTZ;

-- Step 2: Update tenants status constraint to include new states
ALTER TABLE tenants
    DROP CONSTRAINT IF EXISTS tenants_status_check,
    ADD CONSTRAINT tenants_status_check
        CHECK (status IN ('pending', 'active', 'past_due', 'suspended', 'cancelled'));

-- Step 3: Create indexes for Stripe columns
CREATE INDEX idx_tenants_stripe_customer_id ON tenants(stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;
CREATE INDEX idx_tenants_stripe_subscription_id ON tenants(stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL;
CREATE INDEX idx_tenants_grace_period ON tenants(grace_period_started_at)
    WHERE status = 'past_due';

COMMENT ON COLUMN tenants.stripe_customer_id IS 'Stripe customer ID for platform subscription billing';
COMMENT ON COLUMN tenants.stripe_subscription_id IS 'Stripe subscription ID for $149/month platform fee';
COMMENT ON COLUMN tenants.grace_period_started_at IS 'When grace period started after payment failure (7 days before suspension)';

-- Step 4: Create tenant_operators table
CREATE TABLE tenant_operators (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Authentication
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255), -- NULL until setup complete

    -- Profile
    name VARCHAR(255),

    -- Role (for future multi-user support)
    role VARCHAR(50) NOT NULL DEFAULT 'owner',
    -- owner: full access, billing management
    -- admin: full access except billing (future)
    -- staff: limited access (future)

    -- Setup/reset tokens (SHA-256 hashed)
    setup_token_hash VARCHAR(255),
    setup_token_expires_at TIMESTAMPTZ,
    reset_token_hash VARCHAR(255),
    reset_token_expires_at TIMESTAMPTZ,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending: invited, hasn't set password
    -- active: can log in
    -- suspended: access revoked

    last_login_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_operators_tenant_email_unique UNIQUE(tenant_id, email)
);

-- Indexes for tenant_operators
CREATE INDEX idx_tenant_operators_tenant_id ON tenant_operators(tenant_id);
CREATE INDEX idx_tenant_operators_email ON tenant_operators(email);
CREATE INDEX idx_tenant_operators_setup_token ON tenant_operators(setup_token_hash)
    WHERE setup_token_hash IS NOT NULL;
CREATE INDEX idx_tenant_operators_reset_token ON tenant_operators(reset_token_hash)
    WHERE reset_token_hash IS NOT NULL;

-- Auto-update trigger
CREATE TRIGGER update_tenant_operators_updated_at
    BEFORE UPDATE ON tenant_operators
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE tenant_operators IS 'People who manage a tenant (roaster staff who pay for Freyja)';
COMMENT ON COLUMN tenant_operators.role IS 'owner (full access), admin (future), staff (future)';
COMMENT ON COLUMN tenant_operators.setup_token_hash IS 'SHA-256 hash of setup token sent via email (48h expiry)';
COMMENT ON COLUMN tenant_operators.reset_token_hash IS 'SHA-256 hash of password reset token (1h expiry)';

-- Step 5: Create operator_sessions table (separate from customer sessions)
CREATE TABLE operator_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operator_id UUID NOT NULL REFERENCES tenant_operators(id) ON DELETE CASCADE,

    token_hash VARCHAR(255) NOT NULL, -- SHA-256 of session token

    -- Session metadata
    user_agent TEXT,
    ip_address INET,

    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for operator_sessions
CREATE INDEX idx_operator_sessions_token_hash ON operator_sessions(token_hash);
CREATE INDEX idx_operator_sessions_operator_id ON operator_sessions(operator_id);
CREATE INDEX idx_operator_sessions_expires_at ON operator_sessions(expires_at);

COMMENT ON TABLE operator_sessions IS 'Sessions for tenant operators (separate from customer sessions)';
COMMENT ON COLUMN operator_sessions.token_hash IS 'SHA-256 hash of session token (not stored in plaintext)';

-- Step 6: Update existing sessions table to use token_hash for consistency
-- NOTE: This is a breaking change that requires data migration if sessions exist
-- For MVP, we can drop and recreate since no production data exists

-- Rename existing token column to token_hash to reflect security model
ALTER TABLE sessions RENAME COLUMN token TO token_hash;

-- Update index
DROP INDEX IF EXISTS idx_sessions_token;
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);

COMMENT ON COLUMN sessions.token_hash IS 'SHA-256 hash of session token (not stored in plaintext)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse session column rename
ALTER TABLE sessions RENAME COLUMN token_hash TO token;
DROP INDEX IF EXISTS idx_sessions_token_hash;
CREATE INDEX idx_sessions_token ON sessions(token);

-- Drop operator tables
DROP TRIGGER IF EXISTS update_tenant_operators_updated_at ON tenant_operators;
DROP TABLE IF EXISTS operator_sessions CASCADE;
DROP TABLE IF EXISTS tenant_operators CASCADE;

-- Drop tenant Stripe columns
DROP INDEX IF EXISTS idx_tenants_grace_period;
DROP INDEX IF EXISTS idx_tenants_stripe_subscription_id;
DROP INDEX IF EXISTS idx_tenants_stripe_customer_id;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS grace_period_started_at,
    DROP COLUMN IF EXISTS stripe_subscription_id,
    DROP COLUMN IF EXISTS stripe_customer_id;

-- Restore original status constraint
ALTER TABLE tenants
    DROP CONSTRAINT IF EXISTS tenants_status_check,
    ADD CONSTRAINT tenants_status_check
        CHECK (status IN ('active', 'suspended', 'cancelled'));

-- +goose StatementEnd
```

---

## 2. Go Data Structures

### Domain Models (internal/domain/operator.go)

```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

// TenantOperator represents a person who manages a tenant (roaster staff).
// This is separate from users (customers) who buy coffee.
type TenantOperator struct {
    ID       uuid.UUID
    TenantID uuid.UUID
    Email    string
    Name     string
    Role     OperatorRole
    Status   OperatorStatus
    LastLoginAt *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
}

// OperatorRole defines access levels for operators
type OperatorRole string

const (
    OperatorRoleOwner OperatorRole = "owner" // Full access including billing
    OperatorRoleAdmin OperatorRole = "admin" // Full access except billing (future)
    OperatorRoleStaff OperatorRole = "staff" // Limited access (future)
)

// OperatorStatus defines the account state
type OperatorStatus string

const (
    OperatorStatusPending   OperatorStatus = "pending"   // Invited, hasn't set password
    OperatorStatusActive    OperatorStatus = "active"    // Can log in
    OperatorStatusSuspended OperatorStatus = "suspended" // Access revoked
)

// TenantStatus defines the subscription state
type TenantStatus string

const (
    TenantStatusPending   TenantStatus = "pending"   // Paid but setup not complete
    TenantStatusActive    TenantStatus = "active"    // Fully operational
    TenantStatusPastDue   TenantStatus = "past_due"  // Payment failed, in grace period
    TenantStatusSuspended TenantStatus = "suspended" // Grace period expired
    TenantStatusCancelled TenantStatus = "cancelled" // Subscription cancelled
)

// OperatorSession represents an authenticated operator session
type OperatorSession struct {
    ID         uuid.UUID
    OperatorID uuid.UUID
    UserAgent  string
    IPAddress  string
    ExpiresAt  time.Time
    CreatedAt  time.Time
}
```

### DTOs (internal/dto/operator.go)

```go
package dto

// CreateOperatorRequest contains data for creating a new operator
type CreateOperatorRequest struct {
    TenantID uuid.UUID
    Email    string
    Name     string
    Role     string // Default: "owner"
}

// SetupPasswordRequest contains data for password setup
type SetupPasswordRequest struct {
    Token    string
    Password string
}

// OperatorLoginRequest contains login credentials
type OperatorLoginRequest struct {
    Email    string
    Password string
}

// RequestPasswordResetRequest contains email for password reset
type RequestPasswordResetRequest struct {
    Email string
}

// ResetPasswordRequest contains reset token and new password
type ResetPasswordRequest struct {
    Token           string
    Password        string
    ConfirmPassword string
}

// OperatorResponse is the public-facing operator data
type OperatorResponse struct {
    ID          uuid.UUID  `json:"id"`
    TenantID    uuid.UUID  `json:"tenant_id"`
    Email       string     `json:"email"`
    Name        string     `json:"name"`
    Role        string     `json:"role"`
    Status      string     `json:"status"`
    LastLoginAt *time.Time `json:"last_login_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
}
```

### Repository Row Types (generated by sqlc)

These will be generated in `internal/repository/` when sqlc runs. Shown here for reference:

```go
// TenantOperator represents a row from tenant_operators table
type TenantOperator struct {
    ID                   pgtype.UUID        `json:"id"`
    TenantID             pgtype.UUID        `json:"tenant_id"`
    Email                string             `json:"email"`
    PasswordHash         pgtype.Text        `json:"password_hash"`
    Name                 pgtype.Text        `json:"name"`
    Role                 string             `json:"role"`
    SetupTokenHash       pgtype.Text        `json:"setup_token_hash"`
    SetupTokenExpiresAt  pgtype.Timestamptz `json:"setup_token_expires_at"`
    ResetTokenHash       pgtype.Text        `json:"reset_token_hash"`
    ResetTokenExpiresAt  pgtype.Timestamptz `json:"reset_token_expires_at"`
    Status               string             `json:"status"`
    LastLoginAt          pgtype.Timestamptz `json:"last_login_at"`
    CreatedAt            pgtype.Timestamptz `json:"created_at"`
    UpdatedAt            pgtype.Timestamptz `json:"updated_at"`
}

// OperatorSession represents a row from operator_sessions table
type OperatorSession struct {
    ID         pgtype.UUID        `json:"id"`
    OperatorID pgtype.UUID        `json:"operator_id"`
    TokenHash  string             `json:"token_hash"`
    UserAgent  pgtype.Text        `json:"user_agent"`
    IpAddress  pgtype.Text        `json:"ip_address"` // INET type maps to string
    ExpiresAt  pgtype.Timestamptz `json:"expires_at"`
    CreatedAt  pgtype.Timestamptz `json:"created_at"`
}
```

---

## 3. sqlc Queries

### sqlc/queries/tenant_operators.sql

```sql
-- name: CreateTenantOperator :one
-- Create a new tenant operator (called after Stripe checkout)
INSERT INTO tenant_operators (
    tenant_id,
    email,
    name,
    role,
    setup_token_hash,
    setup_token_expires_at,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, 'pending'
) RETURNING *;

-- name: GetTenantOperatorByID :one
-- Get operator by ID
SELECT *
FROM tenant_operators
WHERE id = $1
LIMIT 1;

-- name: GetTenantOperatorByEmail :one
-- Get operator by email within a tenant
SELECT *
FROM tenant_operators
WHERE tenant_id = $1
  AND email = $2
LIMIT 1;

-- name: GetTenantOperatorByIDAndTenant :one
-- Get operator by ID within a specific tenant (for session validation)
SELECT *
FROM tenant_operators
WHERE id = $1
  AND tenant_id = $2
LIMIT 1;

-- name: GetTenantOperatorBySetupToken :one
-- Get operator by valid (non-expired) setup token
SELECT *
FROM tenant_operators
WHERE setup_token_hash = $1
  AND setup_token_expires_at > NOW()
  AND status = 'pending'
LIMIT 1;

-- name: GetTenantOperatorByResetToken :one
-- Get operator by valid (non-expired) reset token
SELECT *
FROM tenant_operators
WHERE reset_token_hash = $1
  AND reset_token_expires_at > NOW()
LIMIT 1;

-- name: SetOperatorPassword :exec
-- Set operator password and activate account (called during setup or password reset)
UPDATE tenant_operators
SET
    password_hash = $2,
    status = 'active',
    setup_token_hash = NULL,
    setup_token_expires_at = NULL,
    reset_token_hash = NULL,
    reset_token_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateOperatorPassword :exec
-- Update operator password (for password resets)
UPDATE tenant_operators
SET
    password_hash = $2,
    reset_token_hash = NULL,
    reset_token_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: SetOperatorResetToken :exec
-- Set password reset token for an operator
UPDATE tenant_operators
SET
    reset_token_hash = $2,
    reset_token_expires_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: ClearOperatorSetupToken :exec
-- Clear setup token after successful use
UPDATE tenant_operators
SET
    setup_token_hash = NULL,
    setup_token_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateOperatorLastLogin :exec
-- Update last login timestamp
UPDATE tenant_operators
SET
    last_login_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateOperatorProfile :exec
-- Update operator profile information
UPDATE tenant_operators
SET
    name = COALESCE($2, name),
    email = COALESCE($3, email),
    updated_at = NOW()
WHERE id = $1
  AND tenant_id = $2;

-- name: SuspendOperator :exec
-- Suspend an operator account
UPDATE tenant_operators
SET
    status = 'suspended',
    updated_at = NOW()
WHERE id = $1
  AND tenant_id = $2;

-- name: ActivateOperator :exec
-- Activate a suspended operator account
UPDATE tenant_operators
SET
    status = 'active',
    updated_at = NOW()
WHERE id = $1
  AND tenant_id = $2;

-- name: ListTenantOperators :many
-- List all operators for a tenant (for future multi-user support)
SELECT *
FROM tenant_operators
WHERE tenant_id = $1
ORDER BY created_at ASC;

-- name: CountOperatorsByTenant :one
-- Count operators for a tenant
SELECT COUNT(*)
FROM tenant_operators
WHERE tenant_id = $1;
```

### sqlc/queries/operator_sessions.sql

```sql
-- name: CreateOperatorSession :one
-- Create a new operator session
INSERT INTO operator_sessions (
    operator_id,
    token_hash,
    user_agent,
    ip_address,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetOperatorSessionByTokenHash :one
-- Get a valid (non-expired) operator session by token hash
SELECT *
FROM operator_sessions
WHERE token_hash = $1
  AND expires_at > NOW()
LIMIT 1;

-- name: DeleteOperatorSession :exec
-- Delete an operator session (logout)
DELETE FROM operator_sessions
WHERE token_hash = $1;

-- name: DeleteOperatorSessionsByOperatorID :exec
-- Delete all sessions for an operator (e.g., password change)
DELETE FROM operator_sessions
WHERE operator_id = $1;

-- name: DeleteExpiredOperatorSessions :exec
-- Clean up expired operator sessions (background job)
DELETE FROM operator_sessions
WHERE expires_at <= NOW();

-- name: GetOperatorSessionsForOperator :many
-- Get all active sessions for an operator (for "active sessions" UI)
SELECT *
FROM operator_sessions
WHERE operator_id = $1
  AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: CountActiveOperatorSessions :one
-- Count active sessions for an operator
SELECT COUNT(*)
FROM operator_sessions
WHERE operator_id = $1
  AND expires_at > NOW();
```

### sqlc/queries/tenants.sql (additions)

Add these queries to the existing tenants.sql file:

```sql
-- name: UpdateTenantStripeCustomer :exec
-- Set Stripe customer ID for a tenant
UPDATE tenants
SET
    stripe_customer_id = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateTenantStripeSubscription :exec
-- Set Stripe subscription ID for a tenant
UPDATE tenants
SET
    stripe_subscription_id = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: GetTenantByStripeCustomerID :one
-- Get tenant by Stripe customer ID (for webhook processing)
SELECT *
FROM tenants
WHERE stripe_customer_id = $1
LIMIT 1;

-- name: GetTenantByStripeSubscriptionID :one
-- Get tenant by Stripe subscription ID (for webhook processing)
SELECT *
FROM tenants
WHERE stripe_subscription_id = $1
LIMIT 1;

-- name: SetTenantStatus :exec
-- Update tenant status
UPDATE tenants
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: StartTenantGracePeriod :exec
-- Start grace period after payment failure
UPDATE tenants
SET
    status = 'past_due',
    grace_period_started_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: ClearTenantGracePeriod :exec
-- Clear grace period after successful payment
UPDATE tenants
SET
    status = 'active',
    grace_period_started_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: GetTenantsWithExpiredGracePeriod :many
-- Get tenants whose grace period has expired (for suspension job)
-- Grace period is 7 days (168 hours)
SELECT *
FROM tenants
WHERE status = 'past_due'
  AND grace_period_started_at IS NOT NULL
  AND grace_period_started_at <= NOW() - INTERVAL '168 hours'
ORDER BY grace_period_started_at ASC;

-- name: ActivateTenant :exec
-- Activate a pending tenant after password setup
UPDATE tenants
SET
    status = 'active',
    updated_at = NOW()
WHERE id = $1
  AND status = 'pending';
```

---

## 4. Service Layer Interfaces

### internal/service/operator.go

```go
package service

import (
    "context"
    "time"
    "github.com/google/uuid"
    "github.com/dukerupert/hiri/internal/domain"
    "github.com/dukerupert/hiri/internal/repository"
)

// OperatorService provides business logic for tenant operator operations
type OperatorService interface {
    // Authentication methods

    // Authenticate verifies email/password and returns the operator if valid
    // Returns ErrInvalidPassword, ErrOperatorNotFound, ErrAccountSuspended
    Authenticate(ctx context.Context, tenantID uuid.UUID, email, password string) (*domain.TenantOperator, error)

    // CreateSession creates a new session for an operator
    // Returns the raw session token (not hashed) to be set as cookie
    CreateSession(ctx context.Context, operatorID uuid.UUID, userAgent, ipAddress string) (string, error)

    // GetOperatorBySessionToken retrieves an operator from a session token
    // Validates token hash, expiration, and operator status
    // Returns ErrSessionExpired, ErrOperatorNotFound, ErrAccountSuspended
    GetOperatorBySessionToken(ctx context.Context, rawToken string) (*domain.TenantOperator, error)

    // DeleteSession logs out an operator by deleting their session
    DeleteSession(ctx context.Context, rawToken string) error

    // DeleteAllSessions logs out an operator from all devices
    DeleteAllSessions(ctx context.Context, operatorID uuid.UUID) error

    // Password management methods

    // ValidateSetupToken validates a setup token and returns the associated operator
    // Returns ErrInvalidToken if token is invalid, expired, or already used
    ValidateSetupToken(ctx context.Context, rawToken string) (*domain.TenantOperator, error)

    // SetPassword sets the operator's password and activates their account
    // Called during initial setup or password reset completion
    // Also activates the associated tenant if status is 'pending'
    SetPassword(ctx context.Context, operatorID, tenantID uuid.UUID, password string) error

    // RequestPasswordReset initiates a password reset flow
    // Generates reset token, stores hash, sends email
    // Returns nil (without error) if operator doesn't exist (prevent enumeration)
    RequestPasswordReset(ctx context.Context, tenantID uuid.UUID, email, ipAddress, userAgent string) error

    // ValidateResetToken validates a reset token and returns the associated operator
    // Returns ErrInvalidToken if token is invalid, expired, or already used
    ValidateResetToken(ctx context.Context, rawToken string) (*domain.TenantOperator, error)

    // ResetPassword completes password reset using a valid token
    // Returns ErrInvalidToken, ErrWeakPassword
    ResetPassword(ctx context.Context, rawToken, newPassword string) error

    // Operator CRUD methods

    // GetOperatorByID retrieves an operator by ID
    GetOperatorByID(ctx context.Context, operatorID uuid.UUID) (*domain.TenantOperator, error)

    // GetOperatorByEmail retrieves an operator by email within a tenant
    GetOperatorByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.TenantOperator, error)

    // UpdateProfile updates operator profile information
    UpdateProfile(ctx context.Context, operatorID, tenantID uuid.UUID, name, email string) error

    // UpdateLastLogin updates the last login timestamp
    UpdateLastLogin(ctx context.Context, operatorID uuid.UUID) error
}

// OperatorServiceConfig holds configuration for operator service
type OperatorServiceConfig struct {
    // SetupTokenExpiry is how long setup tokens are valid (default: 48 hours)
    SetupTokenExpiry time.Duration

    // ResetTokenExpiry is how long reset tokens are valid (default: 1 hour)
    ResetTokenExpiry time.Duration

    // SessionExpiry is how long sessions are valid (default: 7 days)
    SessionExpiry time.Duration
}

// NewOperatorService creates a new OperatorService instance
func NewOperatorService(
    repo repository.Querier,
    jobService JobService,
    config OperatorServiceConfig,
) OperatorService {
    // Implementation will be in operator.go
}
```

### internal/service/onboarding.go

```go
package service

import (
    "context"
    "github.com/google/uuid"
    "github.com/dukerupert/hiri/internal/domain"
)

// OnboardingService handles SaaS customer onboarding flows
type OnboardingService interface {
    // CreateCheckoutSession creates a Stripe Checkout session for new tenant signup
    // Returns checkout session URL to redirect customer to
    CreateCheckoutSession(ctx context.Context, params CreateCheckoutParams) (string, error)

    // ProcessCheckoutCompleted handles the checkout.session.completed webhook
    // Creates tenant, operator, setup token, and queues welcome email
    // Returns tenant ID and operator ID
    ProcessCheckoutCompleted(ctx context.Context, session CheckoutSession) (tenantID, operatorID uuid.UUID, err error)

    // ProcessInvoicePaid handles the invoice.paid webhook
    // Clears grace period if tenant is past_due
    ProcessInvoicePaid(ctx context.Context, invoiceID string) error

    // ProcessInvoicePaymentFailed handles the invoice.payment_failed webhook
    // Starts grace period, queues payment failed email
    ProcessInvoicePaymentFailed(ctx context.Context, invoiceID string) error

    // ProcessSubscriptionUpdated handles the customer.subscription.updated webhook
    // Syncs subscription status changes
    ProcessSubscriptionUpdated(ctx context.Context, subscriptionID string) error

    // ProcessSubscriptionDeleted handles the customer.subscription.deleted webhook
    // Sets tenant status to 'cancelled'
    ProcessSubscriptionDeleted(ctx context.Context, subscriptionID string) error

    // CreateBillingPortalSession creates a Stripe Customer Portal session
    // Returns portal URL to redirect operator to
    CreateBillingPortalSession(ctx context.Context, tenantID uuid.UUID, returnURL string) (string, error)

    // ExpireGracePeriods suspends tenants whose grace period has expired
    // Called by background job hourly
    // Returns count of suspended tenants
    ExpireGracePeriods(ctx context.Context) (int, error)
}

// CreateCheckoutParams contains parameters for creating a checkout session
type CreateCheckoutParams struct {
    SuccessURL string // URL to redirect after successful payment
    CancelURL  string // URL to redirect if checkout is cancelled
}

// CheckoutSession represents data from Stripe checkout.session.completed event
type CheckoutSession struct {
    ID           string
    CustomerID   string
    Email        string
    BusinessName string // From custom field
    AmountTotal  int64
}

// NewOnboardingService creates a new OnboardingService instance
func NewOnboardingService(
    repo repository.Querier,
    billingProvider billing.Provider,
    jobService JobService,
    operatorService OperatorService,
) OnboardingService {
    // Implementation will be in onboarding.go
}
```

---

## 5. Handler Signatures

### internal/handler/saas/checkout.go

```go
package saas

import (
    "net/http"
    "github.com/dukerupert/hiri/internal/service"
    "github.com/dukerupert/hiri/internal/handler"
)

// CheckoutHandler handles SaaS checkout flow
type CheckoutHandler struct {
    onboardingService service.OnboardingService
    renderer          *handler.Renderer
}

// NewCheckoutHandler creates a new checkout handler
func NewCheckoutHandler(
    onboardingService service.OnboardingService,
    renderer *handler.Renderer,
) *CheckoutHandler

// CreateCheckoutSession handles POST /api/create-checkout-session
// Creates Stripe Checkout session and returns JSON with session URL
func (h *CheckoutHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request)
```

### internal/handler/saas/setup.go

```go
package saas

import (
    "net/http"
    "github.com/dukerupert/hiri/internal/service"
    "github.com/dukerupert/hiri/internal/handler"
)

// SetupHandler handles account setup after checkout
type SetupHandler struct {
    operatorService   service.OperatorService
    onboardingService service.OnboardingService
    renderer          *handler.Renderer
}

// NewSetupHandler creates a new setup handler
func NewSetupHandler(
    operatorService service.OperatorService,
    onboardingService service.OnboardingService,
    renderer *handler.Renderer,
) *SetupHandler

// ShowSetupForm handles GET /setup?token=xxx
// Validates token and displays password form
func (h *SetupHandler) ShowSetupForm(w http.ResponseWriter, r *http.Request)

// HandleSetup handles POST /setup
// Sets password, activates account, creates session, redirects to /admin
func (h *SetupHandler) HandleSetup(w http.ResponseWriter, r *http.Request)

// ShowResendSetupForm handles GET /resend-setup
// Displays email form to request new setup link
func (h *SetupHandler) ShowResendSetupForm(w http.ResponseWriter, r *http.Request)

// HandleResendSetup handles POST /resend-setup
// Regenerates setup token, sends email (always shows success to prevent enumeration)
func (h *SetupHandler) HandleResendSetup(w http.ResponseWriter, r *http.Request)
```

### internal/handler/saas/auth.go

```go
package saas

import (
    "net/http"
    "github.com/dukerupert/hiri/internal/service"
    "github.com/dukerupert/hiri/internal/handler"
)

const (
    // OperatorCookieName is the cookie name for operator sessions
    OperatorCookieName = "freyja_operator"

    // OperatorCookiePath restricts cookie to admin routes
    OperatorCookiePath = "/admin"

    // OperatorSessionMaxAge is 7 days in seconds
    OperatorSessionMaxAge = 7 * 24 * 60 * 60
)

// AuthHandler handles operator authentication flows
type AuthHandler struct {
    operatorService service.OperatorService
    renderer        *handler.Renderer
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
    operatorService service.OperatorService,
    renderer *handler.Renderer,
) *AuthHandler

// ShowLoginForm handles GET /login
// Displays login form
func (h *AuthHandler) ShowLoginForm(w http.ResponseWriter, r *http.Request)

// HandleLogin handles POST /login
// Authenticates operator, creates session, sets cookie, redirects to /admin
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request)

// HandleLogout handles POST /logout
// Deletes session, clears cookie, redirects to /login
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request)

// ShowForgotPasswordForm handles GET /forgot-password
// Displays email form
func (h *AuthHandler) ShowForgotPasswordForm(w http.ResponseWriter, r *http.Request)

// HandleForgotPassword handles POST /forgot-password
// Requests password reset, always shows success message
func (h *AuthHandler) HandleForgotPassword(w http.ResponseWriter, r *http.Request)

// ShowResetPasswordForm handles GET /reset-password?token=xxx
// Validates token and displays password form
func (h *AuthHandler) ShowResetPasswordForm(w http.ResponseWriter, r *http.Request)

// HandleResetPassword handles POST /reset-password
// Resets password using token, redirects to login
func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request)
```

### internal/handler/saas/billing.go

```go
package saas

import (
    "net/http"
    "github.com/dukerupert/hiri/internal/service"
)

// BillingHandler handles billing portal access
type BillingHandler struct {
    onboardingService service.OnboardingService
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(
    onboardingService service.OnboardingService,
) *BillingHandler

// RedirectToBillingPortal handles GET /billing
// Creates Stripe Customer Portal session and redirects operator
// Requires RequireOperator middleware
func (h *BillingHandler) RedirectToBillingPortal(w http.ResponseWriter, r *http.Request)
```

### internal/handler/saas/webhook.go

```go
package saas

import (
    "net/http"
    "github.com/dukerupert/hiri/internal/service"
    "github.com/dukerupert/hiri/internal/billing"
)

// WebhookHandler handles Stripe webhooks for SaaS subscriptions
type WebhookHandler struct {
    onboardingService service.OnboardingService
    billingProvider   billing.Provider
    webhookSecret     string // Stripe webhook signing secret
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(
    onboardingService service.OnboardingService,
    billingProvider billing.Provider,
    webhookSecret string,
) *WebhookHandler

// HandleStripeWebhook handles POST /webhooks/stripe/saas
// Processes Stripe webhook events for platform subscriptions
// Events: checkout.session.completed, invoice.paid, invoice.payment_failed,
//         customer.subscription.updated, customer.subscription.deleted
func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request)
```

---

## 6. Middleware Signatures

### internal/middleware/operator.go

```go
package middleware

import (
    "context"
    "net/http"
    "github.com/google/uuid"
    "github.com/dukerupert/hiri/internal/service"
    "github.com/dukerupert/hiri/internal/domain"
)

// OperatorContextKey is the context key for storing operator data
type operatorKey struct{}

var OperatorContextKey = operatorKey{}

// RequireOperator is middleware that validates operator session
// Reads freyja_operator cookie, validates session, loads operator
// Redirects to /login if not authenticated or session expired
func RequireOperator(operatorService service.OperatorService) func(http.Handler) http.Handler

// RequireActiveTenant is middleware that ensures tenant subscription is active
// Checks tenant status: allows 'active' and 'past_due', blocks 'pending', 'suspended', 'cancelled'
// Shows banner for 'past_due', blocks access for others
// Should be used after RequireOperator
func RequireActiveTenant() func(http.Handler) http.Handler

// WithOperatorContext adds operator to request context without enforcing authentication
// Used for routes that optionally use operator data (e.g., public pages with admin preview)
func WithOperatorContext(operatorService service.OperatorService) func(http.Handler) http.Handler

// GetOperator retrieves operator from request context
// Returns nil if no operator in context
func GetOperator(ctx context.Context) *domain.TenantOperator

// GetOperatorID retrieves operator ID from request context
// Returns uuid.Nil if no operator in context
func GetOperatorID(ctx context.Context) uuid.UUID

// GetTenantIDFromOperator retrieves tenant ID from operator in context
// Returns uuid.Nil if no operator in context
func GetTenantIDFromOperator(ctx context.Context) uuid.UUID
```

---

## 7. Webhook Handler Signatures

All webhook processing happens in `internal/handler/saas/webhook.go` via the single `HandleStripeWebhook` handler. Event routing is done internally:

```go
// HandleStripeWebhook processes these event types:

// checkout.session.completed
// - Extract email, business_name from session metadata
// - Call onboardingService.ProcessCheckoutCompleted()
// - Creates tenant (status: pending), operator (status: pending)
// - Generates setup token, queues welcome email

// invoice.paid
// - Extract invoice ID
// - Call onboardingService.ProcessInvoicePaid()
// - If tenant is past_due, clear grace period and set to active

// invoice.payment_failed
// - Extract invoice ID
// - Call onboardingService.ProcessInvoicePaymentFailed()
// - Set tenant to past_due, start grace period, queue payment failed email

// customer.subscription.updated
// - Extract subscription ID
// - Call onboardingService.ProcessSubscriptionUpdated()
// - Sync subscription status changes (e.g., from Stripe portal)

// customer.subscription.deleted
// - Extract subscription ID
// - Call onboardingService.ProcessSubscriptionDeleted()
// - Set tenant status to 'cancelled'
```

All webhook events are validated via `billingProvider.VerifyWebhookSignature()` before processing.

---

## 8. Email Job Definitions

### internal/jobs/operator_emails.go

```go
package jobs

import (
    "context"
    "github.com/google/uuid"
)

const (
    // Job type constants for SaaS operator emails
    JobTypeOperatorWelcome           = "email:operator_welcome"
    JobTypeOperatorPasswordReset     = "email:operator_password_reset"
    JobTypeOperatorPaymentFailed     = "email:operator_payment_failed"
    JobTypeOperatorAccountSuspended  = "email:operator_account_suspended"
)

// OperatorWelcomeEmailJob sends welcome email with setup link
type OperatorWelcomeEmailJob struct {
    OperatorID   uuid.UUID `json:"operator_id"`
    TenantID     uuid.UUID `json:"tenant_id"`
    Email        string    `json:"email"`
    Name         string    `json:"name"`
    BusinessName string    `json:"business_name"`
    SetupToken   string    `json:"setup_token"` // Raw token, not hash
}

// OperatorPasswordResetEmailJob sends password reset link
type OperatorPasswordResetEmailJob struct {
    OperatorID uuid.UUID `json:"operator_id"`
    TenantID   uuid.UUID `json:"tenant_id"`
    Email      string    `json:"email"`
    Name       string    `json:"name"`
    ResetToken string    `json:"reset_token"` // Raw token, not hash
}

// OperatorPaymentFailedEmailJob notifies of failed payment and grace period
type OperatorPaymentFailedEmailJob struct {
    OperatorID        uuid.UUID `json:"operator_id"`
    TenantID          uuid.UUID `json:"tenant_id"`
    Email             string    `json:"email"`
    Name              string    `json:"name"`
    AmountDueCents    int64     `json:"amount_due_cents"`
    InvoiceID         string    `json:"invoice_id"`
    BillingPortalURL  string    `json:"billing_portal_url"`
}

// OperatorAccountSuspendedEmailJob notifies of account suspension
type OperatorAccountSuspendedEmailJob struct {
    OperatorID       uuid.UUID `json:"operator_id"`
    TenantID         uuid.UUID `json:"tenant_id"`
    Email            string    `json:"email"`
    Name             string    `json:"name"`
    BillingPortalURL string    `json:"billing_portal_url"`
}

// ProcessOperatorWelcomeEmail processes welcome email job
func ProcessOperatorWelcomeEmail(ctx context.Context, data OperatorWelcomeEmailJob) error {
    // Build setup URL: https://freyja.app/setup?token={token}
    // Render email template: web/templates/email/operator_welcome.html
    // Send via email provider
}

// ProcessOperatorPasswordResetEmail processes password reset email job
func ProcessOperatorPasswordResetEmail(ctx context.Context, data OperatorPasswordResetEmailJob) error {
    // Build reset URL: https://freyja.app/reset-password?token={token}
    // Render email template: web/templates/email/operator_password_reset.html
    // Send via email provider
}

// ProcessOperatorPaymentFailedEmail processes payment failed email job
func ProcessOperatorPaymentFailedEmail(ctx context.Context, data OperatorPaymentFailedEmailJob) error {
    // Render email template: web/templates/email/operator_payment_failed.html
    // Send via email provider
}

// ProcessOperatorAccountSuspendedEmail processes account suspended email job
func ProcessOperatorAccountSuspendedEmail(ctx context.Context, data OperatorAccountSuspendedEmailJob) error {
    // Render email template: web/templates/email/operator_account_suspended.html
    // Send via email provider
}
```

### Add missing wholesale approval email job

```go
const (
    JobTypeWholesaleApproved = "email:wholesale_approved"
)

// WholesaleApprovedEmailJob notifies customer of wholesale account approval
type WholesaleApprovedEmailJob struct {
    UserID   uuid.UUID `json:"user_id"`
    TenantID uuid.UUID `json:"tenant_id"`
    Email    string    `json:"email"`
    Name     string    `json:"name"`
}

// ProcessWholesaleApprovedEmail processes wholesale approval email job
func ProcessWholesaleApprovedEmail(ctx context.Context, data WholesaleApprovedEmailJob) error {
    // Render email template: web/templates/email/wholesale_approved.html
    // Send via email provider
}
```

**Implementation location for wholesale approval email enqueuing:**

In `internal/handler/admin/customers.go`, add to the wholesale approval handler:

```go
// After updating account_type to 'wholesale'...
jobData := WholesaleApprovedEmailJob{
    UserID:   user.ID,
    TenantID: tenantID,
    Email:    user.Email,
    Name:     user.FirstName.String + " " + user.LastName.String,
}
h.jobService.EnqueueJob(ctx, tenantID, JobTypeWholesaleApproved, jobData)
```

---

## 9. Implementation Notes

### 9.1 Cookie Scoping Strategy

**Problem:** Operators and customers need separate authentication without interference.

**Solution:** Path-based cookie scoping

```go
// Operator session cookie (admin dashboard)
http.SetCookie(w, &http.Cookie{
    Name:     "freyja_operator",
    Value:    token,
    Path:     "/admin",        // Only sent to /admin/* routes
    MaxAge:   7 * 24 * 60 * 60, // 7 days
    HttpOnly: true,
    Secure:   true, // HTTPS only in production
    SameSite: http.SameSiteLaxMode,
})

// Customer session cookie (storefront)
http.SetCookie(w, &http.Cookie{
    Name:     "freyja_session",
    Value:    token,
    Path:     "/",             // Sent to all routes
    MaxAge:   30 * 24 * 60 * 60, // 30 days
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteLaxMode,
})
```

**Why this works:**
- Browser only sends `freyja_operator` cookie to `/admin/*` routes
- Browser sends `freyja_session` cookie to all routes
- No cookie conflicts or cross-contamination
- Operator can be logged into admin and shop as customer simultaneously

### 9.2 Token Security (Hashing and Expiry)

**All tokens must be hashed before storage:**

```go
import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
)

// GenerateToken creates a cryptographically secure random token
func GenerateToken(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

// HashToken creates SHA-256 hash of token for storage
func HashToken(rawToken string) string {
    hash := sha256.Sum256([]byte(rawToken))
    return hex.EncodeToString(hash[:])
}
```

**Token lifecycle:**

1. **Setup token (48h expiry):**
   - Generated: After checkout.session.completed webhook
   - Raw token: Sent in welcome email link
   - Stored: SHA-256 hash in `setup_token_hash`
   - Used: Once during password setup, then cleared
   - Expires: 48 hours from creation

2. **Reset token (1h expiry):**
   - Generated: During forgot password request
   - Raw token: Sent in password reset email link
   - Stored: SHA-256 hash in `reset_token_hash`
   - Used: Once during password reset, then cleared
   - Expires: 1 hour from creation

3. **Session token (7 days expiry):**
   - Generated: During login or setup completion
   - Raw token: Stored in HTTP-only cookie
   - Stored: SHA-256 hash in `operator_sessions.token_hash`
   - Used: Every request via middleware
   - Expires: 7 days from creation (sliding window possible)

**Security benefits:**
- Raw tokens never stored in database
- Token leakage from DB breach doesn't expose sessions
- Rainbow tables ineffective (each token is unique)
- Consistent with existing customer session security model

### 9.3 Migration Path from Current Admin System

**Current state:**
- Bootstrap admin user in `users` table with `account_type='admin'`
- Uses `sessions` table with shared cookie `freyja_session`
- Created via environment variables on startup

**Migration strategy:**

**Phase 1: Create tables (no behavior change)**
- Run migration 00026_add_saas_onboarding.sql
- Existing admin login continues to work
- New tables exist but unused

**Phase 2: Migrate admin user to tenant_operator**
- Create migration: 00027_migrate_admin_to_operator.sql

```sql
-- +goose Up
-- +goose StatementBegin

-- For each admin user, create corresponding tenant_operator
INSERT INTO tenant_operators (
    tenant_id,
    email,
    password_hash,
    name,
    role,
    status,
    created_at,
    updated_at
)
SELECT
    tenant_id,
    email,
    password_hash,
    COALESCE(first_name || ' ' || last_name, email) as name,
    'owner' as role,
    'active' as status,
    created_at,
    updated_at
FROM users
WHERE account_type = 'admin';

-- Delete admin users from users table
DELETE FROM users WHERE account_type = 'admin';

-- Remove 'admin' from account_type constraint
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_account_type_check,
    ADD CONSTRAINT users_account_type_check
        CHECK (account_type IN ('retail', 'wholesale'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse migration (recreate admin users from tenant_operators)
-- Omitted for brevity
-- +goose StatementEnd
```

**Phase 3: Update code**
- Add operator handlers and middleware
- Update admin routes to use `RequireOperator` middleware
- Deploy code changes
- Existing operator can log in via new `/login` route

**Phase 4: Clean up (optional)**
- Remove bootstrap admin creation code from `internal/bootstrap/admin.go`
- Remove admin-specific code from `internal/handler/admin/auth.go`

### 9.4 Integration with Existing Billing Provider Interface

The billing.Provider interface already supports all necessary operations. Additions needed:

**Add to internal/billing/billing.go:**

```go
// CreateCheckoutSession creates a Stripe Checkout session for SaaS signup
// Returns checkout session ID and URL
CreateCheckoutSession(ctx context.Context, params CreateCheckoutSessionParams) (*CheckoutSession, error)

// CreateCheckoutSessionParams contains parameters for SaaS checkout
type CreateCheckoutSessionParams struct {
    PriceID        string            // Stripe price ID for $149/month
    SuccessURL     string            // Redirect after payment
    CancelURL      string            // Redirect if cancelled
    CouponID       string            // Optional coupon (e.g., "freyja-launch-special")
    Email          string            // Prefill email
    Metadata       map[string]string // Must include business_name from custom field
    TrialPeriodDays int              // Optional trial period
}

// CheckoutSession represents a Stripe Checkout session
type CheckoutSession struct {
    ID         string
    URL        string
    CustomerID string
    Mode       string // "subscription"
    Status     string // "open", "complete", "expired"
}
```

**Stripe configuration needed:**

```bash
# Create product
stripe products create \
  --name="Freyja Platform" \
  --description="E-commerce platform for coffee roasters"

# Create monthly price
stripe prices create \
  --product=prod_xxx \
  --unit-amount=14900 \
  --currency=usd \
  --recurring[interval]=month \
  --lookup-key=freyja_monthly

# Create introductory coupon
stripe coupons create \
  --id=freyja-launch-special \
  --amount-off=14400 \
  --currency=usd \
  --duration=repeating \
  --duration-in-months=3
```

**Webhook configuration:**

```bash
# In Stripe Dashboard, configure webhook endpoint: /webhooks/stripe/saas
# Select events:
# - checkout.session.completed
# - invoice.paid
# - invoice.payment_failed
# - customer.subscription.updated
# - customer.subscription.deleted
```

### 9.5 Slug Generation Utility

**Location:** `internal/onboarding/slug.go`

```go
package onboarding

import (
    "context"
    "fmt"
    "regexp"
    "strings"
    "unicode"
    "golang.org/x/text/unicode/norm"
    "github.com/dukerupert/hiri/internal/repository"
)

var (
    // slugNonAlphanumeric matches non-alphanumeric characters
    slugNonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
    // slugMultipleDashes matches multiple consecutive dashes
    slugMultipleDashes = regexp.MustCompile(`-+`)
)

// GenerateSlug converts a name to a URL-safe slug
// Example: "Café Racer Coffee" -> "cafe-racer-coffee"
func GenerateSlug(name string) string {
    // Normalize unicode (convert café -> cafe)
    normalized := norm.NFKD.String(name)

    // Convert to lowercase
    lower := strings.ToLower(normalized)

    // Remove non-ASCII characters
    ascii := strings.Map(func(r rune) rune {
        if r > unicode.MaxASCII {
            return -1
        }
        return r
    }, lower)

    // Replace non-alphanumeric with dashes
    slug := slugNonAlphanumeric.ReplaceAllString(ascii, "-")

    // Collapse multiple dashes
    slug = slugMultipleDashes.ReplaceAllString(slug, "-")

    // Trim dashes from ends
    slug = strings.Trim(slug, "-")

    // Limit length to 100 characters
    if len(slug) > 100 {
        slug = slug[:100]
        slug = strings.TrimRight(slug, "-")
    }

    return slug
}

// GenerateUniqueSlug generates a unique slug by appending numbers if needed
// Example: "acme-coffee" -> "acme-coffee-2" if "acme-coffee" exists
func GenerateUniqueSlug(ctx context.Context, repo repository.Querier, name string) (string, error) {
    baseSlug := GenerateSlug(name)
    slug := baseSlug

    for i := 2; i < 1000; i++ { // Prevent infinite loop
        // Check if slug exists
        exists, err := repo.TenantSlugExists(ctx, slug)
        if err != nil {
            return "", err
        }

        if !exists {
            return slug, nil
        }

        // Try next number
        slug = fmt.Sprintf("%s-%d", baseSlug, i)
    }

    return "", fmt.Errorf("could not generate unique slug for name: %s", name)
}
```

**Add to sqlc/queries/tenants.sql:**

```sql
-- name: TenantSlugExists :one
-- Check if a slug is already taken
SELECT EXISTS(
    SELECT 1
    FROM tenants
    WHERE slug = $1
) as exists;
```

### 9.6 Password Requirements

**Minimum requirements:**
- At least 8 characters
- No maximum (don't artificially limit)
- No complexity requirements for MVP (avoids user frustration)

**Future considerations:**
- Integrate zxcvbn for strength checking
- Warn on weak passwords without blocking
- Check against common password lists

**Implementation:**

```go
// ValidatePassword checks password requirements
func ValidatePassword(password string) error {
    if len(password) < 8 {
        return fmt.Errorf("password must be at least 8 characters")
    }
    if len(password) > 72 {
        // bcrypt has 72-byte limit
        return fmt.Errorf("password must be less than 72 characters")
    }
    return nil
}
```

### 9.7 Session Expiry Strategy

**Current implementation:** Fixed 7-day expiry

**Considerations for sliding window:**

Pros:
- Better UX (sessions extend with activity)
- Reduces login friction for active users

Cons:
- More database writes (update expiry on each request)
- Slightly more complex middleware

**Decision for MVP:** Fixed 7-day expiry
- Simpler implementation
- Adequate for most use cases
- Can add sliding window later if requested

**Implementation for future sliding window:**

```go
// In middleware, after validating session:
if time.Until(session.ExpiresAt) < 24*time.Hour {
    // Extend session if less than 24 hours remaining
    newExpiry := time.Now().Add(7 * 24 * time.Hour)
    repo.UpdateOperatorSessionExpiry(ctx, sessionID, newExpiry)
}
```

### 9.8 Rate Limiting

**Email-based rate limits:**

All email job enqueuing should check rate limits to prevent abuse:

```sql
-- Add to sqlc/queries/tenant_operators.sql

-- name: CountRecentSetupEmailsByEmail :one
-- Count setup emails sent to an email address in time window
SELECT COUNT(*)
FROM tenant_operators
WHERE email = $1
  AND setup_token_expires_at IS NOT NULL
  AND created_at > $2;

-- name: CountRecentPasswordResetsByEmail :one
-- Count password reset requests for an operator in time window
SELECT COUNT(*)
FROM tenant_operators
WHERE email = $1
  AND reset_token_expires_at IS NOT NULL
  AND updated_at > $2; -- updated_at changes when reset token is set
```

**Rate limit constants:**

```go
const (
    // RateLimitSetupEmailPerEmail is max setup emails per email address per hour
    RateLimitSetupEmailPerEmail = 3

    // RateLimitPasswordResetPerEmail is max reset requests per email per hour
    RateLimitPasswordResetPerEmail = 3

    // RateLimitWindow is the time window for rate limiting
    RateLimitWindow = 1 * time.Hour
)
```

### 9.9 Tenant Status Enforcement

**Status-based access control:**

```go
// In RequireActiveTenant middleware:

tenant := GetTenantFromOperator(ctx)

switch tenant.Status {
case "pending":
    // Operator hasn't completed setup yet
    http.Redirect(w, r, "/setup", http.StatusSeeOther)
    return

case "suspended":
    // Grace period expired, show payment required page
    renderTemplate(w, "account_suspended", data)
    return

case "cancelled":
    // Subscription cancelled, show reactivation page
    renderTemplate(w, "account_cancelled", data)
    return

case "past_due":
    // In grace period, show warning banner but allow access
    // Banner rendered in admin layout template
    next.ServeHTTP(w, r)

case "active":
    // Normal operation
    next.ServeHTTP(w, r)
}
```

**Banner for past_due status:**

In `web/templates/admin/layout.html`:

```html
{{if eq .Tenant.Status "past_due"}}
<div class="bg-amber-100 border-l-4 border-amber-500 p-4">
    <div class="flex">
        <div class="flex-shrink-0">
            <!-- Warning icon -->
        </div>
        <div class="ml-3">
            <p class="text-sm text-amber-700">
                Your payment failed. Please <a href="/billing" class="underline">update your payment method</a>
                to avoid service interruption.
            </p>
        </div>
    </div>
</div>
{{end}}
```

### 9.10 Logging and Telemetry

**Important events to log:**

```go
// In operator service:
logger.Info("operator login successful",
    "operator_id", operatorID,
    "tenant_id", tenantID,
    "ip_address", ipAddress)

logger.Warn("operator login failed",
    "email", email,
    "reason", "invalid_password",
    "ip_address", ipAddress)

logger.Info("password reset requested",
    "operator_id", operatorID,
    "tenant_id", tenantID)

// In onboarding service:
logger.Info("checkout completed",
    "tenant_id", tenantID,
    "operator_id", operatorID,
    "stripe_customer_id", customerID)

logger.Warn("payment failed",
    "tenant_id", tenantID,
    "invoice_id", invoiceID,
    "grace_period_started", gracePeriodStart)

logger.Error("grace period expired, tenant suspended",
    "tenant_id", tenantID,
    "grace_period_days", 7)
```

**Metrics to track (Prometheus):**

```go
// operator_logins_total
// operator_login_failures_total
// checkout_sessions_created_total
// checkout_sessions_completed_total
// tenants_suspended_total
// tenants_reactivated_total
// grace_periods_active_gauge
```

---

## Summary

This specification provides all the components needed to implement SaaS onboarding:

1. **Database migrations** create tables and constraints
2. **Go structures** define domain models and DTOs
3. **sqlc queries** provide type-safe database access
4. **Service interfaces** define business logic contracts
5. **Handler signatures** define HTTP endpoints
6. **Middleware** enforces authentication and authorization
7. **Webhook handlers** process Stripe events
8. **Email jobs** handle transactional emails
9. **Implementation notes** cover security, migration path, and edge cases

**Next steps for developer:**

1. Run migration 00026_add_saas_onboarding.sql
2. Generate sqlc code: `sqlc generate`
3. Implement OperatorService (follow UserService pattern)
4. Implement OnboardingService
5. Implement handlers (follow storefront/auth.go pattern)
6. Add middleware to admin routes
7. Create email templates
8. Test checkout flow with Stripe test mode
9. Run migration 00027_migrate_admin_to_operator.sql
10. Deploy to production

All patterns follow existing codebase conventions. No new dependencies required.
