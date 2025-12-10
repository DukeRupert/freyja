# WTF Dial Pattern Refactor Guide

## Overview

This guide documents the architectural refactor of Freyja to follow the **WTF Dial** pattern articulated by Ben Johnson. The WTF Dial is a pragmatic approach to organizing Go code that balances simplicity with maintainability.

### What is the WTF Dial?

The WTF Dial pattern organizes code based on a simple principle: **domain concepts live at the package root, implementations live in subpackages**.

```
freyja/
├── product.go           # Domain types + ProductService interface (WHAT)
├── customer.go          # Domain types + CustomerService interface (WHAT)
├── postgres/
│   ├── product.go       # PostgreSQL implementation (HOW)
│   └── customer.go      # PostgreSQL implementation (HOW)
└── http/
    ├── product.go       # HTTP handlers (HOW)
    └── customer.go      # HTTP handlers (HOW)
```

**Key insight:** When someone asks "WTF does this package do?", the answer is right there at the root—domain types and interfaces. When they ask "HOW does it work?", they look in the implementation subpackages.

### Why Are We Adopting This Pattern?

**Current problems:**
- `internal/domain/` contains only interfaces and DTOs—domain types are split across `repository.Product`, `service.ProductService`, `handler.ProductHandler`
- Finding all product-related code requires searching 4+ directories
- Business logic is scattered between `service/` and `handler/` layers
- The "onion architecture" adds layers without clear value for a solo-maintained app

**WTF Dial benefits:**
1. **Co-location by domain**: All product code is in `product.go` or `postgres/product.go` or `http/product.go`
2. **Clear boundaries**: The root package defines the contract (interface), subpackages implement it
3. **Flat is better than nested**: No artificial layers (service/repository/handler abstractions)
4. **Easier to navigate**: `grep -r "type Product struct"` finds it at the root immediately
5. **Testable**: Interfaces at the root make mocking straightforward

### Trade-offs We're Accepting

**What we're gaining:**
- Domain-focused organization (all product code together)
- Clearer dependency flow (postgres depends on freyja, not the reverse)
- Less indirection (no service layer wrapping repository calls)
- Simpler mental model for a solo maintainer

**What we're giving up:**
- Traditional layered architecture (we don't need it at this scale)
- Separation between "business logic" and "data access" (the line is blurry anyway)
- Some Go community conventions (internal/ pattern, service/repository split)

**Is this the "right" approach?** For a 10-person team building a generic SaaS platform, probably not. For a solo developer building a domain-specific coffee roaster platform, yes.

---

## Design Principles

### 1. Domain Types at the Root

All domain types live in `freyja/*.go` files. These are the core concepts of the business:
- `Product`, `Customer`, `Order`, `Subscription`, `Invoice`, etc.
- Domain-specific value objects: `RoastLevel`, `GrindOption`, `PaymentTerms`
- Domain errors: `ErrProductNotFound`, `ErrInvalidPrice`

**Rule:** If it's a business concept, it goes at the root.

### 2. Interfaces Define Contracts, Not Implementations

Each domain has **one primary service interface** that defines all operations:

```go
// freyja/product.go
type ProductService interface {
    ListProducts(ctx context.Context, filter ProductFilter) ([]Product, error)
    GetProductBySlug(ctx context.Context, slug string) (*ProductDetail, error)
    CreateProduct(ctx context.Context, params CreateProductParams) (*Product, error)
    UpdateProduct(ctx context.Context, id uuid.UUID, params UpdateProductParams) error
    DeleteProduct(ctx context.Context, id uuid.UUID) error
}
```

**Rule:** Use one comprehensive interface per domain, not many small interfaces. Split only when there's a clear reason (e.g., separate read/write concerns).

### 3. Implementations in Subpackages

Concrete implementations live in subpackages organized by *mechanism*, not by *layer*:

- `freyja/postgres/` - PostgreSQL database implementation
- `freyja/http/` - HTTP handlers (admin + storefront)
- `freyja/worker/` - Background job workers
- `freyja/billing/` - External billing provider adapters (Stripe)
- `freyja/email/` - External email provider adapters
- `freyja/shipping/` - External shipping provider adapters

**Rule:** Subpackage imports `freyja`, never the reverse. The root package has no dependencies on its implementation packages.

### 4. Keep External Adapters Separate

Billing, email, shipping, and storage stay in their own top-level packages:
- These are **horizontal concerns** (used by multiple domains)
- They have their own interface/implementation split already
- Moving them would provide no value and complicate the refactor

**Rule:** Only refactor domain-specific code. Leave cross-cutting concerns alone unless there's a clear reason.

### 5. Use pgtype for Database Types

Domain types use `pgtype.UUID`, `pgtype.Text`, `pgtype.Timestamptz` directly:

```go
type Product struct {
    ID          pgtype.UUID
    TenantID    pgtype.UUID
    Name        string
    Description pgtype.Text  // nullable
    CreatedAt   pgtype.Timestamptz
}
```

**Why?** This keeps sqlc integration simple—no type mapping needed. The nil-safety of `pgtype.Text{Valid: false}` is explicit and intentional.

**Trade-off:** Domain types have a PostgreSQL dependency. For this project, that's acceptable—we're not swapping databases.

### 6. Incremental Migration

Migrate one domain at a time, starting with Product:

1. Create `freyja/product.go` with types and interface
2. Create `freyja/postgres/product.go` with implementation
3. Update `freyja/http/product.go` handlers
4. Update tests
5. Delete old `internal/domain/product.go`, `internal/service/product.go`
6. Repeat for next domain

**Rule:** Each domain migration is a complete, working increment. Don't half-migrate.

---

## Root Domain File Template

Here's what a root domain file should look like, using Product as the reference:

```go
// freyja/product.go
package freyja

import (
    "context"
    "github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// Domain Types
// =============================================================================

// Product represents a coffee product in the catalog.
// Products belong to a tenant and can be public, private, or wholesale-only.
type Product struct {
    ID               pgtype.UUID
    TenantID         pgtype.UUID
    Name             string
    Slug             string
    Description      pgtype.Text
    ShortDescription pgtype.Text

    // Coffee-specific attributes
    Origin       pgtype.Text
    Region       pgtype.Text
    Producer     pgtype.Text
    Process      pgtype.Text
    RoastLevel   pgtype.Text
    TastingNotes []string
    ElevationMin pgtype.Int4
    ElevationMax pgtype.Int4

    // Visibility and status
    Status     ProductStatus
    Visibility ProductVisibility

    // White label support
    IsWhiteLabel         bool
    BaseProductID        pgtype.UUID
    WhiteLabelCustomerID pgtype.UUID

    // Metadata
    SortOrder int32
    CreatedAt pgtype.Timestamptz
    UpdatedAt pgtype.Timestamptz
}

// ProductStatus represents the lifecycle state of a product.
type ProductStatus string

const (
    ProductStatusDraft    ProductStatus = "draft"
    ProductStatusActive   ProductStatus = "active"
    ProductStatusArchived ProductStatus = "archived"
)

// ProductVisibility controls who can see a product.
type ProductVisibility string

const (
    ProductVisibilityPublic    ProductVisibility = "public"     // Anyone
    ProductVisibilityWholesale ProductVisibility = "wholesale"  // Wholesale customers only
    ProductVisibilityPrivate   ProductVisibility = "private"    // Hidden
)

// ProductSKU represents a variant of a product (weight + grind combination).
type ProductSKU struct {
    ID           pgtype.UUID
    ProductID    pgtype.UUID
    SKU          string
    WeightValue  string
    WeightUnit   string
    Grind        string

    // Inventory
    StockQuantity    int32
    StockStatus      StockStatus
    LowStockWarning  pgtype.Int4
    BackorderEnabled bool

    // Metadata
    SortOrder int32
    CreatedAt pgtype.Timestamptz
    UpdatedAt pgtype.Timestamptz
}

// StockStatus represents inventory availability.
type StockStatus string

const (
    StockStatusInStock    StockStatus = "in_stock"
    StockStatusOutOfStock StockStatus = "out_of_stock"
    StockStatusBackorder  StockStatus = "backorder"
)

// ProductImage represents a product image with ordering and alt text.
type ProductImage struct {
    ID        pgtype.UUID
    ProductID pgtype.UUID
    URL       string
    AltText   pgtype.Text
    SortOrder int32
    CreatedAt pgtype.Timestamptz
}

// =============================================================================
// Aggregates (rich domain objects returned by service)
// =============================================================================

// ProductDetail aggregates product information with SKUs, pricing, and images.
// Returned by GetProductBySlug for product detail pages.
type ProductDetail struct {
    Product Product
    SKUs    []ProductSKU
    Images  []ProductImage
}

// ProductWithPrice combines a product with resolved pricing for a price list.
// Returned by ListProducts for catalog pages.
type ProductWithPrice struct {
    Product    Product
    BaseSKU    ProductSKU      // Smallest weight SKU for display
    BasePrice  int32           // Price in cents
    ImageURL   pgtype.Text
}

// SKUCheckoutDetail contains all information needed to display a SKU in checkout.
type SKUCheckoutDetail struct {
    SKUID                   pgtype.UUID
    SKU                     string
    WeightValue             string
    WeightUnit              string
    Grind                   string
    PriceCents              int32
    ProductName             string
    ProductSlug             string
    ProductShortDescription string
    ProductOrigin           string
    ProductRoastLevel       string
    ProductImageURL         string
}

// =============================================================================
// Service Interface
// =============================================================================

// ProductService provides operations for the product catalog.
// Implementations must enforce tenant isolation.
type ProductService interface {
    // Query operations
    ListProducts(ctx context.Context, filter ProductFilter) ([]ProductWithPrice, error)
    GetProductBySlug(ctx context.Context, slug string) (*ProductDetail, error)
    GetProductByID(ctx context.Context, id pgtype.UUID) (*Product, error)
    GetSKUForCheckout(ctx context.Context, skuID pgtype.UUID) (*SKUCheckoutDetail, error)

    // Admin operations
    CreateProduct(ctx context.Context, params CreateProductParams) (*Product, error)
    UpdateProduct(ctx context.Context, id pgtype.UUID, params UpdateProductParams) error
    DeleteProduct(ctx context.Context, id pgtype.UUID) error

    // SKU operations
    CreateSKU(ctx context.Context, params CreateSKUParams) (*ProductSKU, error)
    UpdateSKU(ctx context.Context, id pgtype.UUID, params UpdateSKUParams) error
    DeleteSKU(ctx context.Context, id pgtype.UUID) error

    // Image operations
    AddProductImage(ctx context.Context, productID pgtype.UUID, url string, altText string) (*ProductImage, error)
    DeleteProductImage(ctx context.Context, imageID pgtype.UUID) error
    ReorderProductImages(ctx context.Context, productID pgtype.UUID, imageIDs []pgtype.UUID) error
}

// =============================================================================
// Filter and Parameter Types
// =============================================================================

// ProductFilter defines criteria for listing products.
type ProductFilter struct {
    Status       *ProductStatus       // nil = all
    Visibility   *ProductVisibility   // nil = all
    RoastLevel   *string              // nil = all
    Origin       *string              // nil = all
    IsWhiteLabel *bool                // nil = all
    Search       string               // empty = no search
    Limit        int                  // 0 = default (50)
    Offset       int                  // 0 = from start
}

// CreateProductParams contains fields for creating a product.
type CreateProductParams struct {
    Name             string
    Slug             string
    ShortDescription string
    Description      string
    Status           ProductStatus
    Visibility       ProductVisibility

    // Coffee attributes
    Origin       string
    Region       string
    Producer     string
    Process      string
    RoastLevel   string
    TastingNotes []string
    ElevationMin int32
    ElevationMax int32

    // White label
    IsWhiteLabel         bool
    BaseProductID        *pgtype.UUID
    WhiteLabelCustomerID *pgtype.UUID
}

// UpdateProductParams contains fields that can be updated.
// All fields are pointers—nil means "don't change".
type UpdateProductParams struct {
    Name             *string
    Slug             *string
    ShortDescription *string
    Description      *string
    Status           *ProductStatus
    Visibility       *ProductVisibility

    Origin       *string
    Region       *string
    Producer     *string
    Process      *string
    RoastLevel   *string
    TastingNotes *[]string
    ElevationMin *int32
    ElevationMax *int32

    SortOrder *int32
}

// CreateSKUParams contains fields for creating a product SKU.
type CreateSKUParams struct {
    ProductID        pgtype.UUID
    SKU              string
    WeightValue      string
    WeightUnit       string
    Grind            string
    StockQuantity    int32
    LowStockWarning  int32
    BackorderEnabled bool
}

// UpdateSKUParams contains fields for updating a SKU.
type UpdateSKUParams struct {
    SKU              *string
    WeightValue      *string
    WeightUnit       *string
    Grind            *string
    StockQuantity    *int32
    StockStatus      *StockStatus
    LowStockWarning  *int32
    BackorderEnabled *bool
    SortOrder        *int32
}

// =============================================================================
// Domain Errors
// =============================================================================

// Sentinel errors for product operations.
var (
    ErrProductNotFound     = &Error{Code: ENOTFOUND, Message: "Product not found"}
    ErrProductSlugExists   = &Error{Code: ECONFLICT, Message: "Product slug already exists"}
    ErrSKUNotFound         = &Error{Code: ENOTFOUND, Message: "Product SKU not found"}
    ErrSKUCodeExists       = &Error{Code: ECONFLICT, Message: "SKU code already exists"}
    ErrInvalidRoastLevel   = &Error{Code: EINVALID, Message: "Invalid roast level"}
    ErrInvalidGrindOption  = &Error{Code: EINVALID, Message: "Invalid grind option"}
    ErrCannotDeleteProduct = &Error{Code: ECONFLICT, Message: "Cannot delete product with active orders"}
)
```

**Key observations:**
1. **Types first, then interface, then parameters, then errors**: Logical reading order
2. **Comments on types, not every field**: Self-documenting field names
3. **Enums as string constants**: Simple, readable, database-friendly
4. **Aggregates separate from raw types**: `ProductDetail` vs. `Product`
5. **Sentinel errors as package variables**: Easy to check with `errors.Is(err, freyja.ErrProductNotFound)`

---

## Implementation File Template

Here's what a PostgreSQL implementation should look like:

```go
// freyja/postgres/product.go
package postgres

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "github.com/dukerupert/hiri"
    "github.com/dukerupert/hiri/internal/repository"
    "github.com/jackc/pgx/v5/pgtype"
)

// ProductService is the PostgreSQL implementation of freyja.ProductService.
type ProductService struct {
    repo     repository.Querier
    tenantID pgtype.UUID
}

// NewProductService creates a tenant-scoped product service.
func NewProductService(repo repository.Querier, tenantID string) (*ProductService, error) {
    var tenantUUID pgtype.UUID
    if err := tenantUUID.Scan(tenantID); err != nil {
        return nil, fmt.Errorf("invalid tenant ID: %w", err)
    }

    return &ProductService{
        repo:     repo,
        tenantID: tenantUUID,
    }, nil
}

// ListProducts returns products matching the filter, with pricing resolved.
func (s *ProductService) ListProducts(ctx context.Context, filter freyja.ProductFilter) ([]freyja.ProductWithPrice, error) {
    // For MVP: just list active public products
    // Later: implement full filtering
    rows, err := s.repo.ListActiveProducts(ctx, s.tenantID)
    if err != nil {
        return nil, freyja.Internal(err, "product.list", "failed to query products")
    }

    products := make([]freyja.ProductWithPrice, len(rows))
    for i, row := range rows {
        products[i] = freyja.ProductWithPrice{
            Product: freyja.Product{
                ID:               row.ID,
                TenantID:         row.TenantID,
                Name:             row.Name,
                Slug:             row.Slug,
                ShortDescription: row.ShortDescription,
                Origin:           row.Origin,
                RoastLevel:       row.RoastLevel,
                TastingNotes:     row.TastingNotes,
                Status:           freyja.ProductStatus(row.Status),
                Visibility:       freyja.ProductVisibility(row.Visibility),
                SortOrder:        row.SortOrder,
                CreatedAt:        row.CreatedAt,
                UpdatedAt:        row.UpdatedAt,
            },
            BaseSKU: freyja.ProductSKU{
                // Map from row if available
            },
            BasePrice: row.BasePrice,
            ImageURL:  row.ImageURL,
        }
    }

    return products, nil
}

// GetProductBySlug retrieves a product with full details.
func (s *ProductService) GetProductBySlug(ctx context.Context, slug string) (*freyja.ProductDetail, error) {
    // Get product
    product, err := s.repo.GetProductBySlug(ctx, repository.GetProductBySlugParams{
        TenantID: s.tenantID,
        Slug:     slug,
    })
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, freyja.ErrProductNotFound
        }
        return nil, freyja.Internal(err, "product.get", "failed to query product")
    }

    // Get SKUs
    skuRows, err := s.repo.ListProductSKUs(ctx, product.ID)
    if err != nil {
        return nil, freyja.Internal(err, "product.get", "failed to query SKUs")
    }

    skus := make([]freyja.ProductSKU, len(skuRows))
    for i, row := range skuRows {
        skus[i] = freyja.ProductSKU{
            ID:               row.ID,
            ProductID:        row.ProductID,
            SKU:              row.Sku,
            WeightValue:      row.WeightValue,
            WeightUnit:       row.WeightUnit,
            Grind:            row.Grind,
            StockQuantity:    row.StockQuantity,
            StockStatus:      freyja.StockStatus(row.StockStatus),
            LowStockWarning:  row.LowStockWarning,
            BackorderEnabled: row.BackorderEnabled,
            SortOrder:        row.SortOrder,
            CreatedAt:        row.CreatedAt,
            UpdatedAt:        row.UpdatedAt,
        }
    }

    // Get images
    imageRows, err := s.repo.ListProductImages(ctx, product.ID)
    if err != nil {
        return nil, freyja.Internal(err, "product.get", "failed to query images")
    }

    images := make([]freyja.ProductImage, len(imageRows))
    for i, row := range imageRows {
        images[i] = freyja.ProductImage{
            ID:        row.ID,
            ProductID: row.ProductID,
            URL:       row.URL,
            AltText:   row.AltText,
            SortOrder: row.SortOrder,
            CreatedAt: row.CreatedAt,
        }
    }

    return &freyja.ProductDetail{
        Product: freyja.Product{
            ID:               product.ID,
            TenantID:         product.TenantID,
            Name:             product.Name,
            Slug:             product.Slug,
            Description:      product.Description,
            ShortDescription: product.ShortDescription,
            Origin:           product.Origin,
            Region:           product.Region,
            Producer:         product.Producer,
            Process:          product.Process,
            RoastLevel:       product.RoastLevel,
            TastingNotes:     product.TastingNotes,
            ElevationMin:     product.ElevationMin,
            ElevationMax:     product.ElevationMax,
            Status:           freyja.ProductStatus(product.Status),
            Visibility:       freyja.ProductVisibility(product.Visibility),
            IsWhiteLabel:     product.IsWhiteLabel,
            BaseProductID:    product.BaseProductID,
            WhiteLabelCustomerID: product.WhiteLabelCustomerID,
            SortOrder:        product.SortOrder,
            CreatedAt:        product.CreatedAt,
            UpdatedAt:        product.UpdatedAt,
        },
        SKUs:   skus,
        Images: images,
    }, nil
}

// CreateProduct creates a new product.
func (s *ProductService) CreateProduct(ctx context.Context, params freyja.CreateProductParams) (*freyja.Product, error) {
    // Validate slug uniqueness
    _, err := s.repo.GetProductBySlug(ctx, repository.GetProductBySlugParams{
        TenantID: s.tenantID,
        Slug:     params.Slug,
    })
    if err == nil {
        return nil, freyja.ErrProductSlugExists
    }
    if !errors.Is(err, sql.ErrNoRows) {
        return nil, freyja.Internal(err, "product.create", "failed to check slug")
    }

    // Create product
    product, err := s.repo.CreateProduct(ctx, repository.CreateProductParams{
        TenantID:         s.tenantID,
        Name:             params.Name,
        Slug:             params.Slug,
        ShortDescription: pgtype.Text{String: params.ShortDescription, Valid: params.ShortDescription != ""},
        Description:      pgtype.Text{String: params.Description, Valid: params.Description != ""},
        Status:           string(params.Status),
        Visibility:       string(params.Visibility),
        Origin:           pgtype.Text{String: params.Origin, Valid: params.Origin != ""},
        Region:           pgtype.Text{String: params.Region, Valid: params.Region != ""},
        Producer:         pgtype.Text{String: params.Producer, Valid: params.Producer != ""},
        Process:          pgtype.Text{String: params.Process, Valid: params.Process != ""},
        RoastLevel:       pgtype.Text{String: params.RoastLevel, Valid: params.RoastLevel != ""},
        TastingNotes:     params.TastingNotes,
        ElevationMin:     pgtype.Int4{Int32: params.ElevationMin, Valid: params.ElevationMin > 0},
        ElevationMax:     pgtype.Int4{Int32: params.ElevationMax, Valid: params.ElevationMax > 0},
        IsWhiteLabel:     params.IsWhiteLabel,
        BaseProductID:    valueOrNull(params.BaseProductID),
        WhiteLabelCustomerID: valueOrNull(params.WhiteLabelCustomerID),
        SortOrder:        0,
    })

    if err != nil {
        return nil, freyja.Internal(err, "product.create", "failed to create product")
    }

    return &freyja.Product{
        ID:               product.ID,
        TenantID:         product.TenantID,
        Name:             product.Name,
        Slug:             product.Slug,
        Description:      product.Description,
        ShortDescription: product.ShortDescription,
        Origin:           product.Origin,
        Region:           product.Region,
        Producer:         product.Producer,
        Process:          product.Process,
        RoastLevel:       product.RoastLevel,
        TastingNotes:     product.TastingNotes,
        ElevationMin:     product.ElevationMin,
        ElevationMax:     product.ElevationMax,
        Status:           freyja.ProductStatus(product.Status),
        Visibility:       freyja.ProductVisibility(product.Visibility),
        IsWhiteLabel:     product.IsWhiteLabel,
        BaseProductID:    product.BaseProductID,
        WhiteLabelCustomerID: product.WhiteLabelCustomerID,
        SortOrder:        product.SortOrder,
        CreatedAt:        product.CreatedAt,
        UpdatedAt:        product.UpdatedAt,
    }, nil
}

// Helper function to convert pointer to pgtype.UUID
func valueOrNull(v *pgtype.UUID) pgtype.UUID {
    if v == nil {
        return pgtype.UUID{Valid: false}
    }
    return *v
}

// ... implement remaining interface methods ...
```

**Key observations:**
1. **Package name is `postgres`, not `freyja/postgres`**: Subpackage imports parent
2. **Types imported with package prefix**: `freyja.Product`, `freyja.ProductService`
3. **Mapping from repository types to domain types**: Explicit field-by-field conversion
4. **Domain errors returned**: `freyja.ErrProductNotFound`, not generic errors
5. **Tenant scoping enforced**: Every query includes `s.tenantID`

---

## HTTP Handler File Template

Here's what HTTP handlers should look like:

```go
// freyja/http/product.go
package http

import (
    "net/http"

    "github.com/dukerupert/hiri"
    "github.com/dukerupert/hiri/internal/handler"
    "github.com/jackc/pgx/v5/pgtype"
)

// ProductHandler handles HTTP routes for products (admin + storefront).
type ProductHandler struct {
    productService freyja.ProductService
    renderer       *handler.Renderer
}

// NewProductHandler creates a new product handler.
func NewProductHandler(productService freyja.ProductService, renderer *handler.Renderer) *ProductHandler {
    return &ProductHandler{
        productService: productService,
        renderer:       renderer,
    }
}

// =============================================================================
// Storefront Routes
// =============================================================================

// ListProducts handles GET /products (storefront product listing).
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
    products, err := h.productService.ListProducts(r.Context(), freyja.ProductFilter{
        Status:     &freyja.ProductStatusActive,
        Visibility: &freyja.ProductVisibilityPublic,
    })

    if err != nil {
        handler.ErrorResponse(w, r, err)
        return
    }

    data := map[string]interface{}{
        "Products": products,
    }

    h.renderer.RenderHTTP(w, "storefront/products", data)
}

// ProductDetail handles GET /products/{slug} (storefront product detail).
func (h *ProductHandler) ProductDetail(w http.ResponseWriter, r *http.Request) {
    slug := r.PathValue("slug")
    if slug == "" {
        handler.ErrorResponse(w, r, freyja.Invalid("http.product_detail", "slug required"))
        return
    }

    detail, err := h.productService.GetProductBySlug(r.Context(), slug)
    if err != nil {
        handler.ErrorResponse(w, r, err)
        return
    }

    data := map[string]interface{}{
        "Product": detail.Product,
        "SKUs":    detail.SKUs,
        "Images":  detail.Images,
    }

    h.renderer.RenderHTTP(w, "storefront/product-detail", data)
}

// =============================================================================
// Admin Routes
// =============================================================================

// AdminListProducts handles GET /admin/products (admin product list).
func (h *ProductHandler) AdminListProducts(w http.ResponseWriter, r *http.Request) {
    products, err := h.productService.ListProducts(r.Context(), freyja.ProductFilter{})
    if err != nil {
        handler.ErrorResponse(w, r, err)
        return
    }

    data := map[string]interface{}{
        "CurrentPath": r.URL.Path,
        "Products":    products,
    }

    h.renderer.RenderHTTP(w, "admin/products", data)
}

// AdminCreateProduct handles POST /admin/products (create new product).
func (h *ProductHandler) AdminCreateProduct(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        handler.ErrorResponse(w, r, freyja.Invalid("http.create_product", "invalid form data"))
        return
    }

    params := freyja.CreateProductParams{
        Name:             r.FormValue("name"),
        Slug:             r.FormValue("slug"),
        ShortDescription: r.FormValue("short_description"),
        Description:      r.FormValue("description"),
        Status:           freyja.ProductStatus(r.FormValue("status")),
        Visibility:       freyja.ProductVisibility(r.FormValue("visibility")),
        Origin:           r.FormValue("origin"),
        Region:           r.FormValue("region"),
        RoastLevel:       r.FormValue("roast_level"),
        // ... map remaining fields
    }

    product, err := h.productService.CreateProduct(r.Context(), params)
    if err != nil {
        handler.ErrorResponse(w, r, err)
        return
    }

    // Redirect to product detail
    http.Redirect(w, r, "/admin/products/"+product.ID.String(), http.StatusSeeOther)
}
```

**Key observations:**
1. **Single handler for both admin and storefront**: Routes differentiated by method name
2. **Service interface, not concrete implementation**: Handler depends on `freyja.ProductService`, not `*postgres.ProductService`
3. **Domain types throughout**: `freyja.Product`, `freyja.CreateProductParams`
4. **Error handling delegates to domain errors**: Handler doesn't construct HTTP status codes

---

## Migration Checklist

Use this checklist when migrating each domain:

### Phase 1: Create Root Domain File

- [ ] Create `freyja/{domain}.go`
- [ ] Define core domain types (`Product`, `Customer`, etc.)
- [ ] Define enums and constants (`ProductStatus`, visibility options, etc.)
- [ ] Define aggregate types (`ProductDetail`, `CustomerWithOrders`, etc.)
- [ ] Define service interface (`ProductService`)
- [ ] Define parameter types (`CreateProductParams`, `UpdateProductParams`, `ProductFilter`)
- [ ] Define domain errors (`ErrProductNotFound`, etc.)
- [ ] Add godoc comments to types and interface
- [ ] Run `go build ./...` to check for syntax errors

### Phase 2: Create PostgreSQL Implementation

- [ ] Create `freyja/postgres/{domain}.go`
- [ ] Define service struct with `repo` and `tenantID` fields
- [ ] Create constructor: `NewProductService(repo, tenantID)`
- [ ] Implement all interface methods
- [ ] Map `repository.*` types to `freyja.*` types
- [ ] Add tenant scoping to all queries
- [ ] Return domain errors (`freyja.ErrProductNotFound`), not sql.ErrNoRows
- [ ] Wrap internal errors with `freyja.Internal(err, op, message)`
- [ ] Run `go build ./...` to check compilation

### Phase 3: Update HTTP Handlers

- [ ] Create `freyja/http/{domain}.go` (or update existing)
- [ ] Update handler struct to use `freyja.{Domain}Service` interface
- [ ] Update constructor to accept interface, not concrete type
- [ ] Update route handlers to use `freyja.*` types
- [ ] Update error handling to use `handler.ErrorResponse(w, r, err)`
- [ ] Remove direct repository access (handlers call service only)
- [ ] Run `go build ./...` to check compilation

### Phase 4: Update Main Wiring

- [ ] Update `cmd/server/main.go` (or current main.go):
  - Import `freyja/postgres` (concrete implementation)
  - Import `freyja/http` (handlers)
  - Wire: `productService := postgres.NewProductService(repo, tenantID)`
  - Wire: `productHandler := http.NewProductHandler(productService, renderer)`
- [ ] Run application and test manually
- [ ] Check logs for errors

### Phase 5: Update Tests

- [ ] Create `freyja/{domain}_test.go` for domain type tests (if needed)
- [ ] Create `freyja/postgres/{domain}_test.go` for integration tests
- [ ] Update existing tests to use new types
- [ ] Add tests for new domain errors
- [ ] Run `go test ./...` and verify all pass

### Phase 6: Cleanup

- [ ] Delete `internal/domain/{domain}.go` (if fully migrated)
- [ ] Delete `internal/service/{domain}.go` (if fully migrated)
- [ ] Remove unused imports from old files
- [ ] Run `go mod tidy`
- [ ] Run `golangci-lint run` and fix issues
- [ ] Run `go test ./...` one final time
- [ ] Commit changes with clear commit message

---

## Domain Migration Order

Migrate domains in this order to minimize disruption and dependencies:

### 1. Product (Pilot Domain)
**Why first:** Self-contained, well-understood, no dependencies on other domains.

**Files to create:**
- `freyja/product.go`
- `freyja/postgres/product.go`
- `freyja/http/product.go`

**Files to update:**
- `cmd/server/main.go` (wiring)
- Existing product templates (minimal changes)

**Files to delete:**
- `internal/domain/product.go`
- `internal/service/product.go`
- `internal/handler/admin/products.go`
- `internal/handler/storefront/products.go`

**Expected effort:** 4-6 hours (includes learning curve)

---

### 2. Customer
**Why second:** Depends only on tenant (context). No product dependency.

**Files to create:**
- `freyja/customer.go`
- `freyja/postgres/customer.go`
- `freyja/http/customer.go`

**Dependencies:** None (tenant is infrastructure, not a domain dependency)

**Expected effort:** 3-4 hours (pattern established from Product)

---

### 3. Cart
**Why third:** Depends on Product and Customer (both migrated). Small surface area.

**Files to create:**
- `freyja/cart.go`
- `freyja/postgres/cart.go`
- `freyja/http/cart.go`

**Dependencies:** Product, Customer

**Expected effort:** 2-3 hours

---

### 4. Order
**Why fourth:** Depends on Product, Customer, Cart. Core transactional domain.

**Files to create:**
- `freyja/order.go`
- `freyja/postgres/order.go`
- `freyja/http/order.go`

**Dependencies:** Product, Customer, Cart

**Expected effort:** 5-6 hours (complex domain)

---

### 5. Subscription
**Why fifth:** Depends on Product, Customer, Order. Integrates with billing.

**Files to create:**
- `freyja/subscription.go`
- `freyja/postgres/subscription.go`
- `freyja/http/subscription.go`

**Dependencies:** Product, Customer, billing (external adapter, unchanged)

**Expected effort:** 4-5 hours

---

### 6. Invoice
**Why sixth:** Depends on Order, Customer. Wholesale-specific domain.

**Files to create:**
- `freyja/invoice.go`
- `freyja/postgres/invoice.go`
- `freyja/http/invoice.go`

**Dependencies:** Order, Customer, billing (external adapter)

**Expected effort:** 4-5 hours

---

### 7. PriceList
**Why seventh:** Used by Product queries but can be migrated independently.

**Files to create:**
- `freyja/pricelist.go`
- `freyja/postgres/pricelist.go`
- `freyja/http/pricelist.go`

**Dependencies:** Product (will need to update Product service to use PriceList)

**Expected effort:** 3-4 hours

---

### 8. Tenant / Operator (If Needed)
**Why last:** Infrastructure concern, not core business domain. May not need migration.

**Decision:** Only migrate if treating tenant operations as a domain (multi-tenant SaaS admin). Otherwise, keep in `internal/`.

**Expected effort:** 2-3 hours (if migrating)

---

## Gotchas and Considerations

### 1. Import Cycles

**Problem:** `freyja/postgres` imports `freyja`. If `freyja` imports `freyja/postgres`, you get a cycle.

**Solution:** The root package (`freyja`) should NEVER import implementation packages. Only `main.go` wires them together.

**Example (WRONG):**
```go
// freyja/product.go
package freyja

import "github.com/dukerupert/hiri/postgres" // CYCLE!

func DefaultProductService() ProductService {
    return postgres.NewProductService(...)
}
```

**Example (CORRECT):**
```go
// cmd/server/main.go
package main

import (
    "github.com/dukerupert/hiri"
    "github.com/dukerupert/hiri/postgres"
)

func main() {
    productService := postgres.NewProductService(repo, tenantID)
    // ...
}
```

---

### 2. sqlc Type Mapping

**Problem:** sqlc generates `repository.Product`, but we want `freyja.Product` at the root.

**Solution:** Keep sqlc generating into `internal/repository/`. Map types in `freyja/postgres/` implementation.

**Don't:** Try to make sqlc generate into `freyja/`. sqlc types are tied to database schema; domain types are business concepts. They should differ.

**Pattern:**
```go
// freyja/postgres/product.go
func (s *ProductService) GetProductBySlug(ctx context.Context, slug string) (*freyja.ProductDetail, error) {
    // Query returns repository.Product
    product, err := s.repo.GetProductBySlug(ctx, repository.GetProductBySlugParams{...})
    if err != nil {
        return nil, err
    }

    // Map to freyja.Product
    return &freyja.ProductDetail{
        Product: freyja.Product{
            ID:       product.ID,
            TenantID: product.TenantID,
            Name:     product.Name,
            // ... map all fields
        },
    }, nil
}
```

**Why the mapping?** Domain types may:
- Combine multiple database rows
- Add computed fields
- Use different field names (e.g., `PriceCents` instead of `price`)
- Exclude internal fields (e.g., `tenant_id` not exposed to handlers)

---

### 3. Tenant Context

**Problem:** Every service needs `tenantID`, but passing it to every method is tedious.

**Solution:** Tenant ID is a constructor parameter, not a method parameter. The service is tenant-scoped.

**Pattern:**
```go
// freyja/postgres/product.go
type ProductService struct {
    repo     repository.Querier
    tenantID pgtype.UUID  // Scoped at construction
}

func NewProductService(repo repository.Querier, tenantID string) (*ProductService, error) {
    var tenantUUID pgtype.UUID
    if err := tenantUUID.Scan(tenantID); err != nil {
        return nil, fmt.Errorf("invalid tenant ID: %w", err)
    }

    return &ProductService{
        repo:     repo,
        tenantID: tenantUUID,
    }, nil
}

// All methods use s.tenantID internally
func (s *ProductService) ListProducts(ctx context.Context, filter freyja.ProductFilter) ([]freyja.Product, error) {
    return s.repo.ListActiveProducts(ctx, s.tenantID)  // Scoped!
}
```

**Why?** This prevents accidental cross-tenant queries. The service is inherently scoped. In `main.go`, create one service instance per tenant context (middleware extracts tenant from subdomain/custom domain).

---

### 4. Multiple Query Patterns (N+1 Problem)

**Problem:** Loading `ProductDetail` requires 3 queries (product, SKUs, images). This is the N+1 problem if loading many products.

**Solution (now):** Accept the 3 queries for single-product detail pages. It's fast enough for <100ms response time.

**Solution (later, if needed):** Add a specialized sqlc query that joins all data:
```sql
-- name: GetProductDetailBySlug :many
SELECT
    p.*,
    s.id as sku_id, s.sku, s.weight_value, ...,
    i.id as image_id, i.url, i.alt_text, ...
FROM products p
LEFT JOIN product_skus s ON p.id = s.product_id
LEFT JOIN product_images i ON p.id = i.product_id
WHERE p.tenant_id = $1 AND p.slug = $2
ORDER BY s.sort_order, i.sort_order;
```

Then in `postgres/product.go`, map rows into a single `ProductDetail`.

**Rule:** Start simple (multiple queries). Optimize when profiling shows it's necessary.

---

### 5. Error Wrapping and Context

**Problem:** When multiple layers wrap errors, the operation context is lost.

**Solution:** Use the `Op` field consistently. Prefix with domain: `"product.create"`, `"customer.delete"`.

**Pattern:**
```go
// freyja/postgres/product.go
func (s *ProductService) CreateProduct(ctx context.Context, params freyja.CreateProductParams) (*freyja.Product, error) {
    product, err := s.repo.CreateProduct(ctx, repoParams)
    if err != nil {
        // Wrap with operation context
        return nil, freyja.Internal(err, "product.create", "failed to insert product")
    }
    return mapProduct(product), nil
}
```

**Why?** Logs will show: `ERROR: product.create: failed to insert product: pq: duplicate key violates unique constraint`. The operation context helps debugging.

---

### 6. Testing Strategy

**Problem:** Do we test the interface or the implementation?

**Solution:** Test the implementation. The interface is a contract; tests verify the contract is fulfilled.

**Pattern:**
```go
// freyja/postgres/product_test.go
package postgres_test

import (
    "testing"
    "github.com/dukerupert/hiri"
    "github.com/dukerupert/hiri/postgres"
)

func TestProductService_CreateProduct(t *testing.T) {
    // Setup: Create test database, run migrations
    repo := setupTestDB(t)
    defer cleanupTestDB(t, repo)

    // Create service
    svc, err := postgres.NewProductService(repo, testTenantID)
    if err != nil {
        t.Fatalf("failed to create service: %v", err)
    }

    // Test
    product, err := svc.CreateProduct(ctx, freyja.CreateProductParams{
        Name: "Test Coffee",
        Slug: "test-coffee",
        Status: freyja.ProductStatusActive,
    })

    if err != nil {
        t.Fatalf("CreateProduct() error = %v", err)
    }

    if product.Name != "Test Coffee" {
        t.Errorf("product.Name = %q; want %q", product.Name, "Test Coffee")
    }
}
```

**Why?** Integration tests verify the actual database interaction. Unit tests for business logic can use mocks if needed, but start with integration tests for data access.

---

### 7. Backwards Compatibility During Migration

**Problem:** How do we avoid breaking the application mid-migration?

**Solution:** Migrate one domain at a time, updating `main.go` wiring as you go.

**Pattern:**
1. Create `freyja/product.go` and `freyja/postgres/product.go`
2. Update `main.go` to use new types
3. Old handlers still work (they use the same `repository.Querier`)
4. Update handlers incrementally
5. Delete old code only when fully migrated

**Tip:** Use feature flags or separate branches if needed, but prefer incremental commits to `main` if each step is complete.

---

## Keeping External Adapters Separate

**Billing, Email, Shipping, and Storage are NOT being refactored.** Here's why:

### 1. They're Already Well-Structured

These packages follow interface/implementation pattern:
- `billing/billing.go` - Interface
- `billing/stripe.go` - Implementation
- `billing/mock.go` - Test mock

No improvement from moving to `freyja/billing/`.

### 2. They're Horizontal Concerns

These are cross-cutting adapters used by multiple domains:
- Billing: Used by Order, Subscription, Invoice
- Email: Used by User, Order, Subscription
- Shipping: Used by Order
- Storage: Used by Product (images), User (avatars)

They don't belong to one domain—they're infrastructure.

### 3. They Have External Dependencies

- Billing depends on `github.com/stripe/stripe-go`
- Email will depend on Postmark/Resend/SES SDKs
- Shipping will depend on ShipStation/EasyPost APIs

Keeping them separate makes dependency management clearer.

### 4. Domain Services Use Them via Interfaces

Domain services depend on interfaces, not implementations:

```go
// freyja/order.go
type OrderService interface {
    CreateOrder(ctx context.Context, params CreateOrderParams) (*Order, error)
}

// freyja/postgres/order.go
type OrderService struct {
    repo           repository.Querier
    tenantID       pgtype.UUID
    billingService billing.Provider  // Interface from billing package
}

func NewOrderService(repo repository.Querier, tenantID string, billing billing.Provider) (*OrderService, error) {
    return &OrderService{
        repo:           repo,
        tenantID:       tenantID,
        billingService: billing,
    }, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, params freyja.CreateOrderParams) (*freyja.Order, error) {
    // Create payment intent via billing interface
    intent, err := s.billingService.CreatePaymentIntent(ctx, billing.CreatePaymentIntentParams{...})
    if err != nil {
        return nil, freyja.Internal(err, "order.create", "payment failed")
    }

    // Continue creating order...
}
```

**Key point:** Domain services import `billing.Provider` (interface). Main wiring passes `*stripe.BillingService` (implementation).

---

## Summary

The WTF Dial refactor organizes Freyja by **domain, not layer**:

- **Before:** `internal/domain/`, `internal/service/`, `internal/handler/`, `internal/repository/`
- **After:** `freyja/product.go`, `freyja/postgres/product.go`, `freyja/http/product.go`

**Benefits:**
- Co-location by domain (all product code together)
- Clear contract/implementation split (root vs. subpackages)
- Easier navigation (grep for "Product" finds types at root)
- Simpler mental model (fewer layers)

**Process:**
1. Start with Product domain (pilot)
2. Create root types and interface
3. Create PostgreSQL implementation
4. Update HTTP handlers
5. Update wiring in main.go
6. Test and cleanup
7. Repeat for next domain

**Remember:** This refactor is incremental. Each domain migration is a complete, working change. Don't try to do everything at once.

Now get started with Product!
