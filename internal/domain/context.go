// Package domain provides core business types and context helpers for Freyja.
//
// Context helpers centralize request-scoped data access, making tenant isolation
// bugs harder to write and providing consistent patterns throughout the codebase.
package domain

import (
	"context"

	"github.com/google/uuid"
)

// contextKey is an unexported type for context keys to prevent collisions.
type contextKey int

const (
	// tenantContextKey stores tenant information in context.
	tenantContextKey contextKey = iota

	// userContextKey stores user information in context.
	userContextKey

	// operatorContextKey stores operator (admin user) information in context.
	operatorContextKey

	// requestIDContextKey stores the request ID for tracing.
	requestIDContextKey
)

// Tenant represents tenant information stored in context.
// This is a minimal struct for context storage - the full tenant
// record can be fetched from the database if needed.
type Tenant struct {
	ID     uuid.UUID
	Slug   string
	Name   string
	Status string
}

// User represents user information stored in context.
// This is a minimal struct for context storage.
type User struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Email       string
	AccountType string // "customer", "admin", "wholesale"
}

// Operator represents operator (admin) information stored in context.
// Operators manage the tenant's store through the admin interface.
type Operator struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	Email    string
	Role     string // "owner", "admin", "staff"
	Status   string // "active", "inactive"
}

// --- Tenant Context Helpers ---

// NewContextWithTenant returns a new context with the tenant attached.
func NewContextWithTenant(ctx context.Context, tenant *Tenant) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenant)
}

// TenantFromContext retrieves the tenant from context.
// Returns nil if no tenant is present.
func TenantFromContext(ctx context.Context) *Tenant {
	tenant, _ := ctx.Value(tenantContextKey).(*Tenant)
	return tenant
}

// TenantIDFromContext retrieves the tenant ID from context.
// Returns uuid.Nil if no tenant is present.
func TenantIDFromContext(ctx context.Context) uuid.UUID {
	if tenant := TenantFromContext(ctx); tenant != nil {
		return tenant.ID
	}
	return uuid.Nil
}

// RequireTenantID retrieves the tenant ID from context, panicking if not present.
// Use this in repository/service layers where tenant is required.
// The panic will be caught by error recovery middleware in HTTP handlers.
func RequireTenantID(ctx context.Context) uuid.UUID {
	id := TenantIDFromContext(ctx)
	if id == uuid.Nil {
		panic("tenant_id required in context but not found")
	}
	return id
}

// MustTenant retrieves the tenant from context, panicking if not present.
// Use this when you need the full tenant struct and it must be present.
func MustTenant(ctx context.Context) *Tenant {
	tenant := TenantFromContext(ctx)
	if tenant == nil {
		panic("tenant required in context but not found")
	}
	return tenant
}

// --- User Context Helpers ---

// NewContextWithUser returns a new context with the user attached.
func NewContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the user from context.
// Returns nil if no user is present.
func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey).(*User)
	return user
}

// UserIDFromContext retrieves the user ID from context.
// Returns uuid.Nil if no user is present.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	if user := UserFromContext(ctx); user != nil {
		return user.ID
	}
	return uuid.Nil
}

// RequireUserID retrieves the user ID from context, panicking if not present.
// Use this in service layers where authenticated user is required.
func RequireUserID(ctx context.Context) uuid.UUID {
	id := UserIDFromContext(ctx)
	if id == uuid.Nil {
		panic("user_id required in context but not found")
	}
	return id
}

// MustUser retrieves the user from context, panicking if not present.
func MustUser(ctx context.Context) *User {
	user := UserFromContext(ctx)
	if user == nil {
		panic("user required in context but not found")
	}
	return user
}

// --- Operator Context Helpers ---

// NewContextWithOperator returns a new context with the operator attached.
func NewContextWithOperator(ctx context.Context, operator *Operator) context.Context {
	return context.WithValue(ctx, operatorContextKey, operator)
}

// OperatorFromContext retrieves the operator from context.
// Returns nil if no operator is present.
func OperatorFromContext(ctx context.Context) *Operator {
	operator, _ := ctx.Value(operatorContextKey).(*Operator)
	return operator
}

// OperatorIDFromContext retrieves the operator ID from context.
// Returns uuid.Nil if no operator is present.
func OperatorIDFromContext(ctx context.Context) uuid.UUID {
	if operator := OperatorFromContext(ctx); operator != nil {
		return operator.ID
	}
	return uuid.Nil
}

// RequireOperatorID retrieves the operator ID from context, panicking if not present.
func RequireOperatorID(ctx context.Context) uuid.UUID {
	id := OperatorIDFromContext(ctx)
	if id == uuid.Nil {
		panic("operator_id required in context but not found")
	}
	return id
}

// MustOperator retrieves the operator from context, panicking if not present.
func MustOperator(ctx context.Context) *Operator {
	operator := OperatorFromContext(ctx)
	if operator == nil {
		panic("operator required in context but not found")
	}
	return operator
}

// --- Request ID Context Helpers ---

// NewContextWithRequestID returns a new context with the request ID attached.
func NewContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// RequestIDFromContext retrieves the request ID from context.
// Returns empty string if no request ID is present.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

// --- Convenience Helpers ---

// IsAuthenticated returns true if there is a user in context.
func IsAuthenticated(ctx context.Context) bool {
	return UserFromContext(ctx) != nil
}

// IsOperator returns true if there is an operator in context.
func IsOperator(ctx context.Context) bool {
	return OperatorFromContext(ctx) != nil
}

// IsOwner returns true if the operator in context has the owner role.
func IsOwner(ctx context.Context) bool {
	operator := OperatorFromContext(ctx)
	return operator != nil && operator.Role == "owner"
}

// HasTenant returns true if there is a tenant in context.
func HasTenant(ctx context.Context) bool {
	return TenantFromContext(ctx) != nil
}
