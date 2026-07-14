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
	ListWorkspacesWithCounts(ctx context.Context) ([]db.WorkspaceSummary, error)
	GetWorkspace(ctx context.Context, id uuid.UUID) (db.Workspace, error)
	GetWorkspaceUsage(ctx context.Context, id uuid.UUID) (db.WorkspaceUsage, error)
	CreateWorkspace(ctx context.Context, name, slug, description string) (db.Workspace, error)
	EnsureWorkspaceSlots(ctx context.Context, workspaceID uuid.UUID) error
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

func (s *WorkspaceService) ListWithCounts(ctx context.Context) ([]db.WorkspaceSummary, error) {
	items, err := s.repo.ListWorkspacesWithCounts(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []db.WorkspaceSummary{}
	}
	return items, nil
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
	if err := s.repo.EnsureWorkspaceSlots(ctx, item.ID); err != nil {
		return db.Workspace{}, err
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
	if id == db.DefaultWorkspaceID {
		return apperrors.Forbidden("default workspace cannot be deleted")
	}
	usage, err := s.repo.GetWorkspaceUsage(ctx, id)
	if err != nil {
		return mapWorkspaceError(err)
	}
	if usage.TeamCount > 0 || usage.EscalationPathCount > 0 || usage.IntegrationCount > 0 {
		return apperrors.ConflictWithDetails("workspace is not empty", map[string]any{
			"team_count":            usage.TeamCount,
			"escalation_path_count": usage.EscalationPathCount,
			"integration_count":     usage.IntegrationCount,
		})
	}
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

func WorkspaceSummaryJSON(item db.WorkspaceSummary) map[string]any {
	out := WorkspaceJSON(item.Workspace)
	out["team_count"] = item.TeamCount
	out["routing_rule_count"] = item.RoutingRuleCount
	return out
}
