# Freyja Architecture Overview

This document provides a quick reference guide to the Freyja e-commerce platform architecture. For detailed technical decisions and rationale, see [TECHNICAL.md](./TECHNICAL.md).

---

## System Overview

Freyja is a multi-tenant SaaS e-commerce platform built for small coffee roasters. Each tenant (coffee roaster) gets their own storefront accessible via subdomain (`acme.freyja.app`) or custom domain (`shop.acmecoffee.com`).

```
                                    +------------------+
                                    |     Caddy        |
                                    | (TLS + Routing)  |
                                    +--------+---------+
                                             |
              +------------------------------+------------------------------+
              |                              |                              |
    +---------v---------+        +-----------v-----------+       +----------v----------+
    |   SaaS Marketing  |        |   Tenant Storefronts  |       |    Admin Dashboards |
    |   (freyja.app)    |        | (*.freyja.app + custom)|      |   (/admin routes)   |
    +-------------------+        +-----------+-----------+       +----------+----------+
                                             |                              |
                                             +------------------------------+
                                                          |
                                             +------------v------------+
                                             |      Go Application     |
                                             |   (net/http + Router)   |
                                             +------------+------------+
                                                          |
              +-------------------+-------------------+----+----+-------------------+
              |                   |                   |         |                   |
    +---------v-------+  +--------v--------+  +-------v-------+ | +----------------v---------+
    |   PostgreSQL    |  |     Stripe      |  |   EasyPost    | | |      Cloudflare R2       |
    | (Data + Jobs)   |  |  (Payments)     |  |  (Shipping)   | | |    (File Storage)        |
    +-----------------+  +-----------------+  +---------------+ | +-------------------------+
                                                                |
                                                       +--------v--------+
                                                       | Postmark (Email)|
                                                       +-----------------+
```

---

## Multi-Tenancy Model

Every table with tenant-scoped data includes a `tenant_id` column. Tenant isolation is enforced at multiple layers:

```
Request Flow:

  HTTP Request ──> Middleware ──> Handler ──> Service ──> Repository ──> PostgreSQL
       │               │             │           │             │             │
       │          ResolveTenant      │      TenantID      tenant_id      WHERE
       │          (from host)        │      (context)     (param)      tenant_id = ?
       │               │             │           │             │             │
       └───────────────┴─────────────┴───────────┴─────────────┴─────────────┘
                          Tenant ID flows through entire request
```

**Key Components:**

| Layer | Responsibility |
|-------|---------------|
| `middleware/tenant.go` | Resolves tenant from subdomain or custom domain |
| `domain/context.go` | Context helpers (`TenantIDFromContext`, `RequireTenantID`) |
| `repository/*.sql.go` | All queries include `tenant_id` parameter |

---

## Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           HTTP Layer                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Handlers    │  │  Middleware  │  │   Router     │              │
│  │  (internal/  │  │  (auth,      │  │  (thin wrap  │              │
│  │   handler/)  │  │   tenant,    │  │   over       │              │
│  │              │  │   logging)   │  │   ServeMux)  │              │
│  └──────┬───────┘  └──────────────┘  └──────────────┘              │
│         │                                                           │
├─────────┼───────────────────────────────────────────────────────────┤
│         │              Service Layer                                │
│  ┌──────▼───────────────────────────────────────────────────────┐  │
│  │                    internal/service/                          │  │
│  │  Business logic, orchestration, cross-cutting concerns        │  │
│  │  Examples: OperatorService, AccountService, FulfillmentService│  │
│  └──────┬───────────────────────────────────────────────────────┘  │
│         │                                                           │
├─────────┼───────────────────────────────────────────────────────────┤
│         │              Domain Layer                                 │
│  ┌──────▼───────────────────────────────────────────────────────┐  │
│  │                    internal/domain/                           │  │
│  │  Core types, interfaces, domain errors                        │  │
│  │  Examples: Product, Order, Subscription, User                 │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                     Infrastructure Layer                            │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐   │
│  │ repository │  │  billing   │  │  shipping  │  │   email    │   │
│  │  (sqlc)    │  │  (Stripe)  │  │ (EasyPost) │  │ (Postmark) │   │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘   │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐                   │
│  │  storage   │  │    tax     │  │   jobs     │                   │
│  │   (R2)     │  │ (Stripe/%) │  │  (DB Queue)│                   │
│  └────────────┘  └────────────┘  └────────────┘                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

| Layer | Location | Responsibilities |
|-------|----------|-----------------|
| **Handler** | `internal/handler/` | HTTP request/response, form parsing, template rendering |
| **Service** | `internal/service/` | Business logic, transaction coordination, external service calls |
| **Domain** | `internal/domain/` | Core types, interfaces, domain rules, errors |
| **Repository** | `internal/repository/` | Database access (sqlc-generated), type-safe queries |
| **Infrastructure** | `internal/{billing,shipping,email,storage}/` | External service integrations behind interfaces |

---

## Handler Organization

```
internal/handler/
├── admin/              # Admin dashboard (authenticated operators)
│   ├── auth.go         # Operator login/logout
│   ├── customers.go    # Customer management
│   ├── dashboard.go    # Main dashboard
│   ├── invoices.go     # Invoice management
│   ├── orders.go       # Order fulfillment
│   ├── products.go     # Product catalog CRUD
│   └── ...
├── storefront/         # Customer-facing (tenant storefronts)
│   ├── auth.go         # Customer signup/login
│   ├── cart.go         # Shopping cart
│   ├── checkout.go     # Checkout flow
│   ├── home.go         # Homepage
│   ├── products.go     # Product browsing
│   └── wholesale.go    # B2B ordering
├── saas/               # SaaS platform (freyja.app)
│   ├── auth.go         # Operator signup
│   ├── billing.go      # Platform subscription
│   ├── landing.go      # Marketing pages
│   └── setup.go        # Onboarding wizard
├── webhook/            # Webhook handlers
│   └── stripe.go       # Stripe events
├── api/                # Internal APIs
│   └── domain_validation.go
├── error.go            # Error response handling
└── renderer.go         # Template rendering
```

---

## Middleware Stack

Middleware is applied in order using the custom router wrapper:

```go
// internal/router/router.go - Chi-like API over net/http
r := router.New(
    router.Recovery(logger),    // Panic recovery
    router.Logger(logger),      // Request logging
)

// Route groups with middleware
r.Group(func(r *router.Router) {
    r.Use(middleware.ResolveTenant(cfg))  // Tenant from host
    r.Use(middleware.RequireTenant)       // Ensure tenant exists
    // ... storefront routes
})
```

### Available Middleware

| Middleware | File | Purpose |
|------------|------|---------|
| `ResolveTenant` | `middleware/tenant.go` | Extract tenant from subdomain/custom domain |
| `RequireTenant` | `middleware/tenant.go` | Block requests without tenant context |
| `WithOperator` | `middleware/operator.go` | Load operator from session (optional) |
| `RequireOperator` | `middleware/operator.go` | Require authenticated operator |
| `RequireActiveTenant` | `middleware/operator.go` | Check tenant subscription status |
| `RequireOwner` | `middleware/operator.go` | Require owner role |
| `Logger` | `router/middleware.go` | Request logging |
| `Recovery` | `router/middleware.go` | Panic recovery |
| `RateLimit` | `middleware/ratelimit.go` | Per-IP/per-user rate limiting |
| `CSRF` | `middleware/csrf.go` | CSRF token validation |

---

## Domain Model

### Core Entities and Relationships

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     Tenant      │────<│      User       │────<│    Address      │
│  (coffee roaster)│     │   (customer)    │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │                      │
         │                      │
         ▼                      ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Product      │────<│   ProductSKU    │────<│ PriceListEntry  │
│  (coffee type)  │     │ (weight+grind)  │     │  (per-list $)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │                      │                        │
         │                      │                        │
         ▼                      ▼                        ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  ProductImage   │     │   CartItem      │────>│   PriceList     │
│                 │     │                 │     │ (Retail, Tier1) │
└─────────────────┘     └────────┬────────┘     └─────────────────┘
                                 │
                                 ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │     Order       │────<│   OrderItem     │
                        │                 │     │                 │
                        └────────┬────────┘     └─────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
     ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
     │    Payment      │ │    Shipment     │ │    Invoice      │
     │                 │ │                 │ │  (wholesale)    │
     └─────────────────┘ └─────────────────┘ └─────────────────┘
```

### Key Domain Types

| Type | File | Description |
|------|------|-------------|
| `Product` | `domain/product.go` | Coffee product with origin, roast level, tasting notes |
| `ProductSKU` | `domain/product.go` | Purchasable variant (weight + grind combination) |
| `User` | `domain/user.go` | Customer account (retail or wholesale) |
| `Order` | `domain/order.go` | Purchase record with items, addresses, payment |
| `Subscription` | `domain/subscription.go` | Recurring delivery subscription |
| `Invoice` | `domain/invoice.go` | Wholesale invoice with net terms |
| `Operator` | `domain/operator.go` | Admin user (store owner/staff) |

---

## Database Design

### Approach

- **Schema**: SQL migrations via Goose (`migrations/`)
- **Queries**: Hand-written SQL with sqlc annotations (`sqlc/queries/`)
- **Generated**: Type-safe Go code (`internal/repository/`)

```
migrations/                    # Goose SQL migrations
├── 00001_create_tenants.sql
├── 00002_create_users.sql
├── 00005_create_products.sql
└── ...

sqlc/queries/                  # SQL queries with annotations
├── products.sql              # -- name: GetProduct :one
├── orders.sql                # -- name: CreateOrder :one
└── ...

internal/repository/          # Generated by sqlc
├── db.go                     # DBTX interface, Queries struct
├── models.go                 # Go structs from schema
├── products.sql.go           # Generated query functions
└── ...
```

### Key Tables

| Table | Purpose |
|-------|---------|
| `tenants` | Root entity - each coffee roaster |
| `users` | Customer accounts (scoped by tenant) |
| `tenant_operators` | Admin users (owner, staff) |
| `products` | Coffee products |
| `product_skus` | SKU variants (weight + grind) |
| `price_lists` | Named pricing tiers |
| `price_list_entries` | Per-SKU pricing per list |
| `carts` / `cart_items` | Shopping cart state |
| `orders` / `order_items` | Completed purchases |
| `subscriptions` | Recurring orders |
| `invoices` | Wholesale billing |
| `jobs` | Background job queue |

---

## Background Jobs

Jobs are processed via a PostgreSQL-backed queue using `SKIP LOCKED` for reliable processing:

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Handler    │────>│  EnqueueJob  │────>│   jobs       │
│  (enqueue)   │     │              │     │   table      │
└──────────────┘     └──────────────┘     └──────────────┘
                                                 │
                                          SKIP LOCKED
                                                 │
                                                 ▼
                                          ┌──────────────┐
                                          │   Worker     │
                                          │  goroutine   │
                                          └──────────────┘
```

### Job Types

| Queue | Job Types | File |
|-------|-----------|------|
| `email` | Password reset, order confirmation, shipping notification | `internal/jobs/email.go` |
| `invoice` | Generate consolidated, mark overdue, send reminders | `internal/jobs/invoice.go` |
| `cleanup` | Delete expired tokens | `internal/jobs/cleanup.go` |

### Enqueuing Pattern

```go
// internal/jobs/email.go
func EnqueuePasswordResetEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload PasswordResetPayload) error {
    payloadJSON, _ := json.Marshal(payload)
    _, err := q.EnqueueJob(ctx, repository.EnqueueJobParams{
        TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
        JobType:    JobTypePasswordReset,
        Queue:      "email",
        Payload:    payloadJSON,
        Priority:   50,
        MaxRetries: 3,
    })
    return err
}
```

---

## External Integrations

All external services are accessed through interfaces, allowing provider swapping:

### Billing (Stripe)

```go
// internal/billing/billing.go
type Provider interface {
    CreatePaymentIntent(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error)
    GetPaymentIntent(ctx context.Context, params GetPaymentIntentParams) (*PaymentIntent, error)
    CreateSubscription(ctx context.Context, params CreateSubscriptionParams) (*Subscription, error)
    CreateInvoice(ctx context.Context, params CreateInvoiceParams) (*Invoice, error)
    // ... more methods
}
```

### Shipping (EasyPost)

```go
// internal/shipping/shipping.go
type Provider interface {
    GetRates(ctx context.Context, params RateParams) ([]Rate, error)
    CreateLabel(ctx context.Context, params LabelParams) (*Label, error)
    TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error)
    ValidateAddress(ctx context.Context, params ValidateAddressParams) (*AddressValidation, error)
}
```

### Email (Postmark/SMTP)

```go
// internal/email/email.go
type Sender interface {
    Send(ctx context.Context, email *Email) (string, error)
    SendTemplate(ctx context.Context, templateID string, to []string, data map[string]interface{}) (string, error)
}
```

### Storage (R2/Local)

```go
// internal/storage/storage.go
type Storage interface {
    Put(ctx context.Context, key string, content io.Reader, contentType string) (string, error)
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    URL(key string) string
    Exists(ctx context.Context, key string) (bool, error)
}
```

---

## Webhook Processing

Stripe webhooks are processed through a dedicated handler with signature verification:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Stripe Event   │────>│ VerifySignature │────>│  Route by Type  │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └────────┬────────┘
                                                         │
              ┌──────────────────────────────────────────┼────────────────────┐
              │                    │                     │                    │
              ▼                    ▼                     ▼                    ▼
   payment_intent.succeeded  invoice.payment_succeeded  subscription.updated  ...
              │                    │                     │
              ▼                    ▼                     ▼
   CreateOrderFromPayment   CreateSubscriptionOrder   SyncSubscription
```

### Handled Events

| Event | Handler Action |
|-------|---------------|
| `payment_intent.succeeded` | Create order from cart |
| `payment_intent.payment_failed` | Log failure, notify customer |
| `invoice.payment_succeeded` | Create subscription renewal order |
| `invoice.payment_failed` | Update subscription to past_due |
| `customer.subscription.updated` | Sync subscription status |
| `customer.subscription.deleted` | Mark subscription expired |

---

## Frontend Architecture

```
Server-Rendered HTML
       │
       ├── Go Templates (web/templates/)
       │   ├── layouts/      # Base layouts
       │   ├── partials/     # Reusable components
       │   ├── storefront/   # Customer pages
       │   ├── admin/        # Dashboard pages
       │   └── email/        # Email templates
       │
       ├── htmx              # Dynamic updates without JS
       │   └── hx-get, hx-post, hx-swap
       │
       ├── Alpine.js         # Client-side state
       │   └── x-data, x-show, x-on
       │
       └── Tailwind CSS      # Utility-first styling
```

### When to Use What

| Tool | Use For |
|------|---------|
| **Go Templates** | Initial page render, full page content |
| **htmx** | Dynamic updates (add to cart, pagination, search) |
| **Alpine.js** | Client-side UI state (modals, dropdowns, form validation) |
| **Svelte** | Complex stateful widgets (future, if needed) |

---

## Authentication Flows

### Two Session Systems

| Session Type | Cookie Name | For | Table |
|--------------|-------------|-----|-------|
| Storefront | `freyja_session` | Customers | `sessions` |
| Operator | `freyja_operator` | Admin users | `operator_sessions` |

### Storefront Auth Flow

```
1. Customer signup ──> Create user ──> Send verification email
2. Email verification ──> Activate account
3. Login ──> Verify password ──> Create session
4. Authenticated requests ──> Validate session from cookie
```

### Operator Auth Flow

```
1. Owner signup (via Stripe Checkout) ──> Create tenant + operator
2. Setup wizard ──> Set password ──> Activate
3. Login ──> Verify password ──> Create operator session
4. Admin routes ──> WithOperator ──> RequireOperator ──> RequireActiveTenant
```

---

## Request Lifecycle Example

**Customer adds item to cart:**

```
1. POST /cart/add
   │
2. ResolveTenant middleware
   │ └── Extract tenant from host (acme.freyja.app -> acme)
   │ └── Load tenant from DB, add to context
   │
3. WithUser middleware
   │ └── Load user from session cookie (if present)
   │
4. CartHandler.Add()
   │ └── Parse form (sku_id, quantity)
   │ └── Get tenant_id from context
   │ └── Call repository.AddCartItem(tenant_id, cart_id, sku_id, qty)
   │ └── Return htmx partial with updated cart
   │
5. htmx swaps cart count in header
```

---

## Configuration

Configuration is loaded from environment variables:

```bash
# Core
ENV=development
PORT=3000

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/freyja

# Stripe
STRIPE_SECRET_KEY=sk_...
STRIPE_WEBHOOK_SECRET=whsec_...

# EasyPost
EASYPOST_API_KEY=...

# Email
EMAIL_PROVIDER=postmark  # or smtp
POSTMARK_API_TOKEN=...

# Storage
STORAGE_PROVIDER=local   # or r2
R2_BUCKET_NAME=...
R2_PUBLIC_URL=...

# Encryption (for provider credentials)
ENCRYPTION_KEY=base64-encoded-32-bytes

# Observability
SENTRY_DSN=...
SENTRY_ENABLED=false
```

---

## Key Patterns

### 1. Context-Based Tenant Isolation

```go
// Always get tenant from context, never trust user input
tenantID := domain.TenantIDFromContext(ctx)

// Repository queries always include tenant_id
row, err := queries.GetProduct(ctx, repository.GetProductParams{
    TenantID: tenantID,
    ID:       productID,
})
```

### 2. Interface-Based Dependencies

```go
// Services depend on interfaces, not implementations
type CheckoutService struct {
    billing  billing.Provider   // Interface
    shipping shipping.Provider  // Interface
    queries  *repository.Queries
}

// Easy to swap implementations or mock in tests
```

### 3. Error Handling

```go
// Domain errors carry codes for proper HTTP responses
var ErrProductNotFound = &domain.Error{
    Code:    domain.ENOTFOUND,
    Message: "Product not found",
}

// Handler maps domain errors to HTTP status
handler.ErrorResponse(w, r, err)  // 404 for ENOTFOUND
```

### 4. Idempotency

```go
// Payment processing uses idempotency keys
_, err := billing.CreatePaymentIntent(ctx, billing.CreatePaymentIntentParams{
    IdempotencyKey: cartID,  // Prevents duplicate charges
})

// Order creation checks for existing order by payment_intent_id
```

---

## Directory Structure

```
freyja/
├── cmd/saas/main.go           # Application entry point
├── internal/
│   ├── config/                # Configuration loading
│   ├── domain/                # Core types and interfaces
│   ├── service/               # Business logic
│   ├── handler/               # HTTP handlers (admin, storefront, webhook)
│   ├── middleware/            # HTTP middleware
│   ├── router/                # Custom router wrapper
│   ├── repository/            # sqlc-generated DB access
│   ├── billing/               # Stripe integration
│   ├── shipping/              # EasyPost integration
│   ├── email/                 # Email service
│   ├── storage/               # File storage (R2/local)
│   ├── tax/                   # Tax calculation
│   ├── jobs/                  # Background job definitions
│   ├── telemetry/             # Metrics and error tracking
│   └── crypto/                # Encryption utilities
├── migrations/                # Goose SQL migrations
├── sqlc/queries/              # SQL query files
├── web/
│   ├── templates/             # Go HTML templates
│   └── static/                # CSS, JS, images
├── planning/                  # Architecture documentation
└── scripts/                   # Development utilities
```

---

## Further Reading

| Document | Content |
|----------|---------|
| [TECHNICAL.md](./TECHNICAL.md) | Detailed technical decisions and rationale |
| [ROADMAP.md](./ROADMAP.md) | Feature roadmap and milestones |
| [UI_DIRECTION.md](./UI_DIRECTION.md) | UI/UX guidelines and design principles |
| [BUSINESS.md](./BUSINESS.md) | Market positioning and economics |
| [CLAUDE.md](../CLAUDE.md) | Development conventions and commands |
