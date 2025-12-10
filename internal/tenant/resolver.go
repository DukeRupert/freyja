package tenant

import (
	"context"

	"github.com/dukerupert/hiri/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Resolver resolves tenants from various identifiers.
type Resolver interface {
	// BySlug resolves a tenant by subdomain slug.
	BySlug(ctx context.Context, slug string) (*Tenant, error)

	// ByCustomDomain resolves a tenant by custom domain.
	// Only returns tenants with active custom domain status.
	ByCustomDomain(ctx context.Context, domain string) (*Tenant, error)

	// ByID resolves a tenant by ID.
	ByID(ctx context.Context, id pgtype.UUID) (*Tenant, error)
}

// DBResolver implements Resolver using database queries.
type DBResolver struct {
	queries *repository.Queries
}

// NewDBResolver creates a new database-backed tenant resolver.
func NewDBResolver(queries *repository.Queries) *DBResolver {
	return &DBResolver{queries: queries}
}

// BySlug resolves a tenant by subdomain slug.
func (r *DBResolver) BySlug(ctx context.Context, slug string) (*Tenant, error) {
	row, err := r.queries.GetTenantBySlug(ctx, slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}

	return &Tenant{
		ID:     row.ID,
		Slug:   row.Slug,
		Name:   row.Name,
		Status: row.Status,
	}, nil
}

// ByCustomDomain resolves a tenant by custom domain.
// Only returns tenants with active custom domain status.
func (r *DBResolver) ByCustomDomain(ctx context.Context, domain string) (*Tenant, error) {
	row, err := r.queries.GetTenantByCustomDomain(ctx, pgtype.Text{String: domain, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}

	// The query already filters for custom_domain_status = 'active',
	// but we still need to check tenant status
	return &Tenant{
		ID:     row.ID,
		Slug:   row.Slug,
		Name:   row.Name,
		Status: row.Status,
	}, nil
}

// ByID resolves a tenant by ID.
func (r *DBResolver) ByID(ctx context.Context, id pgtype.UUID) (*Tenant, error) {
	row, err := r.queries.GetTenantByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}

	return &Tenant{
		ID:     row.ID,
		Slug:   row.Slug,
		Name:   row.Name,
		Status: row.Status,
	}, nil
}

// Compile-time check that DBResolver implements Resolver.
var _ Resolver = (*DBResolver)(nil)
