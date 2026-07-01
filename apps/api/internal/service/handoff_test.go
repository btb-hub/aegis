package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type handoffMockRepo struct {
	incident  db.Incident
	team      db.Team
	onCall    []db.OnCallUser
	onCallErr error
	events    []db.TimelineEvent
	stats     db.HandoffStats
	statsErr  error
	handoffErr error
	bounceErr  error
}

func (m *handoffMockRepo) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	if m.incident.ID == uuid.Nil {
		return db.Incident{}, pgx.ErrNoRows
	}
	return m.incident, nil
}

func (m *handoffMockRepo) GetTeam(context.Context, uuid.UUID) (db.Team, error) {
	if m.team.ID == uuid.Nil {
		return db.Team{}, pgx.ErrNoRows
	}
	return m.team, nil
}

func (m *handoffMockRepo) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	if m.onCallErr != nil {
		return nil, m.onCallErr
	}
	return m.onCall, nil
}

func (m *handoffMockRepo) HandoffIncident(_ context.Context, input db.HandoffIncidentInput) (db.Incident, db.Handoff, error) {
	if m.handoffErr != nil {
		return db.Incident{}, db.Handoff{}, m.handoffErr
	}
	assignee := input.ToUserID
	m.incident.AssigneeID = &assignee
	return m.incident, db.Handoff{ID: uuid.New(), IncidentID: input.IncidentID}, nil
}

func (m *handoffMockRepo) BounceIncident(context.Context, db.BounceIncidentInput) (db.Incident, error) {
	if m.bounceErr != nil {
		return db.Incident{}, m.bounceErr
	}
	return m.incident, nil
}

func (m *handoffMockRepo) EnqueueHandoffNotify(context.Context, uuid.UUID) error { return nil }

func (m *handoffMockRepo) HandoffStats(context.Context, time.Time, time.Time) (db.HandoffStats, error) {
	if m.statsErr != nil {
		return db.HandoffStats{}, m.statsErr
	}
	return m.stats, nil
}

func (m *handoffMockRepo) ListTimelineEvents(context.Context, uuid.UUID) ([]db.TimelineEvent, error) {
	return m.events, nil
}

func TestHandoffServiceHandoff(t *testing.T) {
	incidentID := uuid.New()
	teamID := uuid.New()
	l3User := uuid.New()
	svc := NewHandoffService(&handoffMockRepo{
		incident: db.Incident{ID: incidentID, TeamID: teamID, Status: "open"},
		team:     db.Team{ID: teamID, Name: "L3"},
		onCall:   []db.OnCallUser{{UserID: l3User, Email: "l3@example.com"}},
	})
	incident, err := svc.Handoff(context.Background(), incidentID, uuid.New(), teamID, "needs deep dive")
	require.NoError(t, err)
	require.Equal(t, l3User, *incident.AssigneeID)
}

func TestHandoffServiceHandoffNoOnCall(t *testing.T) {
	incidentID := uuid.New()
	teamID := uuid.New()
	svc := NewHandoffService(&handoffMockRepo{
		incident: db.Incident{ID: incidentID, Status: "open"},
		team:     db.Team{ID: teamID},
	})
	_, err := svc.Handoff(context.Background(), incidentID, uuid.New(), teamID, "")
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestHandoffServiceBounceRequiresNote(t *testing.T) {
	svc := NewHandoffService(&handoffMockRepo{
		incident: db.Incident{ID: uuid.New(), Status: "acknowledged"},
	})
	_, err := svc.Bounce(context.Background(), uuid.New(), uuid.New(), "  ")
	require.Error(t, err)
}

func TestHandoffServiceBounceSuccess(t *testing.T) {
	incidentID := uuid.New()
	svc := NewHandoffService(&handoffMockRepo{
		incident: db.Incident{ID: incidentID, Status: "acknowledged"},
	})
	incident, err := svc.Bounce(context.Background(), incidentID, uuid.New(), "needs L2 context")
	require.NoError(t, err)
	require.Equal(t, incidentID, incident.ID)
}

func TestHandoffServiceStatsInvalidRange(t *testing.T) {
	svc := NewHandoffService(&handoffMockRepo{})
	now := time.Now()
	_, err := svc.Stats(context.Background(), now, now)
	require.Error(t, err)
}

func TestHandoffServiceHandoffUnknownTeam(t *testing.T) {
	incidentID := uuid.New()
	svc := NewHandoffService(&handoffMockRepo{
		incident: db.Incident{ID: incidentID, Status: "open"},
	})
	_, err := svc.Handoff(context.Background(), incidentID, uuid.New(), uuid.New(), "note")
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestHandoffServiceHandoffConflict(t *testing.T) {
	incidentID := uuid.New()
	teamID := uuid.New()
	svc := NewHandoffService(&handoffMockRepo{
		incident:   db.Incident{ID: incidentID, Status: "resolved"},
		team:       db.Team{ID: teamID, Name: "L3"},
		onCall:     []db.OnCallUser{{UserID: uuid.New()}},
		handoffErr: pgx.ErrNoRows,
	})
	_, err := svc.Handoff(context.Background(), incidentID, uuid.New(), teamID, "note")
	require.Error(t, err)
}

func TestHandoffServiceBounceConflict(t *testing.T) {
	svc := NewHandoffService(&handoffMockRepo{
		incident:  db.Incident{ID: uuid.New(), Status: "acknowledged"},
		bounceErr: pgx.ErrNoRows,
	})
	_, err := svc.Bounce(context.Background(), uuid.New(), uuid.New(), "reason")
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestHandoffServiceHandoffUnknownIncident(t *testing.T) {
	svc := NewHandoffService(&handoffMockRepo{team: db.Team{ID: uuid.New(), Name: "L3"}})
	_, err := svc.Handoff(context.Background(), uuid.New(), uuid.New(), uuid.New(), "note")
	require.Error(t, err)
}

func TestHandoffServiceHandoffOnCallError(t *testing.T) {
	incidentID := uuid.New()
	teamID := uuid.New()
	svc := NewHandoffService(&handoffMockRepo{
		incident:  db.Incident{ID: incidentID, Status: "open"},
		team:      db.Team{ID: teamID, Name: "L3"},
		onCallErr: errors.New("db down"),
	})
	_, err := svc.Handoff(context.Background(), incidentID, uuid.New(), teamID, "note")
	require.Error(t, err)
}

func TestHandoffServiceStatsError(t *testing.T) {
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	svc := NewHandoffService(&handoffMockRepo{statsErr: errors.New("db down")})
	_, err := svc.Stats(context.Background(), from, to)
	require.Error(t, err)
}

func TestHandoffServiceTimelineNotFound(t *testing.T) {
	svc := NewHandoffService(&handoffMockRepo{})
	_, err := svc.Timeline(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestHandoffServiceTimelineSharedForAllRoles(t *testing.T) {
	incidentID := uuid.New()
	events := []db.TimelineEvent{
		{ID: uuid.New(), Kind: "created"},
		{ID: uuid.New(), Kind: "handoff"},
		{ID: uuid.New(), Kind: "paged"},
	}
	svc := NewHandoffService(&handoffMockRepo{
		incident: db.Incident{ID: incidentID, Status: "open"},
		events:   events,
	})
	got, err := svc.Timeline(context.Background(), incidentID)
	require.NoError(t, err)
	require.Len(t, got, 3)
}

func TestHandoffServiceStats(t *testing.T) {
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	svc := NewHandoffService(&handoffMockRepo{
		stats: db.HandoffStats{Count: 2, MedianResponseSeconds: 120},
	})
	stats, err := svc.Stats(context.Background(), from, to)
	require.NoError(t, err)
	require.Equal(t, 2, stats.Count)
	require.Equal(t, 120.0, stats.MedianResponseSeconds)
}
