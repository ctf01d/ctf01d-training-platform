-- name: CreateResult :one
INSERT INTO results (game_id, team_id, score)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetResultByID :one
SELECT * FROM results WHERE id = $1;

-- name: ListResultsByGame :many
SELECT * FROM results WHERE game_id = $1 ORDER BY score DESC;

-- name: ListResultsByTeam :many
SELECT * FROM results WHERE team_id = $1;

-- name: ListResultsByGameAndTeam :many
SELECT * FROM results WHERE game_id = $1 AND team_id = $2;

-- name: ListAllResults :many
SELECT * FROM results ORDER BY id;

-- name: UpsertResult :one
INSERT INTO results (game_id, team_id, score)
VALUES ($1, $2, $3)
ON CONFLICT (game_id, team_id)
DO UPDATE SET score = EXCLUDED.score, updated_at = now()
RETURNING *;

-- name: UpdateResult :one
UPDATE results SET score = COALESCE($2, score), updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteResult :exec
DELETE FROM results WHERE id = $1;
