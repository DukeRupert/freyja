package domain

import (
	"time"

	"github.com/google/uuid"
)

// OnboardingStatus represents the complete onboarding state for a tenant.
// Status is computed dynamically from actual data (products, configs, etc.)
// rather than stored explicitly.
type OnboardingStatus struct {
	TenantID    uuid.UUID          `json:"tenant_id"`
	LaunchReady bool               `json:"launch_ready"`
	Phase1      OnboardingPhase    `json:"phase1"`
	Phase2      OnboardingPhase    `json:"phase2"`
	Phase3      OnboardingPhase    `json:"phase3"`
	Phase4      OnboardingPhase    `json:"phase4"`
	Progress    OnboardingProgress `json:"progress"`
	ComputedAt  time.Time          `json:"computed_at"`
}

// OnboardingProgress provides summary stats across all phases
type OnboardingProgress struct {
	CompletedCount  int `json:"completed_count"`
	TotalRequired   int `json:"total_required"`
	PercentComplete int `json:"percent_complete"`
}

// OnboardingPhase represents a collection of checklist items
type OnboardingPhase struct {
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	Items          []OnboardingItem `json:"items"`
	AllComplete    bool             `json:"all_complete"`
	CompletedCount int              `json:"completed_count"`
	TotalCount     int              `json:"total_count"`
}

// OnboardingItem represents a single checklist step
type OnboardingItem struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      OnboardingItemStatus `json:"status"`
	Required    bool                 `json:"required"`
	ActionURL   string               `json:"action_url"`
	ActionLabel string               `json:"action_label"`
}

// OnboardingItemStatus represents the state of a checklist item
type OnboardingItemStatus string

const (
	OnboardingStatusNotStarted OnboardingItemStatus = "not_started"
	OnboardingStatusInProgress OnboardingItemStatus = "in_progress"
	OnboardingStatusCompleted  OnboardingItemStatus = "completed"
	OnboardingStatusSkipped    OnboardingItemStatus = "skipped"
)

// OnboardingItemSkip represents a skipped checklist item stored in the database
type OnboardingItemSkip struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	ItemID    string     `json:"item_id"`
	SkippedAt time.Time  `json:"skipped_at"`
	SkippedBy *uuid.UUID `json:"skipped_by,omitempty"`
}

// Phase 1 item IDs (required for launch)
var Phase1ItemIDs = []string{
	"stripe_connected",
	"email_configured",
	"first_product",
	"first_sku",
	"pricing_set",
	"shipping_configured",
	"tax_configured",
}

// Phase 2 item IDs (recommended, optional)
var Phase2ItemIDs = []string{
	"business_info",
	"warehouse_address",
	"product_images",
	"coffee_attributes",
}

// Phase 3 item IDs (wholesale, optional)
var Phase3ItemIDs = []string{
	"wholesale_pricing",
	"payment_terms",
}

// IsSkippable returns true if the item can be skipped (not in Phase 1)
func IsSkippable(itemID string) bool {
	for _, id := range Phase1ItemIDs {
		if itemID == id {
			return false
		}
	}
	return true
}

// IsComplete returns true if the item is completed
func (i OnboardingItem) IsComplete() bool {
	return i.Status == OnboardingStatusCompleted
}

// IsSkipped returns true if the item was explicitly skipped
func (i OnboardingItem) IsSkipped() bool {
	return i.Status == OnboardingStatusSkipped
}

// IsPending returns true if the item hasn't been started or skipped
func (i OnboardingItem) IsPending() bool {
	return i.Status == OnboardingStatusNotStarted || i.Status == OnboardingStatusInProgress
}
