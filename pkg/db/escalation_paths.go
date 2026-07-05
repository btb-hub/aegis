package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListEscalationPathsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]EscalationPath, error) {
	const q = `
SELECT id, from_team_id, to_team_id, workspace_id, cross_workspace, created_at
FROM escalation_paths
WHERE workspace_id = $1
ORDER BY created_at ASC`
	rows, err := s.pool.Query(ctx, q, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []EscalationPath
	for rows.Next() {
		var path EscalationPath
		if err := rows.Scan(&path.ID, &path.FromTeamID, &path.ToTeamID, &path.WorkspaceID, &path.CrossWorkspace, &path.CreatedAt); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, rows.Err()
}

func (s *Store) ListEscalationPathsFromTeam(ctx context.Context, fromTeamID uuid.UUID) ([]EscalationPath, error) {
	const q = `
SELECT id, from_team_id, to_team_id, workspace_id, cross_workspace, created_at
FROM escalation_paths
WHERE from_team_id = $1
ORDER BY created_at ASC`
	rows, err := s.pool.Query(ctx, q, fromTeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []EscalationPath
	for rows.Next() {
		var path EscalationPath
		if err := rows.Scan(&path.ID, &path.FromTeamID, &path.ToTeamID, &path.WorkspaceID, &path.CrossWorkspace, &path.CreatedAt); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, rows.Err()
}

func (s *Store) ListEscalationPathsToTeam(ctx context.Context, toTeamID uuid.UUID) ([]EscalationPath, error) {
	const q = `
SELECT id, from_team_id, to_team_id, workspace_id, cross_workspace, created_at
FROM escalation_paths
WHERE to_team_id = $1
ORDER BY created_at ASC`
	rows, err := s.pool.Query(ctx, q, toTeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []EscalationPath
	for rows.Next() {
		var path EscalationPath
		if err := rows.Scan(&path.ID, &path.FromTeamID, &path.ToTeamID, &path.WorkspaceID, &path.CrossWorkspace, &path.CreatedAt); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, rows.Err()
}

func (s *Store) HasEscalationPath(ctx context.Context, fromTeamID, toTeamID uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM escalation_paths WHERE from_team_id = $1 AND to_team_id = $2)`
	var exists bool
	err := s.pool.QueryRow(ctx, q, fromTeamID, toTeamID).Scan(&exists)
	return exists, err
}

func (s *Store) ReplaceEscalationPaths(ctx context.Context, workspaceID uuid.UUID, paths []EscalationPath) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM escalation_paths WHERE workspace_id = $1`, workspaceID); err != nil {
		return err
	}

	const insertQ = `
INSERT INTO escalation_paths (from_team_id, to_team_id, workspace_id, cross_workspace)
VALUES ($1, $2, $3, $4)`
	for _, path := range paths {
		if _, err := tx.Exec(ctx, insertQ, path.FromTeamID, path.ToTeamID, workspaceID, path.CrossWorkspace); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Store) AddEscalationPath(ctx context.Context, path EscalationPath) (EscalationPath, error) {
	const q = `
INSERT INTO escalation_paths (from_team_id, to_team_id, workspace_id, cross_workspace)
VALUES ($1, $2, $3, $4)
RETURNING id, from_team_id, to_team_id, workspace_id, cross_workspace, created_at`
	var out EscalationPath
	err := s.pool.QueryRow(ctx, q, path.FromTeamID, path.ToTeamID, path.WorkspaceID, path.CrossWorkspace).Scan(
		&out.ID, &out.FromTeamID, &out.ToTeamID, &out.WorkspaceID, &out.CrossWorkspace, &out.CreatedAt,
	)
	return out, err
}

func (s *Store) DeleteEscalationPath(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM escalation_paths WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) ListHandoffTargetTeams(ctx context.Context, fromTeamID uuid.UUID) ([]Team, error) {
	const q = `
SELECT t.id, t.workspace_id, t.name, t.description, t.support_tier, t.created_at, t.updated_at
FROM escalation_paths ep
JOIN teams t ON t.id = ep.to_team_id
WHERE ep.from_team_id = $1
ORDER BY t.name`
	rows, err := s.pool.Query(ctx, q, fromTeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTeams(rows)
}

func scanTeams(rows pgx.Rows) ([]Team, error) {
	var teams []Team
	for rows.Next() {
		var team Team
		if err := rows.Scan(&team.ID, &team.WorkspaceID, &team.Name, &team.Description, &team.SupportTier, &team.CreatedAt, &team.UpdatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, rows.Err()
}
