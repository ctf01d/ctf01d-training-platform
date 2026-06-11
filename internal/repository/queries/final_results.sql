-- name: DeleteFinalResultsByGame :exec
DELETE FROM final_results WHERE game_id = $1;

-- name: InsertFinalResult :one
INSERT INTO final_results (game_id, team_id, score, position)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListFinalResultsByGame :many
SELECT * FROM final_results WHERE game_id = $1 ORDER BY position;
