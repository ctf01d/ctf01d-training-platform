-- name: CreateSession :one
INSERT INTO user_sessions (user_id, jti, ip_address, user_agent, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSessionByJTI :one
SELECT * FROM user_sessions
WHERE jti = $1;

-- name: GetSessionForAuth :one
-- Single read used on every authenticated request: session validity plus the
-- owner's blocked flag, avoiding a second query/lookup.
SELECT s.revoked_at, s.expires_at, s.last_seen_at, u.is_blocked
FROM user_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.jti = $1;

-- name: ListActiveSessionsByUser :many
SELECT * FROM user_sessions
WHERE user_id = $1
  AND revoked_at IS NULL
  AND expires_at > now()
ORDER BY last_seen_at DESC;

-- name: TouchSession :exec
UPDATE user_sessions
SET last_seen_at = now(),
    ip_address = COALESCE($2, ip_address)
WHERE jti = $1;

-- name: RevokeSession :exec
UPDATE user_sessions
SET revoked_at = now()
WHERE jti = $1 AND revoked_at IS NULL;

-- name: RevokeSessionByID :exec
UPDATE user_sessions
SET revoked_at = now()
WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :exec
UPDATE user_sessions
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredSessions :exec
DELETE FROM user_sessions
WHERE expires_at < now();
