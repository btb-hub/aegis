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

type OnCallRepository interface {
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error)
	ListOnCallSlotsInRange(ctx context.Context, teamID uuid.UUID, from, to time.Time) ([]db.OnCallSlot, error)
}

type OnCallService struct {
	repo OnCallRepository
}

func NewOnCallService(repo OnCallRepository) *OnCallService {
	return &OnCallService{repo: repo}
}

func (s *OnCallService) CurrentOnCall(ctx context.Context, teamID uuid.UUID) ([]db.OnCallUser, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return nil, mapOnCallError(err)
	}
	users, err := s.repo.CurrentOnCallUsers(ctx, teamID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []db.OnCallUser{}
	}
	return users, nil
}

func (s *OnCallService) Calendar(ctx context.Context, teamID uuid.UUID, from, to time.Time) ([]db.OnCallSlot, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return nil, mapOnCallError(err)
	}
	if !from.Before(to) {
		return nil, apperrors.Validation("to must be after from", nil)
	}
	slots, err := s.repo.ListOnCallSlotsInRange(ctx, teamID, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	if slots == nil {
		slots = []db.OnCallSlot{}
	}
	return slots, nil
}

func mapOnCallError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("team")
	}
	return err
}

func OnCallUserJSON(user db.OnCallUser) map[string]any {
	return map[string]any{
		"user_id":      user.UserID.String(),
		"email":        user.Email,
		"display_name": user.DisplayName,
		"source":       user.Source,
	}
}

func OnCallSlotJSON(slot db.OnCallSlot) map[string]any {
	return map[string]any{
		"id":         slot.ID.String(),
		"team_id":    slot.TeamID.String(),
		"user_id":    slot.UserID.String(),
		"start_at":   slot.StartAt,
		"end_at":     slot.EndAt,
		"source":     slot.Source,
		"created_at": slot.CreatedAt,
	}
}
