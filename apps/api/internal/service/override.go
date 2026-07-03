package service

import (
	"context"
	"errors"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OverrideRepository interface {
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	TeamMemberUserIDs(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]struct{}, error)
	ListOverridesByTeam(ctx context.Context, teamID uuid.UUID) ([]db.Override, error)
	GetOverrideForTeam(ctx context.Context, teamID, overrideID uuid.UUID) (db.Override, error)
	CreateOverride(ctx context.Context, teamID, userID uuid.UUID, startAt, endAt time.Time) (db.Override, error)
	DeleteOverrideForTeam(ctx context.Context, teamID, overrideID uuid.UUID) error
	MaterialiseOnCallForTeam(ctx context.Context, teamID uuid.UUID) error
}

type CreateOverrideInput struct {
	UserID  uuid.UUID
	StartAt time.Time
	EndAt   time.Time
}

type OverrideService struct {
	repo OverrideRepository
}

func NewOverrideService(repo OverrideRepository) *OverrideService {
	return &OverrideService{repo: repo}
}

func (s *OverrideService) ListOverrides(ctx context.Context, teamID uuid.UUID) ([]db.Override, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return nil, mapOverrideError(err)
	}
	overrides, err := s.repo.ListOverridesByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if overrides == nil {
		overrides = []db.Override{}
	}
	return overrides, nil
}

func (s *OverrideService) CreateOverride(ctx context.Context, teamID uuid.UUID, input CreateOverrideInput) (db.Override, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return db.Override{}, mapOverrideError(err)
	}
	if err := s.validateOverrideInput(ctx, teamID, input); err != nil {
		return db.Override{}, err
	}
	override, err := s.repo.CreateOverride(ctx, teamID, input.UserID, input.StartAt.UTC(), input.EndAt.UTC())
	if err != nil {
		return db.Override{}, err
	}
	if err := s.repo.MaterialiseOnCallForTeam(ctx, teamID); err != nil {
		return db.Override{}, err
	}
	return override, nil
}

func (s *OverrideService) DeleteOverride(ctx context.Context, teamID, overrideID uuid.UUID) error {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return mapOverrideError(err)
	}
	if err := s.repo.DeleteOverrideForTeam(ctx, teamID, overrideID); err != nil {
		return mapOverrideError(err)
	}
	if err := s.repo.MaterialiseOnCallForTeam(ctx, teamID); err != nil {
		return err
	}
	return nil
}

func (s *OverrideService) validateOverrideInput(ctx context.Context, teamID uuid.UUID, input CreateOverrideInput) error {
	if input.UserID == uuid.Nil {
		return apperrors.Validation("user_id is required", nil)
	}
	if !input.StartAt.Before(input.EndAt) {
		return apperrors.Validation("end_at must be after start_at", nil)
	}
	memberIDs, err := s.repo.TeamMemberUserIDs(ctx, teamID)
	if err != nil {
		return err
	}
	if _, ok := memberIDs[input.UserID]; !ok {
		return apperrors.Validation("user must be a team member", map[string]any{"user_id": input.UserID.String()})
	}
	return nil
}

func mapOverrideError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("team or override")
	}
	return err
}

func OverrideJSON(override db.Override) map[string]any {
	return map[string]any{
		"id":         override.ID.String(),
		"team_id":    override.TeamID.String(),
		"user_id":    override.UserID.String(),
		"start_at":   override.StartAt,
		"end_at":     override.EndAt,
		"created_at": override.CreatedAt,
	}
}
