-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- Onboarding Checklist: Track tenant setup progress
-- ============================================================================
-- This migration adds:
-- 1. A table to store explicit "skip" actions for optional checklist items
-- 2. A setup_completed_at column to track when operator finished initial setup
--
-- The onboarding checklist status is computed dynamically from actual data
-- (products, provider configs, etc.) rather than stored explicitly.
-- Only skip flags are stored in the database.
-- ============================================================================

-- Step 1: Add setup completion tracking to tenant_operators
-- This tracks when the operator finished the initial password setup
ALTER TABLE tenant_operators
    ADD COLUMN IF NOT EXISTS setup_completed_at TIMESTAMPTZ;

COMMENT ON COLUMN tenant_operators.setup_completed_at IS 'When operator completed initial password setup via email link';

-- Step 2: Create table for storing skipped checklist items
-- Only optional items can be skipped (Phase 2, 3, 4 items)
CREATE TABLE onboarding_item_skips (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Checklist item identifier (must match step_id from ONBOARDING.md)
    -- Phase 2 (optional): business_info, warehouse_address, product_images, coffee_attributes
    -- Phase 3 (wholesale): wholesale_pricing, payment_terms
    item_id VARCHAR(50) NOT NULL,

    -- Metadata
    skipped_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    skipped_by UUID REFERENCES tenant_operators(id) ON DELETE SET NULL,

    -- Each tenant can only skip an item once
    CONSTRAINT onboarding_item_skips_unique UNIQUE (tenant_id, item_id)
);

-- Index for fast lookup during checklist computation
CREATE INDEX idx_onboarding_item_skips_tenant_id
    ON onboarding_item_skips(tenant_id);

COMMENT ON TABLE onboarding_item_skips IS 'Stores explicit skip flags for optional onboarding steps';
COMMENT ON COLUMN onboarding_item_skips.item_id IS 'Step ID from ONBOARDING.md (e.g., business_info, product_images)';
COMMENT ON COLUMN onboarding_item_skips.skipped_by IS 'Operator who skipped this item (for audit)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_onboarding_item_skips_tenant_id;
DROP TABLE IF EXISTS onboarding_item_skips CASCADE;
ALTER TABLE tenant_operators DROP COLUMN IF EXISTS setup_completed_at;

-- +goose StatementEnd
