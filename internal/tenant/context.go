package tenant

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type contextKey string

const tenantContextKey contextKey = "tenant"

// Tenant represents a resolved tenant from subdomain or custom domain lookup.
type Tenant struct {
	ID     pgtype.UUID
	Slug   string
	Name   string
	Status string // active, pending, suspended, cancelled
}

// NewContext returns a new context with the tenant attached.
func NewContext(ctx context.Context, t *Tenant) context.Context {
	return context.WithValue(ctx, tenantContextKey, t)
}

// FromContext extracts the tenant from the context.
// Returns nil if no tenant is present.
func FromContext(ctx context.Context) *Tenant {
	t, ok := ctx.Value(tenantContextKey).(*Tenant)
	if !ok {
		return nil
	}
	return t
}

// MustFromContext extracts the tenant from the context.
// Panics if no tenant is present. Use only when tenant middleware
// has definitely run (e.g., in handlers behind RequireTenant).
func MustFromContext(ctx context.Context) *Tenant {
	t := FromContext(ctx)
	if t == nil {
		panic("tenant.MustFromContext: no tenant in context")
	}
	return t
}

// IDFromContext returns the tenant ID from context.
// Returns an invalid (zero) UUID if no tenant is present.
func IDFromContext(ctx context.Context) pgtype.UUID {
	t := FromContext(ctx)
	if t == nil {
		return pgtype.UUID{}
	}
	return t.ID
}

// IsActive returns true if the tenant status is "active".
func (t *Tenant) IsActive() bool {
	return t != nil && t.Status == "active"
}
