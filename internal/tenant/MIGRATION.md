# Multi-Tenant Routing Migration Guide

This document summarizes the scaffolding created for the multi-tenant routing migration and provides guidance for implementation.

## Files Created

### 1. `/internal/cookie/cookie.go`
Domain-aware cookie helpers for subdomain routing.

**Key Types:**
- `Config` - Cookie configuration (BaseDomain, Secure)
- `NewConfig(baseDomain, secure)` - Constructor

**Key Functions to Implement:**
- `SetSession(w, name, value, maxAge)` - Set domain-scoped session cookie
- `ClearSession(w, name)` - Clear session cookie
- `SetSessionWithExpiry(w, name, value, expires)` - Set cookie with explicit expiry
- `Get(r, name)` - Get cookie value (already implemented)

**Cookie Constants:**
- `SessionCookieName` = "freyja_session"
- `CSRFCookieName` = "freyja_csrf"
- `CartCookieName` = "freyja_cart"
- `FlashCookieName` = "freyja_flash"

### 2. `/internal/middleware/tenant.go`
Tenant resolution middleware for subdomain/custom domain routing.

**Key Types:**
- `TenantConfig` - Middleware configuration (BaseDomain, AppDomain, Resolver, Logger)

**Key Functions to Implement:**
- `ResolveTenant(cfg)` - Middleware that resolves tenant from host
- `RequireTenant` - Middleware that requires tenant in context

**Helper Functions (implemented):**
- `extractSubdomain(host, baseDomain)` - Extract subdomain from host
- `stripPort(host)` - Remove port from host string
- `respondNotFound/ServiceUnavailable/InternalError` - Error responses

### 3. `/internal/service/tenant_pattern.go`
Documentation and helpers for service refactoring pattern.

**Key Functions:**
- `extractTenantID(ctx)` - Extract tenant ID from context
- `extractTenantIDStr(ctx)` - Extract tenant ID as string

**Pattern Documentation:**
- Shows before/after for service refactoring
- Lists all services that need updating

### 4. `/internal/worker/tenant_context.go`
Helpers for injecting tenant context in background jobs.

**Key Functions to Implement:**
- `withTenantContext(ctx, job)` - Create tenant context from job record
- `withTenantContextFromID(ctx, tenantID)` - Create tenant context from UUID

## Files Modified

### 1. `/internal/config.go`
- Renamed `MarketingDomain` to `BaseDomain` in `DomainConfig`
- Added `BASE_DOMAIN` environment variable
- Kept `MarketingDomain` for backwards compatibility
- Updated documentation comments

## Existing Files (Already Created)

### `/internal/tenant/context.go`
- `Tenant` struct
- `NewContext(ctx, tenant)` - Add tenant to context
- `FromContext(ctx)` - Get tenant from context
- `MustFromContext(ctx)` - Get tenant or panic
- `IDFromContext(ctx)` - Get tenant ID from context

### `/internal/tenant/resolver.go`
- `Resolver` interface
- `DBResolver` implementation
- `BySlug(ctx, slug)` - Resolve by subdomain slug
- `ByCustomDomain(ctx, domain)` - Resolve by custom domain
- `ByID(ctx, id)` - Resolve by UUID

### `/internal/tenant/errors.go`
- `ErrTenantNotFound`
- `ErrTenantInactive`
- `ErrNoTenant`
- `ErrCustomDomainNotActive`

## Services to Refactor

Each service needs to:
1. Remove `tenantID` field from struct
2. Remove `tenantID` parameter from constructor
3. Add `tenant.IDFromContext(ctx)` at start of each method
4. Return `tenant.ErrNoTenant` if tenant missing

### Service Files:
1. `/internal/postgres/product.go` - ProductService
2. `/internal/postgres/cart.go` - CartService
3. `/internal/postgres/user.go` - UserService
4. `/internal/service/order.go` - OrderService
5. `/internal/service/checkout.go` - CheckoutService
6. `/internal/service/subscription.go` - SubscriptionService
7. `/internal/service/invoice.go` - InvoiceService
8. `/internal/service/payment_terms.go` - PaymentTermsService

## Cookie Usage Updates

These handlers need to use the new cookie package:
1. `/internal/handler/storefront/auth.go`
2. `/internal/handler/storefront/cookies.go`
3. `/internal/handler/saas/auth.go`
4. `/internal/handler/admin/auth.go`
5. `/internal/middleware/csrf.go`
6. `/internal/middleware/operator.go`

## Integration Points

### 1. Operator Middleware Integration
The existing operator middleware (`/internal/middleware/operator.go`) uses its own `TenantContextKey` to store `*repository.Tenant`. This needs to be unified with the new tenant package.

**Current:**
```go
ctx = context.WithValue(r.Context(), TenantContextKey, &tenant)
```

**After Migration:**
```go
t := &tenant.Tenant{
    ID:     repoTenant.ID,
    Slug:   repoTenant.Slug,
    Name:   repoTenant.Name,
    Status: repoTenant.Status,
}
ctx = tenant.NewContext(r.Context(), t)
```

### 2. Worker Integration
Background jobs already have `tenant_id` in job records. Workers need to inject tenant context before calling services.

**Pattern:**
```go
func (w *Worker) processJob(ctx context.Context, job *repository.Job) error {
    tenantCtx, err := withTenantContext(ctx, job)
    if err != nil {
        return err
    }
    return w.service.DoSomething(tenantCtx)
}
```

### 3. Route Wiring (`/cmd/server/main.go`)
- Create `tenant.DBResolver` with queries
- Create `cookie.Config` with BaseDomain
- Apply `ResolveTenant` middleware to storefront routes
- Apply `RequireTenant` middleware where needed
- Remove service initialization with `cfg.TenantID`

## Testing Considerations

All service tests need to include tenant in context:

```go
func TestProductService_ListProducts(t *testing.T) {
    // Setup
    tenantID := pgtype.UUID{Bytes: [16]byte{...}, Valid: true}
    ctx := tenant.NewContext(context.Background(), &tenant.Tenant{
        ID:     tenantID,
        Status: "active",
    })

    // Test
    products, err := svc.ListProducts(ctx)
    // ...
}
```

## Implementation Order

1. **Phase 1: Foundation** (Complete)
   - [x] Tenant package (context.go, resolver.go, errors.go)
   - [x] Cookie package scaffolding
   - [x] Config updates (BaseDomain)
   - [x] Middleware scaffolding

2. **Phase 2: Implement Scaffolding**
   - [ ] Implement cookie.SetSession/ClearSession
   - [ ] Implement ResolveTenant middleware
   - [ ] Implement RequireTenant middleware

3. **Phase 3: Service Refactoring**
   - [ ] Refactor each service to use context-based tenant
   - [ ] Update tests to include tenant context

4. **Phase 4: Integration**
   - [ ] Update operator middleware to use tenant package
   - [ ] Update worker to inject tenant context
   - [ ] Update main.go route wiring

5. **Phase 5: Cookie Migration**
   - [ ] Update handlers to use cookie package
   - [ ] Update CSRF middleware
   - [ ] Update operator middleware cookie handling

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `HOST_ROUTING_ENABLED` | Enable multi-tenant subdomain routing | `true` |
| `BASE_DOMAIN` | Root domain for subdomain routing | `hiri.coffee` |
| `APP_DOMAIN` | SaaS application subdomain | `app.hiri.coffee` |
| `MARKETING_DOMAIN` | DEPRECATED - use BASE_DOMAIN | - |

## Development Testing

Use `lvh.me` for local testing (resolves to 127.0.0.1):

```bash
BASE_DOMAIN=lvh.me:3000
APP_DOMAIN=app.lvh.me:3000

# Test URLs:
# Marketing: http://lvh.me:3000
# App: http://app.lvh.me:3000
# Tenant: http://acme.lvh.me:3000
```
