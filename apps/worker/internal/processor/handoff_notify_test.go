package processor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type handoffNotifyTestStore struct {
	incident     db.Incident
	user         db.User
	userErr      error
	incidentErr  error
	listErr      error
	rows         []integrations.IntegrationRow
	appendCalled bool
	notifyCalled bool
}

func (s *handoffNotifyTestStore) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	if s.incidentErr != nil {
		return db.Incident{}, s.incidentErr
	}
	if s.incident.ID == uuid.Nil {
		return db.Incident{}, pgx.ErrNoRows
	}
	return s.incident, nil
}

func (s *handoffNotifyTestStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	if s.userErr != nil {
		return db.User{}, s.userErr
	}
	return s.user, nil
}

func (s *handoffNotifyTestStore) ListEnabledIntegrations(context.Context) ([]integrations.IntegrationRow, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.rows, nil
}

func (s *handoffNotifyTestStore) GetIntegrationByKind(_ context.Context, kind string) (db.Integration, error) {
	for _, row := range s.rows {
		if row.Kind == kind {
			return db.Integration{ID: row.ID, Kind: row.Kind, Config: row.Config, Enabled: row.Enabled}, nil
		}
	}
	return db.Integration{}, pgx.ErrNoRows
}

func (s *handoffNotifyTestStore) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	if s.listErr != nil {
		return uuid.Nil, s.listErr
	}
	return uuid.New(), nil
}

func (s *handoffNotifyTestStore) GetWorkspaceIntegration(_ context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error) {
	for _, row := range s.rows {
		if row.Kind == kind {
			return db.Integration{
				ID: row.ID, Kind: row.Kind, Config: row.Config, Enabled: row.Enabled,
				WorkspaceID: &workspaceID, Mode: strPtr("custom"),
			}, nil
		}
	}
	return db.Integration{}, pgx.ErrNoRows
}

func (s *handoffNotifyTestStore) CreateNotification(context.Context, uuid.UUID, uuid.UUID, string, string) (db.Notification, error) {
	s.notifyCalled = true
	return db.Notification{}, nil
}

func (s *handoffNotifyTestStore) AppendTimelineEvent(context.Context, uuid.UUID, string, *uuid.UUID, []byte) error {
	s.appendCalled = true
	return nil
}

func TestHandoffNotifyProcessorUpdatesJiraAssignee(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/user/search":
			_ = json.NewEncoder(w).Encode([]map[string]string{{"accountId": "acct-1"}})
		case "/rest/api/3/issue/OPS-42":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	incidentID := uuid.New()
	assigneeID := uuid.New()
	jiraKey := "OPS-42"
	cfg, err := json.Marshal(map[string]string{
		"base_url":    server.URL,
		"email":       "ops@example.com",
		"api_token":   "token",
		"project_key": "OPS",
	})
	require.NoError(t, err)

	store := &handoffNotifyTestStore{
		incident: db.Incident{
			ID:           incidentID,
			AssigneeID:   &assigneeID,
			JiraIssueKey: &jiraKey,
			Title:        "CPU",
			Severity:     "critical",
		},
		user: db.User{ID: assigneeID, Email: "l3@example.com", Locale: "en"},
		rows: []integrations.IntegrationRow{
			{ID: uuid.New(), Kind: "jira", Config: cfg, Enabled: true},
		},
	}
	p := NewHandoffNotifyProcessor(nil, store, "http://localhost:8080")
	require.NoError(t, p.Handle(context.Background(), Job{
		ID:      "job-1",
		Kind:    "notify_handoff",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	}))
}

func TestHandoffNotifyProcessorInvalidPayload(t *testing.T) {
	p := NewHandoffNotifyProcessor(nil, handoffNotifyMockStore{}, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{ID: "job-1", Payload: json.RawMessage(`{`)})
	require.Error(t, err)
}

func TestHandoffNotifyProcessorInvalidIncidentID(t *testing.T) {
	p := NewHandoffNotifyProcessor(nil, handoffNotifyMockStore{}, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{ID: "job-1", Payload: json.RawMessage(`{"incident_id":"bad"}`)})
	require.Error(t, err)
}

func TestHandoffNotifyProcessorNoAssignee(t *testing.T) {
	incidentID := uuid.New()
	store := &handoffNotifyTestStore{incident: db.Incident{ID: incidentID}}
	p := NewHandoffNotifyProcessor(nil, store, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID:      "job-1",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	})
	require.NoError(t, err)
}

func TestHandoffNotifyProcessorIncidentLookupError(t *testing.T) {
	p := NewHandoffNotifyProcessor(nil, &handoffNotifyTestStore{incidentErr: pgx.ErrNoRows}, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID:      "job-1",
		Payload: json.RawMessage(`{"incident_id":"` + uuid.New().String() + `"}`),
	})
	require.Error(t, err)
}

func TestHandoffNotifyProcessorUserLookupError(t *testing.T) {
	incidentID := uuid.New()
	assigneeID := uuid.New()
	store := &handoffNotifyTestStore{
		incident: db.Incident{ID: incidentID, AssigneeID: &assigneeID},
		userErr:  pgx.ErrNoRows,
	}
	p := NewHandoffNotifyProcessor(nil, store, "http://localhost:8080")
	require.NoError(t, p.Handle(context.Background(), Job{
		ID:      "job-1",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	}))
}

func TestHandoffNotifyProcessorListIntegrationsError(t *testing.T) {
	incidentID := uuid.New()
	assigneeID := uuid.New()
	store := &handoffNotifyTestStore{
		incident: db.Incident{ID: incidentID, AssigneeID: &assigneeID},
		user:     db.User{ID: assigneeID, Email: "l3@example.com"},
		listErr:  errors.New("db down"),
	}
	p := NewHandoffNotifyProcessor(nil, store, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID:      "job-1",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	})
	require.Error(t, err)
}

func TestHandoffNotifyProcessorSlackPageFailure(t *testing.T) {
	incidentID := uuid.New()
	assigneeID := uuid.New()
	slackID := "U123"
	store := &handoffNotifyTestStore{
		incident: db.Incident{ID: incidentID, AssigneeID: &assigneeID, Title: "CPU", Severity: "critical"},
		user:     db.User{ID: assigneeID, Email: "l3@example.com", Locale: "en", SlackUserID: &slackID},
		rows: []integrations.IntegrationRow{
			{ID: uuid.New(), Kind: "slack", Config: []byte(`{"bot_token":"xoxb-test","signing_secret":"secret"}`), Enabled: true},
		},
	}
	p := NewHandoffNotifyProcessor(nil, store, "http://localhost:8080")
	require.NoError(t, p.Handle(context.Background(), Job{
		ID:      "job-1",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	}))
	require.True(t, store.notifyCalled)
}

func TestHandoffNotifyProcessorSlackPageSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/chat.postMessage", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "1234.5678"})
	}))
	defer server.Close()

	incidentID := uuid.New()
	assigneeID := uuid.New()
	slackID := "U123"
	cfg, err := json.Marshal(map[string]string{
		"bot_token":      "xoxb-test",
		"signing_secret": "secret",
		"api_base_url":   server.URL,
	})
	require.NoError(t, err)

	store := &handoffNotifyTestStore{
		incident: db.Incident{ID: incidentID, AssigneeID: &assigneeID, Title: "CPU", Severity: "critical"},
		user:     db.User{ID: assigneeID, Email: "l3@example.com", Locale: "en", SlackUserID: &slackID},
		rows: []integrations.IntegrationRow{
			{ID: uuid.New(), Kind: "slack", Config: cfg, Enabled: true},
		},
	}
	p := NewHandoffNotifyProcessor(nil, store, "http://localhost:8080")
	require.NoError(t, p.Handle(context.Background(), Job{
		ID:      "job-1",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	}))
	require.True(t, store.appendCalled)
	require.True(t, store.notifyCalled)
}
