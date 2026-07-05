package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// DefaultWorkspaceID is the backfill workspace from migration 000012.
var DefaultWorkspaceID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

const demoRoutingLabelKey = "team"
const demoRoutingLabelValue = "platform"
const demoTeamName = "Platform"

// EnsureDemoRouting creates a Platform team and a routing rule for simulator labels
// (team=platform) when no matching rule exists. Idempotent.
func (s *Store) EnsureDemoRouting(ctx context.Context) (uuid.UUID, error) {
	rules, err := s.ListRoutingRules(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	for _, rule := range rules {
		labels, err := parseMatchLabels(rule.MatchLabels)
		if err != nil {
			continue
		}
		if labels[demoRoutingLabelKey] == demoRoutingLabelValue {
			return rule.TeamID, nil
		}
	}

	teamID, err := s.findOrCreateDemoTeam(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	_, err = s.CreateRoutingRule(ctx, teamID, map[string]string{
		demoRoutingLabelKey: demoRoutingLabelValue,
	}, 10)
	if err != nil {
		return uuid.Nil, err
	}
	return teamID, nil
}

func (s *Store) findOrCreateDemoTeam(ctx context.Context) (uuid.UUID, error) {
	teams, err := s.ListTeamsFiltered(ctx, DefaultWorkspaceID)
	if err != nil {
		return uuid.Nil, err
	}
	for _, team := range teams {
		if team.Name == demoTeamName {
			return team.ID, nil
		}
	}

	tier := "l2"
	team, err := s.CreateTeam(ctx, DefaultWorkspaceID, demoTeamName, "Demo routing target for alert simulator", &tier)
	if err != nil {
		return uuid.Nil, err
	}
	return team.ID, nil
}

func parseMatchLabels(raw []byte) (map[string]string, error) {
	var labels map[string]string
	if err := json.Unmarshal(raw, &labels); err != nil {
		return nil, err
	}
	if labels == nil {
		labels = map[string]string{}
	}
	return labels, nil
}

// ReplayFailedProcessAlertJobs re-queues failed process_alert jobs so the worker can retry
// after routing is configured.
func (s *Store) ReplayFailedProcessAlertJobs(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
UPDATE jobs
SET status = 'pending', last_error = NULL, run_at = now(), updated_at = now()
WHERE kind = 'process_alert' AND status = 'failed'`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// CountAllAlerts returns total alerts in the database.
func (s *Store) CountAllAlerts(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM alerts`).Scan(&count)
	return count, err
}
