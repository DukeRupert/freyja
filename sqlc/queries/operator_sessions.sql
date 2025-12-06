-- Operator Sessions: Sessions for tenant operators (separate from customer sessions)

-- name: CreateOperatorSession :one
-- Create a new operator session
INSERT INTO operator_sessions (
    operator_id,
    token_hash,
    user_agent,
    ip_address,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetOperatorSessionByTokenHash :one
-- Get a valid (non-expired) operator session by token hash
SELECT *
FROM operator_sessions
WHERE token_hash = $1
  AND expires_at > NOW()
LIMIT 1;

-- name: DeleteOperatorSession :exec
-- Delete an operator session (logout)
DELETE FROM operator_sessions
WHERE token_hash = $1;

-- name: DeleteOperatorSessionByID :exec
-- Delete an operator session by ID
DELETE FROM operator_sessions
WHERE id = $1;

-- name: DeleteOperatorSessionsByOperatorID :exec
-- Delete all sessions for an operator (e.g., password change, force logout)
DELETE FROM operator_sessions
WHERE operator_id = $1;

-- name: DeleteExpiredOperatorSessions :exec
-- Clean up expired operator sessions (background job)
DELETE FROM operator_sessions
WHERE expires_at <= NOW();

-- name: GetOperatorSessionsForOperator :many
-- Get all active sessions for an operator (for "active sessions" UI)
SELECT *
FROM operator_sessions
WHERE operator_id = $1
  AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: CountActiveOperatorSessions :one
-- Count active sessions for an operator
SELECT COUNT(*)
FROM operator_sessions
WHERE operator_id = $1
  AND expires_at > NOW();

-- name: UpdateOperatorSessionExpiry :exec
-- Update session expiry (for sliding window sessions)
UPDATE operator_sessions
SET expires_at = $2
WHERE id = $1;
