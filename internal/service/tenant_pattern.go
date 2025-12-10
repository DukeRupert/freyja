// Package service contains business logic services for Freyja.
//
// This file documents the pattern for refactoring services to extract tenant from context
// instead of storing tenantID as a struct field.
//
// MIGRATION GUIDE: Converting services to context-based tenant resolution
//
// Before (current pattern - tenantID stored in struct):
//
//	type ProductService struct {
//	    repo     repository.Querier
//	    tenantID pgtype.UUID  // <-- Remove this field
//	}
//
//	func NewProductService(repo repository.Querier, tenantID string) (*ProductService, error) {
//	    var tenantUUID pgtype.UUID
//	    if err := tenantUUID.Scan(tenantID); err != nil {
//	        return nil, fmt.Errorf("invalid tenant ID: %w", err)
//	    }
//	    return &ProductService{
//	        repo:     repo,
//	        tenantID: tenantUUID,  // <-- Remove this
//	    }, nil
//	}
//
//	func (s *ProductService) ListProducts(ctx context.Context) ([]Product, error) {
//	    return s.repo.ListActiveProducts(ctx, s.tenantID)  // <-- Uses stored tenantID
//	}
//
// After (new pattern - tenant from context):
//
//	type ProductService struct {
//	    repo repository.Querier
//	    // Note: tenantID field REMOVED - extracted from context at runtime
//	}
//
//	func NewProductService(repo repository.Querier) *ProductService {
//	    return &ProductService{repo: repo}
//	}
//
//	func (s *ProductService) ListProducts(ctx context.Context) ([]Product, error) {
//	    tenantID := tenant.IDFromContext(ctx)
//	    if !tenantID.Valid {
//	        return nil, tenant.ErrNoTenant
//	    }
//	    return s.repo.ListActiveProducts(ctx, tenantID)
//	}
//
// SERVICES TO REFACTOR:
//
// The following services need to be updated to extract tenant from context:
//
// 1. /internal/postgres/product.go - ProductService
//    - Remove tenantID field
//    - Update NewProductService to not require tenantID
//    - Add tenant.IDFromContext call at start of each method
//
// 2. /internal/postgres/cart.go - CartService
//    - Same pattern as ProductService
//
// 3. /internal/postgres/user.go - UserService
//    - Same pattern as ProductService
//
// 4. /internal/service/order.go - OrderService
//    - Same pattern as ProductService
//
// 5. /internal/service/checkout.go - CheckoutService
//    - Same pattern as ProductService
//
// 6. /internal/service/subscription.go - SubscriptionService
//    - Same pattern as ProductService
//
// 7. /internal/service/invoice.go - InvoiceService
//    - Same pattern as ProductService
//
// 8. /internal/service/payment_terms.go - PaymentTermsService
//    - Same pattern as ProductService
//
// IMPORTANT NOTES:
//
// 1. Always check tenantID.Valid before using it
//    - Returns tenant.ErrNoTenant if context doesn't have tenant
//    - This forces proper middleware setup
//
// 2. For admin routes, tenant comes from operator session
//    - Operator middleware sets tenant in context from TenantOperator.TenantID
//    - Service code is identical - just extracts from context
//
// 3. For background jobs, worker injects tenant context
//    - Worker reads tenant_id from job record
//    - Creates context with tenant before calling service
//    - Service code is identical - just extracts from context
//
// 4. Testing requires tenant in context
//    - Update tests to use tenant.NewContext
//    - Example: ctx := tenant.NewContext(context.Background(), &tenant.Tenant{ID: testTenantID})
package service

import (
	"context"

	"github.com/dukerupert/hiri/internal/tenant"
	"github.com/jackc/pgx/v5/pgtype"
)

// ExtractTenantID is a helper that extracts tenant ID from context.
// Returns tenant.ErrNoTenant if no tenant in context.
//
// This is the standard pattern for service methods:
//
//	func (s *MyService) DoSomething(ctx context.Context) error {
//	    tenantID, err := service.ExtractTenantID(ctx)
//	    if err != nil {
//	        return err
//	    }
//	    // Use tenantID in queries...
//	}
func ExtractTenantID(ctx context.Context) (pgtype.UUID, error) {
	tenantID := tenant.IDFromContext(ctx)
	if !tenantID.Valid {
		return pgtype.UUID{}, tenant.ErrNoTenant
	}
	return tenantID, nil
}

// ExtractTenantIDStr is a helper that extracts tenant ID as string from context.
// Useful when services need to pass tenant ID to external providers (Stripe, etc.).
func ExtractTenantIDStr(ctx context.Context) (string, error) {
	t := tenant.FromContext(ctx)
	if t == nil {
		return "", tenant.ErrNoTenant
	}
	if !t.ID.Valid {
		return "", tenant.ErrNoTenant
	}
	// Format UUID as string
	b := t.ID.Bytes
	return formatUUID(b), nil
}

// formatUUID formats a UUID byte array as a string.
func formatUUID(b [16]byte) string {
	return string(b[0:4]) + "-" + string(b[4:6]) + "-" + string(b[6:8]) + "-" + string(b[8:10]) + "-" + string(b[10:16])
}

// ExampleRefactoredService demonstrates the new pattern.
// This is not a real service - just documentation.
type ExampleRefactoredService struct {
	// Note: NO tenantID field
	// repo repository.Querier
}

// DoSomething shows the pattern for extracting tenant from context.
func (s *ExampleRefactoredService) DoSomething(ctx context.Context) error {
	tenantID, err := ExtractTenantID(ctx)
	if err != nil {
		return err // Returns tenant.ErrNoTenant
	}

	// Now use tenantID in queries
	_ = tenantID
	return nil
}
