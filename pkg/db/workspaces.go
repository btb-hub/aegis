package db

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var DefaultWorkspaceID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type WorkspaceSummary struct {
	Workspace
	TeamCount        int
	RoutingRuleCount int
}

type WorkspaceUsage struct {
	TeamCount           int
	EscalationPathCount int
	IntegrationCount    int
}

func (s *Store) ListWorkspacesWithCounts(ctx context.Context) ([]WorkspaceSummary, error) {
	const q = `
SELECT w.id, w.name, w.slug, w.description, w.created_at, w.updated_at,
       COUNT(DISTINCT t.id)::int AS team_count,
       COUNT(DISTINCT rr.id)::int AS routing_rule_count
FROM workspaces w
LEFT JOIN teams t ON t.workspace_id = w.id
LEFT JOIN routing_rules rr ON rr.workspace_id = w.id
GROUP BY w.id
ORDER BY w.name`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WorkspaceSummary
	for rows.Next() {
		var item WorkspaceSummary
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt,
			&item.TeamCount, &item.RoutingRuleCount,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetWorkspaceUsage(ctx context.Context, id uuid.UUID) (WorkspaceUsage, error) {
	const q = `
SELECT
  (SELECT COUNT(*)::int FROM teams WHERE workspace_id = $1),
  (SELECT COUNT(*)::int FROM escalation_paths WHERE workspace_id = $1),
  (SELECT COUNT(*)::int FROM integrations WHERE workspace_id = $1)`
	var usage WorkspaceUsage
	err := s.pool.QueryRow(ctx, q, id).Scan(&usage.TeamCount, &usage.EscalationPathCount, &usage.IntegrationCount)
	return usage, err
}

func (s *Store) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	const q = `
SELECT id, name, slug, description, created_at, updated_at
FROM workspaces
ORDER BY name`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Workspace
	for rows.Next() {
		var item Workspace
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetWorkspace(ctx context.Context, id uuid.UUID) (Workspace, error) {
	const q = `
SELECT id, name, slug, description, created_at, updated_at
FROM workspaces
WHERE id = $1`
	var item Workspace
	err := s.pool.QueryRow(ctx, q, id).Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) CreateWorkspace(ctx context.Context, name, slug, description string) (Workspace, error) {
	const q = `
INSERT INTO workspaces (name, slug, description)
VALUES ($1, $2, $3)
RETURNING id, name, slug, description, created_at, updated_at`
	var item Workspace
	err := s.pool.QueryRow(ctx, q, name, slug, description).Scan(
		&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Store) CreateWorkspaceWithSlots(ctx context.Context, name, slug, description string) (Workspace, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Workspace{}, err
	}
	defer tx.Rollback(ctx)

	const createWorkspaceQuery = `
INSERT INTO workspaces (name, slug, description)
VALUES ($1, $2, $3)
RETURNING id, name, slug, description, created_at, updated_at`
	var item Workspace
	err = tx.QueryRow(ctx, createWorkspaceQuery, name, slug, description).Scan(
		&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return Workspace{}, err
	}

	const createSlotQuery = `
INSERT INTO integrations (kind, name, config, enabled, workspace_id, mode)
SELECT $1, $1, '{}'::jsonb, true, $2, 'inherit'
WHERE NOT EXISTS (
	SELECT 1
	FROM integrations
	WHERE workspace_id = $2 AND kind = $1
)`
	for _, kind := range []string{"jira", "slack", "express"} {
		if _, err := tx.Exec(ctx, createSlotQuery, kind, item.ID); err != nil {
			return Workspace{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Workspace{}, err
	}
	return item, nil
}

func (s *Store) UpdateWorkspace(ctx context.Context, id uuid.UUID, name, slug, description string) (Workspace, error) {
	const q = `
UPDATE workspaces
SET name = $2, slug = $3, description = $4, updated_at = now()
WHERE id = $1
RETURNING id, name, slug, description, created_at, updated_at`
	var item Workspace
	err := s.pool.QueryRow(ctx, q, id, name, slug, description).Scan(
		&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Store) DeleteWorkspace(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM workspaces WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func Slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	var b strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "workspace"
	}
	return out
}
