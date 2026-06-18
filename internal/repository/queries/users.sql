-- name: CreateUser :one
INSERT INTO users (user_name, display_name, role, rating, avatar_url, password_digest)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByUserName :one
SELECT * FROM users
WHERE user_name = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE (
  user_name ILIKE '%' || sqlc.narg('search_query') || '%'
  OR display_name ILIKE '%' || sqlc.narg('search_query') || '%'
  OR sqlc.narg('search_query') IS NULL
)
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT count(*) FROM users
WHERE (
  user_name ILIKE '%' || sqlc.narg('search_query') || '%'
  OR display_name ILIKE '%' || sqlc.narg('search_query') || '%'
  OR sqlc.narg('search_query') IS NULL
);

-- name: UpdateUserProfile :one
UPDATE users
SET display_name = $2,
    avatar_url = COALESCE($3, avatar_url),
    password_digest = COALESCE($4, password_digest),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateUserProfileAdmin :one
UPDATE users
SET display_name = $2,
    avatar_url = COALESCE($3, avatar_url),
    password_digest = COALESCE($4, password_digest),
    bio = $5,
    telegram = $6,
    github = $7,
    email = $8,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserAvatar :one
UPDATE users
SET avatar_url = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserBlocked :one
UPDATE users
SET is_blocked = $2,
    blocked_at = CASE WHEN $2 THEN now() ELSE NULL END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserLastLogin :exec
UPDATE users
SET last_login_ip = $2,
    last_login_at = now()
WHERE id = $1;

-- name: ClearUserTeamCaptaincy :exec
UPDATE teams
SET captain_id = NULL
WHERE captain_id = $1;

-- name: UpdateUserRole :one
UPDATE users
SET role = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateUserRating :one
UPDATE users
SET rating = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
