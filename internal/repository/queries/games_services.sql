-- name: AddService :exec
INSERT INTO games_services (game_id, service_id)
VALUES ($1, $2)
ON CONFLICT (game_id, service_id) DO NOTHING;

-- name: RemoveService :exec
DELETE FROM games_services WHERE game_id = $1 AND service_id = $2;

-- name: ListServicesByGame :many
SELECT service_id FROM games_services WHERE game_id = $1;
