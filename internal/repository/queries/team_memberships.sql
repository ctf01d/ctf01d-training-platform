-- name: CreateTeamMembership :one
INSERT INTO team_memberships (team_id, user_id, role, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetTeamMembershipByID :one
SELECT * FROM team_memberships WHERE id = $1;

-- name: UpdateTeamMembership :one
UPDATE team_memberships SET role = COALESCE($2, role),
    status = COALESCE($3, status),
    updated_at = now()
WHERE id = $1 RETURNING *;

-- name: UpdateMembershipStatus :one
UPDATE team_memberships SET status = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: UpdateMembershipRole :one
UPDATE team_memberships SET role = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteTeamMembership :exec
DELETE FROM team_memberships WHERE id = $1;

-- name: ListTeamMembershipsByTeam :many
SELECT * FROM team_memberships WHERE team_id = $1 ORDER BY id;

-- name: ListTeamMembershipsByUser :many
SELECT * FROM team_memberships WHERE user_id = $1 ORDER BY id;

-- name: ListTeamMemberships :many
SELECT * FROM team_memberships ORDER BY id LIMIT $1 OFFSET $2;

-- name: CountTeamMemberships :one
SELECT count(*) FROM team_memberships;

-- name: GetMembership :one
SELECT * FROM team_memberships WHERE team_id = $1 AND user_id = $2;

-- name: CountApprovedManagers :one
SELECT count(*) FROM team_memberships
WHERE team_id = $1 AND status = 'approved' AND role IN ('owner', 'captain', 'vice_captain');
