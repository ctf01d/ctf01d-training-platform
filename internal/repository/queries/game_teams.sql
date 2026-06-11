-- name: CreateGameTeam :one
INSERT INTO game_teams (game_id, team_id, ip_address, ctf01d_id, ctf01d_overrides, team_type, "order")
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetGameTeamByID :one
SELECT * FROM game_teams WHERE id = $1;

-- name: ListGameTeamsByGame :many
SELECT * FROM game_teams WHERE game_id = $1 ORDER BY "order", id;

-- name: ListGameTeamsByTeam :many
SELECT * FROM game_teams WHERE team_id = $1;

-- name: UpdateGameTeam :one
UPDATE game_teams SET
    ip_address = COALESCE(sqlc.narg('ip_address'), ip_address),
    ctf01d_id = COALESCE(sqlc.narg('ctf01d_id'), ctf01d_id),
    ctf01d_overrides = COALESCE(sqlc.narg('ctf01d_overrides'), ctf01d_overrides),
    team_type = COALESCE(sqlc.narg('team_type'), team_type),
    "order" = COALESCE(sqlc.narg('order'), "order"),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteGameTeam :exec
DELETE FROM game_teams WHERE id = $1;

-- name: UpdateGameTeamOrder :exec
UPDATE game_teams SET "order" = $2, updated_at = now()
WHERE id = $1;

-- name: IsUserApprovedInGameTeams :one
SELECT EXISTS(
  SELECT 1 FROM game_teams gt
  JOIN team_memberships tm ON tm.team_id = gt.team_id AND tm.user_id = $2 AND tm.status = 'approved'
  WHERE gt.game_id = $1
);
