# Architecture Decision Records (ADRs)

## ADR-001: Technology Stack Selection

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need to choose a technology stack for a coffee e-commerce platform that supports retail sales, subscriptions, and B2B operations with a focus on observability and maintainability.

### Decision
- **Backend:** Go with Echo framework
- **Database:** PostgreSQL with SQLC
- **Caching:** Valkey (Redis fork)
- **Events:** NATS with JetStream
- **Monitoring:** Prometheus + Grafana
- **Reverse Proxy:** Caddy
- **Containerization:** Docker + Docker Compose

### Rationale
- **Go:** Excellent performance, strong typing, great concurrency support, extensive library ecosystem
- **Echo:** Lightweight, fast, good middleware support, simpler than Gin for our use case
- **PostgreSQL:** ACID compliance, JSON support, excellent performance, mature ecosystem
- **SQLC:** Type-safe database access, compile-time query validation, better than ORMs for performance
- **Valkey:** Community-driven Redis fork, avoiding licensing concerns, same performance characteristics
- **NATS:** Purpose-built for messaging, excellent performance, simpler than Kafka for our scale
- **Caddy:** Automatic HTTPS, simpler configuration than nginx, perfect for modern deployments

### Consequences
- **Positive:** S3-compatible interface, development/production parity, cost-effective
- **Negative:** Additional infrastructure component for development
- **Migration:** Easy migration from MinIO to S3 with same interface

---

## ADR-008: Configuration Management - Environment Variables vs Database vs Config Files

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need to manage application configuration, business rules, feature flags, and environment-specific settings.

### Decision
**Hybrid approach:** Environment variables for infrastructure, database for business configuration.

### Configuration Strategy
```go
// Environment Variables (Infrastructure)
DATABASE_URL, VALKEY_ADDR, NATS_URL, STRIPE_SECRET_KEY

// Database Configuration (Business Rules)
settings table → tax_rates, shipping_costs, feature_flags, email_templates

// Interface Design
type ConfigService interface {
    GetString(key) (string, error)
    GetInt(key) (int, error)
    SetString(key, value) error
    Watch(key) (<-chan ConfigChange, error)
}
```

### Configuration Categories
```yaml
Infrastructure:     Environment variables (12-factor app)
Business Rules:     Database settings table
Feature Flags:      Database with real-time updates
Secrets:           Environment variables + secret management
Templates:         Database with versioning
```

### Rationale
- **Separation of Concerns:** Infrastructure vs business configuration
- **Runtime Changes:** Business rules can be updated without deployment
- **Security:** Secrets in environment variables, not database
- **Audit Trail:** Database configuration changes are logged
- **Development:** Easy to override with environment variables

### Consequences
- **Positive:** Flexible configuration, runtime updates, proper secret handling
- **Negative:** More complex configuration management
- **Monitoring:** Need to track configuration changes and their impact

---

## ADR-009: Search Implementation - PostgreSQL vs Elasticsearch vs Typesense

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
Coffee customers need to search products by name, description, origin, roast level, and other attributes with filtering and faceting.

### Decision
**Start with PostgreSQL full-text search, design for future Elasticsearch** using ProductSearchService interface.

### Search Strategy
```sql
-- Phase 1: PostgreSQL Full-Text Search
CREATE INDEX products_search_idx ON products 
USING gin(to_tsvector('english', name || ' ' || description));

-- Phase 2: Elasticsearch (when search becomes complex)
-- Separate search index with product synchronization
```

### Implementation
```go
type ProductSearchService interface {
    Search(req SearchRequest) (*SearchResponse, error)
    Autocomplete(query, limit) ([]string, error)
    IndexProduct(product) error
}

// Providers
- PostgreSQLSearchService  (MVP)
- ElasticsearchService    (future)
- TypesenseService        (alternative)
```

### Migration Threshold
Switch to Elasticsearch when:
- Product catalog > 10,000 items
- Search queries > 1,000/day
- Complex faceting requirements
- Performance issues with PostgreSQL

### Rationale
- **Simplicity:** PostgreSQL full-text search is powerful for MVP
- **Cost:** No additional infrastructure components initially
- **Performance:** Adequate for small to medium catalogs
- **Future-Proof:** Interface design allows easy migration

### Consequences
- **Positive:** Simple implementation, no additional infrastructure, fast development
- **Negative:** Limited advanced search features initially
- **Migration Path:** Clear criteria for when to upgrade search infrastructure

---

## ADR-010: Monitoring and Observability - Prometheus vs Alternatives

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need comprehensive monitoring for business metrics, technical performance, and infrastructure health.

### Decision
**Prometheus + Grafana + AlertManager** for complete observability stack.

### Monitoring Strategy
```yaml
Business Metrics:
  - Revenue tracking (real-time)
  - Order conversion rates
  - Cart abandonment
  - Product performance
  - Customer lifetime value

Technical Metrics:
  - API response times
  - Database query performance
  - Cache hit rates
  - Event processing delays
  - Error rates and types

Infrastructure Metrics:
  - CPU, Memory, Disk usage
  - Network throughput
  - Container health
  - External API dependencies
```

### Alerting Strategy
```yaml
Critical Alerts:
  - Application down (2 minutes)
  - High error rate (>5% for 5 minutes)
  - Payment processing failures
  - Database connection issues

Business Alerts:
  - Revenue drop (>50% for 30 minutes)
  - High cart abandonment (>85% for 15 minutes)
  - Inventory stock-outs
  - Failed order processing

Warning Alerts:
  - Slow API responses (>2s for 10 minutes)
  - High memory usage (>80% for 10 minutes)
  - Event processing delays (>1 minute)
```

### Rationale
- **Industry Standard:** Prometheus is the de facto standard for modern monitoring
- **Pull Model:** Better for discovering services and handling network issues
- **Query Language:** PromQL is powerful for complex business metrics
- **Ecosystem:** Extensive exporter ecosystem and Grafana integration
- **Cost:** Open source with no licensing fees

### Consequences
- **Positive:** Comprehensive monitoring, industry-standard tooling, excellent alerting
- **Negative:** Additional infrastructure complexity, learning curve for PromQL
- **Operational:** Need proper retention policies and storage management

---

## ADR-011: Testing Strategy - Unit vs Integration vs E2E

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need a comprehensive testing strategy that balances development speed with confidence in deployments.

### Decision
**Test Pyramid:** Heavy unit tests, selective integration tests, minimal E2E tests.

### Testing Strategy
```
E2E Tests (5%)        → Critical user journeys
Integration Tests (25%) → Service boundaries, database
Unit Tests (70%)       → Business logic, utilities
```

### Test Categories
```go
// Unit Tests
- Business logic functions
- Service interfaces
- Utilities and helpers
- Mock external dependencies

// Integration Tests  
- Database operations
- External API integrations
- Event publishing/consuming
- File storage operations

// E2E Tests
- Complete order flow
- Payment processing
- User registration/login
- Critical admin functions
```

### Test Infrastructure
```yaml
Unit Tests:        Go test with testify
Integration Tests: Docker test containers
E2E Tests:         Playwright or similar
Mocking:          gomock for interface mocking
Test Database:     PostgreSQL in tmpfs
CI/CD:            GitHub Actions
```

### Rationale
- **Fast Feedback:** Unit tests run in milliseconds
- **Confidence:** Integration tests catch interface issues
- **Cost/Benefit:** E2E tests are expensive but catch critical issues
- **Maintainability:** Test pyramid is easier to maintain than heavy E2E

### Consequences
- **Positive:** Fast test execution, high confidence, maintainable test suite
- **Negative:** Initial setup complexity, need for good mocking strategy
- **CI/CD:** Tests must complete in <5 minutes for good developer experience

---

## ADR-012: Deployment Strategy - Docker vs Kubernetes vs Serverless

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need a deployment strategy that supports development, staging, and production environments with proper scaling capabilities.

### Decision
**Docker Compose for development, single Docker host for MVP production,** with Kubernetes migration path.

### Deployment Phases
```yaml
Phase 1 (MVP):
  Development: Docker Compose
  Production:  Single VPS with Docker Compose
  
Phase 2 (Scale):
  Production:  Docker Swarm or managed Kubernetes
  
Phase 3 (Enterprise):
  Production:  Full Kubernetes with auto-scaling
```

### Infrastructure Strategy
```yaml
MVP Production Stack:
  - Single VPS (4-8 GB RAM)
  - Docker Compose with production configs
  - Caddy for reverse proxy + auto-HTTPS
  - Automated backups
  - Basic monitoring

Scaling Triggers:
  - >1000 orders/day → Consider container orchestration
  - >10k users → Move to managed Kubernetes
  - Multi-region → Full cloud-native architecture
```

### Rationale
- **Simplicity:** Docker Compose is simple to understand and deploy
- **Cost:** Single VPS is cost-effective for MVP
- **Migration Path:** Easy to move from Compose → Swarm → Kubernetes
- **Development/Production Parity:** Same containers in all environments

### Consequences
- **Positive:** Simple deployment, low operational overhead, clear scaling path
- **Negative:** Single point of failure for MVP, manual scaling initially
- **Risk Mitigation:** Regular backups, monitoring, documented recovery procedures

---

## ADR-013: API Design - REST vs GraphQL vs gRPC

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need API design for web frontend, mobile apps, and potential B2B integrations.

### Decision
**RESTful JSON APIs with OpenAPI specification** for external interfaces.

### API Design Principles
```yaml
REST Endpoints:
  - /api/v1/products       → Product catalog
  - /api/v1/cart          → Shopping cart
  - /api/v1/orders        → Order management
  - /api/v1/customers     → Customer profiles
  - /api/v1/admin/*       → Administrative functions

HTTP Methods:
  - GET:    Retrieve resources
  - POST:   Create resources  
  - PUT:    Update resources (full)
  - PATCH:  Update resources (partial)
  - DELETE: Remove resources

Response Format:
  - JSON with consistent structure
  - Proper HTTP status codes
  - Error responses with details
  - Pagination for collections
```

### API Versioning
```go
// URL Versioning
/api/v1/products  → Current version
/api/v2/products  → Future version

// Header Versioning (future)
Accept: application/vnd.ecommerce.v1+json
```

### Rationale
- **Simplicity:** REST is well-understood and widely supported
- **Tooling:** Excellent tooling ecosystem (OpenAPI, Postman, etc.)
- **Caching:** HTTP caching works naturally with REST
- **Standards:** Industry standard for e-commerce APIs
- **B2B Ready:** Easy for partners to integrate

### Future Considerations
- **GraphQL:** Consider for complex frontend requirements
- **gRPC:** Consider for internal service communication
- **WebSockets:** For real-time features (order tracking)

### Consequences
- **Positive:** Wide compatibility, excellent tooling, easy to document
- **Negative:** Potential over-fetching, multiple requests for complex operations
- **Documentation:** OpenAPI spec provides automatic documentation

---

## ADR-014: Security Strategy - Authentication, Authorization, and Data Protection

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
E-commerce platforms handle sensitive customer data, payment information, and business data requiring comprehensive security measures.

### Decision
**Defense in depth** with multiple security layers and compliance-first approach.

### Security Layers
```yaml
Application Security:
  - Input validation and sanitization
  - SQL injection prevention (SQLC + parameterized queries)
  - XSS protection (output encoding)
  - CSRF protection (tokens)
  - Rate limiting (per user/IP)

Authentication & Authorization:
  - JWT with short expiration (15 minutes)
  - Refresh token rotation
  - Role-based access control (RBAC)
  - MFA for admin accounts
  - Session management in Valkey

Data Protection:
  - Encryption at rest (database, file storage)
  - Encryption in transit (TLS everywhere)
  - PII data minimization
  - Password hashing (bcrypt)
  - Sensitive data tokenization

Infrastructure Security:
  - Container security scanning
  - Network segmentation
  - Secrets management
  - Regular security updates
  - Backup encryption
```

### Compliance Requirements
```yaml
PCI DSS:
  - No card data storage (Stripe handles)
  - Secure transmission
  - Access controls
  - Regular monitoring

GDPR/CCPA:
  - Data minimization
  - Consent management
  - Right to deletion
  - Data breach notification
  - Privacy by design

Audit Requirements:
  - All sensitive actions logged
  - Immutable audit trail
  - Access logging
  - Change tracking
```

### Rationale
- **Compliance First:** Build compliance into architecture from day one
- **Minimize Risk:** Don't store sensitive payment data
- **Defense in Depth:** Multiple security layers reduce risk
- **Audit Trail:** Complete traceability for compliance and debugging

### Consequences
- **Positive:** Strong security posture, compliance ready, customer trust
- **Negative:** Additional complexity, development overhead
- **Operations:** Need security monitoring and incident response procedures

---

## Summary of Key Architectural Decisions

1. **Technology Stack:** Go + PostgreSQL + Valkey + NATS for performance and reliability
2. **Database:** Single database with schema boundaries for MVP simplicity
3. **Caching:** Valkey for performance data, separate from business events
4. **Events:** NATS JetStream for durable business events and workflows
5. **Authentication:** JWT + refresh tokens with RBAC for security and scalability
6. **Payments:** Stripe primary with interface for future providers
7. **File Storage:** MinIO development, S3 production with consistent interface
8. **Configuration:** Hybrid approach - env vars for infrastructure, database for business rules
9. **Search:** PostgreSQL full-text for MVP, interface for future Elasticsearch
10. **Monitoring:** Prometheus + Grafana for comprehensive observability
11. **Testing:** Test pyramid with focus on unit tests and selective integration
12. **Deployment:** Docker Compose for simplicity with Kubernetes migration path
13. **API Design:** RESTful JSON with OpenAPI for broad compatibility
14. **Security:** Defense in depth with compliance-first approach

These decisions provide a solid foundation for MVP development while maintaining flexibility for future scaling and feature additions. High performance, type safety, excellent observability, simple deployment
- **Negative:** Go learning curve for team members unfamiliar with the language
- **Risks:** Smaller talent pool for Go developers compared to Node.js/Python

---

## ADR-002: Database Architecture - Single Database vs Microservices

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need to decide between a single shared database or separate databases per service boundary (products, orders, customers, etc.).

### Decision
**Single PostgreSQL database with clear schema boundaries** for MVP, with the option to split later.

### Rationale
- **Simplified Operations:** Single database to backup, monitor, and maintain
- **ACID Transactions:** Cross-entity operations (order creation affecting inventory) are simpler
- **Development Speed:** Faster development with fewer moving parts
- **Cost Effective:** Single database instance reduces infrastructure costs
- **Schema Migration:** Easier to manage schema changes with single database
- **Future Flexibility:** Can split into separate databases when scale demands it

### Implementation Strategy
```sql
-- Clear schema boundaries
CREATE SCHEMA products;    -- Product catalog, inventory
CREATE SCHEMA customers;   -- User accounts, profiles
CREATE SCHEMA orders;      -- Orders, payments
CREATE SCHEMA analytics;   -- Reporting, metrics
CREATE SCHEMA admin;       -- Configuration, settings
```

### Consequences
- **Positive:** Faster development, simpler operations, stronger consistency
- **Negative:** Potential for coupling between domains
- **Migration Path:** Plan for database splitting when we reach ~100k orders/month

---

## ADR-003: Caching Strategy - Valkey vs Redis vs In-Memory

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need a caching layer for session storage, product catalogs, and performance optimization. Options include Redis, Valkey, or application-level caching.

### Decision
**Valkey for external caching** with clear separation from business logic.

### Rationale
- **Valkey Benefits:** Community-driven Redis fork, no licensing concerns, same performance
- **External vs In-Memory:** Persistent across application restarts, shared between instances
- **Use Cases:**
  - Session storage (shopping carts, user sessions)
  - Product catalog caching (reduce database load)
  - Rate limiting (API throttling)
  - Temporary data (OTP codes, locks)

### Cache Strategy
```go
// Cache TTL Strategy
Product Catalog:     1 hour
Shopping Carts:      24 hours  
User Sessions:       30 days
Rate Limits:         1 minute
OTP Codes:          5 minutes
```

### Consequences
- **Positive:** Improved performance, reduced database load, session persistence
- **Negative:** Additional infrastructure component to manage
- **Fallback:** All cached data can be regenerated from primary database

---

## ADR-004: Event Architecture - NATS vs Valkey Streams vs Database Events

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need an event-driven architecture for order processing, audit logs, and business workflows.

### Decision
**NATS JetStream for durable business events** with clear separation from caching.

### Event Boundaries
- **NATS:** Business-critical events (order created, payment confirmed, inventory updated)
- **Valkey:** Performance events (cache invalidation, session updates)
- **Database:** Audit logs requiring compliance (stored permanently)

### Rationale
- **Durability:** JetStream ensures business events are never lost
- **Ordering:** Maintains event sequence for workflow processing
- **Replay:** Can replay events for debugging or new subscribers
- **Scalability:** Purpose-built for high-throughput messaging
- **Separation:** Clear boundary between caching (Valkey) and eventing (NATS)

### Event Design
```go
// Event Naming Convention
"orders.created"      // Order workflow events
"payments.confirmed"  // Payment processing
"inventory.updated"   // Stock changes
"audit.user_action"   // Compliance logging
```

### Consequences
- **Positive:** Reliable event processing, audit capability, loose coupling
- **Negative:** Additional complexity in error handling and monitoring
- **Monitoring:** Need to track event processing delays and failures

---

## ADR-005: Authentication Strategy - JWT vs Sessions vs OAuth

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need authentication for customers, staff, and B2B users with different access levels.

### Decision
**JWT tokens with refresh tokens stored in Valkey** for stateless authentication.

### Authentication Flow
```
1. Login → JWT access token (15 min) + refresh token (30 days)
2. Store refresh token in Valkey with user session data
3. Access token contains: user_id, roles, permissions, session_id
4. Refresh token rotation on renewal
5. OAuth integration for social login (future)
```

### Role-Based Access Control
```go
Roles:
- customer:     Product browsing, cart, orders
- staff:        Order management, inventory
- admin:        Full system access
- b2b_customer: Wholesale pricing, bulk orders
```

### Rationale
- **Stateless:** JWT allows horizontal scaling without shared state
- **Security:** Short-lived access tokens limit exposure
- **Flexibility:** Easy to add OAuth providers later
- **Session Management:** Valkey stores session metadata for logout/revocation

### Consequences
- **Positive:** Scalable, secure, flexible for future OAuth integration
- **Negative:** Token management complexity, need for refresh token rotation
- **Security:** Must handle token storage securely in frontend

---

## ADR-006: Payment Processing - Stripe vs Multiple Providers

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need payment processing for retail customers and subscription management, with potential future support for B2B invoicing.

### Decision
**Start with Stripe, design for multiple providers** using the PaymentProvider interface.

### Implementation Strategy
```go
// Provider Interface Design
type PaymentProvider interface {
    CreateCheckoutSession(req CheckoutRequest) (*CheckoutResponse, error)
    VerifyWebhook(payload, signature) (*WebhookEvent, error)
    RefundPayment(paymentID, amount) (*RefundResponse, error)
}

// Providers
- StripePaymentProvider  (primary)
- PayPalPaymentProvider  (future)
- SquarePaymentProvider  (future)
```

### Rationale
- **Stripe Benefits:** Excellent developer experience, comprehensive APIs, subscription support
- **Interface Design:** Easy to add PayPal, Square, or other providers later
- **Webhook Handling:** Standardized webhook processing across providers
- **B2B Future:** Stripe Invoicing for NET-30 terms

### Consequences
- **Positive:** Fast implementation, room for growth, provider independence
- **Negative:** Stripe fees (2.9% + 30¢), potential vendor lock-in
- **Risk Mitigation:** Interface design allows easy provider switching

---

## ADR-007: File Storage - Local vs Cloud vs Self-Hosted

**Date:** 2025-06-03  
**Status:** Accepted  
**Deciders:** Development Team  

### Context
We need file storage for product images, invoices, shipping labels, and user-generated content.

### Decision
**MinIO for development, AWS S3 for production** using FileStorage interface.

### Storage Strategy
```go
// Bucket Organization
ecommerce-products/   → Product images, catalogs (public)
ecommerce-invoices/   → Invoices, receipts (private)
ecommerce-assets/     → Static assets, logos (public)
ecommerce-uploads/    → User uploads (private)
```

### Implementation
```go
type FileStorage interface {
    Store(key, data, options) (*FileResult, error)
    Retrieve(key) ([]byte, error)
    GetURL(key) (string, error)
    ProcessImage(key, operations) (*FileResult, error)
}

// Providers
- MinIOFileStorage    (development)
- S3FileStorage      (production)
- LocalFileStorage   (testing)
```

### Rationale
- **MinIO Benefits:** S3-compatible, self-hosted, no vendor lock-in
- **Development Simplicity:** Docker-based local storage
- **Production Scaling:** AWS S3 for reliability and CDN integration
- **Cost Control:** MinIO for development, S3 only for production traffic

### Consequences
- **Positive:**