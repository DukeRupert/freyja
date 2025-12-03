package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dukerupert/freyja/internal/repository"
)

// Job type constants for cleanup jobs
const (
	JobTypeCleanupExpiredTokens = "cleanup:expired_tokens"
)

// CleanupExpiredTokensPayload represents the payload for a token cleanup job
// This is intentionally minimal as the job is self-contained
type CleanupExpiredTokensPayload struct {
	// No fields needed - the job cleans up all expired tokens across all tenants
}

// EnqueueCleanupExpiredTokens enqueues a job to clean up expired tokens
// This should be called on a schedule (e.g., daily) to remove expired
// email verification and password reset tokens
func EnqueueCleanupExpiredTokens(ctx context.Context, q repository.Querier, tenantID uuid.UUID) error {
	payloadJSON, err := json.Marshal(CleanupExpiredTokensPayload{})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeCleanupExpiredTokens,
		Queue:      "cleanup",
		Payload:    payloadJSON,
		Priority:   10, // Low priority - maintenance task
		MaxRetries: 1,  // Don't retry on failure, will run again next scheduled time
		ScheduledAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		TimeoutSeconds: 60, // Allow up to 1 minute for cleanup
		Metadata:       []byte("{}"),
	})

	return err
}

// CleanupResult holds the result of a cleanup operation
type CleanupResult struct {
	EmailVerificationTokensDeleted int64 `json:"email_verification_tokens_deleted"`
	PasswordResetTokensDeleted     int64 `json:"password_reset_tokens_deleted"`
}

// ProcessCleanupJob processes a cleanup job based on its type
func ProcessCleanupJob(ctx context.Context, job *repository.Job, queries *repository.Queries) (*CleanupResult, error) {
	switch job.JobType {
	case JobTypeCleanupExpiredTokens:
		return processCleanupExpiredTokens(ctx, queries)
	default:
		return nil, fmt.Errorf("unknown cleanup job type: %s", job.JobType)
	}
}

// processCleanupExpiredTokens deletes expired email verification and password reset tokens
func processCleanupExpiredTokens(ctx context.Context, queries *repository.Queries) (*CleanupResult, error) {
	result := &CleanupResult{}

	// Delete expired email verification tokens
	err := queries.DeleteExpiredEmailVerificationTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to delete expired email verification tokens: %w", err)
	}
	// Note: The query doesn't return count, so we can't populate this field accurately
	// In a future iteration, we could modify the query to return affected rows

	// Delete expired password reset tokens
	err = queries.DeleteExpiredPasswordResetTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to delete expired password reset tokens: %w", err)
	}

	return result, nil
}

// IsCleanupJob checks if a job type is a cleanup job
func IsCleanupJob(jobType string) bool {
	switch jobType {
	case JobTypeCleanupExpiredTokens:
		return true
	}
	return false
}
