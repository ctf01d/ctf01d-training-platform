-- name: CreateEvent :one
INSERT INTO team_membership_events (team_id, user_id, actor_id, action, from_role, to_role, from_status, to_status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListEventsByTeam :many
SELECT * FROM team_membership_events WHERE team_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetLatestEventForMember :one
SELECT * FROM team_membership_events
WHERE team_id = $1 AND user_id = $2
ORDER BY created_at DESC LIMIT 1;

-- name: CountEventsByTeam :one
SELECT count(*) FROM team_membership_events WHERE team_id = $1;
