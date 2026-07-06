package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type integrationMockRepo struct {
	items     []db.Integration
	deleteErr error
}

func (m *integrationMockRepo) ListIntegrations(context.Context) ([]db.Integration, error) {
	return m.items, nil
}
func (m *integrationMockRepo) GetIntegration(_ context.Context, id uuid.UUID) (db.Integration, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return db.Integration{}, pgx.ErrNoRows
}
func (m *integrationMockRepo) UpsertIntegration(_ context.Context, kind, name string, config json.RawMessage, enabled bool, workspaceID *uuid.UUID) (db.Integration, error) {
	item := db.Integration{ID: uuid.New(), Kind: kind, Name: name, Config: config, Enabled: enabled, WorkspaceID: workspaceID}
	m.items = append(m.items, item)
	return item, nil
}
func (m *integrationMockRepo) DeleteIntegration(context.Context, uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}
func (m *integrationMockRepo) ListEnabledIntegrations(_ context.Context) ([]integrations.IntegrationRow, error) {
	rows := make([]integrations.IntegrationRow, 0, len(m.items))
	for _, item := range m.items {
		if item.Enabled {
			rows = append(rows, integrations.IntegrationRow{ID: item.ID, Kind: item.Kind, Config: item.Config, Enabled: true})
		}
	}
	return rows, nil
}
func (m *integrationMockRepo) GetWorkspace(_ context.Context, id uuid.UUID) (db.Workspace, error) {
	return db.Workspace{ID: id, Name: "Default", Slug: "default"}, nil
}

func TestIntegrationServiceList(t *testing.T) {
	repo := &integrationMockRepo{items: []db.Integration{{ID: uuid.New(), Kind: "jira"}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	items, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestIntegrationServiceDelete(t *testing.T) {
	svc := NewIntegrationService(&integrationMockRepo{}, "http://localhost:8080")
	require.NoError(t, svc.Delete(context.Background(), uuid.New()))
}

func TestIncidentServiceList(t *testing.T) {
	svc := NewIncidentService(&incidentMockRepo{incident: db.Incident{ID: uuid.New(), Status: "open"}})
	items, err := svc.List(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestIncidentServiceGet(t *testing.T) {
	id := uuid.New()
	svc := NewIncidentService(&incidentMockRepo{incident: db.Incident{ID: id, Status: "open"}})
	incident, err := svc.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, id, incident.ID)
}

func TestRoutingServiceListRules(t *testing.T) {
	svc := NewRoutingService(&routingMockRepo{rules: []db.RoutingRule{{ID: uuid.New()}}})
	rules, err := svc.ListRules(context.Background())
	require.NoError(t, err)
	require.Len(t, rules, 1)
}

func TestMapIntegrationErrorNotFound(t *testing.T) {
	err := mapIntegrationError(pgx.ErrNoRows)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestIntegrationServiceUpsertValidation(t *testing.T) {
	svc := NewIntegrationService(&integrationMockRepo{}, "http://localhost:8080")
	_, err := svc.Upsert(context.Background(), "unknown", "x", json.RawMessage(`{}`), true, nil)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestIntegrationServiceUpsertSuccess(t *testing.T) {
	repo := &integrationMockRepo{}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	item, err := svc.Upsert(context.Background(), "jira", "Jira", json.RawMessage(`{"project_key":"OPS"}`), true, nil)
	require.NoError(t, err)
	require.Equal(t, "jira", item.Kind)
}

func TestIntegrationServiceUpsertWorkspaceJira(t *testing.T) {
	repo := &integrationMockRepo{}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	workspaceID := uuid.New()
	item, err := svc.Upsert(context.Background(), "jira", "Platform Jira", json.RawMessage(`{"project_key":"OPS"}`), true, &workspaceID)
	require.NoError(t, err)
	require.NotNil(t, item.WorkspaceID)
	require.Equal(t, workspaceID, *item.WorkspaceID)
}

func TestIntegrationServiceUpsertWorkspaceJiraRequiresProjectKey(t *testing.T) {
	repo := &integrationMockRepo{}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	workspaceID := uuid.New()
	_, err := svc.Upsert(context.Background(), "jira", "Platform Jira", json.RawMessage(`{}`), true, &workspaceID)
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestIntegrationJSONWithWorkspace(t *testing.T) {
	workspaceID := uuid.New()
	out := IntegrationJSON(db.Integration{
		ID: uuid.New(), Kind: "jira", Name: "Platform Jira", Config: []byte(`{"project_key":"OPS"}`), Enabled: true, WorkspaceID: &workspaceID,
	})
	require.Equal(t, workspaceID.String(), out["workspace_id"])
}

func TestIntegrationServiceUpsertWorkspaceNotFound(t *testing.T) {
	repo := &integrationMockRepoMissingWorkspace{}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	workspaceID := uuid.New()
	_, err := svc.Upsert(context.Background(), "jira", "Platform Jira", json.RawMessage(`{"project_key":"OPS"}`), true, &workspaceID)
	require.Error(t, err)
}

type integrationMockRepoMissingWorkspace struct {
	integrationMockRepo
}

func (m *integrationMockRepoMissingWorkspace) GetWorkspace(_ context.Context, _ uuid.UUID) (db.Workspace, error) {
	return db.Workspace{}, pgx.ErrNoRows
}

func TestValidateWorkspaceIntegrationConfigInvalidJSON(t *testing.T) {
	err := validateWorkspaceIntegrationConfig("jira", json.RawMessage(`{`), true)
	require.Error(t, err)
}

func TestIntegrationJSON(t *testing.T) {
	out := IntegrationJSON(db.Integration{ID: uuid.New(), Kind: "slack", Name: "Slack", Config: []byte(`{"bot_token":"x"}`), Enabled: true})
	require.Equal(t, "slack", out["kind"])
}

func TestIncidentServiceResolve(t *testing.T) {
	incidentID := uuid.New()
	svc := NewIncidentService(&incidentMockRepo{incident: db.Incident{ID: incidentID, Status: "open"}})
	incident, err := svc.Resolve(context.Background(), incidentID, uuid.New())
	require.NoError(t, err)
	require.Equal(t, "resolved", incident.Status)
}

func TestIncidentServiceTimelineNotFound(t *testing.T) {
	svc := NewIncidentService(&incidentMockRepo{})
	_, err := svc.Timeline(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestTimelineEventJSON(t *testing.T) {
	actor := uuid.New()
	out := TimelineEventJSON(db.TimelineEvent{
		ID: uuid.New(), Kind: "created", ActorID: &actor, Payload: []byte(`{"k":"v"}`),
	})
	require.Equal(t, "created", out["kind"])
}
