-- name: ListTeams :many
SELECT id, name, description, created_at, updated_at
FROM teams
ORDER BY name;

-- name: GetTeam :one
SELECT id, name, description, created_at, updated_at
FROM teams
WHERE id = $1;

-- name: CreateTeam :one
INSERT INTO teams (name, description)
VALUES ($1, $2)
RETURNING id, name, description, created_at, updated_at;

-- name: UpdateTeam :one
UPDATE teams
SET name = $2, description = $3, updated_at = now()
WHERE id = $1
RETURNING id, name, description, created_at, updated_at;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = $1;

-- name: ListTeamMembers :many
SELECT
    tm.id,
    tm.team_id,
    tm.user_id,
    tm.team_role,
    tm.created_at,
    u.email,
    u.display_name
FROM team_memberships tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = $1
ORDER BY u.display_name;

-- name: AddTeamMember :one
INSERT INTO team_memberships (team_id, user_id, team_role)
VALUES ($1, $2, $3)
RETURNING id, team_id, user_id, team_role, created_at;

-- name: UpdateTeamMemberRole :one
UPDATE team_memberships
SET team_role = $3
WHERE team_id = $1 AND user_id = $2
RETURNING id, team_id, user_id, team_role, created_at;

-- name: RemoveTeamMember :exec
DELETE FROM team_memberships WHERE team_id = $1 AND user_id = $2;
