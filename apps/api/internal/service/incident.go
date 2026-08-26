package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type IncidentRepository interface {
	ListIncidents(ctx context.Context, status string) ([]db.Incident, error)
	GetIncidentByID(ctx context.Context, id uuid.UUID) (db.Incident, error)
	AcknowledgeIncident(ctx context.Context, incidentID, actorID uuid.UUID) (db.Incident, error)
	ResolveIncident(ctx context.Context, incidentID, actorID uuid.UUID) (db.Incident, error)
	ListTimelineEvents(ctx context.Context, incidentID uuid.UUID) ([]db.TimelineEvent, error)
	ListAlertsForIncident(ctx context.Context, incidentID uuid.UUID) ([]db.Alert, error)
	CancelEscalationJobs(ctx context.Context, incidentID uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	GetUserBySlackID(ctx context.Context, slackUserID string) (db.User, error)
	GetUserByExpressHuid(ctx context.Context, expressHuid uuid.UUID) (db.User, error)
	GetAlertByID(ctx context.Context, id uuid.UUID) (db.Alert, error)
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	TeamMemberUserIDs(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]struct{}, error)
	CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error)
	ManualCreateFromAlert(ctx context.Context, input db.ManualCreateFromAlertInput) (db.ManualCreateFromAlertResult, error)
}

type CreateFromAlertInput struct {
	AlertID    uuid.UUID
	TeamID     uuid.UUID
	AssigneeID *uuid.UUID
	ActorID    uuid.UUID
}

type IncidentService struct {
	repo              IncidentRepository
	dedupWindow       time.Duration
	escalationTimeout time.Duration
}

func NewIncidentService(repo IncidentRepository, dedupWindow, escalationTimeout time.Duration) *IncidentService {
	return &IncidentService{
		repo:              repo,
		dedupWindow:       dedupWindow,
		escalationTimeout: escalationTimeout,
	}
}

func (s *IncidentService) List(ctx context.Context, status string) ([]db.Incident, error) {
	return s.repo.ListIncidents(ctx, status)
}

func (s *IncidentService) Get(ctx context.Context, id uuid.UUID) (db.Incident, error) {
	incident, err := s.repo.GetIncidentByID(ctx, id)
	if err != nil {
		return db.Incident{}, mapIncidentError(err)
	}
	return incident, nil
}

func (s *IncidentService) Acknowledge(ctx context.Context, incidentID, actorID uuid.UUID) (db.Incident, error) {
	incident, err := s.repo.AcknowledgeIncident(ctx, incidentID, actorID)
	if err != nil {
		return db.Incident{}, mapIncidentTransitionError(err, "acknowledge")
	}
	_ = s.repo.CancelEscalationJobs(ctx, incidentID)
	return incident, nil
}

func (s *IncidentService) AcknowledgeBySlackUser(ctx context.Context, incidentID uuid.UUID, slackUserID string) (db.Incident, error) {
	user, err := s.repo.GetUserBySlackID(ctx, slackUserID)
	if err != nil {
		return db.Incident{}, mapIncidentError(err)
	}
	return s.Acknowledge(ctx, incidentID, user.ID)
}

func (s *IncidentService) AcknowledgeByExpressHuid(ctx context.Context, incidentID uuid.UUID, expressHuidRaw string) (db.Incident, error) {
	huid, err := db.ParseExpressHuid(expressHuidRaw)
	if err != nil {
		return db.Incident{}, apperrors.Validation("invalid express_user_huid", nil)
	}
	user, err := s.repo.GetUserByExpressHuid(ctx, huid)
	if err != nil {
		return db.Incident{}, mapIncidentError(err)
	}
	return s.Acknowledge(ctx, incidentID, user.ID)
}

func (s *IncidentService) Resolve(ctx context.Context, incidentID, actorID uuid.UUID) (db.Incident, error) {
	incident, err := s.repo.ResolveIncident(ctx, incidentID, actorID)
	if err != nil {
		return db.Incident{}, mapIncidentTransitionError(err, "resolve")
	}
	_ = s.repo.CancelEscalationJobs(ctx, incidentID)
	return incident, nil
}

func (s *IncidentService) Timeline(ctx context.Context, incidentID uuid.UUID) ([]db.TimelineEvent, error) {
	if _, err := s.repo.GetIncidentByID(ctx, incidentID); err != nil {
		return nil, mapIncidentError(err)
	}
	return s.repo.ListTimelineEvents(ctx, incidentID)
}

func (s *IncidentService) Alerts(ctx context.Context, incidentID uuid.UUID) ([]db.Alert, error) {
	if _, err := s.repo.GetIncidentByID(ctx, incidentID); err != nil {
		return nil, mapIncidentError(err)
	}
	return s.repo.ListAlertsForIncident(ctx, incidentID)
}

func (s *IncidentService) CreateFromAlert(ctx context.Context, input CreateFromAlertInput) (db.Incident, error) {
	if input.TeamID == uuid.Nil {
		return db.Incident{}, apperrors.Validation("team_id is required", nil)
	}
	if _, err := s.repo.GetTeam(ctx, input.TeamID); err != nil {
		return db.Incident{}, mapIncidentTeamError(err)
	}
	if _, err := s.repo.GetAlertByID(ctx, input.AlertID); err != nil {
		return db.Incident{}, mapAlertError(err)
	}

	assigneeID, err := s.resolveAssignee(ctx, input.TeamID, input.AssigneeID)
	if err != nil {
		return db.Incident{}, err
	}

	actorID := input.ActorID
	now := time.Now().UTC()
	result, err := s.repo.ManualCreateFromAlert(ctx, db.ManualCreateFromAlertInput{
		AlertID:    input.AlertID,
		TeamID:     input.TeamID,
		AssigneeID: assigneeID,
		ActorID:    &actorID,
		DedupSince: now.Add(-s.dedupWindow),
		PostCreate: &db.IncidentPostCreateJobs{
			EscalationRunAt: now.Add(s.escalationTimeout),
		},
	})
	if err != nil {
		return db.Incident{}, mapCreateFromAlertError(err)
	}
	return result.Incident, nil
}

func (s *IncidentService) resolveAssignee(ctx context.Context, teamID uuid.UUID, assigneeID *uuid.UUID) (*uuid.UUID, error) {
	if assigneeID == nil {
		onCall, err := s.repo.CurrentOnCallUsers(ctx, teamID, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		if len(onCall) == 0 {
			return nil, nil
		}
		id := onCall[0].UserID
		return &id, nil
	}
	if _, err := s.repo.GetUserByID(ctx, *assigneeID); err != nil {
		return nil, mapAssigneeError(err)
	}
	memberIDs, err := s.repo.TeamMemberUserIDs(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if _, ok := memberIDs[*assigneeID]; !ok {
		return nil, apperrors.Validation("assignee must be a team member", map[string]any{"assignee_id": assigneeID.String()})
	}
	return assigneeID, nil
}

func IncidentJSON(incident db.Incident) map[string]any {
	out := map[string]any{
		"id":          incident.ID.String(),
		"team_id":     incident.TeamID.String(),
		"status":      incident.Status,
		"severity":    incident.Severity,
		"title":       incident.Title,
		"fingerprint": incident.Fingerprint,
		"created_at":  incident.CreatedAt,
	}
	if incident.AssigneeID != nil {
		out["assignee_id"] = incident.AssigneeID.String()
	}
	if incident.JiraIssueKey != nil {
		out["jira_issue_key"] = *incident.JiraIssueKey
	}
	if incident.AcknowledgedAt != nil {
		out["acknowledged_at"] = incident.AcknowledgedAt
	}
	if incident.ResolvedAt != nil {
		out["resolved_at"] = incident.ResolvedAt
	}
	return out
}

func TimelineEventJSON(event db.TimelineEvent) map[string]any {
	var payload map[string]any
	_ = json.Unmarshal(event.Payload, &payload)
	out := map[string]any{
		"id":         event.ID.String(),
		"kind":       event.Kind,
		"payload":    payload,
		"created_at": event.CreatedAt,
	}
	if event.ActorID != nil {
		out["actor_id"] = event.ActorID.String()
	}
	return out
}

func mapIncidentError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("incident not found")
	}
	return err
}

func mapIncidentTransitionError(err error, action string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.Conflict("incident cannot be " + action + " in its current state")
	}
	return err
}

func mapAlertError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("alert")
	}
	return err
}

func mapIncidentTeamError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.Validation("team_id is invalid", nil)
	}
	return err
}

func mapAssigneeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.Validation("assignee_id is invalid", nil)
	}
	return err
}

func mapCreateFromAlertError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("alert")
	}
	if errors.Is(err, db.ErrAlertNotFiring) {
		return apperrors.Validation("alert must be firing", map[string]any{"alert_id": "not_firing"})
	}
	if errors.Is(err, db.ErrAlertAlreadyLinked) {
		return apperrors.Conflict("alert already linked to an open incident")
	}
	var teamMismatch *db.FingerprintTeamMismatchError
	if errors.As(err, &teamMismatch) {
		return apperrors.ConflictWithDetails("fingerprint matches an open incident on a different team", map[string]any{
			"incident_id": teamMismatch.IncidentID.String(),
		})
	}
	return err
}
