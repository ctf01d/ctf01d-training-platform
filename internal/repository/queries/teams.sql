-- name: CreateTeam :one
INSERT INTO teams (name, description, website, avatar_url, university_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetTeamByID :one
SELECT * FROM teams WHERE id = $1;

-- name: GetTeamByCaptain :one
SELECT * FROM teams WHERE captain_id = $1;

-- name: ListTeams :many
SELECT * FROM teams ORDER BY id LIMIT $1 OFFSET $2;

-- name: CountTeams :one
SELECT count(*) FROM teams;

-- name: UpdateTeam :one
UPDATE teams SET name = COALESCE($2, name),
    description = COALESCE($3, description),
    website = COALESCE($4, website),
    avatar_url = COALESCE($5, avatar_url),
    university_id = COALESCE($6, university_id),
    updated_at = now()
WHERE id = $1 RETURNING *;

-- name: SetCaptain :one
UPDATE teams SET captain_id = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: ClearCaptain :one
UPDATE teams SET captain_id = NULL, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = $1;
