package domain

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// =============================================================================
// PRODUCT DOMAIN TYPES
// =============================================================================

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
	ProductVisibilityPublic        ProductVisibility = "public"
	ProductVisibilityWholesaleOnly ProductVisibility = "wholesale_only"
	ProductVisibilityHidden        ProductVisibility = "hidden"
)

// GrindOption represents available grind types.
type GrindOption string

const (
	GrindWholeBeans   GrindOption = "whole_beans"
	GrindCoarse       GrindOption = "coarse"
	GrindMediumCoarse GrindOption = "medium_coarse"
	GrindMedium       GrindOption = "medium"
	GrindMediumFine   GrindOption = "medium_fine"
	GrindFine         GrindOption = "fine"
	GrindExtraFine    GrindOption = "extra_fine"
)

// InventoryPolicy controls behavior when inventory reaches zero.
type InventoryPolicy string

const (
	InventoryPolicyDeny  InventoryPolicy = "deny"  // Prevent orders when out of stock
	InventoryPolicyAllow InventoryPolicy = "allow" // Allow backorders
)

// Product represents a coffee product offering.
// This is the domain type - implementations map from repository types.
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
	ElevationMin pgtype.Int4
	ElevationMax pgtype.Int4
	Variety      pgtype.Text
	HarvestYear  pgtype.Int4
	TastingNotes []string

	// Catalog attributes
	Status     ProductStatus
	Visibility ProductVisibility
	SortOrder  int32

	// SEO
	MetaTitle       pgtype.Text
	MetaDescription pgtype.Text

	// White-label support
	IsWhiteLabel         bool
	BaseProductID        pgtype.UUID
	WhiteLabelCustomerID pgtype.UUID

	// Timestamps
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

// ProductSKU represents a purchasable variant of a product (weight + grind combination).
type ProductSKU struct {
	ID        pgtype.UUID
	TenantID  pgtype.UUID
	ProductID pgtype.UUID
	SKU       string

	// Variant attributes
	WeightValue pgtype.Numeric
	WeightUnit  string
	Grind       string

	// Pricing and inventory
	BasePriceCents    int32
	InventoryQuantity int32
	InventoryPolicy   InventoryPolicy
	LowStockThreshold pgtype.Int4

	// Status and shipping
	IsActive         bool
	WeightGrams      pgtype.Int4
	RequiresShipping bool

	// Timestamps
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

// ProductImage represents an image associated with a product.
type ProductImage struct {
	ID        pgtype.UUID
	TenantID  pgtype.UUID
	ProductID pgtype.UUID
	URL       string
	AltText   pgtype.Text
	Width     pgtype.Int4
	Height    pgtype.Int4
	FileSize  pgtype.Int4
	SortOrder int32
	IsPrimary bool
	CreatedAt pgtype.Timestamptz
}

// =============================================================================
// AGGREGATE TYPES (composed domain types)
// =============================================================================

// ProductDetail aggregates product information with SKUs, pricing, and images.
// Used for product detail pages where full information is needed.
type ProductDetail struct {
	Product Product
	SKUs    []ProductSKUWithPrice
	Images  []ProductImage
}

// ProductSKUWithPrice combines SKU information with resolved pricing.
type ProductSKUWithPrice struct {
	SKU              ProductSKU
	PriceCents       int32
	CompareAtCents   pgtype.Int4
	InventoryMessage string
}

// ProductPrice contains pricing information for a specific SKU.
type ProductPrice struct {
	SKUID       pgtype.UUID
	PriceCents  int32
	PriceListID pgtype.UUID
}

// SKUCheckoutDetail contains SKU and product info for checkout display.
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

// ProductListItem represents a product in a listing with minimal info.
type ProductListItem struct {
	ID               pgtype.UUID
	TenantID         pgtype.UUID
	Name             string
	Slug             string
	ShortDescription pgtype.Text
	Origin           pgtype.Text
	RoastLevel       pgtype.Text
	TastingNotes     []string
	Status           ProductStatus
	Visibility       ProductVisibility
	SortOrder        int32
	PrimaryImageURL  pgtype.Text
	PrimaryImageAlt  pgtype.Text
	MinPriceCents    pgtype.Int4 // Lowest SKU price for "from $X" display
}

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// ProductService provides business logic for product catalog operations.
// Implementations should be tenant-scoped at construction time.
type ProductService interface {
	// -------------------------------------------------------------------------
	// Storefront Operations (read-only, public-facing)
	// -------------------------------------------------------------------------

	// ListProducts returns all active public products for the tenant.
	ListProducts(ctx context.Context) ([]ProductListItem, error)

	// ListProductsFiltered returns products matching the given filters.
	ListProductsFiltered(ctx context.Context, filter ProductFilter) ([]ProductListItem, error)

	// GetProductDetail retrieves a product with SKUs, pricing, and images.
	GetProductDetail(ctx context.Context, slug string) (*ProductDetail, error)

	// GetProductPrice retrieves pricing for a specific SKU.
	GetProductPrice(ctx context.Context, skuID string) (*ProductPrice, error)

	// GetSKUForCheckout retrieves SKU details with product info for checkout display.
	GetSKUForCheckout(ctx context.Context, skuID string) (*SKUCheckoutDetail, error)

	// GetFilterOptions returns available filter values (roast levels, origins, etc).
	GetFilterOptions(ctx context.Context) (*ProductFilterOptions, error)

	// -------------------------------------------------------------------------
	// Admin Operations (CRUD)
	// -------------------------------------------------------------------------

	// GetProductByID retrieves a product by ID (includes inactive).
	GetProductByID(ctx context.Context, id pgtype.UUID) (*Product, error)

	// CreateProduct creates a new product.
	CreateProduct(ctx context.Context, params CreateProductParams) (*Product, error)

	// UpdateProduct updates an existing product.
	UpdateProduct(ctx context.Context, id pgtype.UUID, params UpdateProductParams) error

	// DeleteProduct soft-deletes a product (sets status to archived).
	DeleteProduct(ctx context.Context, id pgtype.UUID) error

	// -------------------------------------------------------------------------
	// SKU Operations
	// -------------------------------------------------------------------------

	// ListSKUs returns all SKUs for a product.
	ListSKUs(ctx context.Context, productID pgtype.UUID) ([]ProductSKU, error)

	// GetSKUByID retrieves a SKU by ID.
	GetSKUByID(ctx context.Context, id pgtype.UUID) (*ProductSKU, error)

	// CreateSKU creates a new SKU for a product.
	CreateSKU(ctx context.Context, params CreateSKUParams) (*ProductSKU, error)

	// UpdateSKU updates an existing SKU.
	UpdateSKU(ctx context.Context, id pgtype.UUID, params UpdateSKUParams) error

	// DeleteSKU soft-deletes a SKU (sets is_active to false).
	DeleteSKU(ctx context.Context, id pgtype.UUID) error

	// -------------------------------------------------------------------------
	// Image Operations
	// -------------------------------------------------------------------------

	// ListImages returns all images for a product.
	ListImages(ctx context.Context, productID pgtype.UUID) ([]ProductImage, error)

	// CreateImage adds an image to a product.
	CreateImage(ctx context.Context, params CreateImageParams) (*ProductImage, error)

	// UpdateImage updates image metadata.
	UpdateImage(ctx context.Context, id pgtype.UUID, params UpdateImageParams) error

	// DeleteImage removes an image from a product.
	DeleteImage(ctx context.Context, id pgtype.UUID) error

	// SetPrimaryImage sets an image as the primary image for its product.
	SetPrimaryImage(ctx context.Context, productID, imageID pgtype.UUID) error
}

// =============================================================================
// PARAMETER TYPES
// =============================================================================

// ProductFilter contains optional filters for product listing.
type ProductFilter struct {
	RoastLevel  *string
	Origin      *string
	TastingNote *string
	Status      *ProductStatus
	Visibility  *ProductVisibility
}

// ProductFilterOptions contains available values for each filter.
type ProductFilterOptions struct {
	RoastLevels  []string
	Origins      []string
	TastingNotes []string
}

// CreateProductParams contains parameters for creating a product.
type CreateProductParams struct {
	Name             string
	Slug             string
	Description      pgtype.Text
	ShortDescription pgtype.Text
	Origin           pgtype.Text
	Region           pgtype.Text
	Producer         pgtype.Text
	Process          pgtype.Text
	RoastLevel       pgtype.Text
	ElevationMin     pgtype.Int4
	ElevationMax     pgtype.Int4
	Variety          pgtype.Text
	HarvestYear      pgtype.Int4
	TastingNotes     []string
	Status           ProductStatus
	Visibility       ProductVisibility
	MetaTitle        pgtype.Text
	MetaDescription  pgtype.Text
}

// UpdateProductParams contains parameters for updating a product.
// Pointer fields indicate optional updates (nil = no change).
type UpdateProductParams struct {
	Name             *string
	Slug             *string
	Description      pgtype.Text
	ShortDescription pgtype.Text
	Origin           pgtype.Text
	Region           pgtype.Text
	Producer         pgtype.Text
	Process          pgtype.Text
	RoastLevel       pgtype.Text
	ElevationMin     pgtype.Int4
	ElevationMax     pgtype.Int4
	Variety          pgtype.Text
	HarvestYear      pgtype.Int4
	TastingNotes     []string
	Status           *ProductStatus
	Visibility       *ProductVisibility
	MetaTitle        pgtype.Text
	MetaDescription  pgtype.Text
}

// CreateSKUParams contains parameters for creating a SKU.
type CreateSKUParams struct {
	ProductID         pgtype.UUID
	SKU               string
	WeightValue       pgtype.Numeric
	WeightUnit        string
	Grind             string
	BasePriceCents    int32
	InventoryQuantity int32
	InventoryPolicy   InventoryPolicy
	LowStockThreshold pgtype.Int4
	WeightGrams       pgtype.Int4
	RequiresShipping  bool
}

// UpdateSKUParams contains parameters for updating a SKU.
type UpdateSKUParams struct {
	SKU               *string
	WeightValue       pgtype.Numeric
	WeightUnit        *string
	Grind             *string
	BasePriceCents    *int32
	InventoryQuantity *int32
	InventoryPolicy   *InventoryPolicy
	LowStockThreshold pgtype.Int4
	WeightGrams       pgtype.Int4
	RequiresShipping  *bool
	IsActive          *bool
}

// CreateImageParams contains parameters for creating an image.
type CreateImageParams struct {
	ProductID pgtype.UUID
	URL       string
	AltText   pgtype.Text
	Width     pgtype.Int4
	Height    pgtype.Int4
	FileSize  pgtype.Int4
	SortOrder int32
	IsPrimary bool
}

// UpdateImageParams contains parameters for updating an image.
type UpdateImageParams struct {
	AltText   pgtype.Text
	SortOrder *int32
}

// =============================================================================
// DOMAIN ERRORS
// =============================================================================

// Product-specific errors.
var (
	ErrProductNotFound = &Error{Code: ENOTFOUND, Message: "Product not found"}
	ErrSKUNotFound     = &Error{Code: ENOTFOUND, Message: "SKU not found"}
	ErrPriceNotFound   = &Error{Code: ENOTFOUND, Message: "Price not found for this product"}
	ErrImageNotFound   = &Error{Code: ENOTFOUND, Message: "Image not found"}

	ErrDuplicateSlug = &Error{Code: ECONFLICT, Message: "Product slug already exists"}
	ErrDuplicateSKU  = &Error{Code: ECONFLICT, Message: "SKU code already exists"}

	ErrInvalidProductStatus     = &Error{Code: EINVALID, Message: "Invalid product status"}
	ErrInvalidProductVisibility = &Error{Code: EINVALID, Message: "Invalid product visibility"}
	ErrInvalidInventoryPolicy   = &Error{Code: EINVALID, Message: "Invalid inventory policy"}
)
