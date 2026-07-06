package service

import (
	"context"
	"errors"
	"strings"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type WorkspaceRepository interface {
	ListWorkspaces(ctx context.Context) ([]db.Workspace, error)
	GetWorkspace(ctx context.Context, id uuid.UUID) (db.Workspace, error)
	CreateWorkspace(ctx context.Context, name, slug, description string) (db.Workspace, error)
	UpdateWorkspace(ctx context.Context, id uuid.UUID, name, slug, description string) (db.Workspace, error)
	DeleteWorkspace(ctx context.Context, id uuid.UUID) error
}

type WorkspaceService struct {
	repo WorkspaceRepository
}

func NewWorkspaceService(repo WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{repo: repo}
}

func (s *WorkspaceService) List(ctx context.Context) ([]db.Workspace, error) {
	return s.repo.ListWorkspaces(ctx)
}

func (s *WorkspaceService) Get(ctx context.Context, id uuid.UUID) (db.Workspace, error) {
	item, err := s.repo.GetWorkspace(ctx, id)
	if err != nil {
		return db.Workspace{}, mapWorkspaceError(err)
	}
	return item, nil
}

func (s *WorkspaceService) Create(ctx context.Context, name, slug, description string) (db.Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return db.Workspace{}, apperrors.Validation("workspace name is required", nil)
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		slug = db.Slugify(name)
	}
	item, err := s.repo.CreateWorkspace(ctx, name, slug, strings.TrimSpace(description))
	if err != nil {
		return db.Workspace{}, mapWorkspaceError(err)
	}
	return item, nil
}

func (s *WorkspaceService) Update(ctx context.Context, id uuid.UUID, name, slug, description string) (db.Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return db.Workspace{}, apperrors.Validation("workspace name is required", nil)
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		slug = db.Slugify(name)
	}
	item, err := s.repo.UpdateWorkspace(ctx, id, name, slug, strings.TrimSpace(description))
	if err != nil {
		return db.Workspace{}, mapWorkspaceError(err)
	}
	return item, nil
}

func (s *WorkspaceService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteWorkspace(ctx, id); err != nil {
		return mapWorkspaceError(err)
	}
	return nil
}

func mapWorkspaceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("workspace")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.Conflict("workspace slug already exists")
	}
	return err
}

func WorkspaceJSON(item db.Workspace) map[string]any {
	return map[string]any{
		"id":          item.ID.String(),
		"name":        item.Name,
		"slug":        item.Slug,
		"description": item.Description,
		"created_at":  item.CreatedAt,
		"updated_at":  item.UpdatedAt,
	}
}
