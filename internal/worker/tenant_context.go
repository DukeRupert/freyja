// Package worker provides background job processing for Freyja.
//
// This file contains helpers for injecting tenant context into background jobs.
// When services are refactored to extract tenant from context, workers must
// create tenant context before calling service methods.
package worker

import (
	"context"

	"github.com/dukerupert/hiri/internal/repository"
	"github.com/dukerupert/hiri/internal/tenant"
	"github.com/jackc/pgx/v5/pgtype"
)

// withTenantContext creates a new context with tenant information from a job record.
//
// Background jobs store tenant_id in the job record. Before calling any service
// method, the worker must inject this tenant into the context so services can
// extract it using tenant.IDFromContext().
//
// Usage in processJob:
//
//	func (w *Worker) processJob(ctx context.Context, job *repository.Job) error {
//	    // Create context with tenant from job
//	    tenantCtx, err := withTenantContext(ctx, job)
//	    if err != nil {
//	        return fmt.Errorf("failed to create tenant context: %w", err)
//	    }
//
//	    // Now call service methods - they extract tenant from context
//	    _, err = w.invoiceService.GenerateConsolidatedInvoice(tenantCtx, params)
//	    return err
//	}
//
// TODO: Implement tenant context injection
//   - Extract tenant_id from job record
//   - Create tenant.Tenant with minimal info (ID is sufficient for most operations)
//   - If full tenant info is needed, resolve via tenant.Resolver
//   - Call tenant.NewContext to inject into context
func withTenantContext(ctx context.Context, job *repository.Job) (context.Context, error) {
	// TODO: Implement
	//
	// Option 1: Minimal tenant (just ID) - sufficient for most jobs
	// if !job.TenantID.Valid {
	//     return ctx, tenant.ErrNoTenant
	// }
	// t := &tenant.Tenant{
	//     ID:     job.TenantID,
	//     Status: "active", // Assume active for background jobs
	// }
	// return tenant.NewContext(ctx, t), nil
	//
	// Option 2: Full tenant resolution (if job needs tenant name, slug, etc.)
	// t, err := w.tenantResolver.ByID(ctx, job.TenantID)
	// if err != nil {
	//     return nil, err
	// }
	// return tenant.NewContext(ctx, t), nil

	panic("withTenantContext not implemented")
}

// withTenantContextFromID creates tenant context from a raw UUID.
// Useful when tenant ID comes from sources other than job records.
//
// TODO: Implement
func withTenantContextFromID(ctx context.Context, tenantID pgtype.UUID) (context.Context, error) {
	if !tenantID.Valid {
		return ctx, tenant.ErrNoTenant
	}

	t := &tenant.Tenant{
		ID:     tenantID,
		Status: "active", // Assume active for background processing
	}

	return tenant.NewContext(ctx, t), nil
}

// MIGRATION NOTES FOR worker.go:
//
// After services are refactored to use context-based tenant resolution,
// update the worker's processJob method as follows:
//
// BEFORE (current - services have tenantID field):
//
//	func (w *Worker) processInvoiceJob(ctx context.Context, job *repository.Job) error {
//	    // Service was initialized with tenant ID - no context injection needed
//	    _, err := w.invoiceService.GenerateConsolidatedInvoice(ctx, params)
//	    return err
//	}
//
// AFTER (new - services extract tenant from context):
//
//	func (w *Worker) processInvoiceJob(ctx context.Context, job *repository.Job) error {
//	    // Inject tenant context before calling service
//	    tenantCtx, err := withTenantContext(ctx, job)
//	    if err != nil {
//	        return fmt.Errorf("failed to create tenant context: %w", err)
//	    }
//
//	    _, err = w.invoiceService.GenerateConsolidatedInvoice(tenantCtx, params)
//	    return err
//	}
//
// The key changes:
// 1. Call withTenantContext at the start of processJob (or each job handler)
// 2. Pass tenantCtx to all service method calls
// 3. Services extract tenant using tenant.IDFromContext(ctx)
//
// This ensures:
// - Consistent tenant handling between HTTP requests and background jobs
// - Services remain stateless and testable
// - Tenant context is explicitly passed rather than implicitly stored
