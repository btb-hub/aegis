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
	teams     map[uuid.UUID]db.Team
	workspaces map[uuid.UUID]db.Workspace
	team      db.Team
	rules     []db.RoutingRule
	createErr error
	deleteErr error
	lastCreate RoutingRuleInput
	lastUpdate RoutingRuleInput
}

func (m *routingMockRepo) ListRoutingRules(context.Context) ([]db.RoutingRule, error) {
	return m.rules, nil
}
func (m *routingMockRepo) GetRoutingRule(_ context.Context, id uuid.UUID) (db.RoutingRule, error) {
	for _, rule := range m.rules {
		if rule.ID == id {
			return rule, nil
		}
	}
	return db.RoutingRule{}, pgx.ErrNoRows
}
func (m *routingMockRepo) CreateRoutingRule(
	_ context.Context,
	workspaceID, teamID uuid.UUID,
	matchLabels map[string]string,
	priority int32,
	crossWorkspace bool,
) (db.RoutingRule, error) {
	m.lastCreate = RoutingRuleInput{
		WorkspaceID: workspaceID, TeamID: teamID, MatchLabels: matchLabels,
		Priority: priority, CrossWorkspace: crossWorkspace,
	}
	if m.createErr != nil {
		return db.RoutingRule{}, m.createErr
	}
	raw, _ := json.Marshal(matchLabels)
	return db.RoutingRule{
		ID: uuid.New(), WorkspaceID: workspaceID, TeamID: teamID,
		MatchLabels: raw, Priority: priority, CrossWorkspace: crossWorkspace,
	}, nil
}
func (m *routingMockRepo) UpdateRoutingRule(
	_ context.Context,
	id, workspaceID, teamID uuid.UUID,
	matchLabels map[string]string,
	priority int32,
	crossWorkspace bool,
) (db.RoutingRule, error) {
	m.lastUpdate = RoutingRuleInput{
		WorkspaceID: workspaceID, TeamID: teamID, MatchLabels: matchLabels,
		Priority: priority, CrossWorkspace: crossWorkspace,
	}
	raw, _ := json.Marshal(matchLabels)
	return db.RoutingRule{
		ID: id, WorkspaceID: workspaceID, TeamID: teamID,
		MatchLabels: raw, Priority: priority, CrossWorkspace: crossWorkspace,
	}, nil
}
func (m *routingMockRepo) DeleteRoutingRule(context.Context, uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}
func (m *routingMockRepo) GetTeam(_ context.Context, id uuid.UUID) (db.Team, error) {
	if m.teams != nil {
		team, ok := m.teams[id]
		if !ok {
			return db.Team{}, pgx.ErrNoRows
		}
		return team, nil
	}
	if m.team.ID == uuid.Nil {
		return db.Team{}, pgx.ErrNoRows
	}
	return m.team, nil
}
func (m *routingMockRepo) GetWorkspace(_ context.Context, id uuid.UUID) (db.Workspace, error) {
	if m.workspaces != nil {
		ws, ok := m.workspaces[id]
		if !ok {
			return db.Workspace{}, pgx.ErrNoRows
		}
		return ws, nil
	}
	return db.Workspace{ID: id, Name: "Default", Slug: "default"}, nil
}

func TestRoutingServiceCreateRule(t *testing.T) {
	workspaceID := uuid.New()
	teamID := uuid.New()
	svc := NewRoutingService(&routingMockRepo{
		team: db.Team{ID: teamID, WorkspaceID: workspaceID},
	})
	rule, err := svc.CreateRule(context.Background(), RoutingRuleInput{
		WorkspaceID: workspaceID,
		TeamID:      teamID,
		MatchLabels: map[string]string{"team": "platform"},
		Priority:    10,
	})
	require.NoError(t, err)
	require.Equal(t, teamID, rule.TeamID)
	require.Equal(t, workspaceID, rule.WorkspaceID)
	require.False(t, rule.CrossWorkspace)
}

func TestRoutingServiceCreateRuleSameWorkspace(t *testing.T) {
	workspaceID := uuid.New()
	teamID := uuid.New()
	repo := &routingMockRepo{
		team: db.Team{ID: teamID, WorkspaceID: workspaceID},
	}
	svc := NewRoutingService(repo)
	_, err := svc.CreateRule(context.Background(), RoutingRuleInput{
		WorkspaceID:    workspaceID,
		TeamID:         teamID,
		MatchLabels:    map[string]string{"project": "app"},
		Priority:       5,
		CrossWorkspace: false,
	})
	require.NoError(t, err)
	require.Equal(t, workspaceID, repo.lastCreate.WorkspaceID)
	require.False(t, repo.lastCreate.CrossWorkspace)
}

func TestRoutingServiceCreateRuleCrossWorkspace(t *testing.T) {
	ruleWS := uuid.New()
	teamWS := uuid.New()
	teamID := uuid.New()
	repo := &routingMockRepo{
		team: db.Team{ID: teamID, WorkspaceID: teamWS},
	}
	svc := NewRoutingService(repo)
	rule, err := svc.CreateRule(context.Background(), RoutingRuleInput{
		WorkspaceID:    ruleWS,
		TeamID:         teamID,
		MatchLabels:    map[string]string{"service": "shared"},
		Priority:       20,
		CrossWorkspace: true,
	})
	require.NoError(t, err)
	require.True(t, rule.CrossWorkspace)
	require.Equal(t, ruleWS, rule.WorkspaceID)
	require.Equal(t, teamID, rule.TeamID)
}

func TestRoutingServiceCreateRuleRejectsForeignWithoutFlag(t *testing.T) {
	ruleWS := uuid.New()
	teamWS := uuid.New()
	teamID := uuid.New()
	svc := NewRoutingService(&routingMockRepo{
		team: db.Team{ID: teamID, WorkspaceID: teamWS},
	})
	_, err := svc.CreateRule(context.Background(), RoutingRuleInput{
		WorkspaceID:    ruleWS,
		TeamID:         teamID,
		MatchLabels:    map[string]string{"service": "shared"},
		Priority:       20,
		CrossWorkspace: false,
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestRoutingServiceCreateRuleValidation(t *testing.T) {
	svc := NewRoutingService(&routingMockRepo{})
	_, err := svc.CreateRule(context.Background(), RoutingRuleInput{
		WorkspaceID: uuid.New(),
		TeamID:      uuid.New(),
		MatchLabels: map[string]string{},
		Priority:    1,
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestRoutingServiceCreateRuleMissingWorkspaceID(t *testing.T) {
	svc := NewRoutingService(&routingMockRepo{})
	_, err := svc.CreateRule(context.Background(), RoutingRuleInput{
		TeamID:      uuid.New(),
		MatchLabels: map[string]string{"team": "platform"},
		Priority:    1,
	})
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
	workspaceID := uuid.New()
	repo := &routingMockRepo{team: db.Team{ID: uuid.New(), WorkspaceID: workspaceID}}
	svc := NewRoutingService(repo)
	require.NoError(t, svc.DeleteRule(context.Background(), uuid.New()))
}

func TestRoutingServiceUpdateRule(t *testing.T) {
	workspaceID := uuid.New()
	teamID := uuid.New()
	ruleID := uuid.New()
	raw, _ := json.Marshal(map[string]string{"team": "platform"})
	svc := NewRoutingService(&routingMockRepo{
		team: db.Team{ID: teamID, WorkspaceID: workspaceID},
		rules: []db.RoutingRule{{
			ID: ruleID, WorkspaceID: workspaceID, TeamID: teamID, MatchLabels: raw, Priority: 1,
		}},
	})
	_, err := svc.UpdateRule(context.Background(), ruleID, RoutingRuleInput{
		WorkspaceID: workspaceID,
		TeamID:      teamID,
		MatchLabels: map[string]string{"team": "platform"},
		Priority:    1,
	})
	require.NoError(t, err)
}

func TestRoutingServiceUpdateRuleRejectsWorkspaceChange(t *testing.T) {
	ownedWS := uuid.New()
	otherWS := uuid.New()
	teamID := uuid.New()
	ruleID := uuid.New()
	raw, _ := json.Marshal(map[string]string{"team": "platform"})
	svc := NewRoutingService(&routingMockRepo{
		team: db.Team{ID: teamID, WorkspaceID: ownedWS},
		rules: []db.RoutingRule{{
			ID: ruleID, WorkspaceID: ownedWS, TeamID: teamID, MatchLabels: raw, Priority: 1,
		}},
	})
	_, err := svc.UpdateRule(context.Background(), ruleID, RoutingRuleInput{
		WorkspaceID: otherWS,
		TeamID:      teamID,
		MatchLabels: map[string]string{"team": "platform"},
		Priority:    1,
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestRoutingServiceUpdateRuleRejectsForeignWithoutFlag(t *testing.T) {
	ruleWS := uuid.New()
	teamWS := uuid.New()
	teamID := uuid.New()
	ruleID := uuid.New()
	raw, _ := json.Marshal(map[string]string{"team": "platform"})
	svc := NewRoutingService(&routingMockRepo{
		team: db.Team{ID: teamID, WorkspaceID: teamWS},
		rules: []db.RoutingRule{{
			ID: ruleID, WorkspaceID: ruleWS, TeamID: teamID, MatchLabels: raw, Priority: 1,
		}},
	})
	_, err := svc.UpdateRule(context.Background(), ruleID, RoutingRuleInput{
		WorkspaceID:    ruleWS,
		TeamID:         teamID,
		MatchLabels:    map[string]string{"team": "platform"},
		Priority:       1,
		CrossWorkspace: false,
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestRoutingServiceCreateRuleRequiresTeamID(t *testing.T) {
	workspaceID := uuid.New()
	svc := NewRoutingService(&routingMockRepo{})
	_, err := svc.CreateRule(context.Background(), RoutingRuleInput{
		WorkspaceID: workspaceID,
		MatchLabels: map[string]string{"team": "platform"},
		Priority:    1,
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestMapRoutingTeamErrorNotFound(t *testing.T) {
	err := mapRoutingTeamError(pgx.ErrNoRows)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", appErr.Code)
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
	_, err := svc.CreateRule(context.Background(), RoutingRuleInput{
		WorkspaceID: uuid.New(),
		TeamID:      uuid.New(),
		MatchLabels: map[string]string{"team": "platform"},
		Priority:    1,
	})
	require.Error(t, err)
}

func TestRoutingRuleJSON(t *testing.T) {
	raw, _ := json.Marshal(map[string]string{"team": "platform"})
	ws := uuid.New()
	out := RoutingRuleJSON(db.RoutingRule{
		ID: uuid.New(), WorkspaceID: ws, TeamID: uuid.New(), MatchLabels: raw, Priority: 3, CrossWorkspace: true,
	})
	require.Equal(t, "platform", out["match_labels"].(map[string]string)["team"])
	require.Equal(t, ws.String(), out["workspace_id"])
	require.Equal(t, true, out["cross_workspace"])
}
