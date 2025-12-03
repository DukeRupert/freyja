-- name: CreatePasswordResetToken :one
-- Create a new password reset token
INSERT INTO password_reset_tokens (
    tenant_id,
    user_id,
    token_hash,
    expires_at,
    ip_address,
    user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetPasswordResetToken :one
-- Get a valid (unused, non-expired) password reset token with user details
SELECT
    prt.id,
    prt.tenant_id,
    prt.user_id,
    prt.token_hash,
    prt.used,
    prt.used_at,
    prt.expires_at,
    prt.created_at,
    u.email as user_email,
    u.first_name as user_first_name,
    u.last_name as user_last_name,
    u.status as user_status
FROM password_reset_tokens prt
INNER JOIN users u ON prt.user_id = u.id
WHERE prt.tenant_id = $1
  AND prt.token_hash = $2
  AND prt.used = FALSE
  AND prt.expires_at > NOW()
LIMIT 1;

-- name: MarkPasswordResetTokenUsed :exec
-- Mark a password reset token as used
UPDATE password_reset_tokens
SET
    used = TRUE,
    used_at = NOW()
WHERE tenant_id = $1
  AND token_hash = $2
  AND used = FALSE;

-- name: CountRecentResetRequestsByEmail :one
-- Count recent password reset requests for a specific user (rate limiting)
SELECT COUNT(*)
FROM password_reset_tokens
WHERE user_id = $1
  AND created_at > $2;

-- name: CountRecentResetRequestsByIP :one
-- Count recent password reset requests from a specific IP address (rate limiting)
SELECT COUNT(*)
FROM password_reset_tokens
WHERE ip_address = $1
  AND created_at > $2;

-- name: InvalidateUserPasswordResetTokens :exec
-- Mark all unused password reset tokens for a user as used
-- (Called after successful password reset to invalidate other tokens)
UPDATE password_reset_tokens
SET
    used = TRUE,
    used_at = NOW()
WHERE tenant_id = $1
  AND user_id = $2
  AND used = FALSE;

-- name: DeleteExpiredPasswordResetTokens :exec
-- Delete expired password reset tokens (cleanup job)
DELETE FROM password_reset_tokens
WHERE expires_at <= NOW();
