package service

import (
	"context"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type incidentMockRepo struct {
	incident db.Incident
	events   []db.TimelineEvent
	alerts   []db.Alert
	user     db.User
}

func (m *incidentMockRepo) ListIncidents(context.Context, string) ([]db.Incident, error) {
	return []db.Incident{m.incident}, nil
}
func (m *incidentMockRepo) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	if m.incident.ID == uuid.Nil {
		return db.Incident{}, pgx.ErrNoRows
	}
	return m.incident, nil
}
func (m *incidentMockRepo) AcknowledgeIncident(_ context.Context, incidentID, actorID uuid.UUID) (db.Incident, error) {
	if m.incident.Status != "open" {
		return db.Incident{}, pgx.ErrNoRows
	}
	now := time.Now()
	m.incident.Status = "acknowledged"
	m.incident.AcknowledgedAt = &now
	return m.incident, nil
}
func (m *incidentMockRepo) ResolveIncident(_ context.Context, incidentID, actorID uuid.UUID) (db.Incident, error) {
	if m.incident.Status == "resolved" {
		return db.Incident{}, pgx.ErrNoRows
	}
	now := time.Now()
	m.incident.Status = "resolved"
	m.incident.ResolvedAt = &now
	return m.incident, nil
}
func (m *incidentMockRepo) ListTimelineEvents(context.Context, uuid.UUID) ([]db.TimelineEvent, error) {
	return m.events, nil
}
func (m *incidentMockRepo) ListAlertsForIncident(context.Context, uuid.UUID) ([]db.Alert, error) {
	return m.alerts, nil
}
func (m *incidentMockRepo) CancelEscalationJobs(context.Context, uuid.UUID) error { return nil }
func (m *incidentMockRepo) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return m.user, nil
}
func (m *incidentMockRepo) GetUserBySlackID(context.Context, string) (db.User, error) {
	if m.user.ID == uuid.Nil {
		return db.User{}, pgx.ErrNoRows
	}
	return m.user, nil
}
func (m *incidentMockRepo) GetUserByExpressHuid(_ context.Context, expressHuid uuid.UUID) (db.User, error) {
	if m.user.ID == uuid.Nil || !m.user.ExpressUserHuid.Valid || uuid.UUID(m.user.ExpressUserHuid.Bytes) != expressHuid {
		return db.User{}, pgx.ErrNoRows
	}
	return m.user, nil
}

func (m *incidentMockRepo) GetAlertByID(context.Context, uuid.UUID) (db.Alert, error) {
	return db.Alert{}, pgx.ErrNoRows
}
func (m *incidentMockRepo) GetTeam(context.Context, uuid.UUID) (db.Team, error) {
	return db.Team{}, pgx.ErrNoRows
}
func (m *incidentMockRepo) TeamMemberUserIDs(context.Context, uuid.UUID) (map[uuid.UUID]struct{}, error) {
	return nil, nil
}
func (m *incidentMockRepo) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	return nil, nil
}
func (m *incidentMockRepo) ManualCreateFromAlert(context.Context, db.ManualCreateFromAlertInput) (db.ManualCreateFromAlertResult, error) {
	return db.ManualCreateFromAlertResult{}, nil
}

func TestIncidentServiceAcknowledge(t *testing.T) {
	incidentID := uuid.New()
	actorID := uuid.New()
	svc := NewIncidentService(&incidentMockRepo{incident: db.Incident{ID: incidentID, Status: "open"}}, time.Hour, time.Minute)
	incident, err := svc.Acknowledge(context.Background(), incidentID, actorID)
	require.NoError(t, err)
	require.Equal(t, "acknowledged", incident.Status)
}

func TestIncidentServiceAcknowledgeConflict(t *testing.T) {
	incidentID := uuid.New()
	svc := NewIncidentService(&incidentMockRepo{incident: db.Incident{ID: incidentID, Status: "acknowledged"}}, time.Hour, time.Minute)
	_, err := svc.Acknowledge(context.Background(), incidentID, uuid.New())
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestIncidentServiceAcknowledgeBySlackUser(t *testing.T) {
	incidentID := uuid.New()
	userID := uuid.New()
	svc := NewIncidentService(&incidentMockRepo{
		incident: db.Incident{ID: incidentID, Status: "open"},
		user:     db.User{ID: userID, SlackUserID: strPtr("U123")},
	}, time.Hour, time.Minute)
	incident, err := svc.AcknowledgeBySlackUser(context.Background(), incidentID, "U123")
	require.NoError(t, err)
	require.Equal(t, "acknowledged", incident.Status)
}

func TestIncidentServiceAcknowledgeByExpressHuid(t *testing.T) {
	incidentID := uuid.New()
	userID := uuid.New()
	huid := uuid.MustParse("6fafda2c-6505-57a5-a088-25ea5d1d0364")
	svc := NewIncidentService(&incidentMockRepo{
		incident: db.Incident{ID: incidentID, Status: "open"},
		user:     db.User{ID: userID, ExpressUserHuid: db.ExpressHuidToPg(huid)},
	}, time.Hour, time.Minute)
	incident, err := svc.AcknowledgeByExpressHuid(context.Background(), incidentID, huid.String())
	require.NoError(t, err)
	require.Equal(t, "acknowledged", incident.Status)
}

func TestIncidentServiceAcknowledgeByExpressHuidInvalidHuid(t *testing.T) {
	svc := NewIncidentService(&incidentMockRepo{
		incident: db.Incident{ID: uuid.New(), Status: "open"},
		user:     db.User{ID: uuid.New(), ExpressUserHuid: db.ExpressHuidToPg(uuid.New())},
	}, time.Hour, time.Minute)
	_, err := svc.AcknowledgeByExpressHuid(context.Background(), uuid.New(), "not-a-uuid")
	require.Error(t, err)
}

func TestIncidentServiceAcknowledgeByExpressHuidUserNotFound(t *testing.T) {
	huid := uuid.MustParse("6fafda2c-6505-57a5-a088-25ea5d1d0364")
	svc := NewIncidentService(&incidentMockRepo{
		incident: db.Incident{ID: uuid.New(), Status: "open"},
	}, time.Hour, time.Minute)
	_, err := svc.AcknowledgeByExpressHuid(context.Background(), uuid.New(), huid.String())
	require.Error(t, err)
}

func TestIncidentJSON(t *testing.T) {
	assignee := uuid.New()
	key := "OPS-1"
	out := IncidentJSON(db.Incident{
		ID: uuid.New(), TeamID: uuid.New(), AssigneeID: &assignee,
		Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp",
		JiraIssueKey: &key,
	})
	require.Equal(t, "OPS-1", out["jira_issue_key"])
}

func strPtr(v string) *string { return &v }
