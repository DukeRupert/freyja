-- +goose Up
-- +goose StatementBegin

-- Jobs: background job queue (database-backed)
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL for system-level jobs

    -- Job identification
    job_type VARCHAR(100) NOT NULL, -- e.g., 'send_email', 'process_subscription', 'sync_inventory'
    queue VARCHAR(50) NOT NULL DEFAULT 'default', -- Job queue name

    -- Job status
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending',
        'processing',
        'completed',
        'failed',
        'cancelled'
    )),

    -- Job payload
    payload JSONB NOT NULL DEFAULT '{}',

    -- Priority (lower number = higher priority)
    priority INTEGER NOT NULL DEFAULT 100,

    -- Retry configuration
    max_retries INTEGER NOT NULL DEFAULT 3,
    retry_count INTEGER NOT NULL DEFAULT 0,
    retry_backoff_seconds INTEGER NOT NULL DEFAULT 60,

    -- Scheduling
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processing_started_at TIMESTAMP WITH TIME ZONE,
    processing_completed_at TIMESTAMP WITH TIME ZONE,

    -- Worker tracking
    worker_id VARCHAR(100), -- ID of worker processing this job

    -- Error information
    error_message TEXT,
    error_details JSONB,

    -- Timeout
    timeout_seconds INTEGER NOT NULL DEFAULT 300,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Job history: completed/failed jobs for analysis
CREATE TABLE job_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    -- Job identification
    job_id UUID NOT NULL, -- Original job ID
    job_type VARCHAR(100) NOT NULL,
    queue VARCHAR(50) NOT NULL,

    -- Final status
    status VARCHAR(20) NOT NULL CHECK (status IN ('completed', 'failed', 'cancelled')),

    -- Job payload
    payload JSONB NOT NULL DEFAULT '{}',

    -- Execution details
    retry_count INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER, -- Processing duration in milliseconds

    -- Error information (if failed)
    error_message TEXT,
    error_details JSONB,

    -- Worker tracking
    worker_id VARCHAR(100),

    -- Timestamps
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    processing_started_at TIMESTAMP WITH TIME ZONE,
    processing_completed_at TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_jobs_tenant_id ON jobs(tenant_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_queue ON jobs(queue, status, priority, scheduled_at)
    WHERE status = 'pending';
CREATE INDEX idx_jobs_type ON jobs(job_type);
CREATE INDEX idx_jobs_scheduled_at ON jobs(scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_jobs_processing ON jobs(worker_id, status)
    WHERE status = 'processing';
CREATE INDEX idx_jobs_failed ON jobs(status, retry_count, max_retries)
    WHERE status = 'failed' AND retry_count < max_retries;

CREATE INDEX idx_job_history_tenant_id ON job_history(tenant_id);
CREATE INDEX idx_job_history_job_id ON job_history(job_id);
CREATE INDEX idx_job_history_job_type ON job_history(job_type);
CREATE INDEX idx_job_history_status ON job_history(status);
CREATE INDEX idx_job_history_created_at ON job_history(created_at);

-- Auto-update trigger
CREATE TRIGGER update_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to move completed/failed jobs to history
CREATE OR REPLACE FUNCTION archive_completed_job()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status IN ('completed', 'failed', 'cancelled') AND OLD.status NOT IN ('completed', 'failed', 'cancelled') THEN
        INSERT INTO job_history (
            job_id,
            tenant_id,
            job_type,
            queue,
            status,
            payload,
            retry_count,
            duration_ms,
            error_message,
            error_details,
            worker_id,
            scheduled_at,
            processing_started_at,
            processing_completed_at,
            metadata,
            created_at
        ) VALUES (
            NEW.id,
            NEW.tenant_id,
            NEW.job_type,
            NEW.queue,
            NEW.status,
            NEW.payload,
            NEW.retry_count,
            CASE
                WHEN NEW.processing_started_at IS NOT NULL AND NEW.processing_completed_at IS NOT NULL
                THEN EXTRACT(EPOCH FROM (NEW.processing_completed_at - NEW.processing_started_at)) * 1000
                ELSE NULL
            END,
            NEW.error_message,
            NEW.error_details,
            NEW.worker_id,
            NEW.scheduled_at,
            NEW.processing_started_at,
            NEW.processing_completed_at,
            NEW.metadata,
            NOW()
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER archive_job_on_completion
    AFTER UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION archive_completed_job();

COMMENT ON TABLE jobs IS 'Background job queue';
COMMENT ON TABLE job_history IS 'Completed and failed jobs for analysis';
COMMENT ON COLUMN jobs.priority IS 'Lower number = higher priority (default 100)';
COMMENT ON COLUMN jobs.queue IS 'Job queue name for parallel processing';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS archive_job_on_completion ON jobs;
DROP FUNCTION IF EXISTS archive_completed_job();
DROP TRIGGER IF EXISTS update_jobs_updated_at ON jobs;
DROP TABLE IF EXISTS job_history CASCADE;
DROP TABLE IF EXISTS jobs CASCADE;
-- +goose StatementEnd
