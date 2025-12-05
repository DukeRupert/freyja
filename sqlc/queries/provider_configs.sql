-- name: CreateProviderConfig :one
-- Creates a new tenant provider configuration.
-- If is_default is true, this will be the default provider for this type.
-- The config_encrypted field should contain base64-encoded AES-256-GCM encrypted JSON.
INSERT INTO tenant_provider_configs (
    tenant_id,
    type,
    name,
    is_active,
    is_default,
    priority,
    config_encrypted
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetProviderConfig :one
-- Retrieves a specific provider configuration by ID.
-- Used to load configuration for a known provider config.
SELECT * FROM tenant_provider_configs
WHERE id = $1 AND tenant_id = $2;

-- name: GetActiveProviderConfigs :many
-- Retrieves all active provider configurations for a tenant and type.
-- Results are ordered by is_default DESC (default first), then priority ASC (lower priority number first).
-- Used by registry to load the best provider for a tenant.
SELECT * FROM tenant_provider_configs
WHERE tenant_id = $1
    AND type = $2
    AND is_active = TRUE
ORDER BY is_default DESC, priority ASC;

-- name: GetDefaultProviderConfig :one
-- Retrieves the default provider configuration for a tenant and type.
-- Returns error if no default is configured.
SELECT * FROM tenant_provider_configs
WHERE tenant_id = $1
    AND type = $2
    AND is_default = TRUE
    AND is_active = TRUE;

-- name: ListProviderConfigs :many
-- Lists all provider configurations for a tenant, optionally filtered by type.
-- Used in admin UI to show all configured providers.
-- If type is empty string, returns all types.
SELECT * FROM tenant_provider_configs
WHERE tenant_id = $1
    AND (sqlc.narg('type')::VARCHAR IS NULL OR type = sqlc.narg('type'))
ORDER BY type, is_default DESC, priority ASC;

-- name: UpdateProviderConfig :one
-- Updates an existing provider configuration.
-- Note: Changing is_default requires handling the previous default.
UPDATE tenant_provider_configs
SET
    name = COALESCE(sqlc.narg('name'), name),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    is_default = COALESCE(sqlc.narg('is_default'), is_default),
    priority = COALESCE(sqlc.narg('priority'), priority),
    config_encrypted = COALESCE(sqlc.narg('config_encrypted'), config_encrypted),
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteProviderConfig :exec
-- Deletes a provider configuration.
-- Cascades to tenant_shipping_rates if this is a shipping provider.
DELETE FROM tenant_provider_configs
WHERE id = $1 AND tenant_id = $2;

-- name: UnsetDefaultProvider :exec
-- Removes is_default flag from all providers of a given type for a tenant.
-- Used before setting a new default provider.
UPDATE tenant_provider_configs
SET is_default = FALSE, updated_at = NOW()
WHERE tenant_id = $1 AND type = $2 AND is_default = TRUE;

-- name: CreateShippingRate :one
-- Creates a new shipping rate (manual or cached from provider).
-- For manual rates, valid_until should be NULL.
-- For provider-cached rates, valid_until should be a future timestamp.
INSERT INTO tenant_shipping_rates (
    tenant_id,
    provider_config_id,
    service_code,
    service_name,
    origin_postal_code,
    destination_postal_code,
    weight_grams,
    rate_cents,
    currency,
    valid_until,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: GetShippingRate :one
-- Retrieves a cached shipping rate for a specific route and weight.
-- Only returns rates that are still valid (valid_until is NULL or in future).
SELECT * FROM tenant_shipping_rates
WHERE tenant_id = $1
    AND destination_postal_code = $2
    AND weight_grams = $3
    AND (valid_until IS NULL OR valid_until > NOW())
ORDER BY created_at DESC
LIMIT 1;

-- name: GetShippingRatesByProvider :many
-- Retrieves all shipping rates for a specific provider config.
-- Used to show manual rates configured for a tenant.
SELECT * FROM tenant_shipping_rates
WHERE tenant_id = $1
    AND provider_config_id = $2
ORDER BY service_code, destination_postal_code, weight_grams;

-- name: DeleteExpiredShippingRates :exec
-- Deletes shipping rates that have expired across ALL tenants.
-- IMPORTANT: This is a system-wide cleanup job. Only call from background worker
-- context, never from tenant-scoped handlers. For tenant-specific cleanup,
-- use DeleteExpiredShippingRatesForTenant instead.
DELETE FROM tenant_shipping_rates
WHERE valid_until IS NOT NULL AND valid_until < NOW();

-- name: DeleteExpiredShippingRatesForTenant :exec
-- Deletes expired shipping rates for a specific tenant.
-- Use this when cleaning up in a tenant-scoped context.
DELETE FROM tenant_shipping_rates
WHERE tenant_id = $1
    AND valid_until IS NOT NULL
    AND valid_until < NOW();

-- name: DeleteShippingRate :exec
-- Deletes a specific shipping rate.
DELETE FROM tenant_shipping_rates
WHERE id = $1 AND tenant_id = $2;

-- name: DeleteShippingRatesByProvider :exec
-- Deletes all shipping rates for a specific provider config.
-- Used when removing a shipping provider configuration.
DELETE FROM tenant_shipping_rates
WHERE tenant_id = $1 AND provider_config_id = $2;
