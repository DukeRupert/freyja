# Multi-Tenant Subdomain Routing Architecture

## Overview

Convert Freyja from single-tenant (hardcoded `TENANT_ID` at startup) to multi-tenant subdomain routing.

### Routing Structure

| Domain | Purpose |
|--------|---------|
| `hiri.coffee` | Marketing site (served at apex/BaseDomain) |
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
- Already how operator middleware works (`TenantOperator` has `TenantID` field)
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

**Note:** Signup flow is deferred to a separate milestone (see Implementation Plan).

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

### Decision 6: Domain Configuration

**Choice: BaseDomain as Primary Config**

Rename `MarketingDomain` to `BaseDomain`. The marketing site is implicitly served at the apex domain.

```go
type DomainConfig struct {
    HostRouting bool   // Enable host-based routing
    BaseDomain  string // e.g., "hiri.coffee" - apex domain, also serves marketing site
    AppDomain   string // e.g., "app.hiri.coffee" - SaaS admin/signup
}
```

**Rationale:**
- `BaseDomain` describes what the value *is* (base for subdomain routing)
- Marketing site is always at apex domain, no need for separate field
- Clearer naming

### Decision 7: Special Subdomain Handling

**Choice: Config-Based AppDomain Check + Hardcoded www Redirect**

Before tenant resolution, check if host matches `AppDomain` config. Also handle `www` with a redirect.

```go
// In middleware
if host == cfg.AppDomain {
    // SaaS app routes, skip tenant resolution
    next.ServeHTTP(w, r)
    return
}

if subdomain == "www" {
    // Redirect www.hiri.coffee → hiri.coffee
    http.Redirect(w, r, "https://"+cfg.BaseDomain+r.URL.Path, http.StatusMovedPermanently)
    return
}
```

**Rationale:**
- Prevents DB lookup for `app.hiri.coffee` on every request
- DNS CNAME should handle `www`, but redirect is cheap fallback
- Reserved slugs list still provides defense-in-depth

### Decision 8: Tenant Status Enforcement

**Choice: Block Inactive Tenants in Middleware**

Check tenant status immediately after resolution. Return appropriate HTTP responses for non-active statuses.

```go
// In ResolveTenantByHost, after successful tenant lookup
switch tenant.Status {
case "active":
    // Continue normally
case "pending":
    // Storefront doesn't exist yet
    respondNotFound(w, r)
    return
case "suspended":
    // Temporarily unavailable
    respondServiceUnavailable(w, r, "This store is temporarily unavailable")
    return
case "cancelled":
    // Storefront no longer exists
    respondNotFound(w, r)
    return
}
```

**Rationale:**
- Single enforcement point prevents accidental data leakage
- Different statuses get appropriate HTTP responses
- Downstream code never sees inactive tenants

### Decision 9: Cookie Domain Scoping

**Choice: Centralized Cookie Package**

Create `internal/cookie/` package with domain-aware helpers. All session/auth cookies go through this package.

```go
// internal/cookie/cookie.go
package cookie

type Config struct {
    BaseDomain string // e.g., "hiri.coffee" or "lvh.me"
    Secure     bool   // true in production
}

func (c *Config) SetSession(w http.ResponseWriter, name, value string, maxAge int) {
    http.SetCookie(w, &http.Cookie{
        Name:     name,
        Value:    value,
        Domain:   "." + c.BaseDomain, // Scoped to all subdomains
        Path:     "/",
        MaxAge:   maxAge,
        HttpOnly: true,
        Secure:   c.Secure,
        SameSite: http.SameSiteLaxMode,
    })
}

func (c *Config) ClearSession(w http.ResponseWriter, name string) {
    http.SetCookie(w, &http.Cookie{
        Name:     name,
        Value:    "",
        Domain:   "." + c.BaseDomain,
        Path:     "/",
        MaxAge:   -1,
        HttpOnly: true,
    })
}
```

**Rationale:**
- Centralized cookie creation ensures consistent domain scoping
- Domain configurable for dev (`lvh.me`) vs prod (`hiri.coffee`)
- Handlers don't need to know about subdomain routing

### Decision 10: Background Job Tenant Context

**Choice: Worker Injects Tenant into Context**

Background jobs already store `tenant_id` in the job record. Worker creates tenant context before calling services.

```go
// In worker, before calling service
func (w *Worker) processInvoiceJob(ctx context.Context, job *repository.Job) error {
    // Create context with tenant from job record
    tenantCtx := tenant.NewContext(ctx, &tenant.Tenant{
        ID: job.TenantID,
    })

    // Services extract tenant from context as normal
    _, err := w.invoiceService.GenerateConsolidatedInvoice(tenantCtx, params)
    return err
}
```

**Rationale:**
- Job system already has `tenant_id` on every job
- Services use same pattern regardless of HTTP vs background context
- No special cases needed in service code

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
account, login, signup, register, auth, oauth, callback,
test, demo, staging
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

### Cookie Package

```go
// /internal/cookie/cookie.go

package cookie

type Config struct {
    BaseDomain string
    Secure     bool
}

func (c *Config) SetSession(w http.ResponseWriter, name, value string, maxAge int)
func (c *Config) ClearSession(w http.ResponseWriter, name string)
```

### Middleware

```go
// /internal/middleware/tenant.go

type TenantConfig struct {
    BaseDomain string          // e.g., "hiri.coffee"
    AppDomain  string          // e.g., "app.hiri.coffee"
    Resolver   tenant.Resolver
}

// ResolveTenant resolves tenant from request host
// - Skips resolution for AppDomain
// - Redirects www to BaseDomain
// - Blocks inactive tenants with appropriate HTTP responses
func ResolveTenant(cfg TenantConfig) func(http.Handler) http.Handler

// RequireTenant returns 404 if no tenant in context
func RequireTenant(next http.Handler) http.Handler
```

## Implementation Plan

### Milestone 1: Multi-Tenant Routing (This PR)

#### Phase 1: Foundation

**Create tenant package** (`/internal/tenant/`)
- `context.go` - Context helpers (NewContext, FromContext, MustFromContext)
- `resolver.go` - Resolver interface and DBResolver implementation
- `errors.go` - Domain errors (ErrTenantNotFound, ErrTenantInactive)

**Create cookie package** (`/internal/cookie/`)
- `cookie.go` - Domain-aware cookie helpers (SetSession, ClearSession)

**Update config** (`/internal/config.go`)
- Rename `MarketingDomain` to `BaseDomain` in `DomainConfig`
- Deprecate root-level `TenantID` (keep for backwards compatibility)

**Update middleware** (`/internal/middleware/`)
- Rename/update `custom_domain.go` to `tenant.go`
- Update `ResolveTenantByHost` to use new tenant package
- Add AppDomain check and www redirect
- Add tenant status enforcement
- Add `RequireTenant` middleware

#### Phase 2: Service Refactoring

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

**Update cookie usage:**
- `/internal/handler/storefront/auth.go`
- `/internal/handler/storefront/cookies.go`
- `/internal/handler/saas/auth.go`
- `/internal/handler/admin/auth.go`
- `/internal/middleware/csrf.go`
- `/internal/middleware/operator.go`

#### Phase 3: Route Wiring

**Update main.go** (`/cmd/server/main.go`)
- Create tenant resolver with database queries
- Create cookie config with BaseDomain
- Update host router to handle `{slug}.hiri.coffee` pattern
- Apply `ResolveTenant` middleware to storefront routes
- Remove service initialization with `cfg.TenantID`

**Update route registration** (`/internal/routes/`)
- `storefront.go` - Add `RequireTenant` middleware
- `admin.go` - Keep operator-based tenant (already works)
- `webhook.go` - Extract tenant from Stripe metadata

**Update worker** (`/internal/worker/worker.go`)
- Inject tenant context from job record before calling services

#### Phase 4: Testing and Cleanup

- Update all service tests to include tenant in context
- Add integration tests for subdomain routing
- Add integration tests for custom domain routing
- Remove deprecated `cfg.TenantID` usage
- Update CLAUDE.md with multi-tenant patterns

### Milestone 2: Self-Serve Signup (Future PR)

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

## Files Affected

### New Files
- `/internal/tenant/context.go`
- `/internal/tenant/resolver.go`
- `/internal/tenant/errors.go`
- `/internal/cookie/cookie.go`

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
- `/internal/handler/storefront/auth.go`
- `/internal/handler/storefront/cookies.go`
- `/internal/handler/saas/auth.go`
- `/internal/handler/admin/auth.go`
- `/internal/middleware/csrf.go`
- `/internal/middleware/operator.go`
- `/internal/worker/worker.go`
- `/internal/routes/storefront.go`
- `/internal/handler/webhook/stripe.go`
- `/cmd/server/main.go`

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Tenant data leak (wrong tenant shown) | Medium | Critical | Fail-fast if tenant missing; status check in middleware; integration tests; audit logging |
| Performance regression (per-request DB lookup) | Low | Medium | Benchmark; add caching if needed |
| Breaking existing single-tenant deployment | Medium | High | Feature flag (`HOST_ROUTING_ENABLED`); gradual rollout |
| Slug collisions during signup | Low | Low | Database unique constraint; validation before checkout |
| Cookie scope issues with subdomains | Medium | Medium | Centralized cookie package with domain scoping |

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
// Using the cookie package
cookieCfg := &cookie.Config{
    BaseDomain: cfg.Domain.BaseDomain, // "hiri.coffee" or "lvh.me"
    Secure:     cfg.Env == "prod",
}

// Sets cookie with Domain=".hiri.coffee"
cookieCfg.SetSession(w, "freyja_session", token, 30*24*60*60)
```

This allows:
- User logged in at `acme.hiri.coffee` stays logged in
- Cookie shared across `app.hiri.coffee` for admin access
- Marketing site (`hiri.coffee`) can read auth state
