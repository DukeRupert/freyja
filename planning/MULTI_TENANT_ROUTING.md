# Multi-Tenant Subdomain Routing Architecture

## Overview

Convert Freyja from single-tenant (hardcoded `TENANT_ID` at startup) to multi-tenant subdomain routing.

### Routing Structure

| Domain | Purpose |
|--------|---------|
| `hiri.coffee` | Marketing site |
| `app.hiri.coffee` | SaaS app (admin dashboard, signup, billing) |
| `{slug}.hiri.coffee` | Tenant storefronts (e.g., `acme.hiri.coffee`) |
| `shop.example.com` | Custom domains (upsell feature) |

## Architecture Decisions

### Decision 1: Tenant Resolution Strategy

**Choice: Context-Based Resolution**

Middleware resolves tenant from subdomain/custom domain and stores in request context. Services extract tenant ID from context at call time.

```go
// Middleware sets tenant
ctx = context.WithValue(ctx, TenantContextKey, tenant)

// Service extracts at call time
func (s *ProductService) ListProducts(ctx context.Context) ([]Product, error) {
    tenantID := tenant.IDFromContext(ctx)
    return s.repo.ListActiveProducts(ctx, tenantID)
}
```

**Rationale:**
- Aligns with existing `middleware.GetUserFromContext()` pattern
- Services remain single instance (memory efficient)
- Clean separation between routing and business logic
- Works naturally with htmx partials

### Decision 2: Service Initialization

**Choice: Stateless Services**

Remove `tenantID` field from services. Services extract tenant from context at runtime.

```go
type ProductService struct {
    repo repository.Querier
    // Note: tenantID field REMOVED
}

func (s *ProductService) ListProducts(ctx context.Context) ([]domain.ProductListItem, error) {
    tenantID := tenant.IDFromContext(ctx)
    if !tenantID.Valid {
        return nil, domain.ErrNoTenant
    }
    return s.repo.ListActiveProducts(ctx, tenantID)
}
```

**Rationale:**
- Simpler mental model: tenant always comes from context
- Forces proper middleware setup (missing middleware causes immediate failure)
- No ambiguity about tenant source

### Decision 3: Admin Dashboard URL Structure

**Choice: Session-Based Tenant**

Tenant determined by authenticated operator session. All admin URLs are the same regardless of tenant.

```
app.hiri.coffee/admin/dashboard
app.hiri.coffee/admin/products
```

**Rationale:**
- Already how operator middleware works
- Better security (no URL manipulation risk)
- Clean URLs
- Supports future multi-tenant operators with tenant picker

### Decision 4: Signup Flow

**Choice: Create Pending Tenant → Stripe Checkout → Webhook Activates**

```
1. User fills form on hiri.coffee/signup (name, email, slug)
2. API validates slug availability
3. API creates tenant with status='pending' + reserves slug
4. Redirect to Stripe Checkout with tenant_id in metadata
5. Stripe checkout.session.completed webhook fires
6. Webhook updates tenant status='active'
7. Webhook creates operator account + sends welcome email
8. User redirected to app.hiri.coffee/welcome (set password)
```

**Rationale:**
- Slug reserved immediately (no race condition)
- Better error recovery (tenant exists, just not active)
- Enables future trial periods if needed

### Decision 5: Development Workflow

**Choice: lvh.me Domain**

`*.lvh.me` resolves to 127.0.0.1. Use `acme.lvh.me:3000` for local testing.

```bash
# Development
BASE_DOMAIN=lvh.me:3000

# Test URLs
# Marketing: lvh.me:3000
# App: app.lvh.me:3000
# Tenant: acme.lvh.me:3000
```

**Rationale:**
- Zero setup required
- Works immediately for most developers
- Fallback: `/etc/hosts` entries for offline development

## Configuration Decisions

| Setting | Value | Rationale |
|---------|-------|-----------|
| Tenant caching | None (MVP) | Simple, always fresh. Add caching if performance becomes an issue |
| Multi-tenant operators | No (1:1) | Each operator belongs to exactly one tenant |
| Trial period | None | Payment required upfront to create tenant |

### Reserved Slugs

The following slugs are blocked from tenant registration:

```
www, app, api, admin, mail, smtp, ftp, static, assets, cdn,
status, help, support, docs, blog, news, shop, store, my,
account, login, signup, register, auth, oauth, callback
```

## Interface Definitions

### Tenant Package

```go
// /internal/tenant/context.go

package tenant

type Tenant struct {
    ID     pgtype.UUID
    Slug   string
    Name   string
    Status string // active, pending, suspended, cancelled
}

// NewContext returns a context with tenant attached
func NewContext(ctx context.Context, t *Tenant) context.Context

// FromContext extracts tenant from context, returns nil if not present
func FromContext(ctx context.Context) *Tenant

// MustFromContext extracts tenant, panics if not present
func MustFromContext(ctx context.Context) *Tenant

// IDFromContext returns tenant ID, or zero UUID if not present
func IDFromContext(ctx context.Context) pgtype.UUID
```

### Tenant Resolver

```go
// /internal/tenant/resolver.go

type Resolver interface {
    BySlug(ctx context.Context, slug string) (*Tenant, error)
    ByCustomDomain(ctx context.Context, domain string) (*Tenant, error)
    ByID(ctx context.Context, id pgtype.UUID) (*Tenant, error)
}
```

### Middleware

```go
// /internal/middleware/tenant.go

type TenantConfig struct {
    BaseDomain   string          // e.g., "hiri.coffee"
    AppSubdomain string          // e.g., "app"
    Resolver     tenant.Resolver
}

// ResolveTenant resolves tenant from request host
func ResolveTenant(cfg TenantConfig) func(http.Handler) http.Handler

// RequireTenant returns 404 if no tenant in context
func RequireTenant(next http.Handler) http.Handler
```

## Implementation Plan

### Phase 1: Foundation

**Create tenant package** (`/internal/tenant/`)
- `context.go` - Context helpers (NewContext, FromContext, MustFromContext)
- `resolver.go` - Resolver interface and DBResolver implementation
- `errors.go` - Domain errors (ErrTenantNotFound, ErrTenantInactive)

**Update middleware** (`/internal/middleware/`)
- Rename/update `custom_domain.go` to `tenant.go`
- Update `ResolveTenantByHost` to use new tenant package
- Add `RequireTenant` middleware

**Update config** (`/internal/config.go`)
- Add `BaseDomain` to `DomainConfig`
- Deprecate root-level `TenantID` (keep for backwards compatibility)

### Phase 2: Service Refactoring

**Refactor services to extract tenant from context:**
- `/internal/postgres/product.go`
- `/internal/postgres/cart.go`
- `/internal/postgres/user.go`
- `/internal/service/order.go`
- `/internal/service/checkout.go`
- `/internal/service/subscription.go`
- `/internal/service/invoice.go`
- `/internal/service/payment_terms.go`

**Pattern for each service:**
1. Remove `tenantID` field from struct
2. Remove `tenantID` parameter from constructor
3. Add `tenant.IDFromContext(ctx)` call at start of each method
4. Return `domain.ErrNoTenant` if tenant missing

### Phase 3: Route Wiring

**Update main.go** (`/cmd/server/main.go`)
- Create tenant resolver with database queries
- Update host router to handle `{slug}.hiri.coffee` pattern
- Apply `ResolveTenant` middleware to storefront routes
- Remove service initialization with `cfg.TenantID`

**Update route registration** (`/internal/routes/`)
- `storefront.go` - Add `RequireTenant` middleware
- `admin.go` - Keep operator-based tenant (already works)
- `webhook.go` - Extract tenant from Stripe metadata

### Phase 4: Signup Flow

**Create signup handler** (`/internal/handler/saas/signup.go`)
- `GET /signup` - Render signup form
- `POST /api/validate-slug` - Check slug availability
- `POST /signup` - Create pending tenant, redirect to Stripe

**Update webhook handler** (`/internal/handler/webhook/stripe_saas.go`)
- Handle `checkout.session.completed` for new tenants
- Activate tenant (status = 'active')
- Create operator account
- Send welcome email with password setup link

**Create welcome flow** (`/internal/handler/saas/welcome.go`)
- `GET /welcome?token=...` - Password setup form
- `POST /welcome` - Set password, redirect to admin

### Phase 5: Testing and Cleanup

- Update all service tests to include tenant in context
- Add integration tests for subdomain routing
- Add integration tests for custom domain routing
- Remove deprecated `cfg.TenantID` usage
- Update CLAUDE.md with multi-tenant patterns

## Files Affected

### New Files
- `/internal/tenant/context.go`
- `/internal/tenant/resolver.go`
- `/internal/tenant/errors.go`
- `/internal/handler/saas/signup.go`
- `/internal/handler/saas/welcome.go`

### Modified Files
- `/internal/config.go`
- `/internal/middleware/tenant.go` (rename from custom_domain.go)
- `/internal/postgres/product.go`
- `/internal/postgres/cart.go`
- `/internal/postgres/user.go`
- `/internal/service/order.go`
- `/internal/service/checkout.go`
- `/internal/service/subscription.go`
- `/internal/service/invoice.go`
- `/internal/service/payment_terms.go`
- `/internal/routes/storefront.go`
- `/internal/handler/webhook/stripe.go`
- `/cmd/server/main.go`

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Tenant data leak (wrong tenant shown) | Medium | Critical | Fail-fast if tenant missing; integration tests; audit logging |
| Performance regression (per-request DB lookup) | Low | Medium | Benchmark; add caching if needed |
| Breaking existing single-tenant deployment | Medium | High | Feature flag (`HOST_ROUTING_ENABLED`); gradual rollout |
| Slug collisions during signup | Low | Low | Database unique constraint; validation before checkout |
| Cookie scope issues with subdomains | Medium | Medium | Use `.hiri.coffee` domain for session cookies |

## Migration Strategy

### Feature Flag

```go
// Enable with HOST_ROUTING_ENABLED=true
if cfg.Domain.HostRouting {
    // Multi-tenant subdomain routing
} else {
    // Legacy single-tenant mode (uses cfg.TenantID)
}
```

### Backwards Compatibility

- `TENANT_ID` env var continues to work for 2 releases
- Existing single-tenant deployments unaffected
- New multi-tenant: Enable `HOST_ROUTING_ENABLED=true`

## Cookie Configuration

To support subdomain routing, session cookies must be scoped to the base domain:

```go
// Session cookie configuration
cookie.Domain = ".hiri.coffee"  // Note leading dot
cookie.SameSite = http.SameSiteLaxMode
cookie.Secure = true  // Production only
```

This allows:
- User logged in at `acme.hiri.coffee` stays logged in
- Cookie shared across `app.hiri.coffee` for admin access
- Marketing site (`hiri.coffee`) can read auth state
