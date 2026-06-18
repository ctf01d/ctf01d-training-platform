-- name: CreateGame :one
INSERT INTO games (name, organizer, starts_at, ends_at, avatar_url, site_url, ctftime_url,
    finalized, finalized_at, registration_opens_at, registration_closes_at,
    scoreboard_opens_at, scoreboard_closes_at, vpn_url, vpn_config_url,
    access_instructions, access_secret)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING *;

-- name: GetGameByID :one
SELECT * FROM games WHERE id = $1;

-- name: ListGames :many
SELECT * FROM games
WHERE (name ILIKE '%' || sqlc.narg('search_query') || '%' OR sqlc.narg('search_query') IS NULL)
ORDER BY starts_at DESC NULLS LAST, created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: CountGames :one
SELECT count(*) FROM games
WHERE (name ILIKE '%' || sqlc.narg('search_query') || '%' OR sqlc.narg('search_query') IS NULL);

-- name: UpdateGame :one
UPDATE games SET
    name = COALESCE($2, name),
    organizer = COALESCE($3, organizer),
    starts_at = COALESCE($4, starts_at),
    ends_at = COALESCE($5, ends_at),
    avatar_url = COALESCE($6, avatar_url),
    site_url = COALESCE($7, site_url),
    ctftime_url = COALESCE($8, ctftime_url),
    registration_opens_at = COALESCE($9, registration_opens_at),
    registration_closes_at = COALESCE($10, registration_closes_at),
    scoreboard_opens_at = COALESCE($11, scoreboard_opens_at),
    scoreboard_closes_at = COALESCE($12, scoreboard_closes_at),
    vpn_url = COALESCE($13, vpn_url),
    vpn_config_url = COALESCE($14, vpn_config_url),
    access_instructions = COALESCE($15, access_instructions),
    access_secret = COALESCE($16, access_secret),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteGame :exec
DELETE FROM games WHERE id = $1;

-- name: SetFinalized :one
UPDATE games SET finalized = $2, finalized_at = $3, updated_at = now()
WHERE id = $1
RETURNING *;
