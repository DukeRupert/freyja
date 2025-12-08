# WTF Patterns Implementation Checklist

This document tracks the adoption of architectural patterns from Ben Bjohnson's WTF Dial project into Freyja.

## Overview

These patterns improve code organization, testability, and maintainability—particularly important for a multi-tenant SaaS application maintained by a solo developer.

---

## 1. Application Error Codes

**Status:** [x] Complete (Phase 1)

**Goal:** Consistent error handling with typed error codes that map cleanly to HTTP status codes while hiding internal details from users.

### Tasks

- [x] Create `internal/domain/error.go` with error codes and types
- [x] Implement `Error` struct with `Code`, `Message`, `Op`, and `Err` fields
- [x] Add helper functions: `ErrorCode()`, `ErrorMessage()`, `ErrorOp()`, `Errorf()`, `WrapError()`
- [x] Define standard codes: `ECONFLICT`, `EINTERNAL`, `EINVALID`, `ENOTFOUND`, `EUNAUTHORIZED`, `EFORBIDDEN`, `EPAYMENT`, `ERATELIMIT`, `EGONE`, `ENOTIMPL`
- [x] Create HTTP error response helper (`internal/handler/error.go`) that maps codes to status codes
- [x] Add `ValidationError` for field-level form validation errors
- [x] Add convenience functions: `NotFound()`, `Unauthorized()`, `Forbidden()`, `Invalid()`, `Conflict()`, `Internal()`
- [x] Add multi-tenant errors: `ErrTenantMismatch`, `ErrTenantRequired`
- [x] Add tests for domain errors and HTTP response helpers
- [ ] Update existing handlers to use the new error system (incremental migration)
- [ ] Add error reporting hooks for observability integration (future)

### Implementation Reference

```go
// internal/domain/errors.go
package domain

const (
    ECONFLICT       = "conflict"
    EINTERNAL       = "internal"
    EINVALID        = "invalid"
    ENOTFOUND       = "not_found"
    EUNAUTHORIZED   = "unauthorized"
    EFORBIDDEN      = "forbidden"
)

type Error struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func (e *Error) Error() string {
    return e.Message
}

func ErrorCode(err error) string {
    if err == nil {
        return ""
    }
    if e, ok := err.(*Error); ok {
        return e.Code
    }
    return EINTERNAL
}

func ErrorMessage(err error) string {
    if err == nil {
        return ""
    }
    if e, ok := err.(*Error); ok {
        return e.Message
    }
    return "Internal error."
}

func Errorf(code string, format string, args ...interface{}) *Error {
    return &Error{
        Code:    code,
        Message: fmt.Sprintf(format, args...),
    }
}
```

---

## 2. Service Interfaces in Domain Package

**Status:** [x] Complete (Phase 1 - Interface Definitions)

**Goal:** Define service contracts in the domain package, keeping implementations separate for clean dependency flow.

### Tasks

- [x] Audit existing service interfaces and their locations
- [x] Create service interfaces in `internal/domain/`
- [ ] Add compile-time interface checks to implementations (deferred - requires type alignment)
- [ ] Update import paths throughout codebase (incremental migration)
- [x] Document interface contracts with comments

### Files Created

- `internal/domain/product.go` - ProductService interface + types
- `internal/domain/user.go` - UserService interface + SessionData type
- `internal/domain/cart.go` - CartService interface + Cart/CartSummary/CartItem types
- `internal/domain/order.go` - OrderService interface + OrderDetail type
- `internal/domain/checkout.go` - CheckoutService interface + supporting types

### Migration Notes

The domain interfaces are now the canonical definitions. The service package still has its own
type definitions that match these interfaces. Future work:
- Migrate handlers to use `domain.ProductService` instead of `service.ProductService`
- Add compile-time checks once service implementations use domain types
- This is an incremental migration - both patterns coexist during transition

### Implementation Reference

```go
// internal/domain/product.go
package domain

type ProductService interface {
    FindProductByID(ctx context.Context, id uuid.UUID) (*Product, error)
    FindProducts(ctx context.Context, filter ProductFilter) ([]*Product, int, error)
    CreateProduct(ctx context.Context, product *Product) error
    UpdateProduct(ctx context.Context, id uuid.UUID, update ProductUpdate) (*Product, error)
    DeleteProduct(ctx context.Context, id uuid.UUID) error
}

// Compile-time check in implementation file:
// var _ domain.ProductService = (*ProductService)(nil)
```

---

## 3. Tenant Context Helpers

**Status:** [ ] Not Started

**Goal:** Centralized tenant extraction from context, making tenant isolation bugs harder to write.

### Tasks

- [ ] Create `internal/domain/context.go` with context helpers
- [ ] Implement `NewContextWithTenant()` and `TenantFromContext()`
- [ ] Add `TenantIDFromContext()` convenience function
- [ ] Update middleware to set tenant in context after authentication
- [ ] Update repository layer to use context helpers
- [ ] Add panic/error if tenant missing where required

### Implementation Reference

```go
// internal/domain/context.go
package domain

type contextKey int

const (
    tenantContextKey contextKey = iota
    userContextKey
    flashContextKey
)

func NewContextWithTenant(ctx context.Context, tenant *Tenant) context.Context {
    return context.WithValue(ctx, tenantContextKey, tenant)
}

func TenantFromContext(ctx context.Context) *Tenant {
    tenant, _ := ctx.Value(tenantContextKey).(*Tenant)
    return tenant
}

func TenantIDFromContext(ctx context.Context) uuid.UUID {
    if tenant := TenantFromContext(ctx); tenant != nil {
        return tenant.ID
    }
    return uuid.Nil
}

// RequireTenantID panics if tenant is not in context - use in repository layer
func RequireTenantID(ctx context.Context) uuid.UUID {
    id := TenantIDFromContext(ctx)
    if id == uuid.Nil {
        panic("tenant_id required in context but not found")
    }
    return id
}
```

---

## 4. Filter/Update Structs

**Status:** [ ] Not Started

**Goal:** Flexible, extensible query filtering and partial updates using pointer fields.

### Tasks

- [ ] Create filter structs for major entities (Product, Customer, Order)
- [ ] Create update structs with pointer fields for partial updates
- [ ] Update repository methods to accept filter/update structs
- [ ] Add query builder helpers for constructing WHERE clauses
- [ ] Document nil vs zero-value semantics

### Implementation Reference

```go
// internal/domain/product.go
package domain

type ProductFilter struct {
    TenantID    uuid.UUID   // Required - always filter by tenant
    ID          *uuid.UUID
    CategoryID  *uuid.UUID
    RoastLevel  *string
    IsActive    *bool
    PriceListID *uuid.UUID  // Filter by visibility in price list
    Search      *string     // Full-text search
    Offset      int
    Limit       int
}

type ProductUpdate struct {
    Name        *string
    Description *string
    RoastLevel  *string
    Origin      *string
    IsActive    *bool
    // nil = no change, pointer to value = update
}

type OrderFilter struct {
    TenantID    uuid.UUID
    CustomerID  *uuid.UUID
    Status      *OrderStatus
    CreatedFrom *time.Time
    CreatedTo   *time.Time
    Offset      int
    Limit       int
}
```

---

## 5. Compile-Time Interface Checks

**Status:** [ ] Not Started

**Goal:** Catch missing interface methods at compile time rather than runtime.

### Tasks

- [ ] Add interface checks to all service implementations
- [ ] Add interface checks to billing provider implementations
- [ ] Add interface checks to email provider implementations
- [ ] Add interface checks to storage implementations
- [ ] Document pattern in CLAUDE.md

### Implementation Reference

```go
// internal/billing/stripe.go
package billing

import "freyja/internal/domain"

var _ domain.BillingService = (*StripeProvider)(nil)

type StripeProvider struct {
    // ...
}
```

---

## 6. Function-Injection Mocks

**Status:** [ ] Not Started

**Goal:** Simple, dependency-free mocks for unit testing.

### Tasks

- [ ] Create `internal/mock/` package
- [ ] Implement mock structs for major services
- [ ] Add compile-time interface checks to mocks
- [ ] Write example tests using the mock pattern
- [ ] Document mock usage in test files

### Implementation Reference

```go
// internal/mock/product.go
package mock

import (
    "context"
    "freyja/internal/domain"
    "github.com/google/uuid"
)

var _ domain.ProductService = (*ProductService)(nil)

type ProductService struct {
    FindProductByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Product, error)
    FindProductsFn    func(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int, error)
    CreateProductFn   func(ctx context.Context, product *domain.Product) error
    UpdateProductFn   func(ctx context.Context, id uuid.UUID, update domain.ProductUpdate) (*domain.Product, error)
    DeleteProductFn   func(ctx context.Context, id uuid.UUID) error
}

func (s *ProductService) FindProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
    return s.FindProductByIDFn(ctx, id)
}

func (s *ProductService) FindProducts(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int, error) {
    return s.FindProductsFn(ctx, filter)
}

// ... other methods
```

### Example Test Usage

```go
func TestProductHandler_Get(t *testing.T) {
    productSvc := &mock.ProductService{
        FindProductByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
            return &domain.Product{
                ID:   id,
                Name: "Test Espresso Blend",
            }, nil
        },
    }

    handler := NewProductHandler(productSvc)
    // ... test the handler
}
```

---

## 7. Transaction Helper with Testable Time

**Status:** [ ] Not Started

**Goal:** Consistent timestamps within transactions and injectable time for testing.

### Tasks

- [ ] Add `Now func() time.Time` field to database struct
- [ ] Create transaction wrapper that captures timestamp at start
- [ ] Update repository methods to use transaction timestamp
- [ ] Add tests demonstrating time injection

---

## Implementation Priority

1. **Error Codes** - Foundation for consistent error handling
2. **Context Helpers** - Critical for multi-tenant safety
3. **Service Interfaces** - Clean architecture foundation
4. **Filter/Update Structs** - Improve query flexibility
5. **Interface Checks** - Low effort, high safety
6. **Mocks** - Enable better testing
7. **Transaction Helpers** - Nice to have for testing

---

## References

- [WTF Dial Repository](https://github.com/benbjohnson/wtf)
- [Standard Package Layout](https://www.gobeyond.dev/standard-package-layout/) - Ben Bjohnson's blog post
- [WTF Dial Architecture](https://www.gobeyond.dev/wtf-dial/) - Detailed walkthrough
