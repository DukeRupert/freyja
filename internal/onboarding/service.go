package onboarding

import (
	"context"
	"errors"
	"time"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Errors
var (
	ErrCannotSkipRequiredItem = errors.New("cannot skip required checklist item")
	ErrInvalidItemID          = errors.New("invalid checklist item ID")
)

// Service handles onboarding checklist logic
type Service struct {
	repo repository.Querier
}

// NewService creates a new onboarding service
func NewService(repo repository.Querier) *Service {
	return &Service{repo: repo}
}

// GetStatus returns the complete onboarding status for a tenant.
// This is the main entry point - it computes all validation checks
// and combines them with skip flags to build the full checklist.
func (s *Service) GetStatus(ctx context.Context, tenantID uuid.UUID) (*domain.OnboardingStatus, error) {
	pgTenantID := uuidToPgtype(tenantID)

	// Get all validation results in a single query
	validations, err := s.repo.GetAllOnboardingValidations(ctx, pgTenantID)
	if err != nil {
		return nil, err
	}

	// Get skipped items
	skips, err := s.repo.GetSkippedItems(ctx, pgTenantID)
	if err != nil {
		return nil, err
	}

	// Build skip map for fast lookup
	skipMap := make(map[string]bool)
	for _, skip := range skips {
		skipMap[skip.ItemID] = true
	}

	// Build the checklist structure
	status := s.buildChecklistStatus(tenantID, validations, skipMap)

	return status, nil
}

// SkipItem marks an optional item as skipped
func (s *Service) SkipItem(ctx context.Context, tenantID uuid.UUID, itemID string, operatorID *uuid.UUID) error {
	// Validate that item exists and is skippable
	if !isValidItemID(itemID) {
		return ErrInvalidItemID
	}
	if !domain.IsSkippable(itemID) {
		return ErrCannotSkipRequiredItem
	}

	pgTenantID := uuidToPgtype(tenantID)
	var pgOperatorID pgtype.UUID
	if operatorID != nil {
		pgOperatorID = uuidToPgtype(*operatorID)
	}

	_, err := s.repo.SkipItem(ctx, repository.SkipItemParams{
		TenantID:  pgTenantID,
		ItemID:    itemID,
		SkippedBy: pgOperatorID,
	})
	return err
}

// UnskipItem removes skip flag from an item
func (s *Service) UnskipItem(ctx context.Context, tenantID uuid.UUID, itemID string) error {
	if !isValidItemID(itemID) {
		return ErrInvalidItemID
	}

	pgTenantID := uuidToPgtype(tenantID)
	return s.repo.UnskipItem(ctx, repository.UnskipItemParams{
		TenantID: pgTenantID,
		ItemID:   itemID,
	})
}

// IsLaunchReady returns true if all Phase 1 items are complete
func (s *Service) IsLaunchReady(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	status, err := s.GetStatus(ctx, tenantID)
	if err != nil {
		return false, err
	}
	return status.LaunchReady, nil
}

// buildChecklistStatus constructs the full OnboardingStatus from validation results
func (s *Service) buildChecklistStatus(
	tenantID uuid.UUID,
	v repository.GetAllOnboardingValidationsRow,
	skips map[string]bool,
) *domain.OnboardingStatus {
	// Phase 1: Critical Path (all required)
	// URLs must match actual routes in internal/routes/admin.go
	// Note: "account_activated" removed - if user is viewing dashboard, they're already logged in
	phase1Items := []domain.OnboardingItem{
		buildItem("stripe_connected", "Connect Stripe", "Configure payment processing", v.StripeConnected, skips, true, "/admin/settings/integrations/billing", "Connect Stripe"),
		buildItem("email_configured", "Configure Email", "Set up transactional email provider", v.EmailConfigured, skips, true, "/admin/settings/integrations/email", "Configure Email"),
		buildItem("first_product", "Add First Product", "Create at least one active product", v.FirstProduct, skips, true, "/admin/products/new", "Add Product"),
		buildItem("first_sku", "Create Product SKU", "Add weight/grind variant to product", v.FirstSku, skips, true, "/admin/products", "Manage Products"),
		buildItem("pricing_set", "Set Up Pricing", "Configure retail prices for SKUs", v.PricingSet, skips, true, "/admin/price-lists", "Set Prices"),
		buildItem("shipping_configured", "Configure Shipping", "Add at least one shipping method", v.ShippingConfigured, skips, true, "/admin/settings/integrations/shipping", "Configure Shipping"),
		buildItem("tax_configured", "Configure Taxes", "Select tax calculation method", v.TaxConfigured, skips, true, "/admin/settings/integrations/tax", "Configure Taxes"),
	}

	// Phase 2: Recommended (optional)
	phase2Items := []domain.OnboardingItem{
		buildItem("business_info", "Business Information", "Store name, contact details", v.BusinessInfo, skips, false, "/admin/settings/integrations", "Edit Settings"),
		buildItem("warehouse_address", "Warehouse Address", "Fulfillment origin for shipping", v.WarehouseAddress, skips, false, "/admin/settings/integrations", "Add Address"),
		buildItem("product_images", "Product Images", "Upload images for products", v.ProductImages, skips, false, "/admin/products", "Add Images"),
		buildItem("coffee_attributes", "Coffee Details", "Origin, roast level, tasting notes", v.CoffeeAttributes, skips, false, "/admin/products", "Edit Products"),
	}

	// Phase 3: Wholesale (optional)
	phase3Items := []domain.OnboardingItem{
		buildItem("wholesale_pricing", "Wholesale Pricing", "Create B2B price list", v.WholesalePricing, skips, false, "/admin/price-lists/new", "Create Price List"),
		buildItem("payment_terms", "Payment Terms", "Configure net terms for wholesale", v.PaymentTerms, skips, false, "/admin/settings/integrations", "Configure Terms"),
	}

	phase1 := buildPhase("Critical Path", "Required to launch your store", phase1Items)
	phase2 := buildPhase("Recommended", "Improve customer experience", phase2Items)
	phase3 := buildPhase("Wholesale", "Enable B2B sales", phase3Items)
	phase4 := domain.OnboardingPhase{
		Name:        "Advanced",
		Description: "Optional enhancements",
		Items:       []domain.OnboardingItem{},
		AllComplete: true,
		TotalCount:  0,
	}

	// Launch ready = all Phase 1 complete
	launchReady := phase1.AllComplete

	// Calculate overall progress (Phase 1 only for progress bar)
	progress := domain.OnboardingProgress{
		CompletedCount:  phase1.CompletedCount,
		TotalRequired:   phase1.TotalCount,
		PercentComplete: 0,
	}
	if phase1.TotalCount > 0 {
		progress.PercentComplete = (phase1.CompletedCount * 100) / phase1.TotalCount
	}

	return &domain.OnboardingStatus{
		TenantID:    tenantID,
		LaunchReady: launchReady,
		Phase1:      phase1,
		Phase2:      phase2,
		Phase3:      phase3,
		Phase4:      phase4,
		Progress:    progress,
		ComputedAt:  time.Now(),
	}
}

// buildItem creates an OnboardingItem with computed status
func buildItem(
	id, name, description string,
	isComplete bool,
	skips map[string]bool,
	required bool,
	actionURL, actionLabel string,
) domain.OnboardingItem {
	var status domain.OnboardingItemStatus

	if skips[id] {
		status = domain.OnboardingStatusSkipped
	} else if isComplete {
		status = domain.OnboardingStatusCompleted
	} else {
		status = domain.OnboardingStatusNotStarted
	}

	return domain.OnboardingItem{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      status,
		Required:    required,
		ActionURL:   actionURL,
		ActionLabel: actionLabel,
	}
}

// buildPhase creates an OnboardingPhase with summary stats
func buildPhase(name, description string, items []domain.OnboardingItem) domain.OnboardingPhase {
	completed := 0
	for _, item := range items {
		if item.Status == domain.OnboardingStatusCompleted {
			completed++
		}
	}

	allComplete := completed == len(items)

	return domain.OnboardingPhase{
		Name:           name,
		Description:    description,
		Items:          items,
		AllComplete:    allComplete,
		CompletedCount: completed,
		TotalCount:     len(items),
	}
}

// isValidItemID returns true if the item ID is a known checklist item
func isValidItemID(itemID string) bool {
	allItems := append(append(domain.Phase1ItemIDs, domain.Phase2ItemIDs...), domain.Phase3ItemIDs...)
	for _, id := range allItems {
		if itemID == id {
			return true
		}
	}
	return false
}

// uuidToPgtype converts a google/uuid.UUID to pgtype.UUID
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}
