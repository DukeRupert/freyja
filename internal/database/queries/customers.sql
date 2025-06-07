-- internal/database/queries/customers.sql

-- name: CreateCustomer :one
INSERT INTO customers (
  email, first_name, last_name, password_hash
) VALUES (
  $1, $2, $3, $4
)
RETURNING id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at;

-- name: GetCustomer :one
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at, archived_at
FROM customers
WHERE id = $1;

-- name: GetCustomerByEmail :one
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at, archived_at
FROM customers
WHERE LOWER(email) = LOWER($1);

-- name: GetCustomerByStripeID :one
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at, archived_at
FROM customers
WHERE stripe_customer_id = $1;

-- name: GetCustomersWithoutStripeID :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at, archived_at
FROM customers
WHERE stripe_customer_id IS NULL
  AND archived_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetCustomerCount :one
SELECT COUNT(*) FROM customers
WHERE archived_at IS NULL;

-- name: GetCustomerCountWithStripeID :one
SELECT COUNT(*) FROM customers
WHERE stripe_customer_id IS NOT NULL
  AND archived_at IS NULL;

-- name: ListCustomers :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at
FROM customers
WHERE archived_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListActiveCustomers :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at
FROM customers
WHERE created_at >= NOW() - INTERVAL '1 year'
  AND archived_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetRecentCustomers :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at
FROM customers
WHERE archived_at IS NULL
ORDER BY created_at DESC
LIMIT $1;

-- name: SearchCustomers :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at
FROM customers
WHERE
    archived_at IS NULL
    AND (
        email ILIKE $1 OR
        first_name ILIKE $1 OR
        last_name ILIKE $1
    )
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: SearchCustomersByEmail :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at
FROM customers
WHERE
    archived_at IS NULL
    AND email ILIKE $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetCustomersWithOrderStats :many
SELECT
    c.id as customer_id,
    c.email,
    c.first_name,
    c.last_name,
    COALESCE(COUNT(o.id), 0)::bigint as order_count,
    COALESCE(SUM(o.total), 0)::bigint as total_spent,
    MAX(o.created_at) as last_order_at
FROM customers c
LEFT JOIN orders o ON c.id = o.customer_id
    AND o.status IN ('confirmed', 'processing', 'shipped', 'delivered')
WHERE c.archived_at IS NULL
GROUP BY c.id, c.email, c.first_name, c.last_name
HAVING COUNT(o.id) > 0
ORDER BY total_spent DESC, order_count DESC
LIMIT $1;

-- name: GetCustomerOrderStats :one
SELECT
    COALESCE(COUNT(o.id), 0)::bigint as order_count,
    COALESCE(SUM(o.total), 0)::bigint as total_spent,
    MAX(o.created_at) as last_order_at
FROM orders o
WHERE o.customer_id = $1
    AND o.status IN ('confirmed', 'processing', 'shipped', 'delivered');

-- name: UpdateCustomer :one
UPDATE customers
SET
  email = $2,
  first_name = $3,
  last_name = $4,
  updated_at = NOW()
WHERE id = $1
RETURNING id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at;

-- name: UpdateCustomerPassword :one
UPDATE customers
SET
  password_hash = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at;

-- name: UpdateCustomerStripeID :one
UPDATE customers
SET
  stripe_customer_id = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at;

-- name: ArchiveCustomer :one
UPDATE customers
SET
    archived_at = NOW(),
    email = CONCAT('archived_', id, '_', email),
    updated_at = NOW()
WHERE id = $1
RETURNING id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at;

-- name: DeleteCustomer :exec
DELETE FROM customers
WHERE id = $1;

-- name: GetArchivedCustomers :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at, archived_at
FROM customers
WHERE archived_at IS NOT NULL
ORDER BY archived_at DESC
LIMIT $1 OFFSET $2;

-- name: RestoreCustomer :one
UPDATE customers
SET
    archived_at = NULL,
    email = REPLACE(email, CONCAT('archived_', id, '_'), ''),
    updated_at = NOW()
WHERE id = $1
RETURNING id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at;
