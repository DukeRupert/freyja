# Freyja Technical Choices

## Overview

This document records the technical decisions for Freyja, including the rationale for each choice. The guiding principles are: simplicity, maintainability by a solo developer, and minimal operational overhead.

---

## Language & Runtime

### Backend: Go

**Choice:** Go (latest stable)

**Rationale:**
- Single binary deployment eliminates dependency management in production
- Strong standard library reduces external dependencies
- Excellent performance characteristics without tuning
- Static typing catches errors at compile time
- Straightforward concurrency model for background tasks
- The developer's preferred language

**Alternatives Considered:**
- Rust: Higher learning curve, slower development velocity for this domain
- Node.js: Viable, but less preference and runtime dependency management
- Python: Performance concerns at scale, runtime dependencies

---

## Web Framework

### Choice: Standard Library with Thin Router Wrapper

**Package:** None (stdlib `net/http` only)

**Rationale:**
- Go 1.22+ added method matching and path parameters to `http.ServeMux`
- A ~50 line wrapper provides chi-like ergonomics (`r.Get()`, `r.Post()`, middleware groups)
- Zero external dependencies for routing
- Full compatibility with any `net/http` middleware
- No framework lock-in or upgrade concerns
- One less dependency to audit and maintain

**Implementation:**
The router wrapper provides:
- Method-specific handlers: `r.Get(path, handler)`, `r.Post(path, handler)`, etc.
- Global middleware: `r.Use(middleware...)`
- Grouped middleware: `r.Group(func(r *router) { ... })`
- Per-route middleware: `r.Get(path, handler, middleware...)`

```go
type (
    middleware func(http.Handler) http.Handler
    router struct {
        *http.ServeMux
        chain []middleware
    }
)

func NewRouter(mx ...middleware) *router {
    return &router{ServeMux: &http.ServeMux{}, chain: mx}
}

func (r *router) Use(mx ...middleware) {
    r.chain = append(r.chain, mx...)
}

func (r *router) Group(fn func(r *router)) {
    fn(&router{ServeMux: r.ServeMux, chain: slices.Clone(r.chain)})
}

func (r *router) Get(path string, fn http.HandlerFunc, mx ...middleware) {
    r.handle(http.MethodGet, path, fn, mx)
}

// Post, Put, Delete, etc. follow the same pattern

func (r *router) handle(method, path string, fn http.HandlerFunc, mx []middleware) {
    r.Handle(method+" "+path, r.wrap(fn, mx))
}

func (r *router) wrap(fn http.HandlerFunc, mx []middleware) (out http.Handler) {
    out, mx = http.Handler(fn), append(slices.Clone(r.chain), mx...)
    slices.Reverse(mx)
    for _, m := range mx {
        out = m(out)
    }
    return
}
```

**Path Parameters:**
Go 1.22+ supports path parameters natively:
```go
r.Get("/products/{id}", productHandler)

// In handler:
id := r.PathValue("id")
```

**Alternatives Considered:**
- Chi: Excellent, but now redundant given stdlib improvements
- Echo: More features built-in, but adds dependency and opinions
- Fiber: Uses fasthttp (non-standard), potential compatibility issues

---

## Database

### Choice: PostgreSQL

**Version:** 15 or 16 (latest stable)

**Rationale:**
- Robust, battle-tested relational database
- Excellent JSON support for semi-structured data (e.g., product metadata)
- Strong data integrity guarantees
- Rich indexing options for query optimization
- Well-supported in the Go ecosystem
- Can handle significant scale before becoming a bottleneck

**What PostgreSQL Handles:**
- All application data (products, customers, orders, etc.)
- Session storage
- Background job queue (simple table-based approach)
- Full-text search for products (using tsvector)

**What We're Not Adding (Yet):**
- Redis: Not needed at this scale; PostgreSQL handles sessions and simple caching
- Elasticsearch: PostgreSQL full-text search is sufficient for product catalog

---

## Database Access

### Choice: sqlc + pgx

**Packages:**
- `github.com/sqlc-dev/sqlc` (code generation)
- `github.com/jackc/pgx/v5` (PostgreSQL driver)

**Rationale:**
- Write plain SQL, get type-safe Go code
- No ORM abstraction to fight or debug
- Queries are explicit and optimizable
- Compile-time verification of SQL syntax and types
- pgx is the most performant and feature-complete PostgreSQL driver for Go

**Workflow:**
1. Define schema in SQL migration files
2. Write queries in `.sql` files with sqlc annotations
3. Run `sqlc generate` to produce Go code
4. Call generated functions from application code

**Alternatives Considered:**
- GORM: Too much magic, hard to debug, performance overhead
- ent: Interesting but adds complexity, code generation is heavier
- Raw database/sql: Loses type safety, more boilerplate

---

## Schema Migrations

### Choice: Goose

**Package:** `github.com/pressly/goose/v3`

**Rationale:**
- Simple, file-based migrations
- SQL migrations (not Go code) for transparency
- Embeddable in the application binary
- Supports up, down, and migration status
- Active maintenance

**Migration Strategy:**
- Sequential, timestamped migration files
- All migrations in `/migrations` directory
- Migrations run automatically on application startup (with locking)
- Down migrations provided for development; production rollback via new migration

---

## Frontend

### Primary: Server-Rendered HTML + htmx + Alpine.js + Tailwind CSS

**Rationale:**
- Server-side rendering simplifies state management
- htmx enables dynamic updates without a JavaScript build step
- Alpine.js handles UI interactions (modals, dropdowns, form state)
- Tailwind CSS provides utility-first styling with minimal custom CSS
- No Node.js required in production
- Fast initial page loads, good SEO by default

**When This Approach Applies:**
- All admin dashboard functionality
- Storefront product browsing and detail pages
- Cart management
- Checkout flow
- Customer account pages
- Wholesale portal

### Secondary: Svelte (If Needed)

**When to Reach for Svelte:**
- Complex, stateful interfaces that would be unwieldy in htmx
- Real-time, highly interactive features
- Significant client-side computation or validation

**Current Candidates (None Required for MVP):**
- Subscription management with preview calculations
- Build-your-own-box product configurations
- Complex filtering and sorting UIs

**Integration Approach:**
- Compile Svelte components as standalone widgets
- Embed into server-rendered pages via script tag
- Pass initial data via JSON in a script block or data attributes
- Communicate with backend via REST endpoints

---

## CSS Framework

### Choice: Tailwind CSS

**Rationale:**
- Utility-first approach speeds up development
- Consistent spacing, colors, and typography
- Small production bundle with purging
- Works well with server-rendered HTML
- No context switching between HTML and CSS files

**Build Process:**
- Tailwind CLI in watch mode during development
- Production build with minification and purging
- Output to `/web/static/css/`

**Component Library:** daisyUI (optional)
- Provides pre-built component classes (buttons, cards, forms)
- Reduces custom CSS for common patterns
- Can be added if development speed is a concern

---

## Authentication

### Choice: Cookie-Based Sessions

**Packages:**
- `github.com/gorilla/sessions` (session management)
- Custom middleware for auth checks

**Rationale:**
- Simpler than JWT for server-rendered applications
- Automatic handling by browsers (no client-side token management)
- Easy to invalidate (delete session from database)
- Works seamlessly with htmx

**Session Storage:** PostgreSQL
- Sessions stored in database table
- Allows session invalidation and multi-device logout
- No additional infrastructure (Redis not needed)

**Authentication Methods:**
- Email/password with bcrypt hashing
- Magic link (email-based passwordless login) — planned

**Email Verification:**
- Required before login (prevents account enumeration via timing)
- Cryptographically secure tokens (32 bytes via `crypto/rand`)
- Tokens stored as SHA-256 hashes (not plaintext)
- 24-hour expiration window
- Rate limiting: 5 requests/user/hour, 10 requests/IP/hour (tenant-scoped)
- Atomic verification flow via database transactions
- Resend capability with same security constraints

**Password Reset:**
- Secure token generation and hashing (same pattern as email verification)
- Rate limiting per user and IP address
- Token expiration after 1 hour
- Single-use tokens (invalidated after use)

---

## Payment Processing

### Choice: Stripe

**Package:** `github.com/stripe/stripe-go/v76`

**Rationale:**
- Industry-standard API and documentation
- Comprehensive Go SDK
- Handles subscriptions, invoicing, and one-time payments
- PCI compliance handled by Stripe
- Extensive webhook system for event-driven updates

**Integration Points:**
- Customer synchronization
- Payment Intents for one-time purchases
- Stripe Billing for subscriptions
- Stripe Invoicing for wholesale net terms
- Webhook handling for payment events

**Abstraction:**
- Billing provider interface defined in application
- Stripe implementation behind the interface
- Allows future alternative implementations if needed

---

## Email

### Choice: Postmark (Primary) + SMTP (Development)

**Package:** Custom `email.Sender` interface with implementations

**Rationale:**
- Postmark: Best deliverability reputation, simple API, chosen for production
- SMTP: Local development with Mailhog for email inspection
- Interface abstraction allows provider swapping

**Implementation:**
- `email.Sender` interface defines `Send(ctx, to, subject, htmlBody, textBody)`
- `PostmarkSender` implementation using Postmark API token
- `SMTPSender` implementation for local development
- `email.Service` wraps sender with template rendering

**Email Types Implemented:**
- Email verification (signup flow)
- Password reset
- Order confirmation
- Shipping confirmation with tracking
- Subscription welcome
- Subscription payment failed
- Subscription cancelled

**Template System:**
- Go HTML templates in `/web/templates/email/`
- Base layout with consistent branding
- Template functions for date/currency formatting
- Both HTML and plain text versions

---

## Background Jobs

### Choice: Database-Backed Job Queue

**Rationale:**
- No additional infrastructure (no Redis, no RabbitMQ)
- PostgreSQL SKIP LOCKED provides reliable job processing
- Simple to implement and debug
- Sufficient for the workload scale

**Use Cases:**
- Webhook processing (deferred for reliability)
- Email sending
- Invoice generation on billing cycle
- Report generation
- Cleanup tasks

**Implementation:**
- Jobs table with status, payload, retry count, scheduled time
- Worker goroutines polling the queue with configurable concurrency
- Exponential backoff on failure
- Dead letter handling for inspection
- Job history table for completed/failed jobs

**Job Types Implemented:**
- Email jobs: password reset, email verification, order confirmation, shipping, subscription lifecycle
- Invoice jobs: generate consolidated, mark overdue, send reminders, sync from Stripe
- Cleanup jobs: delete expired verification/reset tokens

**Worker Configuration:**
- Configurable poll interval (default 1s)
- Configurable max concurrency (default 5)
- Queue-specific processing
- Tenant-scoped job processing
- Graceful shutdown on context cancellation

**Future Migration:**
- If scale demands, can migrate to dedicated queue (e.g., River for Go/PostgreSQL)
- Interface abstraction allows swap without application changes

---

## Shipping

### Choice: EasyPost

**Package:** `github.com/EasyPost/easypost-go/v5`

**Rationale:**
- Pay-per-label pricing (no monthly minimums, good for MVP)
- First-class Go SDK with good documentation
- USPS Commercial Plus rates (similar discounts to PirateShip)
- Simple REST API with comprehensive features
- Address verification included

**Capabilities:**
- Real-time rates from USPS, UPS, FedEx, DHL, and 100+ carriers
- Label purchasing with automatic tracking numbers
- Label voiding/refunds
- Shipment tracking with event history
- Address validation with suggestions

**Abstraction:**
- `shipping.Provider` interface defined in application
- EasyPost implementation behind the interface
- FlatRate provider for simple configurations
- Mock provider for testing

### Configuration

**Environment Variables:**
```bash
EASYPOST_API_KEY=your_api_key_here
```

### Usage

**Initialize the provider:**
```go
import "github.com/dukerupert/freyja/internal/shipping"

provider, err := shipping.NewEasyPostProvider(shipping.EasyPostConfig{
    APIKey: os.Getenv("EASYPOST_API_KEY"),
    Logger: slog.Default(), // Optional, defaults to slog.Default()
})
```

**Get shipping rates:**
```go
rates, err := provider.GetRates(ctx, shipping.RateParams{
    TenantID: tenantID,  // Required for multi-tenant security
    OriginAddress: shipping.ShippingAddress{
        Name:       "My Coffee Roaster",
        Line1:      "123 Roaster Lane",
        City:       "Portland",
        State:      "OR",
        PostalCode: "97201",
        Country:    "US",
    },
    DestinationAddress: shipping.ShippingAddress{
        Name:       "John Doe",
        Line1:      "456 Customer St",
        City:       "Seattle",
        State:      "WA",
        PostalCode: "98101",
        Country:    "US",
    },
    Packages: []shipping.Package{
        {
            WeightGrams: 340,  // 12oz coffee
            LengthCm:    20,
            WidthCm:     15,
            HeightCm:    8,
        },
    },
})
```

**Purchase a label:**
```go
label, err := provider.CreateLabel(ctx, shipping.LabelParams{
    TenantID:           tenantID,
    RateID:             selectedRate.RateID,  // From GetRates
    OriginAddress:      origin,
    DestinationAddress: destination,
    Package:            pkg,
})
// label.TrackingNumber, label.LabelURL available
```

**Track a shipment:**
```go
tracking, err := provider.TrackShipment(ctx, "9400111899223456789012")
// tracking.Status, tracking.Events available
```

**Validate an address:**
```go
result, err := provider.ValidateAddress(ctx, shipping.ValidateAddressParams{
    TenantID: tenantID,
    Address:  customerAddress,
})
// result.Status: AddressValid, AddressValidWithChanges, or AddressInvalid
// result.SuggestedAddress available if changes recommended
```

### Security Features

- **Tenant isolation:** TenantID stored in EasyPost shipment reference, validated on all operations
- **Idempotency:** CreateLabel returns existing label if already purchased (prevents duplicates)
- **Rate expiration:** Rates include ExpiresAt field (24 hours from creation)

### Alternatives Considered

- **Shippo:** Good alternative, similar pricing, but EasyPost Go SDK is more mature
- **ShipEngine:** Owned by Stamps.com, good rates but less Go community support
- **PirateShip:** No API available (why we chose EasyPost)

---

## Provider Configuration System

### Choice: Database-Backed Registry with Encrypted Credentials

**Packages:**
- `internal/provider` (registry, factory, validator)
- `internal/crypto` (AES-256-GCM encryption)

**Rationale:**
- Tenants can select their preferred providers for tax, shipping, billing, and email
- Credentials stored encrypted at rest (no plaintext API keys in database)
- Lazy loading with TTL-based caching reduces database queries
- Interface-based design allows adding new providers without architecture changes

**Architecture Components:**

1. **Provider Registry** (`internal/provider/registry.go`)
   - Central access point for provider instances
   - TTL-based caching (1 hour default) for performance
   - Automatic cache invalidation on config changes
   - Thread-safe concurrent access

2. **Provider Factory** (`internal/provider/factory.go`)
   - Creates provider instances from decrypted configurations
   - Validates credentials before instantiation
   - Returns appropriate errors for invalid configs

3. **Configuration Validator** (`internal/provider/validator.go`)
   - Validates provider-specific configuration requirements
   - Checks API key formats, required fields
   - Used both on save and before instantiation

4. **Encryption Service** (`internal/crypto/encrypt.go`)
   - AES-256-GCM authenticated encryption
   - Base64-encoded keys for environment variable storage
   - Key generation utility for production setup

**Database Schema:**
```sql
CREATE TABLE tenant_provider_configs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    provider_type VARCHAR(50) NOT NULL,  -- 'tax', 'shipping', 'billing', 'email'
    provider_name VARCHAR(50) NOT NULL,  -- 'stripe', 'easypost', 'percentage', etc.
    config_encrypted BYTEA,              -- AES-256-GCM encrypted JSON
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    UNIQUE(tenant_id, provider_type)     -- One active provider per type per tenant
);
```

**Provider Types:**
- **Tax:** none, stripe_tax, percentage, taxjar, avalara
- **Shipping:** flat_rate, easypost, shipstation, shippo
- **Billing:** stripe (required)
- **Email:** smtp, postmark, resend, ses

**Usage Example:**
```go
// Get the tax calculator for a tenant
taxCalc, err := registry.GetTaxCalculator(ctx, tenantID)
if err != nil {
    // Handle error (no config, invalid credentials, etc.)
}

// Use the tax calculator
tax, err := taxCalc.Calculate(ctx, order)
```

**Security Considerations:**
- Encryption key stored in environment variable (`ENCRYPTION_KEY`)
- Keys never logged or exposed in error messages
- Failed decryption logs warning but doesn't expose key material
- Admin UI masks existing credentials (shows "••••" not actual values)

---

## Observability & Telemetry

### Choice: Prometheus + Sentry

**Packages:**
- `github.com/prometheus/client_golang` (metrics)
- `github.com/getsentry/sentry-go` (error tracking)

**Rationale:**
- Prometheus is the industry standard for metrics; integrates with Grafana for dashboards
- Sentry provides structured error tracking with context (user, tenant, request info)
- Free tier (5K errors/month) sufficient for solo developer MVP
- Both can be disabled in development to avoid noise

### Business Metrics (Prometheus)

**Package:** `internal/telemetry/business_metrics.go`

All business metrics include `tenant_id` label for per-tenant Grafana dashboards.

**Metric Categories:**
- **Product Engagement:** views, add-to-cart events, cart value
- **Checkout Funnel:** started, abandoned, completed, conversion tracking
- **Orders:** created, value distribution, payment success/failure
- **Revenue:** collected revenue, refunds issued, refund amounts
- **Subscriptions:** created, renewed, cancelled, failed payments
- **Webhooks:** processed count, latency histogram, failures
- **Email:** sent, failed (by type)
- **External APIs:** Stripe API latency histogram

**Usage Example:**
```go
// Record a product view
telemetry.Business.ProductViews.WithLabelValues(tenantID, productID).Inc()

// Record checkout completion
telemetry.Business.CheckoutCompleted.WithLabelValues(tenantID).Inc()
telemetry.Business.OrderValue.WithLabelValues(tenantID, "retail").Observe(float64(totalCents))
telemetry.Business.RevenueCollected.WithLabelValues(tenantID, "retail").Add(float64(totalCents))
```

### Error Tracking (Sentry)

**Package:** `internal/telemetry/sentry.go`

**Features:**
- Disabled by default (`SENTRY_ENABLED=false`) for development
- Automatic tenant and user context via middleware
- Context-aware error capture preserves request information
- Panic recovery middleware for HTTP handlers

**Environment Variables:**
```bash
SENTRY_DSN=https://xxx@sentry.io/xxx
SENTRY_ENABLED=true          # false by default
SENTRY_ENVIRONMENT=production
SENTRY_RELEASE=v1.0.0
SENTRY_SAMPLE_RATE=1.0       # 0.0-1.0, percentage of errors to capture
SENTRY_TRACES_SAMPLE_RATE=0  # 0 to disable performance monitoring
```

**Middleware Pattern:**
```go
// Applied in main.go router setup
r.Use(
    telemetry.SentryMiddleware(),                           // Panic recovery
    telemetry.SentryContextMiddleware(tenantID, userExtractor), // Tenant/user context
)
```

**Error Capture:**
```go
// In HTTP handlers (uses context for tenant/user)
telemetry.CaptureErrorFromContext(r.Context(), err, map[string]interface{}{
    "order_id": orderID,
})

// In background jobs (explicit tenant)
telemetry.CaptureErrorWithTenant(err, tenantID, map[string]interface{}{
    "job_type": "invoice:generate",
})
```

**Why This Architecture:**
- Global singleton for metrics (simple, single Prometheus registry)
- Middleware for HTTP context (automatic, no boilerplate in handlers)
- Explicit context for non-HTTP paths (webhooks, background jobs)
- Disable flag prevents consuming free tier during development

---

## File Storage

### Choice: Cloudflare R2 (Production) + Local Filesystem (Development)

**Package:** `github.com/aws/aws-sdk-go-v2/service/s3`

**Rationale:**
- **Cloudflare R2:** S3-compatible with zero egress fees (critical for serving images)
- **Platform-controlled:** Single storage backend managed by Freyja (not per-tenant configuration)
- **Simplicity for target market:** Coffee roasters don't want to manage AWS credentials
- **Cost-effective:** ~$0.015/GB/month storage, free egress, generous free tier (10GB)

**Architecture:**
- `storage.Storage` interface with `Put`, `Get`, `Delete`, `URL`, `Exists` methods
- `LocalStorage` implementation for development (stores in `./web/static/uploads/`)
- `R2Storage` implementation for production (uses AWS SDK v2 with R2 endpoint)
- Factory function `storage.NewStorage(cfg)` selects implementation based on config

**Configuration (Environment Variables):**
```bash
# Development (default)
STORAGE_PROVIDER=local
LOCAL_STORAGE_PATH=./web/static/uploads
LOCAL_STORAGE_URL=/uploads

# Production
STORAGE_PROVIDER=r2
R2_ACCOUNT_ID=your_account_id
R2_ACCESS_KEY_ID=your_access_key
R2_SECRET_ACCESS_KEY=your_secret_key
R2_BUCKET_NAME=freyja-files
R2_PUBLIC_URL=https://files.your-domain.com
```

**Multi-Tenant Isolation:**
- Storage keys include tenant ID prefix: `{tenant_id}/products/{product_id}/image.jpg`
- All tenants share one R2 bucket with key-based isolation
- Tenant ID validated on all storage operations

**Why Not Per-Tenant Storage Configuration:**
- Target customers (small coffee roasters) aren't technical
- Storage costs are negligible (~$1/month for 100 tenants)
- Reduces support burden (no "my S3 credentials stopped working" tickets)
- Consistent UX: just upload images, no cloud provider setup

**Alternatives Considered:**
- AWS S3: Expensive egress ($0.09/GB)
- DigitalOcean Spaces: Good but R2 egress is free
- MinIO (self-hosted): Adds operational burden
- Per-tenant buckets: Unnecessary complexity for target market

---

## Deployment

### Choice: Docker + Caddy on VPS

**Components:**
- Single Go binary in Docker container
- PostgreSQL in Docker container (or managed database)
- Caddy as reverse proxy and TLS termination

**Rationale:**
- Simple, reproducible deployment
- Caddy handles HTTPS automatically via Let's Encrypt
- No Kubernetes complexity for a single-instance application
- Easy to move to any VPS provider

**Docker Compose Setup:**
- Application container
- PostgreSQL container with volume persistence
- Caddy container with config volume
- Internal network for container communication

**Deployment Process:**
1. Build Docker image (locally or in CI)
2. Push to container registry
3. Pull and restart on VPS
4. Migrations run automatically on startup

---

## Development Tooling

### Live Reload

**Tool:** Air (`github.com/cosmtrek/air`)

**Purpose:** Rebuild and restart Go application on file changes

### Database Management

**Tools:**
- Goose CLI for manual migration commands
- pgcli or DBeaver for database inspection

### Code Quality

**Tools:**
- `gofmt` / `goimports` for formatting
- `golangci-lint` for linting
- `go test` for testing

### Local Services

**Tool:** Docker Compose

**Services:**
- PostgreSQL
- Mailhog (email testing)
- Stripe CLI (webhook forwarding)

---

## Project Structure

```
freyja/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration loading
│   ├── domain/                  # Core business types
│   │   ├── product.go
│   │   ├── customer.go
│   │   ├── order.go
│   │   ├── subscription.go
│   │   └── pricelist.go
│   ├── billing/                 # Billing abstraction
│   │   ├── billing.go           # Interface definition
│   │   └── stripe/              # Stripe implementation
│   ├── shipping/                # Shipping abstraction
│   │   ├── shipping.go          # Interface definition
│   │   └── manual/              # Manual fulfillment implementation
│   ├── email/                   # Email abstraction
│   │   ├── email.go             # Interface definition
│   │   └── smtp/                # SMTP implementation
│   ├── storage/                 # File storage abstraction
│   ├── repository/              # Database queries (sqlc generated)
│   ├── handler/                 # HTTP handlers
│   │   ├── storefront/          # Customer-facing routes
│   │   ├── admin/               # Admin routes
│   │   └── webhook/             # Webhook handlers
│   ├── middleware/              # HTTP middleware
│   ├── jobs/                    # Background job definitions
│   └── worker/                  # Job processing
├── migrations/                  # SQL migration files
├── sqlc/                        # sqlc query files
│   ├── queries/
│   │   ├── products.sql
│   │   ├── customers.sql
│   │   └── orders.sql
│   └── schema.sql               # Reference schema for sqlc
├── web/
│   ├── templates/               # Go HTML templates
│   │   ├── layouts/
│   │   ├── partials/
│   │   ├── storefront/
│   │   └── admin/
│   └── static/                  # Static assets
│       ├── css/
│       ├── js/
│       └── images/
├── docs/                        # Project documentation
├── scripts/                     # Development and deployment scripts
├── sqlc.yaml                    # sqlc configuration
├── tailwind.config.js           # Tailwind configuration
├── docker-compose.yml           # Local development services
├── Dockerfile                   # Production container build
└── Caddyfile                    # Caddy configuration
```

---

## Dependency Summary

### Go Packages

| Package | Purpose | License |
|---------|---------|---------|
| pgx | PostgreSQL driver | MIT |
| sqlc (generated) | Type-safe queries | MIT |
| goose | Migrations | MIT |
| stripe-go | Stripe API | MIT |
| easypost-go | Shipping API | MIT |
| gorilla/sessions | Session management | BSD-3 |
| validator | Input validation | MIT |
| slog (stdlib) | Logging | (stdlib) |
| prometheus/client_golang | Metrics | Apache 2.0 |
| sentry-go | Error tracking | MIT |

### Frontend

| Tool | Purpose | License |
|------|---------|---------|
| htmx | Dynamic HTML updates | BSD-2 |
| Alpine.js | UI interactions | MIT |
| Tailwind CSS | Styling | MIT |
| daisyUI (optional) | Component classes | MIT |

### Infrastructure

| Tool | Purpose | License |
|------|---------|---------|
| Docker | Containerization | Apache 2.0 |
| PostgreSQL | Database | PostgreSQL License |
| Caddy | Reverse proxy / TLS | Apache 2.0 |

---

## Decision Log

Significant decisions should be recorded here as the project evolves.

| Date | Decision | Rationale |
|------|----------|-----------|
| (Project Start) | Initial technical choices documented | Baseline architecture |
| 2024-12-03 | EasyPost for shipping integration | Pay-per-label pricing, mature Go SDK, USPS Commercial Plus rates, PirateShip has no API |
| 2024-12-03 | Postmark for transactional email | Best deliverability, simple API, good DX |
| 2024-12-03 | Email verification required before login | Prevents abuse, ensures valid contact info, industry standard |
| 2024-12-03 | Token hashing with SHA-256 | Tokens stored as hashes, not plaintext; prevents database breach from exposing valid tokens |
| 2024-12-03 | Rate limiting scoped by tenant_id | Multi-tenant isolation for rate limits; prevents cross-tenant interference |
| 2024-12-03 | Atomic verification via transactions | Prevents race conditions on concurrent verification attempts |
| 2024-12-03 | Client IP extraction via proxy headers | Supports X-Forwarded-For/X-Real-IP for Caddy/nginx deployments |
| 2024-12-04 | Provider configuration system | Tenant-selectable providers with encrypted credentials for multi-tenant SaaS flexibility |
| 2024-12-04 | AES-256-GCM for credential encryption | Industry standard authenticated encryption; prevents tampering and provides confidentiality |
| 2024-12-04 | TTL-based provider caching | 1-hour cache reduces database queries while allowing timely config updates |
| 2024-12-04 | Stripe Tax + percentage-based tax | Offers tenants choice between automatic (Stripe) and manual (state rates) tax calculation |
| 2024-12-05 | Cloudflare R2 for file storage | Zero egress fees critical for serving product images; S3-compatible API; generous free tier |
| 2024-12-05 | Platform-controlled storage (not per-tenant) | Target market (coffee roasters) aren't technical; storage costs negligible; reduces support burden |
| 2024-12-06 | Prometheus for business metrics with tenant_id | Multi-tenant dashboards require per-tenant labels; global singleton pattern for simplicity |
| 2024-12-06 | Sentry for error tracking with disable flag | Free tier (5K errors/month) sufficient for MVP; SENTRY_ENABLED=false by default for development |
| 2024-12-06 | Sentry context middleware pattern | Automatic tenant/user context via middleware for HTTP handlers; explicit context for webhooks/background jobs |

---

## Review Schedule

This document should be reviewed:
- Before each major phase begins
- When a significant technical challenge is encountered
- Every 3 months during active development

Updates should include rationale for any changes from the original choices.