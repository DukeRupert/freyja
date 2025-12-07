-- ============================================================================
-- ONBOARDING SKIP TRACKING
-- ============================================================================

-- name: GetSkippedItems :many
-- Get all skipped items for a tenant
SELECT id, tenant_id, item_id, skipped_at, skipped_by
FROM onboarding_item_skips
WHERE tenant_id = $1;

-- name: IsItemSkipped :one
-- Check if a specific item is skipped
SELECT EXISTS(
    SELECT 1 FROM onboarding_item_skips
    WHERE tenant_id = $1 AND item_id = $2
) as is_skipped;

-- name: SkipItem :one
-- Mark an item as skipped (idempotent - updates timestamp if already skipped)
INSERT INTO onboarding_item_skips (tenant_id, item_id, skipped_by)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id, item_id) DO UPDATE SET
    skipped_at = NOW(),
    skipped_by = EXCLUDED.skipped_by
RETURNING *;

-- name: UnskipItem :exec
-- Remove skip flag (if user wants to complete the item)
DELETE FROM onboarding_item_skips
WHERE tenant_id = $1 AND item_id = $2;

-- ============================================================================
-- ONBOARDING VALIDATION QUERIES
-- ============================================================================
-- These queries check actual data to determine completion status.
-- Each returns a single boolean indicating if the step is complete.

-- name: CheckAccountActivated :one
-- Validation: account_activated
-- True if any operator for this tenant is active (has set password and can log in)
SELECT EXISTS(
    SELECT 1 FROM tenant_operators
    WHERE tenant_id = $1
      AND status = 'active'
      AND password_hash IS NOT NULL
) as is_complete;

-- name: CheckStripeConnected :one
-- Validation: stripe_connected
-- True if Stripe billing provider is configured and active
SELECT EXISTS(
    SELECT 1 FROM tenant_provider_configs
    WHERE tenant_id = $1
      AND type = 'billing'
      AND name = 'stripe'
      AND is_active = true
      AND config_encrypted IS NOT NULL
) as is_complete;

-- name: CheckEmailConfigured :one
-- Validation: email_configured
-- True if any email provider is configured and active
SELECT EXISTS(
    SELECT 1 FROM tenant_provider_configs
    WHERE tenant_id = $1
      AND type = 'email'
      AND is_active = true
) as is_complete;

-- name: CheckFirstProduct :one
-- Validation: first_product
-- True if at least one active product exists
SELECT EXISTS(
    SELECT 1 FROM products
    WHERE tenant_id = $1
      AND status = 'active'
) as is_complete;

-- name: CheckFirstSKU :one
-- Validation: first_sku
-- True if at least one SKU exists for an active product
SELECT EXISTS(
    SELECT 1 FROM product_skus ps
    JOIN products p ON p.id = ps.product_id
    WHERE p.tenant_id = $1
      AND p.status = 'active'
) as is_complete;

-- name: CheckPricingSet :one
-- Validation: pricing_set
-- True if all SKUs of active products have price list entries in the default price list
-- Returns true if there are no active products (vacuous truth - nothing to price)
SELECT NOT EXISTS(
    SELECT 1 FROM product_skus ps
    JOIN products p ON p.id = ps.product_id
    JOIN price_lists pl ON pl.tenant_id = p.tenant_id AND pl.list_type = 'default'
    LEFT JOIN price_list_entries ple ON ple.price_list_id = pl.id AND ple.product_sku_id = ps.id
    WHERE p.tenant_id = $1
      AND p.status = 'active'
      AND ple.id IS NULL
) as is_complete;

-- name: CheckShippingConfigured :one
-- Validation: shipping_configured
-- True if at least one active shipping method exists
SELECT EXISTS(
    SELECT 1 FROM shipping_methods
    WHERE tenant_id = $1
      AND is_active = true
) as is_complete;

-- name: CheckTaxConfigured :one
-- Validation: tax_configured
-- True if any tax provider is configured and active (including 'no_tax')
SELECT EXISTS(
    SELECT 1 FROM tenant_provider_configs
    WHERE tenant_id = $1
      AND type = 'tax'
      AND is_active = true
) as is_complete;

-- name: CheckBusinessInfo :one
-- Validation: business_info
-- True if tenant has name and email set
SELECT EXISTS(
    SELECT 1 FROM tenants
    WHERE id = $1
      AND name IS NOT NULL
      AND name != ''
      AND email IS NOT NULL
      AND email != ''
) as is_complete;

-- name: CheckWarehouseAddress :one
-- Validation: warehouse_address
-- True if a warehouse address exists
SELECT EXISTS(
    SELECT 1 FROM addresses
    WHERE tenant_id = $1
      AND address_type = 'warehouse'
) as is_complete;

-- name: CheckProductImages :one
-- Validation: product_images
-- True if all active products have at least one image
-- Returns true if there are no active products (vacuous truth)
SELECT NOT EXISTS(
    SELECT p.id FROM products p
    LEFT JOIN product_images pi ON pi.product_id = p.id
    WHERE p.tenant_id = $1
      AND p.status = 'active'
    GROUP BY p.id
    HAVING COUNT(pi.id) = 0
) as is_complete;

-- name: CheckCoffeeAttributes :one
-- Validation: coffee_attributes
-- True if all active products have origin and roast_level set
-- Returns true if there are no active products (vacuous truth)
SELECT NOT EXISTS(
    SELECT 1 FROM products
    WHERE tenant_id = $1
      AND status = 'active'
      AND (origin IS NULL OR origin = '' OR roast_level IS NULL OR roast_level = '')
) as is_complete;

-- name: CheckWholesalePricing :one
-- Validation: wholesale_pricing
-- True if at least one wholesale price list with entries exists
SELECT EXISTS(
    SELECT 1 FROM price_lists pl
    JOIN price_list_entries ple ON ple.price_list_id = pl.id
    WHERE pl.tenant_id = $1
      AND pl.list_type = 'wholesale'
) as is_complete;

-- name: CheckPaymentTerms :one
-- Validation: payment_terms
-- True if at least one payment term is marked as default
SELECT EXISTS(
    SELECT 1 FROM payment_terms
    WHERE tenant_id = $1
      AND is_default = true
) as is_complete;

-- ============================================================================
-- COMBINED VALIDATION QUERY (PERFORMANCE OPTIMIZATION)
-- ============================================================================
-- Single query that returns all validation results at once.
-- Use this instead of 14 separate queries for better performance.
-- Uses a CTE to anchor the tenant_id parameter and avoid ambiguous references.

-- name: GetAllOnboardingValidations :one
WITH tenant_param AS (
    SELECT $1::uuid AS tid
)
SELECT
    -- Phase 1: Critical Path (Required)
    EXISTS(
        SELECT 1 FROM tenant_operators, tenant_param tp
        WHERE tenant_operators.tenant_id = tp.tid AND status = 'active' AND password_hash IS NOT NULL
    ) as account_activated,

    EXISTS(
        SELECT 1 FROM tenant_provider_configs, tenant_param tp
        WHERE tenant_provider_configs.tenant_id = tp.tid AND type = 'billing' AND name = 'stripe' AND is_active = true
    ) as stripe_connected,

    EXISTS(
        SELECT 1 FROM tenant_provider_configs, tenant_param tp
        WHERE tenant_provider_configs.tenant_id = tp.tid AND type = 'email' AND is_active = true
    ) as email_configured,

    EXISTS(
        SELECT 1 FROM products, tenant_param tp
        WHERE products.tenant_id = tp.tid AND status = 'active'
    ) as first_product,

    EXISTS(
        SELECT 1 FROM product_skus ps
        JOIN products p ON p.id = ps.product_id
        CROSS JOIN tenant_param tp
        WHERE p.tenant_id = tp.tid AND p.status = 'active'
    ) as first_sku,

    NOT EXISTS(
        SELECT 1 FROM product_skus ps
        JOIN products p ON p.id = ps.product_id
        JOIN price_lists pl ON pl.tenant_id = p.tenant_id AND pl.list_type = 'default'
        LEFT JOIN price_list_entries ple ON ple.price_list_id = pl.id AND ple.product_sku_id = ps.id
        CROSS JOIN tenant_param tp
        WHERE p.tenant_id = tp.tid AND p.status = 'active' AND ple.id IS NULL
    ) as pricing_set,

    EXISTS(
        SELECT 1 FROM shipping_methods, tenant_param tp
        WHERE shipping_methods.tenant_id = tp.tid AND is_active = true
    ) as shipping_configured,

    EXISTS(
        SELECT 1 FROM tenant_provider_configs, tenant_param tp
        WHERE tenant_provider_configs.tenant_id = tp.tid AND type = 'tax' AND is_active = true
    ) as tax_configured,

    -- Phase 2: Recommended (Optional)
    EXISTS(
        SELECT 1 FROM tenants, tenant_param tp
        WHERE tenants.id = tp.tid AND name IS NOT NULL AND name != '' AND email IS NOT NULL AND email != ''
    ) as business_info,

    EXISTS(
        SELECT 1 FROM addresses, tenant_param tp
        WHERE addresses.tenant_id = tp.tid AND address_type = 'warehouse'
    ) as warehouse_address,

    NOT EXISTS(
        SELECT p.id FROM products p
        CROSS JOIN tenant_param tp
        LEFT JOIN product_images pi ON pi.product_id = p.id
        WHERE p.tenant_id = tp.tid AND p.status = 'active'
        GROUP BY p.id HAVING COUNT(pi.id) = 0
    ) as product_images,

    NOT EXISTS(
        SELECT 1 FROM products, tenant_param tp
        WHERE products.tenant_id = tp.tid AND status = 'active'
        AND (origin IS NULL OR origin = '' OR roast_level IS NULL OR roast_level = '')
    ) as coffee_attributes,

    -- Phase 3: Wholesale (Optional)
    EXISTS(
        SELECT 1 FROM price_lists pl
        JOIN price_list_entries ple ON ple.price_list_id = pl.id
        CROSS JOIN tenant_param tp
        WHERE pl.tenant_id = tp.tid AND pl.list_type = 'wholesale'
    ) as wholesale_pricing,

    EXISTS(
        SELECT 1 FROM payment_terms, tenant_param tp
        WHERE payment_terms.tenant_id = tp.tid AND is_default = true
    ) as payment_terms;
