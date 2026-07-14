package processor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type flowAlertStore struct {
	alert                   db.Alert
	teamID                  uuid.UUID
	workspaceID             uuid.UUID
	assigneeID              uuid.UUID
	existing                *db.Incident
	created                 bool
	linked                  bool
	linkErr                 error
	createErr               error
	enqueueErr              error
	workspaceIntegrationErr error
	jiraIntegration         db.Integration
	slackIntegration        db.Integration
	workspaceIntegrations   map[string]db.Integration
	timelineEvents          []timelineEventCall
	jiraServer              *httptest.Server
	slackServer             *httptest.Server
}

type timelineEventCall struct {
	kind    string
	payload []byte
}

func newFlowAlertStore(t *testing.T) *flowAlertStore {
	t.Helper()
	teamID := uuid.New()
	workspaceID := uuid.New()
	assignee := uuid.New()
	alertID := uuid.New()
	labels, _ := json.Marshal(map[string]string{"team": "platform", "alertname": "HighCPU"})

	jiraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" {
			_ = json.NewEncoder(w).Encode(map[string]string{"key": "OPS-9"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "1.2"})
	}))

	jiraCfg, _ := json.Marshal(map[string]string{
		"base_url": jiraServer.URL, "email": "ops@example.com", "api_token": "token", "project_key": "OPS",
	})
	slackCfg, _ := json.Marshal(map[string]string{
		"bot_token": "xoxb-test", "signing_secret": "secret", "api_base_url": slackServer.URL,
	})

	store := &flowAlertStore{
		alert: db.Alert{
			ID: alertID, Fingerprint: "fp-1", Status: "firing", Severity: "critical",
			Title: "CPU high", Labels: labels,
		},
		teamID:           teamID,
		workspaceID:      workspaceID,
		assigneeID:       assignee,
		jiraIntegration:  db.Integration{ID: uuid.New(), Kind: "jira", Config: jiraCfg, Enabled: true},
		slackIntegration: db.Integration{ID: uuid.New(), Kind: "slack", Config: slackCfg, Enabled: true},
		workspaceIntegrations: map[string]db.Integration{
			"jira":    {ID: uuid.New(), Kind: "jira", Config: []byte(`{}`), Enabled: true, WorkspaceID: &workspaceID, Mode: strPtr("inherit")},
			"slack":   {ID: uuid.New(), Kind: "slack", Config: []byte(`{}`), Enabled: true, WorkspaceID: &workspaceID, Mode: strPtr("inherit")},
			"express": {ID: uuid.New(), Kind: "express", Config: []byte(`{}`), Enabled: false, WorkspaceID: &workspaceID, Mode: strPtr("inherit")},
		},
		jiraServer:  jiraServer,
		slackServer: slackServer,
	}
	t.Cleanup(func() {
		jiraServer.Close()
		slackServer.Close()
	})
	return store
}

func (s *flowAlertStore) GetIncidentForAlert(context.Context, uuid.UUID) (db.Incident, error) {
	return db.Incident{}, pgx.ErrNoRows
}
func (s *flowAlertStore) GetAlertByID(context.Context, uuid.UUID) (db.Alert, error) {
	return s.alert, nil
}
func (s *flowAlertStore) FindOpenIncidentByFingerprint(context.Context, string, time.Time) (db.Incident, error) {
	if s.existing != nil {
		return *s.existing, nil
	}
	return db.Incident{}, pgx.ErrNoRows
}
func (s *flowAlertStore) CreateIncidentWithAlert(_ context.Context, input db.CreateIncidentInput) (db.Incident, error) {
	if s.createErr != nil {
		return db.Incident{}, s.createErr
	}
	s.created = true
	return db.Incident{
		ID: uuid.New(), TeamID: input.TeamID, AssigneeID: input.AssigneeID,
		Status: "open", Severity: input.Severity, Title: input.Title, Fingerprint: input.Fingerprint,
		CreatedAt: time.Now(),
	}, nil
}
func (s *flowAlertStore) LinkAlertToIncident(context.Context, uuid.UUID, uuid.UUID) error {
	if s.linkErr != nil {
		return s.linkErr
	}
	s.linked = true
	return nil
}
func (s *flowAlertStore) ListRoutingRules(context.Context) ([]db.RoutingRule, error) {
	matchLabels, _ := json.Marshal(map[string]string{"team": "platform"})
	return []db.RoutingRule{{TeamID: s.teamID, MatchLabels: matchLabels, Priority: 5}}, nil
}
func (s *flowAlertStore) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	if s.assigneeID == uuid.Nil {
		return nil, nil
	}
	return []db.OnCallUser{{UserID: s.assigneeID, Email: "oncall@example.com", DisplayName: "On Call"}}, nil
}
func (s *flowAlertStore) UpdateIncidentJiraKey(context.Context, uuid.UUID, string) error { return nil }
func (s *flowAlertStore) AppendTimelineEvent(_ context.Context, _ uuid.UUID, kind string, _ *uuid.UUID, payload []byte) error {
	s.timelineEvents = append(s.timelineEvents, timelineEventCall{kind: kind, payload: payload})
	return nil
}
func (s *flowAlertStore) GetIntegrationByKind(_ context.Context, kind string) (db.Integration, error) {
	switch kind {
	case "jira":
		if s.jiraIntegration.ID == uuid.Nil {
			return db.Integration{}, pgx.ErrNoRows
		}
		return s.jiraIntegration, nil
	case "slack":
		if s.slackIntegration.ID == uuid.Nil {
			return db.Integration{}, pgx.ErrNoRows
		}
		return s.slackIntegration, nil
	default:
		return db.Integration{}, pgx.ErrNoRows
	}
}
func (s *flowAlertStore) CreateNotification(context.Context, uuid.UUID, uuid.UUID, string, string) (db.Notification, error) {
	return db.Notification{}, nil
}
func (s *flowAlertStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	slackID := "U123"
	return db.User{ID: s.assigneeID, Email: "oncall@example.com", DisplayName: "On Call", Locale: "en", SlackUserID: &slackID}, nil
}
func (s *flowAlertStore) EnqueueEscalation(context.Context, uuid.UUID, time.Time) error {
	return s.enqueueErr
}
func (s *flowAlertStore) ListEnabledIntegrationsForWorkspace(context.Context, uuid.UUID) ([]integrations.IntegrationRow, error) {
	return []integrations.IntegrationRow{
		{ID: s.jiraIntegration.ID, Kind: "jira", Config: s.jiraIntegration.Config, Enabled: true},
		{ID: s.slackIntegration.ID, Kind: "slack", Config: s.slackIntegration.Config, Enabled: true},
	}, nil
}
func (s *flowAlertStore) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return s.workspaceID, nil
}
func (s *flowAlertStore) GetWorkspaceIntegration(_ context.Context, _ uuid.UUID, kind string) (db.Integration, error) {
	if s.workspaceIntegrationErr != nil {
		return db.Integration{}, s.workspaceIntegrationErr
	}
	integration, ok := s.workspaceIntegrations[kind]
	if !ok {
		return db.Integration{}, pgx.ErrNoRows
	}
	return integration, nil
}

func TestAlertProcessorEnqueueEscalationError(t *testing.T) {
	store := newFlowAlertStore(t)
	store.enqueueErr = errors.New("enqueue failed")
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.Error(t, err)
}

func TestAlertProcessorLoadRegistryError(t *testing.T) {
	store := newFlowAlertStore(t)
	store.workspaceIntegrationErr = errors.New("db down")
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.Error(t, err)
}

func TestAlertProcessorLinkError(t *testing.T) {
	store := newFlowAlertStore(t)
	store.existing = &db.Incident{ID: uuid.New()}
	store.linkErr = errors.New("link failed")
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.Error(t, err)
}

func TestAlertProcessorCreateIncidentError(t *testing.T) {
	store := newFlowAlertStore(t)
	store.createErr = errors.New("create failed")
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.Error(t, err)
}

func TestAlertProcessorCreatesIncidentWithoutOnCall(t *testing.T) {
	store := newFlowAlertStore(t)
	store.assigneeID = uuid.Nil
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Kind: "process_alert",
		Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.NoError(t, err)
	require.True(t, store.created)
}

func TestAlertProcessorJiraFailureStillCompletes(t *testing.T) {
	store := newFlowAlertStore(t)
	store.jiraServer.Close()
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Kind: "process_alert",
		Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.NoError(t, err)
	require.True(t, store.created)
}

func TestAlertProcessorCreatesIncident(t *testing.T) {
	store := newFlowAlertStore(t)
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Kind: "process_alert",
		Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.NoError(t, err)
	require.True(t, store.created)
}

func TestAlertProcessorCreatesIncidentAndRecordsUnavailableConnector(t *testing.T) {
	store := newFlowAlertStore(t)
	store.jiraIntegration = db.Integration{}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "http://localhost:8080")

	err := p.Handle(context.Background(), Job{
		ID: "j1", Kind: "process_alert",
		Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})

	require.NoError(t, err)
	require.True(t, store.created)
	var payload map[string]string
	for _, event := range store.timelineEvents {
		if event.kind != "integration_skipped" {
			continue
		}
		require.NoError(t, json.Unmarshal(event.payload, &payload))
		if payload["kind"] == "jira" {
			break
		}
	}
	require.Equal(t, map[string]string{
		"kind":    "jira",
		"reason":  "no_global",
		"message": "Jira skipped: no global connector. Configure global Jira or set the workspace slot to Custom.",
	}, payload)
}

func TestAlertProcessorLinksExistingIncident(t *testing.T) {
	store := newFlowAlertStore(t)
	store.existing = &db.Incident{ID: uuid.New()}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Kind: "process_alert",
		Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.NoError(t, err)
	require.True(t, store.linked)
	require.False(t, store.created)
}

func TestAlertProcessorSkipsResolvedAlert(t *testing.T) {
	store := newFlowAlertStore(t)
	store.alert.Status = "resolved"
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Kind: "process_alert",
		Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.NoError(t, err)
	require.False(t, store.created)
}

func TestAlertProcessorNoRoutingMatch(t *testing.T) {
	store := newFlowAlertStore(t)
	store.alert.Labels = []byte(`{"team":"unknown"}`)
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Kind: "process_alert",
		Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.Error(t, err)
}

func TestEscalateProcessorPagesOpenIncident(t *testing.T) {
	incidentID := uuid.New()
	assignee := uuid.New()
	workspaceID := uuid.New()
	pageCalls := 0
	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCalls++
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "9.9"})
	}))
	defer slackServer.Close()
	slackCfg, _ := json.Marshal(map[string]string{
		"bot_token": "xoxb-test", "signing_secret": "secret", "api_base_url": slackServer.URL,
	})
	slackID := "U123"

	store := &escalateFlowStore{
		incident: db.Incident{
			ID: incidentID, Status: "open", AssigneeID: &assignee,
			TeamID: uuid.New(), Title: "CPU", Severity: "critical", CreatedAt: time.Now(),
		},
		user:        db.User{ID: assignee, Locale: "en", SlackUserID: &slackID},
		workspaceID: workspaceID,
		workspaceIntegrations: map[string]db.Integration{
			"slack": {
				ID: uuid.New(), Kind: "slack", Config: slackCfg, Enabled: true,
				WorkspaceID: &workspaceID, Mode: strPtr("custom"),
			},
		},
	}
	p := NewEscalateProcessor(nil, store, "http://localhost:8080")
	require.NoError(t, p.Handle(context.Background(), Job{
		ID: "j1", Kind: "escalate_incident",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	}))
	require.Equal(t, 1, pageCalls)
}

func TestEscalateProcessorSkipsInheritSlotWithoutGlobal(t *testing.T) {
	incidentID := uuid.New()
	assignee := uuid.New()
	workspaceID := uuid.New()
	store := &escalateFlowStore{
		incident: db.Incident{
			ID: incidentID, Status: "open", AssigneeID: &assignee,
			TeamID: uuid.New(), Title: "CPU", Severity: "critical", CreatedAt: time.Now(),
		},
		user:        db.User{ID: assignee, Locale: "en", SlackUserID: strPtr("U123")},
		workspaceID: workspaceID,
		workspaceIntegrations: map[string]db.Integration{
			"slack": {
				ID: uuid.New(), Kind: "slack", Config: []byte(`{}`), Enabled: true,
				WorkspaceID: &workspaceID, Mode: strPtr("inherit"),
			},
		},
	}
	p := NewEscalateProcessor(nil, store, "http://localhost:8080")

	require.NoError(t, p.Handle(context.Background(), Job{
		ID: "j1", Kind: "escalate_incident",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	}))

	var notice map[string]string
	for _, event := range store.timelineEvents {
		if event.kind == "integration_skipped" {
			require.NoError(t, json.Unmarshal(event.payload, &notice))
			if notice["kind"] == "slack" {
				break
			}
		}
	}
	require.Equal(t, "no_global", notice["reason"])
}

type escalateFlowStore struct {
	incident              db.Incident
	user                  db.User
	workspaceID           uuid.UUID
	globalIntegrations    map[string]db.Integration
	workspaceIntegrations map[string]db.Integration
	timelineEvents        []timelineEventCall
}

func (s escalateFlowStore) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	return s.incident, nil
}
func (s escalateFlowStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return s.user, nil
}
func (s *escalateFlowStore) AppendTimelineEvent(_ context.Context, _ uuid.UUID, kind string, _ *uuid.UUID, payload []byte) error {
	s.timelineEvents = append(s.timelineEvents, timelineEventCall{kind: kind, payload: payload})
	return nil
}
func (s escalateFlowStore) GetIntegrationByKind(_ context.Context, kind string) (db.Integration, error) {
	integration, ok := s.globalIntegrations[kind]
	if !ok {
		return db.Integration{}, pgx.ErrNoRows
	}
	return integration, nil
}
func (s escalateFlowStore) CreateNotification(context.Context, uuid.UUID, uuid.UUID, string, string) (db.Notification, error) {
	return db.Notification{}, nil
}
func (s escalateFlowStore) ListEnabledIntegrations(context.Context) ([]integrations.IntegrationRow, error) {
	return nil, nil
}
func (s escalateFlowStore) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return s.workspaceID, nil
}
func (s escalateFlowStore) GetWorkspaceIntegration(_ context.Context, _ uuid.UUID, kind string) (db.Integration, error) {
	integration, ok := s.workspaceIntegrations[kind]
	if !ok {
		return db.Integration{}, pgx.ErrNoRows
	}
	return integration, nil
}
