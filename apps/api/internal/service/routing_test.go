package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type routingMockRepo struct {
	team      db.Team
	rules     []db.RoutingRule
	createErr error
	deleteErr error
}

func (m *routingMockRepo) ListRoutingRules(context.Context) ([]db.RoutingRule, error) {
	return m.rules, nil
}
func (m *routingMockRepo) GetRoutingRule(context.Context, uuid.UUID) (db.RoutingRule, error) {
	return db.RoutingRule{}, pgx.ErrNoRows
}
func (m *routingMockRepo) CreateRoutingRule(_ context.Context, teamID uuid.UUID, matchLabels map[string]string, priority int32) (db.RoutingRule, error) {
	if m.createErr != nil {
		return db.RoutingRule{}, m.createErr
	}
	raw, _ := json.Marshal(matchLabels)
	return db.RoutingRule{ID: uuid.New(), TeamID: teamID, MatchLabels: raw, Priority: priority}, nil
}
func (m *routingMockRepo) UpdateRoutingRule(context.Context, uuid.UUID, uuid.UUID, map[string]string, int32) (db.RoutingRule, error) {
	return db.RoutingRule{}, nil
}
func (m *routingMockRepo) DeleteRoutingRule(context.Context, uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}
func (m *routingMockRepo) GetTeam(context.Context, uuid.UUID) (db.Team, error) {
	if m.team.ID == uuid.Nil {
		return db.Team{}, pgx.ErrNoRows
	}
	return m.team, nil
}

func TestRoutingServiceCreateRule(t *testing.T) {
	teamID := uuid.New()
	svc := NewRoutingService(&routingMockRepo{team: db.Team{ID: teamID}})
	rule, err := svc.CreateRule(context.Background(), teamID, map[string]string{"team": "platform"}, 10)
	require.NoError(t, err)
	require.Equal(t, teamID, rule.TeamID)
}

func TestRoutingServiceCreateRuleValidation(t *testing.T) {
	svc := NewRoutingService(&routingMockRepo{})
	_, err := svc.CreateRule(context.Background(), uuid.New(), map[string]string{}, 1)
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestRoutingServiceMatchTeam(t *testing.T) {
	teamID := uuid.New()
	labels, _ := json.Marshal(map[string]string{"team": "platform"})
	svc := NewRoutingService(&routingMockRepo{rules: []db.RoutingRule{
		{TeamID: teamID, MatchLabels: labels, Priority: 5},
	}})
	got, err := svc.MatchTeam(context.Background(), map[string]string{"team": "platform", "alertname": "HighCPU"})
	require.NoError(t, err)
	require.Equal(t, teamID, got)
}

func TestIntegrationServiceTestSlackMissingConfig(t *testing.T) {
	id := uuid.New()
	cfg, _ := json.Marshal(map[string]string{"bot_token": "x"})
	repo := &integrationMockRepo{items: []db.Integration{{ID: id, Kind: "slack", Config: cfg, Enabled: true}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	err := svc.Test(context.Background(), id)
	require.Error(t, err)
}

func TestIntegrationServiceLoadRegistrySlack(t *testing.T) {
	cfg, _ := json.Marshal(map[string]string{"bot_token": "xoxb-test", "signing_secret": "secret"})
	repo := &integrationMockRepo{items: []db.Integration{{ID: uuid.New(), Kind: "slack", Config: cfg, Enabled: true}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	reg, err := svc.LoadRegistry(context.Background())
	require.NoError(t, err)
	_, ok := reg.Chat("slack")
	require.True(t, ok)
}

func TestIncidentServiceAcknowledgeBySlackUserNotFound(t *testing.T) {
	svc := NewIncidentService(&incidentMockRepo{incident: db.Incident{ID: uuid.New(), Status: "open"}})
	_, err := svc.AcknowledgeBySlackUser(context.Background(), uuid.New(), "U404")
	require.Error(t, err)
}

func TestRoutingServiceMatchTeamNotFound(t *testing.T) {
	svc := NewRoutingService(&routingMockRepo{rules: []db.RoutingRule{}})
	_, err := svc.MatchTeam(context.Background(), map[string]string{"team": "platform"})
	require.Error(t, err)
}

func TestRoutingServiceDeleteRule(t *testing.T) {
	repo := &routingMockRepo{team: db.Team{ID: uuid.New()}}
	svc := NewRoutingService(repo)
	require.NoError(t, svc.DeleteRule(context.Background(), uuid.New()))
}

func TestRoutingServiceUpdateRule(t *testing.T) {
	teamID := uuid.New()
	svc := NewRoutingService(&routingMockRepo{team: db.Team{ID: teamID}})
	_, err := svc.UpdateRule(context.Background(), uuid.New(), teamID, map[string]string{"team": "platform"}, 1)
	require.NoError(t, err)
}

func TestRoutingServiceMatchTeamSkipsInvalidStoredRule(t *testing.T) {
	teamID := uuid.New()
	good, _ := json.Marshal(map[string]string{"team": "platform"})
	svc := NewRoutingService(&routingMockRepo{rules: []db.RoutingRule{
		{TeamID: uuid.New(), MatchLabels: []byte(`{`), Priority: 1},
		{TeamID: teamID, MatchLabels: good, Priority: 2},
	}})
	got, err := svc.MatchTeam(context.Background(), map[string]string{"team": "platform"})
	require.NoError(t, err)
	require.Equal(t, teamID, got)
}

func TestMapRoutingErrorNotFound(t *testing.T) {
	err := mapRoutingError(pgx.ErrNoRows)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestMapRoutingErrorPassthrough(t *testing.T) {
	inner := errors.New("db down")
	require.Equal(t, inner, mapRoutingError(inner))
}

func TestRoutingServiceCreateRuleTeamNotFound(t *testing.T) {
	svc := NewRoutingService(&routingMockRepo{})
	_, err := svc.CreateRule(context.Background(), uuid.New(), map[string]string{"team": "platform"}, 1)
	require.Error(t, err)
}

func TestRoutingRuleJSON(t *testing.T) {
	raw, _ := json.Marshal(map[string]string{"team": "platform"})
	out := RoutingRuleJSON(db.RoutingRule{
		ID: uuid.New(), TeamID: uuid.New(), MatchLabels: raw, Priority: 3,
	})
	require.Equal(t, "platform", out["match_labels"].(map[string]string)["team"])
}
