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
	openID := uuid.New()
	store := &alertMockStore{
		linkedIncident: db.Incident{ID: openID, Status: "open"},
		alertID:        alertID,
		alert:          db.Alert{ID: alertID, Status: "firing", Labels: []byte(`{"team":"platform"}`)},
		manualErr:      db.ErrAlertAlreadyLinked,
	}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute)
	err := p.Handle(context.Background(), Job{
		ID:      "job-1",
		Kind:    "process_alert",
		Payload: json.RawMessage(`{"alert_id":"` + alertID.String() + `"}`),
	})
	require.NoError(t, err)
}

func TestAlertProcessorInvalidPayload(t *testing.T) {
	p := NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute)
	err := p.Handle(context.Background(), Job{ID: "1", Payload: json.RawMessage(`{`)})
	require.Error(t, err)
}

func TestAlertProcessorInvalidAlertID(t *testing.T) {
	p := NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute)
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

func noopNotifyIncident() *NotifyIncidentProcessor {
	return NewNotifyIncidentProcessor(nil, notifyIncidentMockStore{}, "http://localhost:8080")
}

type notifyIncidentMockStore struct {
	incident db.Incident
}

func (notifyIncidentMockStore) GetIncidentByID(_ context.Context, _ uuid.UUID) (db.Incident, error) {
	return db.Incident{ID: uuid.New(), TeamID: uuid.New(), Status: "open", Title: "CPU", Severity: "critical", CreatedAt: time.Now()}, nil
}
func (notifyIncidentMockStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}
func (notifyIncidentMockStore) UpdateIncidentJiraKey(context.Context, uuid.UUID, string) error { return nil }
func (notifyIncidentMockStore) AppendTimelineEvent(context.Context, uuid.UUID, string, *uuid.UUID, []byte) error {
	return nil
}
func (notifyIncidentMockStore) CreateNotification(context.Context, uuid.UUID, uuid.UUID, string, string) (db.Notification, error) {
	return db.Notification{}, nil
}
func (notifyIncidentMockStore) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (notifyIncidentMockStore) GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error) {
	return db.Integration{}, pgx.ErrNoRows
}
func (notifyIncidentMockStore) HasNotification(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}
func (notifyIncidentMockStore) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	return db.Integration{}, pgx.ErrNoRows
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
func (handoffNotifyMockStore) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (handoffNotifyMockStore) GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error) {
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
	teamID         uuid.UUID
	linkedIncident db.Incident
	alertID        uuid.UUID
	alert          db.Alert
	getAlertErr    error
	manualErr      error
	ensureErr      error
}

func (m *alertMockStore) GetOpenIncidentForAlert(_ context.Context, alertID uuid.UUID) (db.Incident, error) {
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
	return db.Alert{Status: "firing"}, nil
}
func (m *alertMockStore) ManualCreateFromAlert(context.Context, db.ManualCreateFromAlertInput) (db.ManualCreateFromAlertResult, error) {
	if m.manualErr != nil {
		return db.ManualCreateFromAlertResult{}, m.manualErr
	}
	return db.ManualCreateFromAlertResult{Incident: db.Incident{ID: uuid.New(), Status: "open"}, Created: true}, nil
}
func (m *alertMockStore) EnsureIncidentPostCreateJobs(context.Context, uuid.UUID, time.Time) error {
	return m.ensureErr
}
func (m *alertMockStore) ListRoutingRules(context.Context) ([]db.RoutingRule, error) {
	teamID := m.teamID
	if teamID == uuid.Nil {
		teamID = uuid.New()
	}
	matchLabels, _ := json.Marshal(map[string]string{"team": "platform"})
	return []db.RoutingRule{{TeamID: teamID, MatchLabels: matchLabels, Priority: 1}}, nil
}
func (m *alertMockStore) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	return nil, nil
}

func TestWorkerNoJob(t *testing.T) {
	w := NewWorker(nil, &mockStore{claim: false}, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute), noopMaterialise(), noopEscalate(), noopHandoffNotify(), noopNotifyIncident())
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

func TestAlertProcessorEnsurePostCreateJobsError(t *testing.T) {
	alertID := uuid.New()
	openID := uuid.New()
	store := &alertMockStore{
		linkedIncident: db.Incident{ID: openID, Status: "open"},
		alertID:        alertID,
		alert:          db.Alert{ID: alertID, Status: "firing", Labels: []byte(`{"team":"platform"}`)},
		manualErr:      db.ErrAlertAlreadyLinked,
		ensureErr:      errors.New("ensure failed"),
	}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute)
	err := p.Handle(context.Background(), Job{
		Payload: json.RawMessage(`{"alert_id":"` + alertID.String() + `"}`),
	})
	require.Error(t, err)
}

func TestDecodeAlertLabelsDefaultsNilMap(t *testing.T) {
	labels, err := decodeAlertLabels([]byte(`null`))
	require.NoError(t, err)
	require.NotNil(t, labels)
	require.Empty(t, labels)
}

func TestDecodeAlertLabelsInvalidJSON(t *testing.T) {
	_, err := decodeAlertLabels([]byte(`{`))
	require.Error(t, err)
}

func TestAlertProcessorNonFiringAlertNoops(t *testing.T) {
	alertID := uuid.New()
	store := &alertMockStore{
		alert: db.Alert{ID: alertID, Status: "resolved", Labels: []byte(`{"team":"platform"}`)},
	}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute)
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"alert_id":"` + alertID.String() + `"}`)})
	require.NoError(t, err)
}

func TestWorkerProcessesJob(t *testing.T) {
	alertID := uuid.New()
	store := &mockStore{
		claim: true,
		job:   Job{ID: "j1", Kind: "process_alert", Payload: json.RawMessage(`{"alert_id":"` + alertID.String() + `"}`)},
	}
	alertStore := &alertMockStore{
		linkedIncident: db.Incident{ID: uuid.New(), Status: "open"},
		alertID:        alertID,
		alert:          db.Alert{ID: alertID, Status: "firing", Labels: []byte(`{"team":"platform"}`)},
		manualErr:      db.ErrAlertAlreadyLinked,
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil, alertStore, time.Hour, time.Minute), noopMaterialise(), noopEscalate(), noopHandoffNotify(), noopNotifyIncident())
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
	w := NewWorker(nil, store, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute), NewMaterialiseProcessor(nil, &materialiseMockStore{}), noopEscalate(), noopHandoffNotify(), noopNotifyIncident())
	require.NoError(t, w.RunOnce(context.Background()))
}

type claimErrorStore struct {
	mockStore
}

func (m *claimErrorStore) ClaimNextJob(ctx context.Context) (bool, Job, error) {
	return false, Job{}, errors.New("claim failed")
}

func TestWorkerClaimError(t *testing.T) {
	w := NewWorker(nil, &claimErrorStore{}, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute), noopMaterialise(), noopEscalate(), noopHandoffNotify(), noopNotifyIncident())
	err := w.RunOnce(context.Background())
	require.Error(t, err)
}

func TestAlertProcessorGetAlertError(t *testing.T) {
	store := &alertMockStore{getAlertErr: errors.New("db down")}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute)
	err := p.Handle(context.Background(), Job{
		ID: "j1", Payload: json.RawMessage(`{"alert_id":"` + uuid.New().String() + `"}`),
	})
	require.Error(t, err)
}

func TestAlertProcessorInvalidLabelsJSON(t *testing.T) {
	store := &alertMockStore{alert: db.Alert{ID: uuid.New(), Status: "firing", Labels: []byte(`{`)}}
	p := NewAlertProcessor(nil, store, time.Hour, time.Minute)
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

func TestWorkerProcessesNotifyIncidentJob(t *testing.T) {
	incidentID := uuid.New()
	store := &mockStore{
		claim: true,
		job: Job{
			ID:      "j1",
			Kind:    "notify_incident",
			Payload: json.RawMessage(`{"incident_id":"` + incidentID.String() + `"}`),
		},
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute), noopMaterialise(), noopEscalate(), noopHandoffNotify(), noopNotifyIncident())
	require.NoError(t, w.RunOnce(context.Background()))
}

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
	w := NewWorker(nil, store, NewAlertProcessor(nil, &alertMockStore{}, time.Hour, time.Minute), noopMaterialise(), NewEscalateProcessor(nil, escalateMockStore{incident: db.Incident{ID: incidentID, Status: "acknowledged"}}, ""), noopHandoffNotify(), noopNotifyIncident())
	require.NoError(t, w.RunOnce(context.Background()))
}
