package service

import (
	"context"
	"errors"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var validTierPairs = map[string][]string{
	"noc": {"l1", "l2"},
	"l1":  {"l2"},
	"l2":  {"l3"},
}

type EscalationRepository interface {
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	ListEscalationPathsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.EscalationPath, error)
	ListEscalationPathsFromTeam(ctx context.Context, fromTeamID uuid.UUID) ([]db.EscalationPath, error)
	ListEscalationPathsToTeam(ctx context.Context, toTeamID uuid.UUID) ([]db.EscalationPath, error)
	ReplaceEscalationPaths(ctx context.Context, workspaceID uuid.UUID, paths []db.EscalationPath) error
	AddEscalationPath(ctx context.Context, path db.EscalationPath) (db.EscalationPath, error)
	DeleteEscalationPath(ctx context.Context, id uuid.UUID) error
	HasEscalationPath(ctx context.Context, fromTeamID, toTeamID uuid.UUID) (bool, error)
	ListHandoffTargetTeams(ctx context.Context, fromTeamID uuid.UUID) ([]db.Team, error)
}

type EscalationService struct {
	repo EscalationRepository
}

func NewEscalationService(repo EscalationRepository) *EscalationService {
	return &EscalationService{repo: repo}
}

func (s *EscalationService) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.EscalationPath, error) {
	paths, err := s.repo.ListEscalationPathsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if paths == nil {
		paths = []db.EscalationPath{}
	}
	return paths, nil
}

func (s *EscalationService) ListFromTeam(ctx context.Context, fromTeamID uuid.UUID) ([]db.EscalationPath, error) {
	paths, err := s.repo.ListEscalationPathsFromTeam(ctx, fromTeamID)
	if err != nil {
		return nil, err
	}
	if paths == nil {
		paths = []db.EscalationPath{}
	}
	return paths, nil
}

func (s *EscalationService) ListToTeam(ctx context.Context, toTeamID uuid.UUID) ([]db.EscalationPath, error) {
	paths, err := s.repo.ListEscalationPathsToTeam(ctx, toTeamID)
	if err != nil {
		return nil, err
	}
	if paths == nil {
		paths = []db.EscalationPath{}
	}
	return paths, nil
}

func (s *EscalationService) HandoffTargets(ctx context.Context, fromTeamID uuid.UUID) ([]db.Team, error) {
	if _, err := s.repo.GetTeam(ctx, fromTeamID); err != nil {
		return nil, mapEscalationTeamError(err)
	}
	teams, err := s.repo.ListHandoffTargetTeams(ctx, fromTeamID)
	if err != nil {
		return nil, err
	}
	if teams == nil {
		teams = []db.Team{}
	}
	return teams, nil
}

type EscalationPathInput struct {
	FromTeamID     uuid.UUID
	ToTeamID       uuid.UUID
	CrossWorkspace bool
}

func (s *EscalationService) ReplaceWorkspacePaths(ctx context.Context, workspaceID uuid.UUID, inputs []EscalationPathInput) ([]db.EscalationPath, error) {
	paths := make([]db.EscalationPath, 0, len(inputs))
	for _, input := range inputs {
		path, err := s.validatePath(ctx, workspaceID, input)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	if err := s.repo.ReplaceEscalationPaths(ctx, workspaceID, paths); err != nil {
		return nil, err
	}
	return s.repo.ListEscalationPathsByWorkspace(ctx, workspaceID)
}

func (s *EscalationService) AddPath(ctx context.Context, workspaceID uuid.UUID, input EscalationPathInput) (db.EscalationPath, error) {
	path, err := s.validatePath(ctx, workspaceID, input)
	if err != nil {
		return db.EscalationPath{}, err
	}
	return s.repo.AddEscalationPath(ctx, path)
}

func (s *EscalationService) DeletePath(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteEscalationPath(ctx, id); err != nil {
		return mapEscalationPathError(err)
	}
	return nil
}

func (s *EscalationService) ValidateHandoffTarget(ctx context.Context, fromTeamID, toTeamID uuid.UUID) error {
	ok, err := s.repo.HasEscalationPath(ctx, fromTeamID, toTeamID)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.Validation("handoff target is not configured in escalation paths", map[string]any{
			"from_team_id": fromTeamID.String(),
			"to_team_id":   toTeamID.String(),
		})
	}
	return nil
}

func (s *EscalationService) validatePath(ctx context.Context, workspaceID uuid.UUID, input EscalationPathInput) (db.EscalationPath, error) {
	if input.FromTeamID == input.ToTeamID {
		return db.EscalationPath{}, apperrors.Validation("from and to team must differ", nil)
	}
	fromTeam, err := s.repo.GetTeam(ctx, input.FromTeamID)
	if err != nil {
		return db.EscalationPath{}, mapEscalationTeamError(err)
	}
	toTeam, err := s.repo.GetTeam(ctx, input.ToTeamID)
	if err != nil {
		return db.EscalationPath{}, mapEscalationTeamError(err)
	}
	if !input.CrossWorkspace && fromTeam.WorkspaceID != toTeam.WorkspaceID {
		return db.EscalationPath{}, apperrors.Validation("teams must belong to the same workspace unless cross_workspace is true", nil)
	}
	if fromTeam.WorkspaceID != workspaceID && toTeam.WorkspaceID != workspaceID {
		return db.EscalationPath{}, apperrors.Validation("at least one team must belong to the workspace", nil)
	}
	if err := validateTierAdjacency(fromTeam.SupportTier, toTeam.SupportTier); err != nil {
		return db.EscalationPath{}, err
	}
	return db.EscalationPath{
		FromTeamID:     input.FromTeamID,
		ToTeamID:       input.ToTeamID,
		WorkspaceID:    workspaceID,
		CrossWorkspace: input.CrossWorkspace,
	}, nil
}

func validateTierAdjacency(fromTier, toTier *string) error {
	if fromTier == nil || toTier == nil {
		return apperrors.Validation("both teams must have a support tier configured", nil)
	}
	allowed, ok := validTierPairs[*fromTier]
	if !ok {
		return apperrors.Validation("source team tier cannot initiate escalation", map[string]any{"tier": *fromTier})
	}
	for _, target := range allowed {
		if target == *toTier {
			return nil
		}
	}
	return apperrors.Validation("invalid escalation tier pair", map[string]any{
		"from_tier": *fromTier,
		"to_tier":   *toTier,
	})
}

func mapEscalationTeamError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("team")
	}
	return err
}

func mapEscalationPathError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("escalation path")
	}
	return err
}

func EscalationPathJSON(path db.EscalationPath) map[string]any {
	return map[string]any{
		"id":              path.ID.String(),
		"from_team_id":    path.FromTeamID.String(),
		"to_team_id":      path.ToTeamID.String(),
		"workspace_id":    path.WorkspaceID.String(),
		"cross_workspace": path.CrossWorkspace,
		"created_at":      path.CreatedAt,
	}
}
