// internal/service/admin.go
package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type AdminService struct {
	customerService interfaces.CustomerService
	productService  interfaces.ProductService
	variantService  interfaces.VariantService
	events          interfaces.EventPublisher

	// Job tracking
	jobs    map[string]*interfaces.BackfillJobStatus
	jobsMux sync.RWMutex
}

func NewAdminService(
	customerService interfaces.CustomerService,
	productService interfaces.ProductService,
	variantService interfaces.VariantService,
	events interfaces.EventPublisher,
) interfaces.AdminService {
	return &AdminService{
		customerService: customerService,
		productService:  productService,
		variantService:  variantService,
		events:          events,
		jobs:            make(map[string]*interfaces.BackfillJobStatus),
	}
}

// BackfillCustomerStripeSync starts customer Stripe sync in background
func (s *AdminService) BackfillCustomerStripeSync(ctx context.Context, req interfaces.BackfillCustomersRequest) (*interfaces.BackfillResult, error) {
	// Set defaults
	if req.BatchSize <= 0 {
		req.BatchSize = 50
	}
	if req.BatchSize > 100 {
		req.BatchSize = 100 // Safety limit
	}

	// Generate job ID
	jobID := generateJobID("customer")

	// Initialize job status
	job := &interfaces.BackfillJobStatus{
		JobID:     jobID,
		Status:    "started",
		StartedAt: time.Now().Format(time.RFC3339),
		Errors:    []interfaces.BackfillError{},
	}

	s.jobsMux.Lock()
	s.jobs[jobID] = job
	s.jobsMux.Unlock()

	// Start background processing
	go s.processCustomerBackfill(context.Background(), jobID, req)

	return &interfaces.BackfillResult{
		JobID:     jobID,
		Status:    "started",
		StartedAt: job.StartedAt,
	}, nil
}

// BackfillProductStripeSync starts product Stripe sync in background
func (s *AdminService) BackfillProductStripeSync(ctx context.Context, req interfaces.BackfillProductsRequest) (*interfaces.BackfillResult, error) {
	// Set defaults
	if req.BatchSize <= 0 {
		req.BatchSize = 20
	}
	if req.BatchSize > 50 {
		req.BatchSize = 50 // Safety limit (Stripe API rate limits)
	}

	// Generate job ID
	jobID := generateJobID("product")

	// Initialize job status
	job := &interfaces.BackfillJobStatus{
		JobID:     jobID,
		Status:    "started",
		StartedAt: time.Now().Format(time.RFC3339),
		Errors:    []interfaces.BackfillError{},
	}

	s.jobsMux.Lock()
	s.jobs[jobID] = job
	s.jobsMux.Unlock()

	// Start background processing
	go s.processProductBackfill(context.Background(), jobID, req)

	return &interfaces.BackfillResult{
		JobID:     jobID,
		Status:    "started",
		StartedAt: job.StartedAt,
	}, nil
}

// GetSyncStatus returns current sync status for customers and products/variants
func (s *AdminService) GetSyncStatus(ctx context.Context) (*interfaces.SyncStatusReport, error) {
	report := &interfaces.SyncStatusReport{}

	// Get customer sync status
	totalCustomers, err := s.customerService.GetCustomerCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer count: %w", err)
	}

	customersWithStripe, err := s.customerService.GetCustomersWithStripeCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get customers with Stripe count: %w", err)
	}

	report.Customers.Total = totalCustomers
	report.Customers.WithStripeID = customersWithStripe
	report.Customers.WithoutStripeID = totalCustomers - customersWithStripe
	if totalCustomers > 0 {
		report.Customers.SyncPercentage = float64(customersWithStripe) / float64(totalCustomers) * 100
	}

	// Get product readiness status - use GetAll to count active products
	activeProducts, err := s.productService.GetAll(ctx, interfaces.ProductFilters{
		Active: &[]bool{true}[0], // Active products only
		Limit:  10000,            // Large limit to get all
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get active products: %w", err)
	}

	// Products without variants need attention
	productsWithoutVariants, err := s.productService.GetProductsWithoutVariants(ctx, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get products without variants: %w", err)
	}

	totalProducts := int64(len(activeProducts))
	report.Products.Total = totalProducts
	report.Products.WithoutVariants = int64(len(productsWithoutVariants))
	report.Products.WithVariants = totalProducts - report.Products.WithoutVariants
	if totalProducts > 0 {
		report.Products.SyncPercentage = float64(report.Products.WithVariants) / float64(totalProducts) * 100
	}

	// Get variant sync status using the new methods
	totalVariants, err := s.variantService.GetVariantCount(ctx, true) // Active variants only
	if err != nil {
		return nil, fmt.Errorf("failed to get variant count: %w", err)
	}

	variantsWithStripe, err := s.variantService.GetVariantsWithStripeCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get variants with Stripe count: %w", err)
	}

	report.Variants.Total = totalVariants
	report.Variants.WithStripeSync = variantsWithStripe
	report.Variants.WithoutStripeSync = totalVariants - variantsWithStripe
	if totalVariants > 0 {
		report.Variants.SyncPercentage = float64(variantsWithStripe) / float64(totalVariants) * 100
	}

	return report, nil
}

// GetBackfillStatus returns status of a specific backfill job
func (s *AdminService) GetBackfillStatus(ctx context.Context, jobID string) (*interfaces.BackfillJobStatus, error) {
	s.jobsMux.RLock()
	job, exists := s.jobs[jobID]
	s.jobsMux.RUnlock()

	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	// Calculate progress
	if job.TotalItems > 0 {
		job.Progress = float64(job.ProcessedItems) / float64(job.TotalItems) * 100
	}

	return job, nil
}

// Background processing for customer backfill
func (s *AdminService) processCustomerBackfill(ctx context.Context, jobID string, req interfaces.BackfillCustomersRequest) {
	s.updateJobStatus(jobID, "running", func(job *interfaces.BackfillJobStatus) {
		// Implementation would get customers without Stripe IDs and process them
		// This is a simplified version - you'd need to implement the actual logic

		log.Printf("Starting customer backfill job %s", jobID)

		// Get customers without Stripe IDs (you'd need to implement this method)
		customers, err := s.customerService.GetCustomersWithoutStripeIDs(ctx, req.MaxCustomers, 0)
		if err != nil {
			job.Status = "failed"
			job.Errors = append(job.Errors, interfaces.BackfillError{
				ItemID: 0,
				Error:  fmt.Sprintf("Failed to get customers: %v", err),
			})
			return
		}

		job.TotalItems = len(customers)

		// Process in batches
		for i := 0; i < len(customers); i += req.BatchSize {
			end := i + req.BatchSize
			end = min(end, len(customers))
			batch := customers[i:end]

			for _, customer := range batch {
				if req.DryRun {
					log.Printf("DRY RUN: Would sync customer %d", customer.ID)
				} else {
					// Trigger customer sync by publishing event
					if err := s.publishCustomerSyncEvent(ctx, customer.ID); err != nil {
						job.ErrorCount++
						job.Errors = append(job.Errors, interfaces.BackfillError{
							ItemID: customer.ID,
							Error:  err.Error(),
						})
					} else {
						job.SuccessCount++
					}
				}
				job.ProcessedItems++
			}

			// Small delay between batches to avoid overwhelming Stripe API
			time.Sleep(1 * time.Second)
		}

		job.Status = "completed"
		log.Printf("Completed customer backfill job %s: %d/%d successful",
			jobID, job.SuccessCount, job.TotalItems)
	})
}

// Background processing for product backfill
func (s *AdminService) processProductBackfill(ctx context.Context, jobID string, req interfaces.BackfillProductsRequest) {
	s.updateJobStatus(jobID, "running", func(job *interfaces.BackfillJobStatus) {
		log.Printf("Starting product backfill job %s", jobID)

		// Get products without Stripe sync
		products, err := s.productService.GetProductsWithoutStripeSync(ctx, req.MaxProducts, 0)
		if err != nil {
			job.Status = "failed"
			job.Errors = append(job.Errors, interfaces.BackfillError{
				ItemID: 0,
				Error:  fmt.Sprintf("Failed to get products: %v", err),
			})
			return
		}

		job.TotalItems = len(products)

		// Process in batches
		for i := 0; i < len(products); i += req.BatchSize {
			end := i + req.BatchSize
			end = min(end, len(products))
			batch := products[i:end]

			for _, product := range batch {
				if req.DryRun {
					log.Printf("DRY RUN: Would sync product %d", product.ID)
				} else {
					// Trigger product sync
					if err := s.productService.EnsureStripeProduct(ctx, product.ID); err != nil {
						job.ErrorCount++
						job.Errors = append(job.Errors, interfaces.BackfillError{
							ItemID: product.ID,
							Error:  err.Error(),
						})
					} else {
						job.SuccessCount++
					}
				}
				job.ProcessedItems++
			}

			// Delay between batches (longer for products due to multiple Stripe API calls)
			time.Sleep(2 * time.Second)
		}

		job.Status = "completed"
		log.Printf("Completed product backfill job %s: %d/%d successful",
			jobID, job.SuccessCount, job.TotalItems)
	})
}

// Helper methods
func (s *AdminService) updateJobStatus(jobID string, status string, updateFunc func(*interfaces.BackfillJobStatus)) {
	s.jobsMux.Lock()
	defer s.jobsMux.Unlock()

	job := s.jobs[jobID]
	if job != nil {
		job.Status = status
		if updateFunc != nil {
			updateFunc(job)
		}
		if status == "completed" || status == "failed" {
			job.CompletedAt = time.Now().Format(time.RFC3339)
		}
	}
}

func (s *AdminService) publishCustomerSyncEvent(ctx context.Context, customerID int32) error {
	event := interfaces.BuildCustomerEvent("customer.stripe_sync_requested", customerID, map[string]interface{}{
		"sync_requested": true,
		"triggered_by":   "admin_backfill",
	})
	return s.events.PublishEvent(ctx, event)
}

func generateJobID(prefix string) string {
	return fmt.Sprintf("%s_backfill_%d", prefix, time.Now().UnixNano())
}
