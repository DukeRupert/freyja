// internal/interfaces/admin.go
package interfaces

import "context"

type AdminService interface {
	// Backfill operations
	BackfillCustomerStripeSync(ctx context.Context, req BackfillCustomersRequest) (*BackfillResult, error)
	BackfillProductStripeSync(ctx context.Context, req BackfillProductsRequest) (*BackfillResult, error)

	// Status checks
	GetSyncStatus(ctx context.Context) (*SyncStatusReport, error)
	GetBackfillStatus(ctx context.Context, jobID string) (*BackfillJobStatus, error)
}

// Request types
type BackfillCustomersRequest struct {
	BatchSize    int  `json:"batch_size,omitempty"`    // Default: 50
	MaxCustomers int  `json:"max_customers,omitempty"` // Default: unlimited
	DryRun       bool `json:"dry_run,omitempty"`       // Default: false
}

type BackfillProductsRequest struct {
	BatchSize   int  `json:"batch_size,omitempty"`   // Default: 20
	MaxProducts int  `json:"max_products,omitempty"` // Default: unlimited
	DryRun      bool `json:"dry_run,omitempty"`      // Default: false
}

// Response types
type BackfillResult struct {
	JobID          string          `json:"job_id"`
	Status         string          `json:"status"` // "started", "running", "completed", "failed"
	TotalItems     int             `json:"total_items"`
	ProcessedItems int             `json:"processed_items"`
	SuccessCount   int             `json:"success_count"`
	ErrorCount     int             `json:"error_count"`
	Errors         []BackfillError `json:"errors,omitempty"`
	StartedAt      string          `json:"started_at"`
	CompletedAt    string          `json:"completed_at,omitempty"`
}

type BackfillError struct {
	ItemID int32  `json:"item_id"`
	Error  string `json:"error"`
}

type SyncStatusReport struct {
	Customers struct {
		Total           int64   `json:"total"`
		WithStripeID    int64   `json:"with_stripe_id"`
		WithoutStripeID int64   `json:"without_stripe_id"`
		SyncPercentage  float64 `json:"sync_percentage"`
	} `json:"customers"`

	Products struct {
		Total            int64   `json:"total"`
		WithVariants     int64   `json:"with_variants"`     // NEW: Products that have variants
		WithoutVariants  int64   `json:"without_variants"`  // NEW: Products needing variants
		SyncPercentage   float64 `json:"sync_percentage"`   // Now represents "readiness" (has variants)
	} `json:"products"`

	Variants struct {
		Total             int64   `json:"total"`              // NEW: Total active variants
		WithStripeSync    int64   `json:"with_stripe_sync"`   // NEW: Variants synced to Stripe
		WithoutStripeSync int64   `json:"without_stripe_sync"` // NEW: Variants not synced
		SyncPercentage    float64 `json:"sync_percentage"`    // NEW: Variant sync percentage
	} `json:"variants"` // NEW: Variant sync status section
}

type VariantSyncStatus struct {
    Total            int64   `json:"total"`
    WithStripeSync   int64   `json:"with_stripe_sync"`
    WithoutStripeSync int64  `json:"without_stripe_sync"`
    SyncPercentage   float64 `json:"sync_percentage"`
}

type ProductSyncStatus struct {
    Total            int64   `json:"total"`
    WithVariants     int64   `json:"with_variants"`     // NEW
    WithoutVariants  int64   `json:"without_variants"`  // NEW
    SyncPercentage   float64 `json:"sync_percentage"`   // Now represents "readiness"
}

type BackfillJobStatus struct {
	JobID          string          `json:"job_id"`
	Status         string          `json:"status"`
	Progress       float64         `json:"progress"` // 0-100
	TotalItems     int             `json:"total_items"`
	ProcessedItems int             `json:"processed_items"`
	SuccessCount   int             `json:"success_count"`
	ErrorCount     int             `json:"error_count"`
	Errors         []BackfillError `json:"errors"`
	StartedAt      string          `json:"started_at"`
	CompletedAt    string          `json:"completed_at,omitempty"`
	EstimatedETA   string          `json:"estimated_eta,omitempty"`
}
