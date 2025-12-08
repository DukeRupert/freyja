package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Page-related domain errors.
var (
	ErrPageNotFound = &Error{Code: ENOTFOUND, Message: "Page not found"}
	ErrInvalidSlug  = &Error{Code: EINVALID, Message: "Invalid page slug"}
)

// PageService provides business logic for tenant page operations.
type PageService interface {
	// GetPage retrieves a page by tenant and slug.
	// Returns ErrPageNotFound if page doesn't exist.
	GetPage(ctx context.Context, params GetPageParams) (*Page, error)

	// GetPublishedPage retrieves only published pages (for storefront).
	// Returns ErrPageNotFound if page doesn't exist or isn't published.
	GetPublishedPage(ctx context.Context, params GetPageParams) (*Page, error)

	// ListPages retrieves all pages for a tenant (for admin).
	ListPages(ctx context.Context, tenantID pgtype.UUID) ([]Page, error)

	// UpdatePage updates an existing page's content.
	UpdatePage(ctx context.Context, params UpdatePageParams) (*Page, error)

	// EnsureDefaultPages creates default legal pages if they don't exist.
	// Called when a new tenant is created or pages are missing.
	EnsureDefaultPages(ctx context.Context, tenantID pgtype.UUID, storeName, contactEmail string) error
}

// GetPageParams contains parameters for retrieving a page.
type GetPageParams struct {
	TenantID pgtype.UUID
	Slug     string
}

// UpdatePageParams contains parameters for updating a page.
type UpdatePageParams struct {
	TenantID         pgtype.UUID
	Slug             string
	Title            string
	Content          string
	MetaDescription  string
	LastUpdatedLabel string
	IsPublished      bool
}

// Page represents an editable content page.
type Page struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	Slug             string
	Title            string
	Content          string // HTML content from Tiptap
	MetaDescription  string
	LastUpdatedLabel string // e.g., "December 2024"
	IsPublished      bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PageSlug constants for known page types.
const (
	PageSlugPrivacy  = "privacy"
	PageSlugTerms    = "terms"
	PageSlugShipping = "shipping"
	PageSlugAbout    = "about"
	PageSlugContact  = "contact"
)

// ValidPageSlugs returns the list of valid page slugs.
func ValidPageSlugs() []string {
	return []string{
		PageSlugPrivacy,
		PageSlugTerms,
		PageSlugShipping,
		PageSlugAbout,
		PageSlugContact,
	}
}

// IsValidPageSlug checks if a slug is a valid page type.
func IsValidPageSlug(slug string) bool {
	for _, valid := range ValidPageSlugs() {
		if slug == valid {
			return true
		}
	}
	return false
}

// PageMetadata contains display metadata for known page types.
type PageMetadata struct {
	Slug        string
	Title       string
	Description string
}

// GetPageMetadata returns metadata for known page types.
func GetPageMetadata() []PageMetadata {
	return []PageMetadata{
		{Slug: PageSlugPrivacy, Title: "Privacy Policy", Description: "How we collect and use customer data"},
		{Slug: PageSlugTerms, Title: "Terms of Service", Description: "Terms and conditions for using the store"},
		{Slug: PageSlugShipping, Title: "Shipping & Returns", Description: "Shipping methods, rates, and return policy"},
		{Slug: PageSlugAbout, Title: "About Us", Description: "Your company story and mission"},
		{Slug: PageSlugContact, Title: "Contact", Description: "How customers can reach you"},
	}
}
