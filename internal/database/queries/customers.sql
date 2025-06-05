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
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetCustomerCount :one
SELECT COUNT(*) FROM customers;

-- name: ListCustomers :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at
FROM customers
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListActiveCustomers :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at
FROM customers
WHERE created_at >= NOW() - INTERVAL '1 year'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: SearchCustomers :many
SELECT id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at
FROM customers
WHERE 
    email ILIKE $1 OR 
    first_name ILIKE $1 OR 
    last_name ILIKE $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

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

-- name: DeleteCustomer :exec
DELETE FROM customers
WHERE id = $1;

-- name: ArchiveCustomer :one
UPDATE customers
SET 
    email = CONCAT('archived_', id, '_', email),
    updated_at = NOW()
WHERE id = $1
RETURNING id, email, first_name, last_name, password_hash, stripe_customer_id, created_at, updated_at;


