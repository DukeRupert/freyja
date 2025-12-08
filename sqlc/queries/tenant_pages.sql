-- name: GetTenantPage :one
-- Get a single page by tenant and slug
SELECT id, tenant_id, slug, title, content, meta_description, last_updated_label, is_published, created_at, updated_at
FROM tenant_pages
WHERE tenant_id = $1 AND slug = $2;

-- name: GetPublishedTenantPage :one
-- Get a published page by tenant and slug (for storefront)
SELECT id, tenant_id, slug, title, content, meta_description, last_updated_label, is_published, created_at, updated_at
FROM tenant_pages
WHERE tenant_id = $1 AND slug = $2 AND is_published = true;

-- name: ListTenantPages :many
-- List all pages for a tenant (for admin)
SELECT id, tenant_id, slug, title, content, meta_description, last_updated_label, is_published, created_at, updated_at
FROM tenant_pages
WHERE tenant_id = $1
ORDER BY slug;

-- name: CreateTenantPage :one
-- Create a new page
INSERT INTO tenant_pages (
    tenant_id,
    slug,
    title,
    content,
    meta_description,
    last_updated_label,
    is_published
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, tenant_id, slug, title, content, meta_description, last_updated_label, is_published, created_at, updated_at;

-- name: UpdateTenantPage :one
-- Update an existing page
UPDATE tenant_pages
SET
    title = $3,
    content = $4,
    meta_description = $5,
    last_updated_label = $6,
    is_published = $7
WHERE tenant_id = $1 AND slug = $2
RETURNING id, tenant_id, slug, title, content, meta_description, last_updated_label, is_published, created_at, updated_at;

-- name: UpsertTenantPage :one
-- Create or update a page (useful for seeding defaults)
INSERT INTO tenant_pages (
    tenant_id,
    slug,
    title,
    content,
    meta_description,
    last_updated_label,
    is_published
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    title = EXCLUDED.title,
    content = EXCLUDED.content,
    meta_description = EXCLUDED.meta_description,
    last_updated_label = EXCLUDED.last_updated_label,
    is_published = EXCLUDED.is_published
RETURNING id, tenant_id, slug, title, content, meta_description, last_updated_label, is_published, created_at, updated_at;

-- name: DeleteTenantPage :exec
-- Delete a page
DELETE FROM tenant_pages
WHERE tenant_id = $1 AND slug = $2;

-- name: TenantPageExists :one
-- Check if a page exists
SELECT EXISTS(
    SELECT 1 FROM tenant_pages WHERE tenant_id = $1 AND slug = $2
) AS exists;
