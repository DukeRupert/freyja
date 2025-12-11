package admin

import (
	"context"

	"github.com/dukerupert/hiri/internal/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// getTenantID extracts the tenant ID from the request context (set by operator middleware)
// and converts it to pgtype.UUID for database queries.
// Returns a zero-value pgtype.UUID if no tenant is in context.
func getTenantID(ctx context.Context) pgtype.UUID {
	tenantUUID := middleware.GetTenantIDFromOperator(ctx)
	if tenantUUID == uuid.Nil {
		return pgtype.UUID{}
	}

	return pgtype.UUID{
		Bytes: tenantUUID,
		Valid: true,
	}
}

// hasTenantContext checks if a valid tenant ID is present in the context.
func hasTenantContext(ctx context.Context) bool {
	return middleware.GetTenantIDFromOperator(ctx) != uuid.Nil
}
