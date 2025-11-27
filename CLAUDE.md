# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Freyja is an e-commerce platform built exclusively for small coffee roasters (1-5 locations, $200k-2M annual revenue). It provides integrated B2C and B2B functionality including subscriptions and invoicing—all in one system without requiring separate plugins or apps.

**Key differentiator:** Built specifically for coffee products with domain-specific features (roast level, origin metadata, tasting notes) while treating retail and wholesale as first-class, equally important sales channels.

**Business model:** Multi-tenant SaaS at $149/month flat fee (no transaction fees beyond Stripe's standard rates).

## Development Commands

### Running the Application

```bash
# Run the main application
go run main.go

# The application will start on port :3000
# Example routes are defined in main.go
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests in a specific package
go test ./internal/handler/...
```

### Code Quality

```bash
# Format code
go fmt ./...
goimports -w .

# Run linter
golangci-lint run

# Vet code for issues
go vet ./...
```

### Database (When Implemented)

```bash
# Run migrations (will be automatic on startup when implemented)
# Manual migration command via goose CLI:
goose -dir migrations postgres "connection-string" up

# Generate sqlc code from SQL queries
sqlc generate
```

### Development Workflow

When the project includes Air for live reload:
```bash
# Start development server with auto-reload
air
```

## Architecture

### Technology Stack

**Backend:**
- Language: Go (latest stable)
- Web Framework: Standard library `net/http` with custom thin router wrapper (~50 lines)
- Database: PostgreSQL 15/16 with pgx driver
- Database Access: sqlc (type-safe SQL code generation) + pgx
- Migrations: Goose (SQL-based, embeddable)
- Sessions: Cookie-based with PostgreSQL storage
- Background Jobs: Database-backed queue (PostgreSQL SKIP LOCKED)
- Payments: Stripe
- Email: Provider TBD (Postmark/Resend/SES) - interface-based abstraction
- File Storage: Local filesystem (MVP) → S3-compatible (post-MVP)

**Frontend:**
- Primary: Server-rendered HTML + htmx + Alpine.js + Tailwind CSS
- Secondary: Svelte (only for complex stateful UIs if needed)
- No Node.js required in production

**Infrastructure:**
- Deployment: Docker + Caddy on VPS
- Development: Docker Compose for local services (PostgreSQL, Mailhog, Stripe CLI)

### Custom Router Implementation

The codebase uses a thin wrapper around Go's standard library `http.ServeMux` (Go 1.22+) to provide chi-like ergonomics. Implementation is in `main.go` (will move to dedicated package):

```go
// Key features:
- Method-specific handlers: r.Get(), r.Post(), r.Put(), r.Delete()
- Global middleware: r.Use(middleware...)
- Grouped middleware: r.Group(func(r *router) { ... })
- Per-route middleware: r.Get(path, handler, middleware...)
- Path parameters: r.PathValue("id") in handlers

// Middleware are applied in order and execute as a chain
```

**Important:** Path parameters use Go 1.22+ syntax: `/products/{id}` accessed via `r.PathValue("id")`

### Project Structure (Planned)

```
freyja/
├── cmd/server/main.go           # Application entry point (currently at root)
├── internal/
│   ├── config/                  # Configuration loading
│   ├── domain/                  # Core business types (product, customer, order, etc.)
│   ├── billing/                 # Billing abstraction + Stripe implementation
│   ├── shipping/                # Shipping abstraction + implementations
│   ├── email/                   # Email abstraction + provider implementation
│   ├── storage/                 # File storage abstraction
│   ├── repository/              # Database queries (sqlc generated)
│   ├── handler/                 # HTTP handlers
│   │   ├── storefront/          # Customer-facing routes
│   │   ├── admin/               # Admin dashboard routes
│   │   └── webhook/             # Webhook handlers (Stripe, etc.)
│   ├── middleware/              # HTTP middleware
│   ├── jobs/                    # Background job definitions
│   └── worker/                  # Job processing
├── migrations/                  # Goose SQL migrations
├── sqlc/
│   ├── queries/                 # SQL query files for sqlc
│   └── schema.sql               # Reference schema for sqlc
├── web/
│   ├── templates/               # Go HTML templates
│   └── static/                  # CSS, JS, images
├── planning/                    # Project documentation (context docs)
├── sqlc.yaml                    # sqlc configuration (when created)
├── docker-compose.yml           # Local development services (when created)
└── Dockerfile                   # Production build (when created)
```

### Key Architectural Patterns

**1. Interface-Based Abstractions:**
All external services (billing, email, shipping, storage) are defined as interfaces with concrete implementations. This allows:
- Easy testing with mocks
- Provider swapping without application changes
- Clear service boundaries

**2. Database-First Approach:**
- Schema defined in SQL migrations
- Queries written in plain SQL with sqlc annotations
- Type-safe Go code generated from SQL
- No ORM abstraction layer

**3. Server-Rendered HTML as Default:**
- Use htmx for dynamic updates
- Alpine.js for local UI state
- Reach for Svelte only when absolutely necessary (complex client-side state)

**4. Multi-Tenancy:**
- All customers share one database and application instance
- Strict `tenant_id` scoping on all queries
- Row-level security considerations in PostgreSQL

## Domain Concepts

### Product Catalog
Coffee products have specialized attributes:
- Standard e-commerce: name, description, images, SKUs, pricing, inventory
- Coffee-specific: origin, region, producer, process method, roast level, tasting notes, elevation
- SKU variants: weight + grind option combinations
- Visibility: public, wholesale-only, or restricted by price list

### Price Lists
Multiple named price lists (e.g., "Retail", "Café Tier 1", "Restaurant Tier 2"):
- Each customer assigned to one price list
- Per-product pricing per list
- Price list membership controls product visibility
- Default retail price list for anonymous/new customers

### Customer Types
- **Retail:** Standard consumers, pay immediately, subscriptions available
- **Wholesale:** Business accounts with application/approval flow, net terms (Net 15/30), minimum order quantities, consolidated billing cycles

### Order Workflows
- **B2C:** Immediate payment → order created → fulfillment → shipped
- **B2B:** Order placed → invoice generated → payment due by terms → fulfillment after payment

### Subscriptions
- Frequency options: weekly, every 2 weeks, monthly, every 6 weeks, every 2 months
- Managed via Stripe Billing
- Customer portal for pause/resume/skip/cancel
- Failed payment handling with dunning

### Invoicing
- **Net terms:** Payment due N days after invoice date
- **Consolidated billing:** Accumulate orders within billing cycle, generate single invoice
- Managed via Stripe Invoicing with webhook synchronization

## Development Principles

### Simplicity Over Complexity
- Use standard library when sufficient
- Minimize external dependencies
- Prefer explicit code over magic
- No premature abstractions

### Solo Developer Maintainability
- Code should be straightforward to understand 6 months later
- Explicit is better than clever
- Comprehensive planning docs in `/planning` directory
- Inline comments only when logic isn't self-evident

### Coffee-Specific, Not Generic
- Domain models reflect coffee roasting business
- Don't build generic e-commerce abstractions
- Hard-code coffee-specific assumptions when appropriate
- This constraint is a feature, not a limitation

### Test Strategically
- Focus tests on business logic and data integrity
- Test database queries (sqlc generates correct types, but test logic)
- Integration tests for critical flows (checkout, invoicing)
- Don't test framework or library code

## UI/UX Guidelines

**Design philosophy:** Pragmatic craft—reliable, clear, quietly confident. Respects the operator's time and expertise.

**Visual language:**
- Color: Predominantly neutral with muted teal primary accent (#2A7D7D), warm amber secondary (#B5873A)
- Typography: System sans-serif stack, clear hierarchy
- Spacing: Generous padding, 4px-based scale
- Components: Subtle borders/shadows, 6-8px border radius

**Voice:**
- Direct and concise
- Plain language, no jargon
- Calm confidence, not excitement
- Example: "Product created" not "Awesome! Your product is ready!"

**Responsive:**
- Mobile: < 640px
- Tablet: 640px - 1024px
- Desktop: > 1024px

See `planning/UI_DIRECTION.md` for comprehensive guidance.

## Important Constraints

**Security:**
- Never bypass authentication checks
- Always scope queries by `tenant_id` when multi-tenancy is implemented
- Validate all user input
- Use parameterized queries (sqlc handles this)
- Be mindful of OWASP top 10 vulnerabilities

**Performance:**
- Database queries should use indexes appropriately
- Avoid N+1 queries (use joins or batch fetches)
- Background jobs for long-running tasks (email, webhooks)
- Defer optimization until profiling identifies bottlenecks

**Payments:**
- All payment operations are idempotent (use Stripe idempotency keys)
- Webhook events may arrive multiple times—handle gracefully
- Never store credit card details (Stripe handles PCI compliance)

## Current Status

**Implemented:**
- Custom router wrapper with middleware support (main.go:10-68)
- Basic HTTP server setup (main.go:85-103)

**Next phases (see planning/ROADMAP.md):**
1. Foundation: Product catalog, customer accounts, price lists (Weeks 1-2)
2. Storefront & Cart (Weeks 3-4)
3. Billing & Payments (Weeks 5-6)
4. Shipping (Weeks 7-8)
5. Subscriptions (Weeks 9-10)
6. Wholesale & Invoicing (Weeks 11-12)

## Reference Documentation

For detailed context, reference these planning documents:
- `planning/PURPOSE.md` - Project vision and target customer
- `planning/TECHNICAL.md` - Comprehensive technical decisions and rationale
- `planning/ROADMAP.md` - Feature roadmap and milestones
- `planning/BUSINESS.md` - Market positioning and economics
- `planning/UI_DIRECTION.md` - Complete UI/UX guidelines
