package domain

import (
	"time"

	"github.com/google/uuid"
)

// TenantOperator represents a person who manages a tenant (roaster staff).
// This is separate from users (customers) who buy coffee.
type TenantOperator struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Email       string
	Name        string
	Role        OperatorRole
	Status      OperatorStatus
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
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

// IsActive returns true if the operator can log in
func (o *TenantOperator) IsActive() bool {
	return o.Status == OperatorStatusActive
}

// IsPending returns true if the operator hasn't completed setup
func (o *TenantOperator) IsPending() bool {
	return o.Status == OperatorStatusPending
}

// IsSuspended returns true if the operator's access is revoked
func (o *TenantOperator) IsSuspended() bool {
	return o.Status == OperatorStatusSuspended
}

// IsOwner returns true if the operator has owner role
func (o *TenantOperator) IsOwner() bool {
	return o.Role == OperatorRoleOwner
}
