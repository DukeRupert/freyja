-- name: CreateEmailVerificationToken :one
-- Create a new email verification token
INSERT INTO email_verification_tokens (
    tenant_id,
    user_id,
    token_hash,
    expires_at,
    ip_address,
    user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetEmailVerificationToken :one
-- Get a valid (unused, non-expired) email verification token with user details
SELECT
    evt.id,
    evt.tenant_id,
    evt.user_id,
    evt.token_hash,
    evt.used,
    evt.used_at,
    evt.expires_at,
    evt.created_at,
    u.email as user_email,
    u.first_name as user_first_name,
    u.last_name as user_last_name,
    u.status as user_status,
    u.email_verified as user_email_verified
FROM email_verification_tokens evt
INNER JOIN users u ON evt.user_id = u.id
WHERE evt.tenant_id = $1
  AND evt.token_hash = $2
  AND evt.used = FALSE
  AND evt.expires_at > NOW()
LIMIT 1;

-- name: MarkEmailVerificationTokenUsed :exec
-- Mark an email verification token as used
UPDATE email_verification_tokens
SET
    used = TRUE,
    used_at = NOW()
WHERE tenant_id = $1
  AND token_hash = $2
  AND used = FALSE;

-- name: CountRecentVerificationRequestsByUser :one
-- Count recent email verification requests for a specific user (rate limiting)
SELECT COUNT(*)
FROM email_verification_tokens
WHERE user_id = $1
  AND created_at > $2;

-- name: CountRecentVerificationRequestsByIP :one
-- Count recent email verification requests from a specific IP address (rate limiting)
SELECT COUNT(*)
FROM email_verification_tokens
WHERE ip_address = $1
  AND created_at > $2;

-- name: InvalidateUserEmailVerificationTokens :exec
-- Mark all unused email verification tokens for a user as used
-- (Called after successful email verification to invalidate other tokens)
UPDATE email_verification_tokens
SET
    used = TRUE,
    used_at = NOW()
WHERE tenant_id = $1
  AND user_id = $2
  AND used = FALSE;

-- name: DeleteExpiredEmailVerificationTokens :exec
-- Delete expired email verification tokens (cleanup job)
DELETE FROM email_verification_tokens
WHERE expires_at <= NOW();

-- name: GetPendingVerificationByUser :one
-- Check if user has any pending (unused, non-expired) verification token
SELECT EXISTS(
    SELECT 1
    FROM email_verification_tokens
    WHERE user_id = $1
      AND used = FALSE
      AND expires_at > NOW()
) as has_pending_token;
