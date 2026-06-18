-- name: CreateUniversity :one
INSERT INTO universities (name, site_url, avatar_url)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUniversityByID :one
SELECT * FROM universities WHERE id = $1;

-- name: ListUniversities :many
SELECT * FROM universities
WHERE (name ILIKE '%' || sqlc.narg('search_query') || '%' OR sqlc.narg('search_query') IS NULL)
ORDER BY id LIMIT $1 OFFSET $2;

-- name: CountUniversities :one
SELECT count(*) FROM universities
WHERE (name ILIKE '%' || sqlc.narg('search_query') || '%' OR sqlc.narg('search_query') IS NULL);

-- name: UpdateUniversity :one
UPDATE universities SET name = COALESCE($2, name),
    site_url = COALESCE($3, site_url),
    avatar_url = COALESCE($4, avatar_url),
    updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteUniversity :exec
DELETE FROM universities WHERE id = $1;
