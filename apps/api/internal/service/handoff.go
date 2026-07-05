package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type HandoffRepository interface {
	GetIncidentByID(ctx context.Context, id uuid.UUID) (db.Incident, error)
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error)
	HandoffIncident(ctx context.Context, input db.HandoffIncidentInput) (db.Incident, db.Handoff, error)
	BounceIncident(ctx context.Context, input db.BounceIncidentInput) (db.Incident, error)
	EnqueueHandoffNotify(ctx context.Context, incidentID uuid.UUID) error
	HandoffStats(ctx context.Context, from, to time.Time) (db.HandoffStats, error)
	ListTimelineEvents(ctx context.Context, incidentID uuid.UUID) ([]db.TimelineEvent, error)
	HasEscalationPath(ctx context.Context, fromTeamID, toTeamID uuid.UUID) (bool, error)
}

type HandoffService struct {
	repo HandoffRepository
}

func NewHandoffService(repo HandoffRepository) *HandoffService {
	return &HandoffService{repo: repo}
}

func (s *HandoffService) Handoff(ctx context.Context, incidentID, actorID, toTeamID uuid.UUID, note string) (db.Incident, error) {
	incident, err := s.repo.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return db.Incident{}, mapHandoffIncidentError(err)
	}
	if _, err := s.repo.GetTeam(ctx, toTeamID); err != nil {
		return db.Incident{}, mapHandoffTeamError(err)
	}

	ok, err := s.repo.HasEscalationPath(ctx, incident.TeamID, toTeamID)
	if err != nil {
		return db.Incident{}, err
	}
	if !ok {
		return db.Incident{}, apperrors.Validation("handoff target is not configured in escalation paths", map[string]any{
			"from_team_id": incident.TeamID.String(),
			"to_team_id":   toTeamID.String(),
		})
	}

	onCall, err := s.repo.CurrentOnCallUsers(ctx, toTeamID, time.Now().UTC())
	if err != nil {
		return db.Incident{}, err
	}
	if len(onCall) == 0 {
		return db.Incident{}, apperrors.Validation("target team has no one on call", nil)
	}

	incident, _, err = s.repo.HandoffIncident(ctx, db.HandoffIncidentInput{
		IncidentID: incidentID,
		ActorID:    actorID,
		ToTeamID:   toTeamID,
		ToUserID:   onCall[0].UserID,
		Note:       strings.TrimSpace(note),
	})
	if err != nil {
		return db.Incident{}, mapHandoffTransitionError(err, "hand off")
	}

	_ = s.repo.EnqueueHandoffNotify(ctx, incidentID)
	return incident, nil
}

func (s *HandoffService) Bounce(ctx context.Context, incidentID, actorID uuid.UUID, note string) (db.Incident, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return db.Incident{}, apperrors.Validation("note is required", nil)
	}

	incident, err := s.repo.BounceIncident(ctx, db.BounceIncidentInput{
		IncidentID: incidentID,
		ActorID:    actorID,
		Note:       note,
	})
	if err != nil {
		return db.Incident{}, mapHandoffTransitionError(err, "bounce")
	}
	return incident, nil
}

func (s *HandoffService) Stats(ctx context.Context, from, to time.Time) (db.HandoffStats, error) {
	if !to.After(from) {
		return db.HandoffStats{}, apperrors.Validation("to must be after from", nil)
	}
	return s.repo.HandoffStats(ctx, from, to)
}

func (s *HandoffService) Timeline(ctx context.Context, incidentID uuid.UUID) ([]db.TimelineEvent, error) {
	if _, err := s.repo.GetIncidentByID(ctx, incidentID); err != nil {
		return nil, mapHandoffIncidentError(err)
	}
	return s.repo.ListTimelineEvents(ctx, incidentID)
}

func mapHandoffIncidentError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("incident not found")
	}
	return err
}

func mapHandoffTeamError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("team not found")
	}
	return err
}

func mapHandoffTransitionError(err error, action string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.Conflict("incident cannot be " + action + " in its current state")
	}
	return err
}
