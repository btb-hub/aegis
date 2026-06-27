package processor

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type failStore struct {
	mockStore
}

func (f *failStore) FailJob(ctx context.Context, id, message string) error {
	return errors.New("stored failure")
}

func TestWorkerUnknownKind(t *testing.T) {
	store := &mockStore{
		claim: true,
		job:   Job{ID: "j1", Kind: "unknown", Payload: json.RawMessage(`{}`)},
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute, ""), noopMaterialise(), noopEscalate())
	err := w.RunOnce(context.Background())
	require.Error(t, err)
}

func TestWorkerHandlerFailure(t *testing.T) {
	store := &failStore{
		mockStore: mockStore{
			claim: true,
			job:   Job{ID: "j1", Kind: "process_alert", Payload: json.RawMessage(`{`)},
		},
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute, ""), noopMaterialise(), noopEscalate())
	err := w.RunOnce(context.Background())
	require.Error(t, err)
}

func TestEscalateProcessorSkipsAcknowledgedIncident(t *testing.T) {
	incidentID := uuid.New()
	store := escalateMockStore{incident: db.Incident{ID: incidentID, Status: "acknowledged"}}
	p := NewEscalateProcessor(nil, store, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID:      "j1",
		Kind:    "escalate_incident",
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	})
	require.NoError(t, err)
}

type escalateMockStore struct {
	incident db.Incident
	user     db.User
	userErr  error
	listErr  error
}

func (m escalateMockStore) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	if m.incident.ID == uuid.Nil {
		return db.Incident{}, pgx.ErrNoRows
	}
	return m.incident, nil
}

func (m escalateMockStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	if m.userErr != nil {
		return db.User{}, m.userErr
	}
	return m.user, nil
}

func (m escalateMockStore) AppendTimelineEvent(context.Context, uuid.UUID, string, *uuid.UUID, []byte) error {
	return nil
}

func (m escalateMockStore) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	return db.Integration{}, pgx.ErrNoRows
}

func (m escalateMockStore) CreateNotification(context.Context, uuid.UUID, uuid.UUID, string, string) (db.Notification, error) {
	return db.Notification{}, nil
}

func (m escalateMockStore) ListEnabledIntegrations(context.Context) ([]integrations.IntegrationRow, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return nil, nil
}
