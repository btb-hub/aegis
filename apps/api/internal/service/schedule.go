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
	"github.com/jackc/pgx/v5/pgconn"
)

type ScheduleRepository interface {
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	TeamMemberUserIDs(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]struct{}, error)
	ListSchedulesWithLayersByTeam(ctx context.Context, teamID uuid.UUID) ([]db.ScheduleWithLayers, error)
	GetScheduleWithLayersForTeam(ctx context.Context, teamID, scheduleID uuid.UUID) (db.ScheduleWithLayers, error)
	CreateScheduleWithLayer(ctx context.Context, teamID uuid.UUID, name, timezone string, layer db.CreateScheduleLayerInput) (db.ScheduleWithLayers, error)
	UpdateScheduleWithLayer(ctx context.Context, teamID, scheduleID uuid.UUID, name, timezone string, layer db.CreateScheduleLayerInput) (db.ScheduleWithLayers, error)
	DeleteScheduleForTeam(ctx context.Context, teamID, scheduleID uuid.UUID) error
	MaterialiseOnCallForTeam(ctx context.Context, teamID uuid.UUID) error
}

type WeeklyRotationInput struct {
	HandoffWeekday int32
	HandoffTime    string
	Participants   []uuid.UUID
}

type CreateScheduleInput struct {
	Name      string
	Timezone  string
	Rotation  WeeklyRotationInput
}

type ScheduleService struct {
	repo ScheduleRepository
}

func NewScheduleService(repo ScheduleRepository) *ScheduleService {
	return &ScheduleService{repo: repo}
}

func (s *ScheduleService) ListSchedules(ctx context.Context, teamID uuid.UUID) ([]db.ScheduleWithLayers, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return nil, mapScheduleError(err)
	}
	schedules, err := s.repo.ListSchedulesWithLayersByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if schedules == nil {
		schedules = []db.ScheduleWithLayers{}
	}
	return schedules, nil
}

func (s *ScheduleService) GetSchedule(ctx context.Context, teamID, scheduleID uuid.UUID) (db.ScheduleWithLayers, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return db.ScheduleWithLayers{}, mapScheduleError(err)
	}
	schedule, err := s.repo.GetScheduleWithLayersForTeam(ctx, teamID, scheduleID)
	if err != nil {
		return db.ScheduleWithLayers{}, mapScheduleError(err)
	}
	return schedule, nil
}

func (s *ScheduleService) CreateSchedule(ctx context.Context, teamID uuid.UUID, input CreateScheduleInput) (db.ScheduleWithLayers, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return db.ScheduleWithLayers{}, mapScheduleError(err)
	}
	name, timezone, layer, err := s.validateScheduleInput(ctx, teamID, input)
	if err != nil {
		return db.ScheduleWithLayers{}, err
	}
	schedule, err := s.repo.CreateScheduleWithLayer(ctx, teamID, name, timezone, layer)
	if err != nil {
		return db.ScheduleWithLayers{}, mapScheduleError(err)
	}
	if err := s.repo.MaterialiseOnCallForTeam(ctx, teamID); err != nil {
		return db.ScheduleWithLayers{}, err
	}
	return schedule, nil
}

func (s *ScheduleService) UpdateSchedule(ctx context.Context, teamID, scheduleID uuid.UUID, input CreateScheduleInput) (db.ScheduleWithLayers, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return db.ScheduleWithLayers{}, mapScheduleError(err)
	}
	name, timezone, layer, err := s.validateScheduleInput(ctx, teamID, input)
	if err != nil {
		return db.ScheduleWithLayers{}, err
	}
	schedule, err := s.repo.UpdateScheduleWithLayer(ctx, teamID, scheduleID, name, timezone, layer)
	if err != nil {
		return db.ScheduleWithLayers{}, mapScheduleError(err)
	}
	if err := s.repo.MaterialiseOnCallForTeam(ctx, teamID); err != nil {
		return db.ScheduleWithLayers{}, err
	}
	return schedule, nil
}

func (s *ScheduleService) DeleteSchedule(ctx context.Context, teamID, scheduleID uuid.UUID) error {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return mapScheduleError(err)
	}
	if err := s.repo.DeleteScheduleForTeam(ctx, teamID, scheduleID); err != nil {
		return mapScheduleError(err)
	}
	if err := s.repo.MaterialiseOnCallForTeam(ctx, teamID); err != nil {
		return err
	}
	return nil
}

func (s *ScheduleService) validateScheduleInput(ctx context.Context, teamID uuid.UUID, input CreateScheduleInput) (string, string, db.CreateScheduleLayerInput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return "", "", db.CreateScheduleLayerInput{}, apperrors.Validation("schedule name is required", nil)
	}
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		return "", "", db.CreateScheduleLayerInput{}, apperrors.Validation("timezone is required", nil)
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return "", "", db.CreateScheduleLayerInput{}, apperrors.Validation("timezone must be a valid IANA name", map[string]any{"timezone": timezone})
	}
	if len(input.Rotation.Participants) == 0 {
		return "", "", db.CreateScheduleLayerInput{}, apperrors.Validation("participants must include at least one user", nil)
	}
	if input.Rotation.HandoffWeekday < 0 || input.Rotation.HandoffWeekday > 6 {
		return "", "", db.CreateScheduleLayerInput{}, apperrors.Validation("handoff_weekday must be between 0 (Sunday) and 6 (Saturday)", nil)
	}
	handoffTime, err := time.Parse("15:04", strings.TrimSpace(input.Rotation.HandoffTime))
	if err != nil {
		return "", "", db.CreateScheduleLayerInput{}, apperrors.Validation("handoff_time must be HH:MM in 24-hour format", nil)
	}

	memberIDs, err := s.repo.TeamMemberUserIDs(ctx, teamID)
	if err != nil {
		return "", "", db.CreateScheduleLayerInput{}, err
	}
	seen := map[uuid.UUID]struct{}{}
	for _, participantID := range input.Rotation.Participants {
		if participantID == uuid.Nil {
			return "", "", db.CreateScheduleLayerInput{}, apperrors.Validation("participants must be valid user ids", nil)
		}
		if _, ok := seen[participantID]; ok {
			return "", "", db.CreateScheduleLayerInput{}, apperrors.Validation("participants must be unique", nil)
		}
		seen[participantID] = struct{}{}
		if _, ok := memberIDs[participantID]; !ok {
			return "", "", db.CreateScheduleLayerInput{}, apperrors.Validation("participants must be team members", map[string]any{"user_id": participantID.String()})
		}
	}

	layer := db.CreateScheduleLayerInput{
		Priority:           0,
		RotationType:       "weekly",
		HandoffWeekday:     input.Rotation.HandoffWeekday,
		HandoffTime:        handoffTime,
		ParticipantUserIDs: input.Rotation.Participants,
	}
	return name, timezone, layer, nil
}

func mapScheduleError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("team or schedule")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.Conflict("schedule name already exists for this team")
	}
	return err
}

func ScheduleJSON(schedule db.ScheduleWithLayers) map[string]any {
	layers := make([]map[string]any, 0, len(schedule.Layers))
	for _, layer := range schedule.Layers {
		layers = append(layers, ScheduleLayerJSON(layer))
	}
	return map[string]any{
		"id":         schedule.Schedule.ID.String(),
		"team_id":    schedule.Schedule.TeamID.String(),
		"name":       schedule.Schedule.Name,
		"timezone":   schedule.Schedule.Timezone,
		"created_at": schedule.Schedule.CreatedAt,
		"updated_at": schedule.Schedule.UpdatedAt,
		"layers":     layers,
	}
}

func ScheduleLayerJSON(layer db.ScheduleLayer) map[string]any {
	participants := make([]string, 0, len(layer.ParticipantUserIDs))
	for _, id := range layer.ParticipantUserIDs {
		participants = append(participants, id.String())
	}
	return map[string]any{
		"id":                   layer.ID.String(),
		"schedule_id":          layer.ScheduleID.String(),
		"priority":             layer.Priority,
		"rotation_type":        layer.RotationType,
		"handoff_weekday":      layer.HandoffWeekday,
		"handoff_time":         layer.HandoffTime.Format("15:04"),
		"participant_user_ids": participants,
		"created_at":           layer.CreatedAt,
		"updated_at":           layer.UpdatedAt,
	}
}

func ParseParticipantIDs(raw []string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return nil, apperrors.Validation("participants must include at least one user", nil)
	}
	ids := make([]uuid.UUID, 0, len(raw))
	for _, value := range raw {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, apperrors.Validation("participants must be valid user ids", nil)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
