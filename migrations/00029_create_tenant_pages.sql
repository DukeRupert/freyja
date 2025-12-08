-- +goose Up
-- +goose StatementBegin

-- Tenant pages table: stores editable content pages (legal, about, etc.)
CREATE TABLE tenant_pages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Page identification
    slug VARCHAR(50) NOT NULL,  -- e.g., 'privacy', 'terms', 'shipping', 'about', 'contact'
    title VARCHAR(255) NOT NULL,

    -- Content (HTML from Tiptap editor)
    content TEXT NOT NULL DEFAULT '',

    -- Metadata
    meta_description VARCHAR(500),
    last_updated_label VARCHAR(50),  -- e.g., "December 2024" for legal pages

    -- Publishing status
    is_published BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Each tenant can only have one page per slug
    UNIQUE (tenant_id, slug)
);

-- Indexes
CREATE INDEX idx_tenant_pages_tenant_id ON tenant_pages(tenant_id);
CREATE INDEX idx_tenant_pages_slug ON tenant_pages(tenant_id, slug) WHERE is_published = true;

-- Auto-update trigger
CREATE TRIGGER update_tenant_pages_updated_at
    BEFORE UPDATE ON tenant_pages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE tenant_pages IS 'Editable content pages per tenant (privacy, terms, shipping, etc.)';
COMMENT ON COLUMN tenant_pages.slug IS 'URL slug for the page (privacy, terms, shipping, about, contact)';
COMMENT ON COLUMN tenant_pages.content IS 'HTML content from Tiptap rich text editor';
COMMENT ON COLUMN tenant_pages.last_updated_label IS 'Human-readable date shown on legal pages (e.g., December 2024)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_tenant_pages_updated_at ON tenant_pages;
DROP TABLE IF EXISTS tenant_pages CASCADE;
-- +goose StatementEnd
