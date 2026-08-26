package processor

import (
	"context"
	"encoding/json"
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
func (s *notifyIncidentTestStore) UpdateIncidentJiraKey(context.Context, uuid.UUID, string) error { return nil }
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
