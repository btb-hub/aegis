package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type SavedViewRepository interface {
	ListSavedViewsForUser(ctx context.Context, userID uuid.UUID) ([]db.SavedView, error)
	GetSavedView(ctx context.Context, id uuid.UUID) (db.SavedView, error)
	CreateSavedView(ctx context.Context, ownerID uuid.UUID, name string, filter json.RawMessage, shared bool) (db.SavedView, error)
	UpdateSavedView(ctx context.Context, id, ownerID uuid.UUID, name string, filter json.RawMessage, shared bool) (db.SavedView, error)
	DeleteSavedView(ctx context.Context, id, ownerID uuid.UUID) error
}

type SavedViewService struct {
	repo SavedViewRepository
}

func NewSavedViewService(repo SavedViewRepository) *SavedViewService {
	return &SavedViewService{repo: repo}
}

func (s *SavedViewService) List(ctx context.Context, userID uuid.UUID) ([]db.SavedView, error) {
	return s.repo.ListSavedViewsForUser(ctx, userID)
}

func (s *SavedViewService) Get(ctx context.Context, userID, id uuid.UUID) (db.SavedView, error) {
	view, err := s.repo.GetSavedView(ctx, id)
	if err != nil {
		return db.SavedView{}, mapSavedViewError(err)
	}
	if view.OwnerID != userID && !view.Shared {
		return db.SavedView{}, apperrors.NotFound("saved view not found")
	}
	return view, nil
}

func (s *SavedViewService) Create(ctx context.Context, userID uuid.UUID, name string, filter json.RawMessage, shared bool) (db.SavedView, error) {
	if err := validateSavedViewInput(name, filter); err != nil {
		return db.SavedView{}, err
	}
	view, err := s.repo.CreateSavedView(ctx, userID, name, filter, shared)
	if err != nil {
		return db.SavedView{}, mapSavedViewError(err)
	}
	return view, nil
}

func (s *SavedViewService) Update(ctx context.Context, userID, id uuid.UUID, name string, filter json.RawMessage, shared bool) (db.SavedView, error) {
	if err := validateSavedViewInput(name, filter); err != nil {
		return db.SavedView{}, err
	}
	view, err := s.repo.UpdateSavedView(ctx, id, userID, name, filter, shared)
	if err != nil {
		return db.SavedView{}, mapSavedViewError(err)
	}
	return view, nil
}

func (s *SavedViewService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.repo.DeleteSavedView(ctx, id, userID); err != nil {
		return mapSavedViewError(err)
	}
	return nil
}

func validateSavedViewInput(name string, filter json.RawMessage) error {
	if strings.TrimSpace(name) == "" {
		return apperrors.Validation("name must not be empty", nil)
	}
	if len(filter) == 0 {
		return apperrors.Validation("filter must be a JSON object", nil)
	}
	var obj map[string]any
	if err := json.Unmarshal(filter, &obj); err != nil {
		return apperrors.Validation("filter must be a JSON object", nil)
	}
	return nil
}

func SavedViewJSON(view db.SavedView) map[string]any {
	var filter map[string]any
	_ = json.Unmarshal(view.Filter, &filter)
	if filter == nil {
		filter = map[string]any{}
	}
	return map[string]any{
		"id":         view.ID.String(),
		"owner_id":   view.OwnerID.String(),
		"name":       view.Name,
		"filter":     filter,
		"shared":     view.Shared,
		"created_at": view.CreatedAt,
		"updated_at": view.UpdatedAt,
	}
}

func mapSavedViewError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("saved view not found")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.Validation("saved view name already exists", nil)
	}
	return err
}
