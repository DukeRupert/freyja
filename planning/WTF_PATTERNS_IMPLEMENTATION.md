# WTF Patterns Implementation Checklist

This document tracks the adoption of architectural patterns from Ben Johnson's WTF Dial project into Freyja.

**Last Updated:** December 8, 2024

## Current Status Summary

| Pattern | Status |
|---------|--------|
| 1. Application Error Codes | ✅ Complete |
| 2. Service Interfaces in Domain Package | ✅ Complete |
| 3. Tenant Context Helpers | ✅ Complete |
| 4. Filter/Update Structs | ⏳ Future |
| 5. Compile-Time Interface Checks | ⏳ Future |
| 6. Function-Injection Mocks | ⏳ Future |
| 7. Transaction Helper with Testable Time | ⏳ Future |

The core WTF Dial patterns (error handling, service interfaces, context helpers) have been fully implemented. The remaining patterns are optional enhancements that can be added as needed.

## Overview

These patterns improve code organization, testability, and maintainability—particularly important for a multi-tenant SaaS application maintained by a solo developer.

---

## 1. Application Error Codes

**Status:** ✅ Complete

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
- [x] Handlers use domain error system via `handler.ErrorResponse()`
- [x] Provider packages (billing, tax, storage, shipping) migrated to typed errors

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

**Status:** ✅ Complete

**Goal:** Define service contracts in the domain package, keeping implementations separate for clean dependency flow.

### Tasks

- [x] Audit existing service interfaces and their locations
- [x] Create service interfaces in `internal/domain/`
- [x] Migrate handlers to use `domain.XxxService` interfaces
- [x] Service implementations use type aliases for backwards compatibility
- [x] Document interface contracts with comments
- [x] Domain errors defined per-domain and re-exported from `service/errors.go`

### Files Created

- `internal/domain/product.go` - ProductService interface + types + errors
- `internal/domain/user.go` - UserService interface + SessionData type + errors
- `internal/domain/cart.go` - CartService interface + Cart/CartSummary/CartItem types + errors
- `internal/domain/order.go` - OrderService interface + OrderDetail type + errors
- `internal/domain/subscription.go` - SubscriptionService interface + types + errors
- `internal/domain/invoice.go` - InvoiceService interface + types + errors
- `internal/domain/checkout.go` - CheckoutService interface + supporting types

### Migration Completed (December 8, 2024)

All service interfaces migrated to domain package:
1. Product domain - `internal/domain/product.go`
2. Customer/User domain - `internal/domain/user.go`
3. Cart domain - `internal/domain/cart.go`
4. Order domain - `internal/domain/order.go`
5. Subscription domain - `internal/domain/subscription.go`
6. Invoice domain - `internal/domain/invoice.go`

Handlers now depend on `domain.XxxService` interfaces. Service implementations use type aliases:
```go
type ProductService = domain.ProductService
```

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

**Status:** ✅ Complete

**Goal:** Centralized tenant extraction from context, making tenant isolation bugs harder to write.

### Tasks

- [x] Create `internal/domain/context.go` with context helpers
- [x] Implement `NewContextWithTenant()` and `TenantFromContext()`
- [x] Add `TenantIDFromContext()` and `RequireTenantID()` convenience functions
- [x] Add `MustTenant()` for cases needing full tenant struct
- [x] Add user context helpers (`NewContextWithUser()`, `UserFromContext()`, etc.)
- [x] Add operator context helpers (`NewContextWithOperator()`, `OperatorFromContext()`, etc.)
- [x] Add request ID context helpers
- [x] Add convenience helpers (`IsAuthenticated()`, `IsOperator()`, `IsOwner()`, `HasTenant()`)
- [x] Add tests for all context helpers

### Files Created

- `internal/domain/context.go` - Context helpers for tenant, user, operator, request ID
- `internal/domain/context_test.go` - Comprehensive tests for all helpers

### Domain Types

```go
// Minimal structs for context storage (full records fetched from DB if needed)
type Tenant struct {
    ID     uuid.UUID
    Slug   string
    Name   string
    Status string
}

type User struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    Email       string
    AccountType string // "customer", "admin", "wholesale"
}

type Operator struct {
    ID       uuid.UUID
    TenantID uuid.UUID
    Email    string
    Role     string // "owner", "admin", "staff"
    Status   string // "active", "inactive"
}
```

### Migration Notes

The middleware package (`internal/middleware/`) currently defines its own context keys
and helper functions. These will be migrated to use `domain.NewContextWithTenant()` etc.
in future work. Both patterns coexist during the transition.

### Implementation Reference

```go
// internal/domain/context.go
package domain

type contextKey int

const (
    tenantContextKey contextKey = iota
    userContextKey
    operatorContextKey
    requestIDContextKey
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

**Status:** ⏳ Future Enhancement

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

**Status:** ⏳ Future Enhancement

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

**Status:** ⏳ Future Enhancement

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

**Status:** ⏳ Future Enhancement

**Goal:** Consistent timestamps within transactions and injectable time for testing.

### Tasks

- [ ] Add `Now func() time.Time` field to database struct
- [ ] Create transaction wrapper that captures timestamp at start
- [ ] Update repository methods to use transaction timestamp
- [ ] Add tests demonstrating time injection

---

## Implementation Priority

**Completed (December 8, 2024):**
1. ✅ **Error Codes** - Foundation for consistent error handling
2. ✅ **Context Helpers** - Critical for multi-tenant safety
3. ✅ **Service Interfaces** - Clean architecture foundation

**Future Enhancements (as needed):**
4. ⏳ **Filter/Update Structs** - Improve query flexibility
5. ⏳ **Interface Checks** - Low effort, high safety
6. ⏳ **Mocks** - Enable better testing
7. ⏳ **Transaction Helpers** - Nice to have for testing

---

## References

- [WTF Dial Repository](https://github.com/benbjohnson/wtf)
- [Standard Package Layout](https://www.gobeyond.dev/standard-package-layout/) - Ben Bjohnson's blog post
- [WTF Dial Architecture](https://www.gobeyond.dev/wtf-dial/) - Detailed walkthrough
