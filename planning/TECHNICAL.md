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
- Magic link (email-based passwordless login)

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

### Choice: Transactional Email Service (Provider TBD)

**Candidates:**
- Postmark: Best deliverability reputation, simple API
- Resend: Modern API, good developer experience
- AWS SES: Cheapest at volume, more configuration required

**Rationale:**
- Reliable delivery for transactional emails (orders, invoices)
- Deliverability monitoring
- Template management options

**Abstraction:**
- Email sender interface in application
- Provider implementation behind interface
- Templates stored in application (Go templates)

**Implementation Note:** Defer provider selection until needed. Start with a simple interface and implement the chosen provider.

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
- Worker goroutines polling the queue
- Exponential backoff on failure
- Dead letter handling for inspection

**Future Migration:**
- If scale demands, can migrate to dedicated queue (e.g., River for Go/PostgreSQL)
- Interface abstraction allows swap without application changes

---

## File Storage

### Choice: Local Filesystem (MVP) → S3-Compatible (Post-MVP)

**MVP Approach:**
- Store product images on local filesystem
- Serve via Caddy static file handling
- Simple, no additional services

**Post-MVP:**
- Migrate to S3-compatible storage (AWS S3, DigitalOcean Spaces, MinIO)
- Enables CDN integration
- Better for multi-instance deployment if needed

**Abstraction:**
- File storage interface from the start
- Local implementation for MVP
- S3 implementation added later

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
| gorilla/sessions | Session management | BSD-3 |
| validator | Input validation | MIT |
| slog (stdlib) | Logging | (stdlib) |

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

---

## Review Schedule

This document should be reviewed:
- Before each major phase begins
- When a significant technical challenge is encountered
- Every 3 months during active development

Updates should include rationale for any changes from the original choices.