-- Subscription queries for the SubscriptionService

-- name: CreateSubscription :one
-- Creates a new subscription record
-- Returns the complete subscription with generated ID and timestamps
INSERT INTO subscriptions (
    tenant_id,
    user_id,
    subscription_plan_id,
    billing_interval,
    status,
    billing_customer_id,
    provider,
    provider_subscription_id,
    subtotal_cents,
    tax_cents,
    total_cents,
    currency,
    shipping_address_id,
    shipping_method_id,
    shipping_cents,
    payment_method_id,
    current_period_start,
    current_period_end,
    next_billing_date,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
)
RETURNING *;

-- name: CreateSubscriptionItem :one
-- Creates a subscription item (product in subscription)
-- Captures pricing at time of subscription creation
INSERT INTO subscription_items (
    tenant_id,
    subscription_id,
    product_sku_id,
    quantity,
    unit_price_cents,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetSubscriptionByID :one
-- Retrieves subscription by database ID with tenant scoping
SELECT * FROM subscriptions
WHERE id = $1 AND tenant_id = $2;

-- name: GetSubscriptionByProviderID :one
-- Retrieves subscription by Stripe subscription ID
-- Used for webhook processing to find local subscription
SELECT * FROM subscriptions
WHERE provider_subscription_id = $1
  AND provider = $2
  AND tenant_id = $3;

-- name: GetSubscriptionWithDetails :one
-- Retrieves subscription with joined user, address, and payment method details
-- Used for displaying subscription information to customers
SELECT
    s.*,
    u.email as user_email,
    u.first_name,
    u.last_name,
    a.full_name as shipping_full_name,
    a.company as shipping_company,
    a.address_line1 as shipping_address_line1,
    a.address_line2 as shipping_address_line2,
    a.city as shipping_city,
    a.state as shipping_state,
    a.postal_code as shipping_postal_code,
    a.country as shipping_country,
    a.phone as shipping_phone,
    pm.method_type as payment_method_type,
    pm.display_brand as payment_display_brand,
    pm.display_last4 as payment_display_last4,
    pm.display_exp_month as payment_display_exp_month,
    pm.display_exp_year as payment_display_exp_year,
    bc.provider_customer_id
FROM subscriptions s
JOIN users u ON s.user_id = u.id
JOIN addresses a ON s.shipping_address_id = a.id
LEFT JOIN payment_methods pm ON s.payment_method_id = pm.id
LEFT JOIN billing_customers bc ON s.billing_customer_id = bc.id
WHERE s.id = $1 AND s.tenant_id = $2;

-- name: ListSubscriptionsForUser :many
-- Lists all subscriptions for a customer with pagination
-- Returns newest first
SELECT * FROM subscriptions
WHERE user_id = $1
  AND tenant_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListActiveSubscriptionsForUser :many
-- Lists only active/trial subscriptions for a customer
-- Used for checking if user has active subscriptions
SELECT * FROM subscriptions
WHERE user_id = $1
  AND tenant_id = $2
  AND status IN ('active', 'trial')
ORDER BY created_at DESC;

-- name: ListSubscriptionsByStatus :many
-- Lists subscriptions filtered by status with pagination
-- Used for admin filtering
SELECT * FROM subscriptions
WHERE tenant_id = $1
  AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountSubscriptionsByStatus :one
-- Counts subscriptions by status for pagination
SELECT COUNT(*) FROM subscriptions
WHERE tenant_id = $1
  AND status = $2;

-- name: ListSubscriptionItemsForSubscription :many
-- Lists all items in a subscription with product details
-- Includes product name, SKU, and image for display
SELECT
    si.*,
    p.name as product_name,
    ps.sku,
    ps.weight_value,
    ps.weight_unit,
    ps.grind,
    pi.url as product_image_url
FROM subscription_items si
JOIN product_skus ps ON si.product_sku_id = ps.id
JOIN products p ON ps.product_id = p.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = true
WHERE si.subscription_id = $1
  AND si.tenant_id = $2
ORDER BY si.created_at;

-- name: UpdateSubscriptionStatus :one
-- Updates subscription status and related timestamps
-- Used when syncing from Stripe webhooks
UPDATE subscriptions
SET
    status = $3,
    current_period_start = COALESCE(sqlc.narg('current_period_start'), current_period_start),
    current_period_end = COALESCE(sqlc.narg('current_period_end'), current_period_end),
    next_billing_date = COALESCE(sqlc.narg('next_billing_date'), next_billing_date),
    cancel_at_period_end = COALESCE(sqlc.narg('cancel_at_period_end'), cancel_at_period_end),
    cancelled_at = COALESCE(sqlc.narg('cancelled_at'), cancelled_at),
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: UpdateSubscriptionProviderID :one
-- Updates subscription with Stripe subscription ID after creation
-- Called after Stripe subscription is created
UPDATE subscriptions
SET
    provider_subscription_id = $3,
    status = $4,
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: UpdateSubscriptionCancellation :one
-- Marks subscription as cancelled or scheduled for cancellation
UPDATE subscriptions
SET
    cancel_at_period_end = $3,
    cancelled_at = CASE WHEN $3 = false THEN NOW() ELSE cancelled_at END,
    cancellation_reason = COALESCE(sqlc.narg('cancellation_reason'), cancellation_reason),
    status = CASE WHEN $3 = false THEN 'cancelled' ELSE status END,
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: UpdateSubscriptionPauseResume :one
-- Updates subscription status for pause/resume operations
UPDATE subscriptions
SET
    status = $3,
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- Billing customer queries

-- name: GetBillingCustomerForUser :one
-- Retrieves billing customer record for user
-- Used to get Stripe customer ID for subscription creation
SELECT * FROM billing_customers
WHERE user_id = $1
  AND tenant_id = $2
  AND provider = $3;

-- name: GetBillingCustomerByProviderID :one
-- Retrieves billing customer by Stripe customer ID
-- Used for webhook processing
SELECT * FROM billing_customers
WHERE provider_customer_id = $1
  AND provider = $2
  AND tenant_id = $3;

-- Payment method queries

-- name: GetDefaultPaymentMethodForUser :one
-- Retrieves user's default payment method
-- Required for subscription creation
SELECT pm.* FROM payment_methods pm
JOIN billing_customers bc ON pm.billing_customer_id = bc.id
WHERE bc.user_id = $1
  AND bc.tenant_id = $2
  AND bc.provider = $3
  AND pm.is_default = true;

-- Subscription schedule queries

-- name: CreateSubscriptionScheduleEvent :one
-- Records a subscription schedule event (billing, pause, resume, cancel, etc.)
INSERT INTO subscription_schedule (
    tenant_id,
    subscription_id,
    event_type,
    status,
    scheduled_at,
    order_id,
    payment_id,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpdateSubscriptionScheduleEvent :one
-- Updates subscription schedule event status after processing
UPDATE subscription_schedule
SET
    status = $3,
    processed_at = COALESCE(sqlc.narg('processed_at'), processed_at),
    failed_at = COALESCE(sqlc.narg('failed_at'), failed_at),
    error_message = COALESCE(sqlc.narg('error_message'), error_message),
    retry_count = COALESCE(sqlc.narg('retry_count'), retry_count),
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: GetSubscriptionScheduleEventByInvoiceID :one
-- Checks if an invoice has already been processed (idempotency)
-- Invoice ID is stored in metadata->>'invoice_id'
SELECT * FROM subscription_schedule
WHERE tenant_id = $1
  AND subscription_id = $2
  AND event_type = 'billing'
  AND metadata->>'invoice_id' = $3
LIMIT 1;

-- name: ListUpcomingScheduleEvents :many
-- Lists upcoming scheduled events for processing
-- Used by background job to process subscription renewals
SELECT
    ss.*,
    s.user_id,
    s.provider_subscription_id
FROM subscription_schedule ss
JOIN subscriptions s ON ss.subscription_id = s.id
WHERE ss.tenant_id = $1
  AND ss.status = 'scheduled'
  AND ss.scheduled_at <= $2
ORDER BY ss.scheduled_at ASC
LIMIT $3;

-- Admin queries

-- name: ListSubscriptions :many
-- List all subscriptions for admin with pagination
SELECT
    s.id,
    s.tenant_id,
    s.status,
    s.billing_interval,
    s.total_cents,
    s.currency,
    s.next_billing_date,
    s.cancel_at_period_end,
    s.created_at,
    s.updated_at,
    u.email as customer_email,
    CONCAT(u.first_name, ' ', u.last_name)::TEXT as customer_name
FROM subscriptions s
LEFT JOIN users u ON u.id = s.user_id
WHERE s.tenant_id = $1
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSubscriptions :one
-- Count total subscriptions for pagination
SELECT COUNT(*)
FROM subscriptions
WHERE tenant_id = $1;

-- name: GetSubscriptionStats :one
-- Get subscription statistics for dashboard
SELECT
    COUNT(*) as total_subscriptions,
    COUNT(*) FILTER (WHERE status = 'active') as active_subscriptions,
    COUNT(*) FILTER (WHERE status = 'paused') as paused_subscriptions,
    COUNT(*) FILTER (WHERE status = 'past_due') as past_due_subscriptions,
    COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled_subscriptions,
    COALESCE(SUM(total_cents) FILTER (WHERE status = 'active'), 0) as monthly_recurring_revenue_cents
FROM subscriptions
WHERE tenant_id = $1;

-- Subscription summary for customer account page

-- name: GetSubscriptionCountsForUser :one
-- Get subscription counts by status for a user (for account dashboard)
SELECT
    COUNT(*) as total_count,
    COUNT(*) FILTER (WHERE status = 'active') as active_count,
    COUNT(*) FILTER (WHERE status = 'paused') as paused_count,
    COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled_count
FROM subscriptions
WHERE user_id = $1
  AND tenant_id = $2;

-- name: GetSubscriptionSummariesForUser :many
-- Get subscription summaries with primary product info for customer display
SELECT
    s.id,
    s.status,
    s.billing_interval,
    s.total_cents,
    s.currency,
    s.next_billing_date,
    s.cancel_at_period_end,
    s.created_at,
    p.name as product_name,
    pi.url as product_image_url
FROM subscriptions s
JOIN subscription_items si ON si.subscription_id = s.id
JOIN product_skus ps ON si.product_sku_id = ps.id
JOIN products p ON ps.product_id = p.id
LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = true
WHERE s.user_id = $1
  AND s.tenant_id = $2
ORDER BY s.created_at DESC;

-- Webhook event queries for subscription idempotency

-- name: GetWebhookEventByProviderID :one
-- Check if webhook event was already processed
SELECT * FROM webhook_events
WHERE provider_event_id = $1
  AND provider = $2
  AND tenant_id = $3;

-- name: CreateWebhookEvent :one
-- Record incoming webhook event for idempotency
INSERT INTO webhook_events (
    tenant_id,
    provider,
    provider_event_id,
    event_type,
    status,
    payload
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (provider, provider_event_id) DO NOTHING
RETURNING *;

-- name: UpdateWebhookEventStatus :exec
-- Update webhook event status after processing
UPDATE webhook_events
SET
    status = $3,
    processed_at = CASE WHEN $3 = 'processed' THEN NOW() ELSE processed_at END,
    error_message = sqlc.narg('error_message'),
    retry_count = COALESCE(sqlc.narg('retry_count'), retry_count),
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2;
