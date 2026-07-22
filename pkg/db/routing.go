package db

import (
	"context"
	"encoding/json"

	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type routingRuleScanner interface {
	Scan(dest ...any) error
}

func scanRoutingRule(row routingRuleScanner) (RoutingRule, error) {
	var rule RoutingRule
	err := row.Scan(
		&rule.ID, &rule.WorkspaceID, &rule.TeamID, &rule.MatchLabels, &rule.Priority,
		&rule.CrossWorkspace, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return rule, err
}

func (s *Store) ListRoutingRules(ctx context.Context) ([]RoutingRule, error) {
	const q = `
SELECT id, workspace_id, team_id, match_labels, priority, cross_workspace, created_at, updated_at
FROM routing_rules
ORDER BY priority DESC, created_at ASC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []RoutingRule
	for rows.Next() {
		rule, err := scanRoutingRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Store) GetRoutingRule(ctx context.Context, id uuid.UUID) (RoutingRule, error) {
	const q = `
SELECT id, workspace_id, team_id, match_labels, priority, cross_workspace, created_at, updated_at
FROM routing_rules
WHERE id = $1`
	return scanRoutingRule(s.pool.QueryRow(ctx, q, id))
}

func (s *Store) CreateRoutingRule(
	ctx context.Context,
	workspaceID, teamID uuid.UUID,
	matchLabels map[string]string,
	priority int32,
	crossWorkspace bool,
) (RoutingRule, error) {
	labelsJSON, err := json.Marshal(matchLabels)
	if err != nil {
		return RoutingRule{}, err
	}
	const q = `
INSERT INTO routing_rules (workspace_id, team_id, match_labels, priority, cross_workspace)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, workspace_id, team_id, match_labels, priority, cross_workspace, created_at, updated_at`
	return scanRoutingRule(s.pool.QueryRow(ctx, q, workspaceID, teamID, labelsJSON, priority, crossWorkspace))
}

func (s *Store) UpdateRoutingRule(
	ctx context.Context,
	id, workspaceID, teamID uuid.UUID,
	matchLabels map[string]string,
	priority int32,
	crossWorkspace bool,
) (RoutingRule, error) {
	labelsJSON, err := json.Marshal(matchLabels)
	if err != nil {
		return RoutingRule{}, err
	}
	const q = `
UPDATE routing_rules
SET workspace_id = $2, team_id = $3, match_labels = $4, priority = $5, cross_workspace = $6, updated_at = now()
WHERE id = $1
RETURNING id, workspace_id, team_id, match_labels, priority, cross_workspace, created_at, updated_at`
	return scanRoutingRule(s.pool.QueryRow(ctx, q, id, workspaceID, teamID, labelsJSON, priority, crossWorkspace))
}

func (s *Store) DeleteRoutingRule(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM routing_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) ListEnabledIntegrations(ctx context.Context) ([]integrations.IntegrationRow, error) {
	const q = `
SELECT id, kind, name, config, enabled, mode
FROM integrations
WHERE enabled = true AND workspace_id IS NULL
ORDER BY kind`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []integrations.IntegrationRow
	for rows.Next() {
		var row integrations.IntegrationRow
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &row.Kind, &name, &row.Config, &row.Enabled, &row.Mode); err != nil {
			return nil, err
		}
		row.ID = id
		row.Name = name
		items = append(items, row)
	}
	return items, rows.Err()
}

func (s *Store) ListEnabledIntegrationsForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]integrations.IntegrationRow, error) {
	globalRows, err := s.ListEnabledIntegrations(ctx)
	if err != nil {
		return nil, err
	}
	if workspaceID == uuid.Nil {
		return globalRows, nil
	}

	const q = `
SELECT id, kind, name, config, enabled, mode
FROM integrations
WHERE enabled = true AND workspace_id = $1
ORDER BY kind`
	rows, err := s.pool.Query(ctx, q, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	overrides := map[string]integrations.IntegrationRow{}
	for rows.Next() {
		var row integrations.IntegrationRow
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &row.Kind, &name, &row.Config, &row.Enabled, &row.Mode); err != nil {
			return nil, err
		}
		row.ID = id
		row.Name = name
		overrides[row.Kind] = row
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]integrations.IntegrationRow, 0, len(globalRows))
	for _, global := range globalRows {
		if override, ok := overrides[global.Kind]; ok {
			merged, err := mergeIntegrationConfig(global.Config, override.Config)
			if err != nil {
				return nil, err
			}
			global.Config = merged
			global.ID = override.ID
			global.Mode = override.Mode
		}
		out = append(out, global)
	}
	for kind, override := range overrides {
		found := false
		for _, global := range globalRows {
			if global.Kind == kind {
				found = true
				break
			}
		}
		if !found {
			out = append(out, override)
		}
	}
	return out, nil
}

func mergeIntegrationConfig(global, override []byte) ([]byte, error) {
	var globalMap map[string]any
	if err := json.Unmarshal(global, &globalMap); err != nil {
		return nil, err
	}
	var overrideMap map[string]any
	if err := json.Unmarshal(override, &overrideMap); err != nil {
		return nil, err
	}
	if globalMap == nil {
		globalMap = map[string]any{}
	}
	for key, value := range overrideMap {
		globalMap[key] = value
	}
	return json.Marshal(globalMap)
}

func (s *Store) ListIntegrations(ctx context.Context) ([]Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, workspace_id, mode, created_at, updated_at
FROM integrations
ORDER BY workspace_id NULLS FIRST, kind`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Integration
	for rows.Next() {
		var item Integration
		if err := rows.Scan(&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.Mode, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetIntegration(ctx context.Context, id uuid.UUID) (Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, workspace_id, mode, created_at, updated_at
FROM integrations
WHERE id = $1`
	var item Integration
	err := s.pool.QueryRow(ctx, q, id).Scan(&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.Mode, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) GetIntegrationByKind(ctx context.Context, kind string) (Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, workspace_id, mode, created_at, updated_at
FROM integrations
WHERE kind = $1 AND workspace_id IS NULL`
	var item Integration
	err := s.pool.QueryRow(ctx, q, kind).Scan(&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.Mode, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) GetWorkspaceIntegration(ctx context.Context, workspaceID uuid.UUID, kind string) (Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, workspace_id, mode, created_at, updated_at
FROM integrations
WHERE workspace_id = $1 AND kind = $2`
	var item Integration
	err := s.pool.QueryRow(ctx, q, workspaceID, kind).Scan(
		&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.Mode, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Store) GetIntegrationByKindForWorkspace(ctx context.Context, kind string, workspaceID uuid.UUID) (Integration, error) {
	global, err := s.GetIntegrationByKind(ctx, kind)
	if err != nil && !isNoRows(err) {
		return Integration{}, err
	}
	if workspaceID == uuid.Nil {
		return global, err
	}

	override, overrideErr := s.GetWorkspaceIntegration(ctx, workspaceID, kind)
	if overrideErr != nil {
		if isNoRows(overrideErr) {
			return global, err
		}
		return Integration{}, overrideErr
	}
	if isNoRows(err) {
		return override, nil
	}
	merged, mergeErr := mergeIntegrationConfig(global.Config, override.Config)
	if mergeErr != nil {
		return Integration{}, mergeErr
	}
	global.Config = merged
	global.ID = override.ID
	global.WorkspaceID = override.WorkspaceID
	global.Mode = override.Mode
	return global, nil
}

func (s *Store) UpsertIntegration(ctx context.Context, kind, name string, config json.RawMessage, enabled bool, workspaceID *uuid.UUID, mode *string) (Integration, error) {
	if workspaceID == nil {
		const q = `
INSERT INTO integrations (kind, name, config, enabled, workspace_id, mode)
VALUES ($1, $2, $3, $4, NULL, NULL)
ON CONFLICT (kind) WHERE workspace_id IS NULL DO UPDATE
SET name = EXCLUDED.name, config = EXCLUDED.config, enabled = EXCLUDED.enabled, mode = NULL, updated_at = now()
RETURNING id, kind, name, config, enabled, workspace_id, mode, created_at, updated_at`
		var item Integration
		err := s.pool.QueryRow(ctx, q, kind, name, config, enabled).Scan(
			&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.Mode, &item.CreatedAt, &item.UpdatedAt,
		)
		return item, err
	}

	const q = `
INSERT INTO integrations (kind, name, config, enabled, workspace_id, mode)
VALUES ($1, $2, $3, $4, $5, COALESCE($6, 'inherit'))
ON CONFLICT (workspace_id, kind) WHERE workspace_id IS NOT NULL DO UPDATE
SET name = EXCLUDED.name, config = EXCLUDED.config, enabled = EXCLUDED.enabled, mode = EXCLUDED.mode, updated_at = now()
RETURNING id, kind, name, config, enabled, workspace_id, mode, created_at, updated_at`
	var item Integration
	err := s.pool.QueryRow(ctx, q, kind, name, config, enabled, *workspaceID, mode).Scan(
		&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.Mode, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Store) UpdateIntegration(ctx context.Context, id uuid.UUID, name string, config json.RawMessage, enabled bool, mode *string) (Integration, error) {
	const q = `
UPDATE integrations
SET name = $2, config = $3, enabled = $4, mode = $5, updated_at = now()
WHERE id = $1
RETURNING id, kind, name, config, enabled, workspace_id, mode, created_at, updated_at`
	var item Integration
	err := s.pool.QueryRow(ctx, q, id, name, config, enabled, mode).Scan(
		&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.Mode, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Store) EnsureWorkspaceSlots(ctx context.Context, workspaceID uuid.UUID) error {
	const q = `
INSERT INTO integrations (kind, name, config, enabled, workspace_id, mode)
SELECT $1, $1, '{}'::jsonb, true, $2, 'inherit'
WHERE NOT EXISTS (
	SELECT 1
	FROM integrations
	WHERE workspace_id = $2 AND kind = $1
)`
	for _, kind := range []string{"jira", "slack", "express"} {
		if _, err := s.pool.Exec(ctx, q, kind, workspaceID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteIntegration(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM integrations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
