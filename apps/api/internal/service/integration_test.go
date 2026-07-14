package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type integrationMockRepo struct {
	items          []db.Integration
	deleteErr      error
	lastUpsertMode *string
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
func (m *integrationMockRepo) GetIntegrationByKind(_ context.Context, kind string) (db.Integration, error) {
	for _, item := range m.items {
		if item.Kind == kind && item.WorkspaceID == nil {
			return item, nil
		}
	}
	return db.Integration{}, pgx.ErrNoRows
}
func (m *integrationMockRepo) GetWorkspaceIntegration(_ context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error) {
	for _, item := range m.items {
		if item.Kind == kind && item.WorkspaceID != nil && *item.WorkspaceID == workspaceID {
			return item, nil
		}
	}
	return db.Integration{}, pgx.ErrNoRows
}
func (m *integrationMockRepo) UpsertIntegration(_ context.Context, kind, name string, config json.RawMessage, enabled bool, workspaceID *uuid.UUID, mode *string) (db.Integration, error) {
	m.lastUpsertMode = mode
	item := db.Integration{ID: uuid.New(), Kind: kind, Name: name, Config: config, Enabled: enabled, WorkspaceID: workspaceID, Mode: mode}
	m.items = append(m.items, item)
	return item, nil
}
func (m *integrationMockRepo) UpdateIntegration(_ context.Context, id uuid.UUID, name string, config json.RawMessage, enabled bool, mode *string) (db.Integration, error) {
	for i, item := range m.items {
		if item.ID == id {
			item.Name = name
			item.Config = config
			item.Enabled = enabled
			item.Mode = mode
			m.items[i] = item
			return item, nil
		}
	}
	return db.Integration{}, pgx.ErrNoRows
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
	item, err := svc.Upsert(context.Background(), "jira", "Jira", json.RawMessage(`{"base_url":"https://jira.example.com","email":"ops@example.com","api_token":"token","project_key":"OPS"}`), true, nil)
	require.NoError(t, err)
	require.Equal(t, "jira", item.Kind)
	require.Nil(t, repo.lastUpsertMode)
}

func TestIntegrationServiceUpsertWorkspaceUsesInheritMode(t *testing.T) {
	repo := &integrationMockRepo{}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	workspaceID := uuid.New()

	item, err := svc.Upsert(context.Background(), "jira", "Jira", json.RawMessage(`{"project_key":"OPS"}`), true, &workspaceID)

	require.NoError(t, err)
	require.NotNil(t, repo.lastUpsertMode)
	require.Equal(t, "inherit", *repo.lastUpsertMode)
	require.Equal(t, "inherit", *item.Mode)
}

func TestIntegrationServiceUpsertWorkspacePreservesExistingMode(t *testing.T) {
	workspaceID := uuid.New()
	custom := "custom"
	repo := &integrationMockRepo{items: []db.Integration{{
		ID: uuid.New(), Kind: "jira", Name: "Jira", Config: json.RawMessage(`{"project_key":"OLD"}`),
		Enabled: true, WorkspaceID: &workspaceID, Mode: &custom,
	}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")

	item, err := svc.Upsert(context.Background(), "jira", "Jira", json.RawMessage(`{"project_key":"OPS"}`), true, &workspaceID)

	require.NoError(t, err)
	require.NotNil(t, repo.lastUpsertMode)
	require.Equal(t, "custom", *repo.lastUpsertMode)
	require.Equal(t, "custom", *item.Mode)
}

func TestIntegrationServiceUpsertGlobalRequiresCredentials(t *testing.T) {
	svc := NewIntegrationService(&integrationMockRepo{}, "http://localhost:8080")
	_, err := svc.Upsert(context.Background(), "jira", "Jira", json.RawMessage(`{"project_key":"OPS"}`), true, nil)
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
	require.Contains(t, appErr.Message, "jira config incomplete")
}

func TestIntegrationServiceUpsertSlackAndExpress(t *testing.T) {
	repo := &integrationMockRepo{}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	_, err := svc.Upsert(context.Background(), "slack", "Slack", json.RawMessage(`{"bot_token":"xoxb","signing_secret":"s"}`), true, nil)
	require.NoError(t, err)
	_, err = svc.Upsert(context.Background(), "express", "eXpress", json.RawMessage(`{"bot_id":"bot","host":"https://cts.example.com","secret_key":"secret"}`), true, nil)
	require.NoError(t, err)
}

func TestIntegrationServiceTestIncompleteConfig(t *testing.T) {
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{
		ID: id, Kind: "jira", Name: "Jira", Config: []byte(`{}`), Enabled: true,
	}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	err := svc.Test(context.Background(), id)
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
	require.NotEqual(t, "integration provider is not configured", appErr.Message)
	require.Contains(t, appErr.Message, "jira config incomplete")
}

func TestIntegrationServiceUpdateKeepsSecrets(t *testing.T) {
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{
		ID: id, Kind: "jira", Name: "Jira",
		Config:  []byte(`{"base_url":"https://jira.example.com","email":"ops@example.com","api_token":"keep-me","project_key":"OPS"}`),
		Enabled: true,
	}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	name := "Jira Prod"
	enabled := false
	item, err := svc.Update(context.Background(), id, &name, &enabled, json.RawMessage(`{"base_url":"https://jira.example.com","email":"ops@example.com","api_token":"","project_key":"OPS"}`))
	require.NoError(t, err)
	require.Equal(t, "Jira Prod", item.Name)
	require.False(t, item.Enabled)
	var cfg map[string]any
	require.NoError(t, json.Unmarshal(item.Config, &cfg))
	require.Equal(t, "keep-me", cfg["api_token"])
}

func TestIntegrationJSONRedactsSecrets(t *testing.T) {
	out := IntegrationJSON(db.Integration{
		ID: uuid.New(), Kind: "slack", Name: "Slack",
		Config: []byte(`{"bot_token":"xoxb-live","signing_secret":"shh","api_base_url":"https://slack.com"}`), Enabled: true,
	})
	cfg, ok := out["config"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, SecretRedacted, cfg["bot_token"])
	require.Equal(t, SecretRedacted, cfg["signing_secret"])
	require.Equal(t, "https://slack.com", cfg["api_base_url"])
	require.Equal(t, true, out["config_complete"])
}

func TestIntegrationJSONIncomplete(t *testing.T) {
	out := IntegrationJSON(db.Integration{
		ID: uuid.New(), Kind: "jira", Name: "Jira", Config: []byte(`{}`), Enabled: true,
	})
	require.Equal(t, false, out["config_complete"])
}

func TestIntegrationServiceGet(t *testing.T) {
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{ID: id, Kind: "slack", Name: "Slack", Config: []byte(`{"bot_token":"x","signing_secret":"y"}`), Enabled: true}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	item, err := svc.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, id, item.ID)

	_, err = svc.Get(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestIntegrationServiceTestWorkspaceMergesGlobalCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	workspaceID := uuid.New()
	globalID := uuid.New()
	overrideID := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{
		{
			ID: globalID, Kind: "jira", Name: "Jira", Enabled: true,
			Config: []byte(`{"base_url":"` + server.URL + `","email":"ops@example.com","api_token":"token","project_key":"GLOBAL"}`),
		},
		{
			ID: overrideID, Kind: "jira", Name: "Workspace Jira", Enabled: true, WorkspaceID: &workspaceID,
			Config: []byte(`{"project_key":"OPS"}`),
		},
	}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	require.NoError(t, svc.Test(context.Background(), overrideID))
}

func TestIntegrationServiceUpdateValidation(t *testing.T) {
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{
		ID: id, Kind: "slack", Name: "Slack", Enabled: true,
		Config: []byte(`{"bot_token":"x","signing_secret":"y"}`),
	}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	empty := "   "
	_, err := svc.Update(context.Background(), id, &empty, nil, nil)
	require.Error(t, err)

	_, err = svc.Update(context.Background(), id, nil, nil, json.RawMessage(`{`))
	require.Error(t, err)

	enabled := true
	_, err = svc.Update(context.Background(), uuid.New(), nil, &enabled, nil)
	require.Error(t, err)
}

func TestMergeIntegrationPatchKeepsSentinel(t *testing.T) {
	merged, err := mergeIntegrationPatchConfig(
		json.RawMessage(`{"bot_token":"keep","signing_secret":"s"}`),
		json.RawMessage(`{"bot_token":"***","signing_secret":"new"}`),
	)
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, json.Unmarshal(merged, &cfg))
	require.Equal(t, "keep", cfg["bot_token"])
	require.Equal(t, "new", cfg["signing_secret"])
}

func TestShouldKeepExistingSecret(t *testing.T) {
	require.True(t, shouldKeepExistingSecret(nil))
	require.True(t, shouldKeepExistingSecret(""))
	require.True(t, shouldKeepExistingSecret("***"))
	require.False(t, shouldKeepExistingSecret("new-token"))
	require.False(t, shouldKeepExistingSecret(123))
}

func TestParseProviderConfigUnknownKind(t *testing.T) {
	err := parseProviderConfig("mattermost", json.RawMessage(`{}`), "")
	require.Error(t, err)
}

func TestMergeIntegrationConfigMaps(t *testing.T) {
	merged, err := mergeIntegrationConfigMaps(
		json.RawMessage(`{"base_url":"https://a","api_token":"t"}`),
		json.RawMessage(`{"project_key":"OPS"}`),
	)
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, json.Unmarshal(merged, &cfg))
	require.Equal(t, "OPS", cfg["project_key"])
	require.Equal(t, "https://a", cfg["base_url"])

	_, err = mergeIntegrationConfigMaps(json.RawMessage(`{`), json.RawMessage(`{}`))
	require.Error(t, err)
	_, err = mergeIntegrationConfigMaps(json.RawMessage(`{}`), json.RawMessage(`{`))
	require.Error(t, err)
}

func TestIntegrationServiceUpsertNilConfig(t *testing.T) {
	svc := NewIntegrationService(&integrationMockRepo{}, "http://localhost:8080")
	_, err := svc.Upsert(context.Background(), "slack", "Slack", nil, true, nil)
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Contains(t, appErr.Message, "slack config incomplete")
}

func TestIntegrationServiceUpdateWorkspaceJira(t *testing.T) {
	workspaceID := uuid.New()
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{
		ID: id, Kind: "jira", Name: "Workspace Jira", Enabled: true, WorkspaceID: &workspaceID,
		Config: []byte(`{"project_key":"OLD"}`),
	}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	item, err := svc.Update(context.Background(), id, nil, nil, json.RawMessage(`{"project_key":"NEW"}`))
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, json.Unmarshal(item.Config, &cfg))
	require.Equal(t, "NEW", cfg["project_key"])
}

func TestIntegrationServiceTestUnsupportedKind(t *testing.T) {
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{ID: id, Kind: "webhook", Name: "Hook", Config: []byte(`{}`), Enabled: true}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	err := svc.Test(context.Background(), id)
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Contains(t, appErr.Message, "unsupported")
}

func TestIntegrationServiceTestWorkspaceWithoutGlobal(t *testing.T) {
	workspaceID := uuid.New()
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{
		ID: id, Kind: "jira", Name: "Override", Enabled: true, WorkspaceID: &workspaceID,
		Config: []byte(`{"project_key":"OPS"}`),
	}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	err := svc.Test(context.Background(), id)
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Contains(t, appErr.Message, "jira config incomplete")
}

func TestIntegrationJSONEmptySecretNotRedacted(t *testing.T) {
	out := IntegrationJSON(db.Integration{
		ID: uuid.New(), Kind: "jira", Name: "Jira",
		Config: []byte(`{"base_url":"https://jira","email":"a@b.c","api_token":"","project_key":"OPS"}`),
		Enabled: true,
	})
	cfg := out["config"].(map[string]any)
	require.Equal(t, "", cfg["api_token"])
}

func TestRedactIntegrationConfigNilMap(t *testing.T) {
	out := redactIntegrationConfig(nil)
	require.Empty(t, out)
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
	err := validateWorkspaceIntegrationConfig("jira", json.RawMessage(`{`))
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
