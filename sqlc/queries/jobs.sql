-- name: EnqueueJob :one
-- Insert a new job into the queue
INSERT INTO jobs (
    tenant_id,
    job_type,
    queue,
    payload,
    priority,
    max_retries,
    scheduled_at,
    timeout_seconds,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: ClaimNextJob :one
-- Claim the next pending job using SKIP LOCKED for safe concurrent access
-- This query finds the highest priority job that's ready to run
UPDATE jobs
SET
    status = 'processing',
    processing_started_at = NOW(),
    worker_id = $1
WHERE id = (
    SELECT j.id
    FROM jobs j
    WHERE j.status = 'pending'
      AND j.scheduled_at <= NOW()
      AND (j.tenant_id = $2 OR $2 IS NULL)
      AND (j.queue = $3 OR $3 = '')
    ORDER BY j.priority ASC, j.scheduled_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: CompleteJob :exec
-- Mark a job as completed
UPDATE jobs
SET
    status = 'completed',
    processing_completed_at = NOW()
WHERE id = $1
  AND status = 'processing';

-- name: FailJob :one
-- Mark a job as failed or reschedule it for retry
-- If retry_count < max_retries, reschedule; otherwise mark as failed
UPDATE jobs
SET
    status = CASE
        WHEN retry_count + 1 < max_retries THEN 'pending'
        ELSE 'failed'
    END,
    retry_count = retry_count + 1,
    scheduled_at = CASE
        WHEN retry_count + 1 < max_retries
        THEN NOW() + (retry_backoff_seconds * POWER(2, retry_count) || ' seconds')::INTERVAL
        ELSE scheduled_at
    END,
    processing_completed_at = CASE
        WHEN retry_count + 1 >= max_retries THEN NOW()
        ELSE NULL
    END,
    worker_id = CASE
        WHEN retry_count + 1 < max_retries THEN NULL
        ELSE worker_id
    END,
    processing_started_at = CASE
        WHEN retry_count + 1 < max_retries THEN NULL
        ELSE processing_started_at
    END,
    error_message = $2,
    error_details = $3
WHERE id = $1
  AND status = 'processing'
RETURNING *;

-- name: GetJobByID :one
-- Fetch a job by ID
SELECT *
FROM jobs
WHERE id = $1
LIMIT 1;

-- name: DeleteOldCompletedJobs :exec
-- Cleanup old completed jobs (history is preserved in job_history table)
-- Delete jobs older than the specified timestamp
DELETE FROM jobs
WHERE status IN ('completed', 'failed', 'cancelled')
  AND processing_completed_at < $1;

-- name: ListJobsByStatus :many
-- List jobs by status for monitoring
SELECT *
FROM jobs
WHERE status = $1
  AND (tenant_id = $2 OR $2 IS NULL)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountJobsByStatus :one
-- Count jobs by status
SELECT COUNT(*)
FROM jobs
WHERE status = $1
  AND (tenant_id = $2 OR $2 IS NULL);

-- name: GetJobStats :one
-- Get job queue statistics
SELECT
    COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE status = 'processing') as processing_count,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled_count
FROM jobs
WHERE tenant_id = $1 OR $1 IS NULL;

-- name: CancelJob :exec
-- Cancel a pending job
UPDATE jobs
SET
    status = 'cancelled',
    processing_completed_at = NOW()
WHERE id = $1
  AND status = 'pending';
