package processor

import (
	"context"
	"encoding/json"
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
	incident db.Incident
	user     db.User
	rows     []integrations.IntegrationRow
}

func (s handoffNotifyTestStore) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	if s.incident.ID == uuid.Nil {
		return db.Incident{}, pgx.ErrNoRows
	}
	return s.incident, nil
}

func (s handoffNotifyTestStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return s.user, nil
}

func (s handoffNotifyTestStore) ListEnabledIntegrations(context.Context) ([]integrations.IntegrationRow, error) {
	return s.rows, nil
}

func (s handoffNotifyTestStore) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	return db.Integration{ID: uuid.New()}, nil
}

func (s handoffNotifyTestStore) CreateNotification(context.Context, uuid.UUID, uuid.UUID, string, string) (db.Notification, error) {
	return db.Notification{}, nil
}

func (s handoffNotifyTestStore) AppendTimelineEvent(context.Context, uuid.UUID, string, *uuid.UUID, []byte) error {
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
		"base_url":     server.URL,
		"email":        "ops@example.com",
		"api_token":    "token",
		"project_key":  "OPS",
	})
	require.NoError(t, err)

	store := handoffNotifyTestStore{
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

func TestHandoffNotifyProcessorNoAssignee(t *testing.T) {
	incidentID := uuid.New()
	store := handoffNotifyTestStore{incident: db.Incident{ID: incidentID}}
	p := NewHandoffNotifyProcessor(nil, store, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID:      "job-1",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	})
	require.NoError(t, err)
}
