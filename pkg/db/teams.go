package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListTeams(ctx context.Context) ([]Team, error) {
	const q = `SELECT id, name, description, created_at, updated_at FROM teams ORDER BY name`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var team Team
		if err := rows.Scan(&team.ID, &team.Name, &team.Description, &team.CreatedAt, &team.UpdatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, rows.Err()
}

func (s *Store) GetTeam(ctx context.Context, id uuid.UUID) (Team, error) {
	const q = `SELECT id, name, description, created_at, updated_at FROM teams WHERE id = $1`
	var team Team
	err := s.pool.QueryRow(ctx, q, id).Scan(&team.ID, &team.Name, &team.Description, &team.CreatedAt, &team.UpdatedAt)
	return team, err
}

func (s *Store) CreateTeam(ctx context.Context, name, description string) (Team, error) {
	const q = `
INSERT INTO teams (name, description)
VALUES ($1, $2)
RETURNING id, name, description, created_at, updated_at`
	var team Team
	err := s.pool.QueryRow(ctx, q, name, description).Scan(
		&team.ID, &team.Name, &team.Description, &team.CreatedAt, &team.UpdatedAt,
	)
	return team, err
}

func (s *Store) UpdateTeam(ctx context.Context, id uuid.UUID, name, description string) (Team, error) {
	const q = `
UPDATE teams
SET name = $2, description = $3, updated_at = now()
WHERE id = $1
RETURNING id, name, description, created_at, updated_at`
	var team Team
	err := s.pool.QueryRow(ctx, q, id, name, description).Scan(
		&team.ID, &team.Name, &team.Description, &team.CreatedAt, &team.UpdatedAt,
	)
	return team, err
}

func (s *Store) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM teams WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]TeamMember, error) {
	const q = `
SELECT tm.id, tm.team_id, tm.user_id, tm.team_role, tm.created_at, u.email, u.display_name
FROM team_memberships tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = $1
ORDER BY u.display_name`
	rows, err := s.pool.Query(ctx, q, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []TeamMember
	for rows.Next() {
		var member TeamMember
		if err := rows.Scan(
			&member.ID, &member.TeamID, &member.UserID, &member.TeamRole, &member.CreatedAt,
			&member.Email, &member.DisplayName,
		); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (s *Store) AddTeamMember(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (TeamMembership, error) {
	const q = `
INSERT INTO team_memberships (team_id, user_id, team_role)
VALUES ($1, $2, $3)
RETURNING id, team_id, user_id, team_role, created_at`
	var membership TeamMembership
	err := s.pool.QueryRow(ctx, q, teamID, userID, teamRole).Scan(
		&membership.ID, &membership.TeamID, &membership.UserID, &membership.TeamRole, &membership.CreatedAt,
	)
	return membership, err
}

func (s *Store) UpdateTeamMemberRole(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (TeamMembership, error) {
	const q = `
UPDATE team_memberships
SET team_role = $3
WHERE team_id = $1 AND user_id = $2
RETURNING id, team_id, user_id, team_role, created_at`
	var membership TeamMembership
	err := s.pool.QueryRow(ctx, q, teamID, userID, teamRole).Scan(
		&membership.ID, &membership.TeamID, &membership.UserID, &membership.TeamRole, &membership.CreatedAt,
	)
	return membership, err
}

func (s *Store) RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM team_memberships WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func IsNotFound(err error) bool {
	return isNoRows(err)
}
