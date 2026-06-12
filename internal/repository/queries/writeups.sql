-- name: CreateWriteup :one
INSERT INTO writeups (game_id, team_id, title, url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetWriteupByID :one
SELECT * FROM writeups WHERE id = $1;

-- name: ListWriteupsByGame :many
SELECT * FROM writeups WHERE game_id = $1 ORDER BY created_at DESC, id DESC;

-- name: ListWriteupsByTeam :many
SELECT * FROM writeups WHERE team_id = $1 ORDER BY created_at DESC, id DESC;

-- name: ListWriteupsByGameAndTeam :many
SELECT * FROM writeups WHERE game_id = $1 AND team_id = $2 ORDER BY created_at DESC, id DESC;

-- name: ListAllWriteups :many
SELECT * FROM writeups ORDER BY created_at DESC, id DESC;

-- name: DeleteWriteup :exec
DELETE FROM writeups WHERE id = $1;
