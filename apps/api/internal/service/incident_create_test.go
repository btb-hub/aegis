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

type createIncidentMockRepo struct {
	incidentMockRepo
	alert           db.Alert
	team            db.Team
	members         map[uuid.UUID]struct{}
	onCall          []db.OnCallUser
	manualResult    db.ManualCreateFromAlertResult
	manualErr       error
	enqueued        []string
	enqueueEscalateErr error
	enqueueNotifyErr   error
}

func (m *createIncidentMockRepo) GetAlertByID(context.Context, uuid.UUID) (db.Alert, error) {
	if m.alert.ID == uuid.Nil {
		return db.Alert{}, pgx.ErrNoRows
	}
	return m.alert, nil
}

func (m *createIncidentMockRepo) GetTeam(context.Context, uuid.UUID) (db.Team, error) {
	if m.team.ID == uuid.Nil {
		return db.Team{}, pgx.ErrNoRows
	}
	return m.team, nil
}

func (m *createIncidentMockRepo) TeamMemberUserIDs(context.Context, uuid.UUID) (map[uuid.UUID]struct{}, error) {
	return m.members, nil
}

func (m *createIncidentMockRepo) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	return m.onCall, nil
}

func (m *createIncidentMockRepo) GetUserByID(_ context.Context, id uuid.UUID) (db.User, error) {
	if m.user.ID == uuid.Nil || m.user.ID != id {
		return db.User{}, pgx.ErrNoRows
	}
	return m.user, nil
}

func (m *createIncidentMockRepo) ManualCreateFromAlert(_ context.Context, input db.ManualCreateFromAlertInput) (db.ManualCreateFromAlertResult, error) {
	if m.manualErr != nil {
		return db.ManualCreateFromAlertResult{}, m.manualErr
	}
	if m.enqueueEscalateErr != nil && input.PostCreate != nil {
		return db.ManualCreateFromAlertResult{}, m.enqueueEscalateErr
	}
	if m.enqueueNotifyErr != nil && input.PostCreate != nil {
		return db.ManualCreateFromAlertResult{}, m.enqueueNotifyErr
	}
	if input.PostCreate != nil && m.manualResult.Created {
		m.enqueued = append(m.enqueued,
			"escalate:"+m.manualResult.Incident.ID.String(),
			"notify:"+m.manualResult.Incident.ID.String(),
		)
	}
	return m.manualResult, nil
}

func TestIncidentServiceCreateFromAlertEnqueuesJobs(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	incidentID := uuid.New()
	repo := &createIncidentMockRepo{
		alert: db.Alert{ID: alertID, Status: "firing", Fingerprint: "fp"},
		team:  db.Team{ID: teamID, Name: "Platform"},
		manualResult: db.ManualCreateFromAlertResult{
			Incident: db.Incident{ID: incidentID, TeamID: teamID, Status: "open"},
			Created:  true,
		},
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	incident, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.NoError(t, err)
	require.Equal(t, incidentID, incident.ID)
	require.Equal(t, []string{"escalate:" + incidentID.String(), "notify:" + incidentID.String()}, repo.enqueued)
}

func TestIncidentServiceCreateFromAlertLinkOnlySkipsEnqueue(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	incidentID := uuid.New()
	repo := &createIncidentMockRepo{
		alert: db.Alert{ID: alertID, Status: "firing", Fingerprint: "fp"},
		team:  db.Team{ID: teamID, Name: "Platform"},
		manualResult: db.ManualCreateFromAlertResult{
			Incident: db.Incident{ID: incidentID, TeamID: teamID, Status: "open"},
			Created:  false,
		},
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	incident, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.NoError(t, err)
	require.Equal(t, incidentID, incident.ID)
	require.Empty(t, repo.enqueued)
}

func TestIncidentServiceCreateFromAlertAssigneeNotOnTeam(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	assigneeID := uuid.New()
	repo := &createIncidentMockRepo{
		incidentMockRepo: incidentMockRepo{user: db.User{ID: assigneeID}},
		alert:            db.Alert{ID: alertID, Status: "firing"},
		team:             db.Team{ID: teamID, Name: "Platform"},
		members:          map[uuid.UUID]struct{}{},
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, AssigneeID: &assigneeID, ActorID: uuid.New(),
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestIncidentServiceCreateFromAlertUsesOnCall(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	incidentID := uuid.New()
	onCallID := uuid.New()
	repo := &createIncidentMockRepo{
		alert:  db.Alert{ID: alertID, Status: "firing", Fingerprint: "fp"},
		team:   db.Team{ID: teamID, Name: "Platform"},
		onCall: []db.OnCallUser{{UserID: onCallID}},
		manualResult: db.ManualCreateFromAlertResult{
			Incident: db.Incident{ID: incidentID, TeamID: teamID, Status: "open", AssigneeID: &onCallID},
			Created:  true,
		},
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	incident, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.NoError(t, err)
	require.Equal(t, incidentID, incident.ID)
}

func TestIncidentServiceCreateFromAlertNotFiring(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	repo := &createIncidentMockRepo{
		alert:     db.Alert{ID: alertID, Status: "firing"},
		team:      db.Team{ID: teamID, Name: "Platform"},
		manualErr: db.ErrAlertNotFiring,
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestIncidentServiceCreateFromAlertAlreadyLinked(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	repo := &createIncidentMockRepo{
		alert:     db.Alert{ID: alertID, Status: "firing"},
		team:      db.Team{ID: teamID, Name: "Platform"},
		manualErr: db.ErrAlertAlreadyLinked,
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestIncidentServiceCreateFromAlertMissingTeam(t *testing.T) {
	svc := NewIncidentService(&createIncidentMockRepo{}, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: uuid.New(), TeamID: uuid.Nil, ActorID: uuid.New(),
	})
	require.Error(t, err)
}

func TestIncidentServiceCreateFromAlertAlertNotFound(t *testing.T) {
	teamID := uuid.New()
	repo := &createIncidentMockRepo{
		team: db.Team{ID: teamID, Name: "Platform"},
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: uuid.New(), TeamID: teamID, ActorID: uuid.New(),
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestIncidentServiceCreateFromAlertInvalidTeam(t *testing.T) {
	alertID := uuid.New()
	repo := &createIncidentMockRepo{
		alert: db.Alert{ID: alertID, Status: "firing"},
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: uuid.New(), ActorID: uuid.New(),
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestIncidentServiceCreateFromAlertInvalidAssigneeUser(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	assigneeID := uuid.New()
	repo := &createIncidentMockRepo{
		alert:   db.Alert{ID: alertID, Status: "firing"},
		team:    db.Team{ID: teamID, Name: "Platform"},
		members: map[uuid.UUID]struct{}{assigneeID: {}},
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, AssigneeID: &assigneeID, ActorID: uuid.New(),
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestIncidentServiceCreateFromAlertFingerprintTeamMismatch(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	repo := &createIncidentMockRepo{
		alert: db.Alert{ID: alertID, Status: "firing"},
		team:  db.Team{ID: teamID, Name: "Platform"},
		manualErr: &db.FingerprintTeamMismatchError{IncidentID: uuid.New()},
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "CONFLICT", appErr.Code)
	require.NotNil(t, appErr.Details["incident_id"])
}

func TestIncidentServiceCreateFromAlertEnqueueEscalationError(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	incidentID := uuid.New()
	repo := &createIncidentMockRepo{
		alert: db.Alert{ID: alertID, Status: "firing", Fingerprint: "fp"},
		team:  db.Team{ID: teamID, Name: "Platform"},
		manualResult: db.ManualCreateFromAlertResult{
			Incident: db.Incident{ID: incidentID, TeamID: teamID, Status: "open"},
			Created:  true,
		},
		enqueueEscalateErr: errors.New("enqueue failed"),
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.Error(t, err)
}

func TestIncidentServiceCreateFromAlertEnqueueNotifyError(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	incidentID := uuid.New()
	repo := &createIncidentMockRepo{
		alert: db.Alert{ID: alertID, Status: "firing", Fingerprint: "fp"},
		team:  db.Team{ID: teamID, Name: "Platform"},
		manualResult: db.ManualCreateFromAlertResult{
			Incident: db.Incident{ID: incidentID, TeamID: teamID, Status: "open"},
			Created:  true,
		},
		enqueueNotifyErr: errors.New("enqueue failed"),
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.Error(t, err)
}

func TestIncidentServiceCreateFromAlertConcurrentConflict(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	repo := &createIncidentMockRepo{
		alert: db.Alert{ID: alertID, Status: "firing", Fingerprint: "fp"},
		team:  db.Team{ID: teamID, Name: "Platform"},
		manualErr: db.ErrAlertAlreadyLinked,
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestIncidentServiceCreateFromAlertAlertDeletedInTransaction(t *testing.T) {
	alertID := uuid.New()
	teamID := uuid.New()
	repo := &createIncidentMockRepo{
		alert:     db.Alert{ID: alertID, Status: "firing", Fingerprint: "fp"},
		team:      db.Team{ID: teamID, Name: "Platform"},
		manualErr: pgx.ErrNoRows,
	}
	svc := NewIncidentService(repo, time.Hour, time.Minute)
	_, err := svc.CreateFromAlert(context.Background(), CreateFromAlertInput{
		AlertID: alertID, TeamID: teamID, ActorID: uuid.New(),
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}
