package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dukerupert/freyja/internal/email"
	"github.com/dukerupert/freyja/internal/jobs"
	"github.com/dukerupert/freyja/internal/repository"
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
	config       Config
	queries      *repository.Queries
	emailService *email.Service
	logger       *slog.Logger
}

// NewWorker creates a new background job worker
func NewWorker(queries *repository.Queries, emailService *email.Service, config Config, logger *slog.Logger) *Worker {
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
		config:       config,
		queries:      queries,
		emailService: emailService,
		logger:       logger,
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
	// TODO: Implement job claiming and processing
	// 1. Call ClaimNextJob with worker ID, tenant ID, queue
	// 2. If no job available, return immediately
	// 3. Call processJob() with claimed job
	// 4. Handle result (complete or fail the job)

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
			ErrorDetails: []byte("{}"), // TODO: Add structured error details
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
	// TODO: Add timeout handling using job.TimeoutSeconds
	// Create a context with timeout
	// ctx, cancel := context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second)
	// defer cancel()

	// Route to appropriate job processor based on job type
	// For now, we only have email jobs
	if isEmailJob(job.JobType) {
		return jobs.ProcessEmailJob(ctx, job, w.emailService, w.queries)
	}

	return fmt.Errorf("unknown job type: %s", job.JobType)
}

// isEmailJob checks if a job type is an email job
func isEmailJob(jobType string) bool {
	switch jobType {
	case jobs.JobTypePasswordReset,
		jobs.JobTypeOrderConfirmation,
		jobs.JobTypeShippingConfirmation,
		jobs.JobTypeSubscriptionWelcome,
		jobs.JobTypeSubscriptionPaymentFailed,
		jobs.JobTypeSubscriptionCancelled:
		return true
	}
	return false
}
