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
WHERE enabled = true
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

func (s *Store) ListIntegrations(ctx context.Context) ([]Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, created_at, updated_at
FROM integrations
ORDER BY kind`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Integration
	for rows.Next() {
		var item Integration
		if err := rows.Scan(&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetIntegration(ctx context.Context, id uuid.UUID) (Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, created_at, updated_at
FROM integrations
WHERE id = $1`
	var item Integration
	err := s.pool.QueryRow(ctx, q, id).Scan(&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) GetIntegrationByKind(ctx context.Context, kind string) (Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, created_at, updated_at
FROM integrations
WHERE kind = $1`
	var item Integration
	err := s.pool.QueryRow(ctx, q, kind).Scan(&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) UpsertIntegration(ctx context.Context, kind, name string, config json.RawMessage, enabled bool) (Integration, error) {
	const q = `
INSERT INTO integrations (kind, name, config, enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (kind) DO UPDATE
SET name = EXCLUDED.name, config = EXCLUDED.config, enabled = EXCLUDED.enabled, updated_at = now()
RETURNING id, kind, name, config, enabled, created_at, updated_at`
	var item Integration
	err := s.pool.QueryRow(ctx, q, kind, name, config, enabled).Scan(
		&item.ID, &item.Kind, &item.Name, &item.Config, &item.Enabled, &item.CreatedAt, &item.UpdatedAt,
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
