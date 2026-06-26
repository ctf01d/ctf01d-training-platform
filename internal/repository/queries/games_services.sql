-- name: AddService :exec
INSERT INTO games_services (game_id, service_id, status)
VALUES ($1, $2, COALESCE(sqlc.narg('status'), 'planning'))
ON CONFLICT (game_id, service_id) DO NOTHING;

-- name: RemoveService :exec
DELETE FROM games_services WHERE game_id = $1 AND service_id = $2;

-- name: ListServicesByGame :many
SELECT service_id, status FROM games_services WHERE game_id = $1;

-- name: ListServiceIDsByGame :many
SELECT service_id FROM games_services WHERE game_id = $1;

-- name: SetServiceStatus :exec
UPDATE games_services SET status = $3
WHERE game_id = $1 AND service_id = $2;
