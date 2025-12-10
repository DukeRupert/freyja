package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/email"
	"github.com/dukerupert/hiri/internal/jobs"
	"github.com/dukerupert/hiri/internal/repository"
)

// Config holds worker configuration
type Config struct {
	// WorkerID uniquely identifies this worker instance
	WorkerID string

	// PollInterval is how often to check for new jobs
	PollInterval time.Duration

	// MaxConcurrency is the maximum number of jobs to process concurrently
	MaxConcurrency int

	// Queue name to process (empty string = all queues)
	Queue string

	// TenantID to process jobs for (nil = all tenants)
	TenantID *uuid.UUID
}

// Worker processes background jobs
type Worker struct {
	config         Config
	queries        *repository.Queries
	emailService   *email.Service
	invoiceService domain.InvoiceService
	logger         *slog.Logger
}

// NewWorker creates a new background job worker
func NewWorker(
	queries *repository.Queries,
	emailService *email.Service,
	invoiceService domain.InvoiceService,
	config Config,
	logger *slog.Logger,
) *Worker {
	// Set defaults
	if config.WorkerID == "" {
		config.WorkerID = fmt.Sprintf("worker-%s", uuid.New().String()[:8])
	}
	if config.PollInterval == 0 {
		config.PollInterval = 1 * time.Second
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 5
	}

	return &Worker{
		config:         config,
		queries:        queries,
		emailService:   emailService,
		invoiceService: invoiceService,
		logger:         logger,
	}
}

// Start begins processing jobs until the context is cancelled
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("worker starting",
		"worker_id", w.config.WorkerID,
		"queue", w.config.Queue,
		"poll_interval", w.config.PollInterval,
		"max_concurrency", w.config.MaxConcurrency,
	)

	// TODO: Implement main worker loop
	// 1. Create a ticker for polling
	// 2. Use a semaphore or goroutine pool for concurrency control
	// 3. In each poll:
	//    - Claim next job using ClaimNextJob query
	//    - If job found, spawn goroutine to process it
	//    - Process job via processJob()
	//    - Mark as complete or failed
	// 4. Handle graceful shutdown on ctx.Done()

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	// Semaphore for concurrency control
	sem := make(chan struct{}, w.config.MaxConcurrency)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker shutting down", "worker_id", w.config.WorkerID)
			// Wait for in-flight jobs to complete
			// TODO: Implement graceful shutdown with timeout
			return ctx.Err()

		case <-ticker.C:
			// Try to claim a job
			select {
			case sem <- struct{}{}:
				// Acquired semaphore, try to claim job
				go func() {
					defer func() { <-sem }()
					w.claimAndProcess(ctx)
				}()
			default:
				// At max concurrency, skip this poll
			}
		}
	}
}

// claimAndProcess claims and processes a single job
func (w *Worker) claimAndProcess(ctx context.Context) {
	// Recover from panics so the worker doesn't die silently
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("job panicked",
				"panic", fmt.Sprintf("%v", r),
			)
		}
	}()

	var tenantID pgtype.UUID
	if w.config.TenantID != nil {
		tenantID = pgtype.UUID{Bytes: *w.config.TenantID, Valid: true}
	}

	job, err := w.queries.ClaimNextJob(ctx, repository.ClaimNextJobParams{
		WorkerID: pgtype.Text{String: w.config.WorkerID, Valid: true},
		TenantID: tenantID,
		Queue:    w.config.Queue,
	})
	if err != nil {
		// No job available or database error
		return
	}

	w.logger.Info("processing job",
		"job_id", job.ID,
		"job_type", job.JobType,
		"retry_count", job.RetryCount,
	)

	err = w.processJob(ctx, &job)
	if err != nil {
		w.logger.Error("job failed",
			"job_id", job.ID,
			"job_type", job.JobType,
			"error", err,
		)
		// Mark job as failed (will retry or mark as failed based on retry count)
		_, _ = w.queries.FailJob(ctx, repository.FailJobParams{
			ID:           job.ID,
			ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			ErrorDetails: []byte("{}"),
		})
		return
	}

	w.logger.Info("job completed",
		"job_id", job.ID,
		"job_type", job.JobType,
	)

	// Mark job as completed
	_ = w.queries.CompleteJob(ctx, job.ID)
}

// processJob processes a single job
func (w *Worker) processJob(ctx context.Context, job *repository.Job) error {
	jobCtx, cancel := context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second)
	defer cancel()

	// Inject tenant context before calling services
	tenantCtx, err := withTenantContext(jobCtx, job)
	if err != nil {
		return fmt.Errorf("failed to create tenant context: %w", err)
	}

	if isEmailJob(job.JobType) {
		return jobs.ProcessEmailJob(tenantCtx, job, w.emailService, w.queries)
	}

	if isInvoiceJob(job.JobType) {
		return w.processInvoiceJob(tenantCtx, job)
	}

	if jobs.IsCleanupJob(job.JobType) {
		result, err := jobs.ProcessCleanupJob(tenantCtx, job, w.queries)
		if err != nil {
			return err
		}
		w.logger.Info("cleanup job completed",
			"job_id", job.ID,
			"email_tokens_deleted", result.EmailVerificationTokensDeleted,
			"password_tokens_deleted", result.PasswordResetTokensDeleted,
		)
		return nil
	}

	return fmt.Errorf("unknown job type: %s", job.JobType)
}

// processInvoiceJob processes an invoice job based on its type
func (w *Worker) processInvoiceJob(ctx context.Context, job *repository.Job) error {
	switch job.JobType {
	case jobs.JobTypeGenerateConsolidatedInvoice:
		var payload jobs.GenerateConsolidatedInvoicePayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal consolidated invoice payload: %w", err)
		}

		_, err := w.invoiceService.GenerateConsolidatedInvoice(ctx, domain.ConsolidatedInvoiceParams{
			UserID:             payload.UserID.String(),
			BillingPeriodStart: payload.BillingPeriodStart,
			BillingPeriodEnd:   payload.BillingPeriodEnd,
		})
		return err

	case jobs.JobTypeMarkOverdueInvoices:
		count, err := w.invoiceService.MarkInvoicesOverdue(ctx)
		if err != nil {
			return fmt.Errorf("failed to mark overdue invoices: %w", err)
		}
		w.logger.Info("marked invoices as overdue", "count", count)
		return nil

	case jobs.JobTypeSendInvoiceReminder:
		var payload jobs.SendInvoiceReminderPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal reminder payload: %w", err)
		}

		// Get invoice details
		invoice, err := w.invoiceService.GetInvoice(ctx, payload.InvoiceID.String())
		if err != nil {
			return fmt.Errorf("invoice not found: %w", err)
		}

		// Get user info for email
		user, err := w.queries.GetUserByID(ctx, invoice.Invoice.UserID)
		if err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		// Build customer name
		customerName := user.Email
		if user.FirstName.Valid {
			customerName = user.FirstName.String
			if user.LastName.Valid {
				customerName += " " + user.LastName.String
			}
		}

		// Payment URL
		paymentURL := fmt.Sprintf("/invoices/%s", payload.InvoiceID.String())

		// Determine if this is an overdue email or a reminder
		if payload.ReminderType == "past_due" || payload.DaysOverdue > 0 {
			// Enqueue overdue email
			overduePayload := jobs.InvoiceOverduePayload{
				InvoiceID:     payload.InvoiceID,
				Email:         user.Email,
				CustomerName:  customerName,
				InvoiceNumber: invoice.Invoice.InvoiceNumber,
				DueDate:       invoice.Invoice.DueDate.Time,
				BalanceCents:  int64(invoice.Invoice.BalanceCents),
				DaysOverdue:   payload.DaysOverdue,
				PaymentURL:    paymentURL,
			}

			tenantID := uuid.UUID(invoice.Invoice.TenantID.Bytes)
			if err := jobs.EnqueueInvoiceOverdueEmail(ctx, w.queries, tenantID, overduePayload); err != nil {
				return fmt.Errorf("failed to enqueue overdue email: %w", err)
			}
		} else {
			// Enqueue reminder email
			reminderPayload := jobs.InvoiceReminderPayload{
				InvoiceID:     payload.InvoiceID,
				Email:         user.Email,
				CustomerName:  customerName,
				InvoiceNumber: invoice.Invoice.InvoiceNumber,
				DueDate:       invoice.Invoice.DueDate.Time,
				BalanceCents:  int64(invoice.Invoice.BalanceCents),
				ReminderType:  payload.ReminderType,
				DaysBefore:    payload.DaysBefore,
				DaysOverdue:   payload.DaysOverdue,
				PaymentURL:    paymentURL,
			}

			tenantID := uuid.UUID(invoice.Invoice.TenantID.Bytes)
			if err := jobs.EnqueueInvoiceReminderEmail(ctx, w.queries, tenantID, reminderPayload); err != nil {
				return fmt.Errorf("failed to enqueue reminder email: %w", err)
			}
		}

		w.logger.Info("invoice reminder email enqueued",
			"invoice_id", payload.InvoiceID,
			"reminder_type", payload.ReminderType)
		return nil

	case jobs.JobTypeSyncInvoiceFromStripe:
		var payload jobs.SyncInvoiceFromStripePayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal sync payload: %w", err)
		}

		return w.invoiceService.SyncInvoiceFromStripe(ctx, payload.StripeInvoiceID)

	default:
		return fmt.Errorf("unknown invoice job type: %s", job.JobType)
	}
}

// isEmailJob checks if a job type is an email job
func isEmailJob(jobType string) bool {
	switch jobType {
	case jobs.JobTypePasswordReset,
		jobs.JobTypeEmailVerification,
		jobs.JobTypeOrderConfirmation,
		jobs.JobTypeShippingConfirmation,
		jobs.JobTypeSubscriptionWelcome,
		jobs.JobTypeSubscriptionPaymentFailed,
		jobs.JobTypeSubscriptionCancelled,
		jobs.JobTypeInvoiceSent,
		jobs.JobTypeInvoiceReminder,
		jobs.JobTypeInvoiceOverdue:
		return true
	}
	return false
}

// isInvoiceJob checks if a job type is an invoice job
func isInvoiceJob(jobType string) bool {
	switch jobType {
	case jobs.JobTypeGenerateConsolidatedInvoice,
		jobs.JobTypeMarkOverdueInvoices,
		jobs.JobTypeSendInvoiceReminder,
		jobs.JobTypeSyncInvoiceFromStripe:
		return true
	}
	return false
}
