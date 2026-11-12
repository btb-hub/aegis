package processor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type notifyIncidentTestStore struct {
	incident        db.Incident
	hasNotification bool
	createCalls     int
}

func (s *notifyIncidentTestStore) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	return s.incident, nil
}
func (s *notifyIncidentTestStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}
func (s *notifyIncidentTestStore) UpdateIncidentJiraKey(context.Context, uuid.UUID, string) error {
	return nil
}
func (s *notifyIncidentTestStore) AppendTimelineEvent(context.Context, uuid.UUID, string, *uuid.UUID, []byte) error {
	return nil
}
func (s *notifyIncidentTestStore) CreateNotification(context.Context, uuid.UUID, uuid.UUID, string, string) (db.Notification, error) {
	s.createCalls++
	return db.Notification{}, nil
}
func (s *notifyIncidentTestStore) HasNotification(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return s.hasNotification, nil
}
func (s *notifyIncidentTestStore) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (s *notifyIncidentTestStore) GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error) {
	return db.Integration{}, pgx.ErrNoRows
}
func (s *notifyIncidentTestStore) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	return db.Integration{}, pgx.ErrNoRows
}

func TestNotifyIncidentProcessorSkipsResolvedIncident(t *testing.T) {
	store := &notifyIncidentTestStore{
		incident: db.Incident{ID: uuid.New(), Status: "resolved", CreatedAt: time.Now()},
	}
	p := NewNotifyIncidentProcessor(nil, store, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		Payload: json.RawMessage(`{"incident_id":"` + store.incident.ID.String() + `"}`),
	})
	require.NoError(t, err)
	require.Zero(t, store.createCalls)
}

func TestNotifyIncidentProcessorSkipsWhenAlreadyNotified(t *testing.T) {
	store := newFlowAlertStore(t)
	store.hasSentNotification = true
	incidentID := uuid.New()
	store.incidentForNotify = db.Incident{
		ID: incidentID, TeamID: store.teamID, AssigneeID: &store.assigneeID,
		Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp-1", CreatedAt: time.Now(),
	}
	p := NewNotifyIncidentProcessor(nil, store, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	})
	require.NoError(t, err)
}

func TestNotifyIncidentProcessorInvalidPayload(t *testing.T) {
	p := NewNotifyIncidentProcessor(nil, notifyIncidentMockStore{}, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{`)})
	require.Error(t, err)
}

func TestNotifyIncidentProcessorInvalidIncidentID(t *testing.T) {
	p := NewNotifyIncidentProcessor(nil, notifyIncidentMockStore{}, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"incident_id":"bad"}`)})
	require.Error(t, err)
}

func TestNotifyIncidentProcessorSkipsWhenJiraAlreadyLinked(t *testing.T) {
	jiraKey := "OPS-1"
	store := &notifyIncidentTestStore{
		incident: db.Incident{
			ID: uuid.New(), Status: "open", JiraIssueKey: &jiraKey, CreatedAt: time.Now(),
		},
	}
	p := NewNotifyIncidentProcessor(nil, store, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		Payload: json.RawMessage(`{"incident_id":"` + store.incident.ID.String() + `"}`),
	})
	require.NoError(t, err)
	require.Zero(t, store.createCalls)
}

type createdTicketStore struct {
	*flowAlertStore
	key           string
	notifications []string
}

func (s *createdTicketStore) UpdateIncidentJiraKey(_ context.Context, _ uuid.UUID, key string) error {
	s.key = key
	return nil
}
func (s *createdTicketStore) CreateNotification(_ context.Context, _, _ uuid.UUID, status, ref string) (db.Notification, error) {
	s.notifications = append(s.notifications, status+":"+ref)
	return db.Notification{}, nil
}

// JIRA-CREATE-ASSIGN: HTTP fixtures and an in-memory store exercise ticket creation,
// assignment, notification and retry decisions, but not a live Jira or database.
func TestNotifyIncidentAssignsCreatedTicket(t *testing.T) {
	for _, tc := range []struct {
		name     string
		assignee bool
		fail     bool
	}{
		{"assigned", true, false}, {"missing assignee", false, false}, {"assignment failure", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := newFlowAlertStore(t)
			creates, assigns := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
					creates++
					_, _ = w.Write([]byte(`{"key":"OPS-9"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/user/search":
					require.Equal(t, "oncall@example.com", r.URL.Query().Get("query"))
					_, _ = w.Write([]byte(`[{"accountId":"acct"}]`))
				case r.Method == http.MethodPut && r.URL.Path == "/rest/api/3/issue/OPS-9":
					assigns++
					if tc.fail {
						w.WriteHeader(500)
					} else {
						w.WriteHeader(204)
					}
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			cfg, _ := json.Marshal(map[string]string{"base_url": server.URL, "email": "ops@example.com", "api_token": "token", "project_key": "OPS", "deployment": "cloud"})
			base.jiraIntegration.Config = cfg
			base.slackIntegration.ID = uuid.Nil
			incident := db.Incident{ID: uuid.New(), TeamID: base.teamID, Status: "open", Title: "CPU", CreatedAt: time.Now()}
			if tc.assignee {
				incident.AssigneeID = &base.assigneeID
			}
			base.incidentForNotify = incident
			store := &createdTicketStore{flowAlertStore: base}
			require.NoError(t, notifyIncidentIntegrations(t.Context(), nil, store, incident, "http://localhost"))
			require.Equal(t, "OPS-9", store.key)
			require.Contains(t, store.notifications, "sent:OPS-9")
			require.Equal(t, 1, creates)
			if tc.assignee {
				require.Equal(t, 1, assigns)
			} else {
				require.Zero(t, assigns)
			}
			incident.JiraIssueKey = &store.key
			require.NoError(t, notifyIncidentIntegrations(t.Context(), nil, store, incident, "http://localhost"))
			require.Equal(t, 1, creates)
		})
	}
}

// JIRA-CREATE-ASSIGN: A handoff during creation or an in-flight update must leave
// the latest assignee in Jira. HTTP fixtures and the in-memory store model the
// interleavings; they do not exercise a real Jira or database transaction.
func TestNotifyIncidentReconcilesHandoffDuringCreationOrAssignment(t *testing.T) {
	for _, timing := range []string{"during creation", "during assignment"} {
		t.Run(timing, func(t *testing.T) {
			base := newFlowAlertStore(t)
			oldID, newID := uuid.New(), uuid.New()
			incident := db.Incident{ID: uuid.New(), TeamID: base.teamID, Status: "open", Title: "CPU", CreatedAt: time.Now(), AssigneeID: &oldID}
			store := &handoffDuringCreateStore{flowAlertStore: base, incident: incident, oldID: oldID, newID: newID}
			creates := 0
			var assigned []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
					creates++
					if timing == "during creation" {
						store.incident.AssigneeID = &newID
					}
					_, _ = w.Write([]byte(`{"key":"OPS-9"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/user/search":
					_ = json.NewEncoder(w).Encode([]map[string]string{{"accountId": r.URL.Query().Get("query")}})
				case r.Method == http.MethodPut && r.URL.Path == "/rest/api/3/issue/OPS-9":
					var body struct {
						Fields struct {
							Assignee struct {
								AccountID string `json:"accountId"`
							} `json:"assignee"`
						} `json:"fields"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					assigned = append(assigned, body.Fields.Assignee.AccountID)
					if timing == "during assignment" && len(assigned) == 1 {
						store.incident.AssigneeID = &newID
					}
					w.WriteHeader(204)
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			cfg, _ := json.Marshal(map[string]string{"base_url": server.URL, "email": "ops@example.com", "api_token": "token", "project_key": "OPS", "deployment": "cloud"})
			base.jiraIntegration.Config = cfg
			base.slackIntegration.ID = uuid.Nil
			require.NoError(t, notifyIncidentIntegrations(t.Context(), nil, store, incident, "http://localhost"))
			require.Equal(t, 1, creates)
			require.Equal(t, "OPS-9", *store.incident.JiraIssueKey)
			if timing == "during creation" {
				require.Equal(t, []string{"new@example.com"}, assigned)
			} else {
				require.Equal(t, []string{"old@example.com", "new@example.com"}, assigned)
			}
		})
	}
}

type handoffDuringCreateStore struct {
	*flowAlertStore
	incident     db.Incident
	oldID, newID uuid.UUID
}

func (s *handoffDuringCreateStore) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	return s.incident, nil
}
func (s *handoffDuringCreateStore) UpdateIncidentJiraKey(_ context.Context, _ uuid.UUID, key string) error {
	s.incident.JiraIssueKey = &key
	return nil
}
func (s *handoffDuringCreateStore) GetUserByID(_ context.Context, id uuid.UUID) (db.User, error) {
	if id == s.oldID {
		return db.User{ID: id, Email: "old@example.com"}, nil
	}
	if id == s.newID {
		return db.User{ID: id, Email: "new@example.com"}, nil
	}
	return db.User{}, pgx.ErrNoRows
}
