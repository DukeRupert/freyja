package tenant

import "errors"

var (
	// ErrTenantNotFound is returned when a tenant cannot be found by slug or custom domain.
	ErrTenantNotFound = errors.New("tenant not found")

	// ErrTenantInactive is returned when a tenant exists but is not in active status.
	ErrTenantInactive = errors.New("tenant is not active")

	// ErrNoTenant is returned when tenant context is required but not present.
	ErrNoTenant = errors.New("no tenant in context")

	// ErrCustomDomainNotActive is returned when a custom domain exists but is not verified/active.
	ErrCustomDomainNotActive = errors.New("custom domain not active")
)
