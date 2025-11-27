-- name: CreateSession :one
-- Create a new session
INSERT INTO sessions (
    token,
    data,
    expires_at
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetSessionByToken :one
-- Get session by token
SELECT
    id,
    token,
    data,
    expires_at,
    created_at,
    updated_at
FROM sessions
WHERE token = $1
  AND expires_at > NOW()
LIMIT 1;

-- name: UpdateSessionData :exec
-- Update session data and extend expiration
UPDATE sessions
SET
    data = $2,
    expires_at = $3,
    updated_at = NOW()
WHERE token = $1;

-- name: DeleteSession :exec
-- Delete a session
DELETE FROM sessions
WHERE token = $1;

-- name: DeleteExpiredSessions :exec
-- Clean up expired sessions (for background job)
DELETE FROM sessions
WHERE expires_at <= NOW();
