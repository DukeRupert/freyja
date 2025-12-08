package page

import (
	"context"
	"time"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service implements domain.PageService
type Service struct {
	repo repository.Querier
}

// NewService creates a new page service
func NewService(repo repository.Querier) *Service {
	return &Service{repo: repo}
}

// GetPage retrieves a page by tenant and slug.
func (s *Service) GetPage(ctx context.Context, params domain.GetPageParams) (*domain.Page, error) {
	if !domain.IsValidPageSlug(params.Slug) {
		return nil, domain.ErrInvalidSlug
	}

	row, err := s.repo.GetTenantPage(ctx, repository.GetTenantPageParams{
		TenantID: params.TenantID,
		Slug:     params.Slug,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPageNotFound
		}
		return nil, err
	}

	return repoToPage(row), nil
}

// GetPublishedPage retrieves only published pages (for storefront).
func (s *Service) GetPublishedPage(ctx context.Context, params domain.GetPageParams) (*domain.Page, error) {
	if !domain.IsValidPageSlug(params.Slug) {
		return nil, domain.ErrInvalidSlug
	}

	row, err := s.repo.GetPublishedTenantPage(ctx, repository.GetPublishedTenantPageParams{
		TenantID: params.TenantID,
		Slug:     params.Slug,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPageNotFound
		}
		return nil, err
	}

	return repoToPage(row), nil
}

// ListPages retrieves all pages for a tenant (for admin).
func (s *Service) ListPages(ctx context.Context, tenantID pgtype.UUID) ([]domain.Page, error) {
	rows, err := s.repo.ListTenantPages(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	pages := make([]domain.Page, len(rows))
	for i, row := range rows {
		pages[i] = *repoToPage(row)
	}

	return pages, nil
}

// UpdatePage updates an existing page's content.
func (s *Service) UpdatePage(ctx context.Context, params domain.UpdatePageParams) (*domain.Page, error) {
	if !domain.IsValidPageSlug(params.Slug) {
		return nil, domain.ErrInvalidSlug
	}

	row, err := s.repo.UpdateTenantPage(ctx, repository.UpdateTenantPageParams{
		TenantID:         params.TenantID,
		Slug:             params.Slug,
		Title:            params.Title,
		Content:          params.Content,
		MetaDescription:  pgtext(params.MetaDescription),
		LastUpdatedLabel: pgtext(params.LastUpdatedLabel),
		IsPublished:      params.IsPublished,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPageNotFound
		}
		return nil, err
	}

	return repoToPage(row), nil
}

// EnsureDefaultPages creates default legal pages if they don't exist.
func (s *Service) EnsureDefaultPages(ctx context.Context, tenantID pgtype.UUID, storeName, contactEmail string) error {
	currentDate := time.Now().Format("January 2006")

	// Privacy Policy
	_, err := s.repo.UpsertTenantPage(ctx, repository.UpsertTenantPageParams{
		TenantID:         tenantID,
		Slug:             domain.PageSlugPrivacy,
		Title:            "Privacy Policy",
		Content:          defaultPrivacyContent(storeName, contactEmail),
		MetaDescription:  pgtext("Our privacy policy explains how we collect, use, and protect your personal information."),
		LastUpdatedLabel: pgtext(currentDate),
		IsPublished:      true,
	})
	if err != nil {
		return err
	}

	// Terms of Service
	_, err = s.repo.UpsertTenantPage(ctx, repository.UpsertTenantPageParams{
		TenantID:         tenantID,
		Slug:             domain.PageSlugTerms,
		Title:            "Terms of Service",
		Content:          defaultTermsContent(storeName, contactEmail),
		MetaDescription:  pgtext("Terms and conditions for using our website and services."),
		LastUpdatedLabel: pgtext(currentDate),
		IsPublished:      true,
	})
	if err != nil {
		return err
	}

	// Shipping & Returns
	_, err = s.repo.UpsertTenantPage(ctx, repository.UpsertTenantPageParams{
		TenantID:         tenantID,
		Slug:             domain.PageSlugShipping,
		Title:            "Shipping & Returns",
		Content:          defaultShippingContent(storeName, contactEmail),
		MetaDescription:  pgtext("Information about our shipping methods, delivery times, and return policy."),
		LastUpdatedLabel: pgtext(currentDate),
		IsPublished:      true,
	})
	if err != nil {
		return err
	}

	return nil
}

// Helper functions

func repoToPage(row repository.TenantPage) *domain.Page {
	return &domain.Page{
		ID:               uuidFromPgtype(row.ID),
		TenantID:         uuidFromPgtype(row.TenantID),
		Slug:             row.Slug,
		Title:            row.Title,
		Content:          row.Content,
		MetaDescription:  stringFromPgtext(row.MetaDescription),
		LastUpdatedLabel: stringFromPgtext(row.LastUpdatedLabel),
		IsPublished:      row.IsPublished,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}

func uuidFromPgtype(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

func pgtext(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func stringFromPgtext(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
