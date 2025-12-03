-- name: GetBillingCustomerByUserID :one
-- Get billing customer for a user
SELECT *
FROM billing_customers
WHERE tenant_id = $1
  AND user_id = $2
  AND provider = $3
LIMIT 1;

-- name: CreateBillingCustomer :one
-- Create a new billing customer
INSERT INTO billing_customers (
    tenant_id,
    user_id,
    provider,
    provider_customer_id,
    metadata
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateBillingCustomer :one
-- Update a billing customer
UPDATE billing_customers
SET
    provider_customer_id = $4,
    metadata = $5,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
  AND provider = $3
RETURNING *;

-- name: ListPaymentMethodsForUser :many
-- Get all payment methods for a user
SELECT
    pm.id,
    pm.tenant_id,
    pm.billing_customer_id,
    pm.provider,
    pm.provider_payment_method_id,
    pm.method_type,
    pm.display_brand,
    pm.display_last4,
    pm.display_exp_month,
    pm.display_exp_year,
    pm.is_default,
    pm.created_at,
    pm.updated_at
FROM payment_methods pm
INNER JOIN billing_customers bc ON bc.id = pm.billing_customer_id
WHERE bc.tenant_id = $1
  AND bc.user_id = $2
ORDER BY pm.is_default DESC, pm.created_at DESC;

-- name: GetPaymentMethodByID :one
-- Get a single payment method by ID (validates user ownership)
SELECT
    pm.id,
    pm.tenant_id,
    pm.billing_customer_id,
    pm.provider,
    pm.provider_payment_method_id,
    pm.method_type,
    pm.display_brand,
    pm.display_last4,
    pm.display_exp_month,
    pm.display_exp_year,
    pm.is_default,
    pm.created_at,
    pm.updated_at
FROM payment_methods pm
INNER JOIN billing_customers bc ON bc.id = pm.billing_customer_id
WHERE pm.id = $1
  AND bc.tenant_id = $2
  AND bc.user_id = $3
LIMIT 1;

-- name: GetDefaultPaymentMethod :one
-- Get the default payment method for a user
SELECT
    pm.id,
    pm.tenant_id,
    pm.billing_customer_id,
    pm.provider,
    pm.provider_payment_method_id,
    pm.method_type,
    pm.display_brand,
    pm.display_last4,
    pm.display_exp_month,
    pm.display_exp_year,
    pm.is_default,
    pm.created_at,
    pm.updated_at
FROM payment_methods pm
INNER JOIN billing_customers bc ON bc.id = pm.billing_customer_id
WHERE bc.tenant_id = $1
  AND bc.user_id = $2
  AND pm.is_default = TRUE
LIMIT 1;

-- name: CreatePaymentMethod :one
-- Create a new payment method
INSERT INTO payment_methods (
    tenant_id,
    billing_customer_id,
    provider,
    provider_payment_method_id,
    method_type,
    display_brand,
    display_last4,
    display_exp_month,
    display_exp_year,
    is_default,
    metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: SetDefaultPaymentMethod :exec
-- Set a payment method as the default for a billing customer
UPDATE payment_methods
SET is_default = (id = $2)
WHERE billing_customer_id = $1;

-- name: DeletePaymentMethod :exec
-- Delete a payment method
DELETE FROM payment_methods
WHERE tenant_id = $1
  AND id = $2;

-- name: GetPaymentByProviderID :one
-- Get a payment by provider payment ID
SELECT *
FROM payments
WHERE tenant_id = $1
  AND provider = $2
  AND provider_payment_id = $3
LIMIT 1;

-- name: CreatePayment :one
-- Create a new payment record
INSERT INTO payments (
    tenant_id,
    billing_customer_id,
    provider,
    provider_payment_id,
    amount_cents,
    currency,
    status,
    payment_method_id,
    metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdatePaymentStatus :one
-- Update payment status
UPDATE payments
SET
    status = $3,
    succeeded_at = CASE WHEN $3 = 'succeeded' THEN NOW() ELSE succeeded_at END,
    failed_at = CASE WHEN $3 = 'failed' THEN NOW() ELSE failed_at END,
    failure_code = $4,
    failure_message = $5,
    updated_at = NOW()
WHERE tenant_id = $1
  AND id = $2
RETURNING *;

-- name: CountPaymentMethodsForUser :one
-- Count payment methods for a user (for account dashboard)
SELECT
    COUNT(*) as payment_method_count,
    COALESCE(BOOL_OR(pm.is_default), false) as has_default_payment
FROM payment_methods pm
INNER JOIN billing_customers bc ON bc.id = pm.billing_customer_id
WHERE bc.tenant_id = $1
  AND bc.user_id = $2;
