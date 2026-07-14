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

func TestAlertProcessorIdempotentWhenLinked(t *testing.T) {
	alertID := uuid.New()
	store := &alertMockStore{
		linkedIncident: db.Incident{ID: uuid.New()},
		alertID:        alertID,
	}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{
		ID:      "job-1",
		Kind:    "process_alert",
		Payload: json.RawMessage(`{"alert_id":"` + alertID.String() + `"}`),
	})
	require.NoError(t, err)
}

func TestAlertProcessorInvalidPayload(t *testing.T) {
	p := NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{ID: "1", Payload: json.RawMessage(`{`)})
	require.Error(t, err)
}

func TestAlertProcessorInvalidAlertID(t *testing.T) {
	p := NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute, "http://localhost:8080")
	err := p.Handle(context.Background(), Job{ID: "1", Payload: json.RawMessage(`{"alert_id":"bad"}`)})
	require.Error(t, err)
}

func TestKindFor(t *testing.T) {
	require.Equal(t, "process_alert", KindFor(Job{Kind: "process_alert"}))
}

func noopMaterialise() *MaterialiseProcessor {
	return NewMaterialiseProcessor(nil, materialiseStoreStub{})
}

func noopEscalate() *EscalateProcessor {
	return NewEscalateProcessor(nil, escalateMockStore{}, "http://localhost:8080")
}

func noopHandoffNotify() *HandoffNotifyProcessor {
	return NewHandoffNotifyProcessor(nil, handoffNotifyMockStore{}, "http://localhost:8080")
}

type handoffNotifyMockStore struct{}

func (handoffNotifyMockStore) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	return db.Incident{}, pgx.ErrNoRows
}
func (handoffNotifyMockStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}
func (handoffNotifyMockStore) ListEnabledIntegrations(context.Context) ([]integrations.IntegrationRow, error) {
	return nil, nil
}
func (handoffNotifyMockStore) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	return db.Integration{}, pgx.ErrNoRows
}
func (handoffNotifyMockStore) CreateNotification(context.Context, uuid.UUID, uuid.UUID, string, string) (db.Notification, error) {
	return db.Notification{}, nil
}
func (handoffNotifyMockStore) AppendTimelineEvent(context.Context, uuid.UUID, string, *uuid.UUID, []byte) error {
	return nil
}

type materialiseStoreStub struct{}

func (materialiseStoreStub) MaterialiseOnCallForTeam(ctx context.Context, teamID uuid.UUID) error {
	return nil
}
func (materialiseStoreStub) ListTeamIDsWithSchedules(ctx context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

type alertMockStore struct {
	linkedIncident db.Incident
	alertID        uuid.UUID
	alert          db.Alert
	getAlertErr    error
	findErr        error
}

func (m *alertMockStore) GetIncidentForAlert(_ context.Context, alertID uuid.UUID) (db.Incident, error) {
	if m.linkedIncident.ID != uuid.Nil && alertID == m.alertID {
		return m.linkedIncident, nil
	}
	return db.Incident{}, pgx.ErrNoRows
}

func (m *alertMockStore) GetAlertByID(context.Context, uuid.UUID) (db.Alert, error) {
	if m.getAlertErr != nil {
		return db.Alert{}, m.getAlertErr
	}
	if m.alert.ID != uuid.Nil {
		return m.alert, nil
	}
	return db.Alert{}, nil
}
func (m *alertMockStore) FindOpenIncidentByFingerprint(context.Context, string, time.Time) (db.Incident, error) {
	if m.findErr != nil {
		return db.Incident{}, m.findErr
	}
	return db.Incident{}, pgx.ErrNoRows
}
func (m *alertMockStore) CreateIncidentWithAlert(context.Context, db.CreateIncidentInput) (db.Incident, error) {
	return db.Incident{}, nil
}
func (m *alertMockStore) LinkAlertToIncident(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (m *alertMockStore) ListRoutingRules(context.Context) ([]db.RoutingRule, error)      { return nil, nil }
func (m *alertMockStore) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	return nil, nil
}
func (m *alertMockStore) UpdateIncidentJiraKey(context.Context, uuid.UUID, string) error { return nil }
func (m *alertMockStore) AppendTimelineEvent(context.Context, uuid.UUID, string, *uuid.UUID, []byte) error {
	return nil
}
func (m *alertMockStore) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	return db.Integration{}, pgx.ErrNoRows
}
func (m *alertMockStore) CreateNotification(context.Context, uuid.UUID, uuid.UUID, string, string) (db.Notification, error) {
	return db.Notification{}, nil
}
func (m *alertMockStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return db.User{}, nil
}
func (m *alertMockStore) EnqueueEscalation(context.Context, uuid.UUID, time.Time) error { return nil }
func (m *alertMockStore) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *alertMockStore) GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error) {
	return db.Integration{}, pgx.ErrNoRows
}
func (m *alertMockStore) ListEnabledIntegrationsForWorkspace(context.Context, uuid.UUID) ([]integrations.IntegrationRow, error) {
	return nil, nil
}

func TestWorkerNoJob(t *testing.T) {
	w := NewWorker(nil, &mockStore{claim: false}, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute, ""), noopMaterialise(), noopEscalate(), noopHandoffNotify())
	err := w.RunOnce(context.Background())
	require.NoError(t, err)
}

type mockStore struct {
	claim bool
	job   Job
}

func (m *mockStore) ClaimNextJob(ctx context.Context) (bool, Job, error) {
	if !m.claim {
		return false, Job{}, nil
	}
	return true, m.job, nil
}
func (m *mockStore) CompleteJob(ctx context.Context, id string) error { return nil }
func (m *mockStore) FailJob(ctx context.Context, id, message string) error {
	return errors.New("fail")
}

func TestWorkerProcessesJob(t *testing.T) {
	alertID := uuid.New()
	store := &mockStore{
		claim: true,
		job:   Job{ID: "j1", Kind: "process_alert", Payload: json.RawMessage(`{"alert_id":"` + alertID.String() + `"}`)},
	}
	alertStore := &alertMockStore{linkedIncident: db.Incident{ID: uuid.New()}, alertID: alertID}
	w := NewWorker(nil, store, NewAlertProcessor(nil, alertStore, time.Hour, time.Minute, ""), noopMaterialise(), noopEscalate(), noopHandoffNotify())
	require.NoError(t, w.RunOnce(context.Background()))
}

func TestWorkerProcessesMaterialiseJob(t *testing.T) {
	teamID := uuid.New()
	store := &mockStore{
		claim: true,
		job: Job{
			ID:      "j1",
			Kind:    "materialise_oncall",
			Payload: json.RawMessage(`{"team_id":"` + teamID.String() + `"}`),
		},
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute, ""), NewMaterialiseProcessor(nil, &materialiseMockStore{}), noopEscalate(), noopHandoffNotify())
	require.NoError(t, w.RunOnce(context.Background()))
}

type claimErrorStore struct {
	mockStore
}

func (m *claimErrorStore) ClaimNextJob(ctx context.Context) (bool, Job, error) {
	return false, Job{}, errors.New("claim failed")
}

func TestWorkerClaimError(t *testing.T) {
	w := NewWorker(nil, &claimErrorStore{}, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute, ""), noopMaterialise(), noopEscalate(), noopHandoffNotify())
	err := w.RunOnce(context.Background())
	require.Error(t, err)
}

func TestAlertProcessorGetAlertError(t *testing.T) {
	store := &alertMockStore{getAlertErr: errors.New("db down")}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Payload: json.RawMessage(`{"alert_id":"` + uuid.New().String() + `"}`),
	})
	require.Error(t, err)
}

func TestAlertProcessorInvalidLabelsJSON(t *testing.T) {
	store := &alertMockStore{alert: db.Alert{ID: uuid.New(), Status: "firing", Labels: []byte(`{`)}}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute, "")
	err := p.Handle(context.Background(), Job{
		ID: "j1", Payload: json.RawMessage(`{"alert_id":"` + store.alert.ID.String() + `"}`),
	})
	require.Error(t, err)
}

func TestEscalateProcessorInvalidPayload(t *testing.T) {
	p := NewEscalateProcessor(nil, escalateMockStore{}, "")
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{`)}))
}

func TestEscalateProcessorGetUserError(t *testing.T) {
	assignee := uuid.New()
	incidentID := uuid.New()
	store := escalateMockStore{
		incident: db.Incident{ID: incidentID, Status: "open", AssigneeID: &assignee, Title: "CPU", Severity: "critical", CreatedAt: time.Now()},
		userErr:  errors.New("missing user"),
	}
	p := NewEscalateProcessor(nil, store, "")
	err := p.Handle(context.Background(), Job{
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	})
	require.Error(t, err)
}

func TestEscalateProcessorLoadRegistryError(t *testing.T) {
	assignee := uuid.New()
	incidentID := uuid.New()
	store := escalateMockStore{
		incident: db.Incident{ID: incidentID, Status: "open", AssigneeID: &assignee, Title: "CPU", Severity: "critical", CreatedAt: time.Now()},
		user:     db.User{ID: assignee, Locale: "en", SlackUserID: strPtr("U1")},
		listErr:  errors.New("db down"),
	}
	p := NewEscalateProcessor(nil, store, "")
	require.Error(t, p.Handle(context.Background(), Job{
		Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
	}))
}

func strPtr(v string) *string { return &v }

func TestWorkerProcessesEscalateJob(t *testing.T) {
	incidentID := uuid.New()
	store := &mockStore{
		claim: true,
		job: Job{
			ID:      "j1",
			Kind:    "escalate_incident",
			Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
		},
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute, ""), noopMaterialise(), NewEscalateProcessor(nil, escalateMockStore{incident: db.Incident{ID: incidentID, Status: "acknowledged"}}, ""), noopHandoffNotify())
	require.NoError(t, w.RunOnce(context.Background()))
}
