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
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: UpdateUserProfile :one
UPDATE users
SET display_name = $2,
    avatar_url = COALESCE($3, avatar_url),
    password_digest = COALESCE($4, password_digest),
    updated_at = now()
WHERE id = $1
RETURNING *;

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
