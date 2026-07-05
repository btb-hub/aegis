package db

import (
	"context"
	"encoding/json"

	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListRoutingRules(ctx context.Context) ([]RoutingRule, error) {
	const q = `
SELECT id, team_id, match_labels, priority, created_at, updated_at
FROM routing_rules
ORDER BY priority DESC, created_at ASC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []RoutingRule
	for rows.Next() {
		var rule RoutingRule
		if err := rows.Scan(&rule.ID, &rule.TeamID, &rule.MatchLabels, &rule.Priority, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Store) GetRoutingRule(ctx context.Context, id uuid.UUID) (RoutingRule, error) {
	const q = `
SELECT id, team_id, match_labels, priority, created_at, updated_at
FROM routing_rules
WHERE id = $1`
	var rule RoutingRule
	err := s.pool.QueryRow(ctx, q, id).Scan(&rule.ID, &rule.TeamID, &rule.MatchLabels, &rule.Priority, &rule.CreatedAt, &rule.UpdatedAt)
	return rule, err
}

func (s *Store) CreateRoutingRule(ctx context.Context, teamID uuid.UUID, matchLabels map[string]string, priority int32) (RoutingRule, error) {
	labelsJSON, err := json.Marshal(matchLabels)
	if err != nil {
		return RoutingRule{}, err
	}
	const q = `
INSERT INTO routing_rules (team_id, match_labels, priority)
VALUES ($1, $2, $3)
RETURNING id, team_id, match_labels, priority, created_at, updated_at`
	var rule RoutingRule
	err = s.pool.QueryRow(ctx, q, teamID, labelsJSON, priority).Scan(
		&rule.ID, &rule.TeamID, &rule.MatchLabels, &rule.Priority, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return rule, err
}

func (s *Store) UpdateRoutingRule(ctx context.Context, id, teamID uuid.UUID, matchLabels map[string]string, priority int32) (RoutingRule, error) {
	labelsJSON, err := json.Marshal(matchLabels)
	if err != nil {
		return RoutingRule{}, err
	}
	const q = `
UPDATE routing_rules
SET team_id = $2, match_labels = $3, priority = $4, updated_at = now()
WHERE id = $1
RETURNING id, team_id, match_labels, priority, created_at, updated_at`
	var rule RoutingRule
	err = s.pool.QueryRow(ctx, q, id, teamID, labelsJSON, priority).Scan(
		&rule.ID, &rule.TeamID, &rule.MatchLabels, &rule.Priority, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return rule, err
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
SELECT id, kind, name, config, enabled
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
		if err := rows.Scan(&id, &row.Kind, &name, &row.Config, &row.Enabled); err != nil {
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
SELECT id, kind, name, config, enabled
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
		if err := rows.Scan(&id, &row.Kind, &name, &row.Config, &row.Enabled); err != nil {
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
SELECT id, kind, name, config, enabled, workspace_id, created_at, updated_at
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
		if err := rows.Scan(&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetIntegration(ctx context.Context, id uuid.UUID) (Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, workspace_id, created_at, updated_at
FROM integrations
WHERE id = $1`
	var item Integration
	err := s.pool.QueryRow(ctx, q, id).Scan(&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) GetIntegrationByKind(ctx context.Context, kind string) (Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, workspace_id, created_at, updated_at
FROM integrations
WHERE kind = $1 AND workspace_id IS NULL`
	var item Integration
	err := s.pool.QueryRow(ctx, q, kind).Scan(&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.CreatedAt, &item.UpdatedAt)
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

	const q = `
SELECT id, kind, name, config, enabled, workspace_id, created_at, updated_at
FROM integrations
WHERE kind = $1 AND workspace_id = $2`
	var override Integration
	overrideErr := s.pool.QueryRow(ctx, q, kind, workspaceID).Scan(
		&override.ID, &override.Kind, &override.Name, &override.Config, &override.Enabled, &override.WorkspaceID, &override.CreatedAt, &override.UpdatedAt,
	)
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
	return global, nil
}

func (s *Store) UpsertIntegration(ctx context.Context, kind, name string, config json.RawMessage, enabled bool, workspaceID *uuid.UUID) (Integration, error) {
	if workspaceID == nil {
		const q = `
INSERT INTO integrations (kind, name, config, enabled, workspace_id)
VALUES ($1, $2, $3, $4, NULL)
ON CONFLICT (kind) WHERE workspace_id IS NULL DO UPDATE
SET name = EXCLUDED.name, config = EXCLUDED.config, enabled = EXCLUDED.enabled, updated_at = now()
RETURNING id, kind, name, config, enabled, workspace_id, created_at, updated_at`
		var item Integration
		err := s.pool.QueryRow(ctx, q, kind, name, config, enabled).Scan(
			&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.CreatedAt, &item.UpdatedAt,
		)
		return item, err
	}

	const q = `
INSERT INTO integrations (kind, name, config, enabled, workspace_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (workspace_id, kind) WHERE workspace_id IS NOT NULL DO UPDATE
SET name = EXCLUDED.name, config = EXCLUDED.config, enabled = EXCLUDED.enabled, updated_at = now()
RETURNING id, kind, name, config, enabled, workspace_id, created_at, updated_at`
	var item Integration
	err := s.pool.QueryRow(ctx, q, kind, name, config, enabled, *workspaceID).Scan(
		&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.WorkspaceID, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
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
