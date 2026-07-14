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

type TeamRepository interface {
	ListTeamsFiltered(ctx context.Context, workspaceID uuid.UUID) ([]db.Team, error)
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	CreateTeam(ctx context.Context, workspaceID uuid.UUID, name, description string, supportTier *string) (db.Team, error)
	UpdateTeam(ctx context.Context, id uuid.UUID, name, description string, supportTier *string) (db.Team, error)
	MoveTeamsToWorkspace(ctx context.Context, workspaceID uuid.UUID, teamIDs []uuid.UUID) error
	DeleteTeam(ctx context.Context, id uuid.UUID) error
	ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]db.TeamMember, error)
	AddTeamMember(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMembership, error)
	UpdateTeamMemberRole(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMembership, error)
	RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	GetWorkspace(ctx context.Context, id uuid.UUID) (db.Workspace, error)
}

type TeamMoveValidator interface {
	BlockedTeamsForWorkspaceMove(ctx context.Context, targetWorkspaceID uuid.UUID, teamIDs []uuid.UUID) (map[uuid.UUID][]db.EscalationPath, error)
}

type TeamService struct {
	repo          TeamRepository
	moveValidator TeamMoveValidator
}

func NewTeamService(repo TeamRepository, moveValidator TeamMoveValidator) *TeamService {
	return &TeamService{repo: repo, moveValidator: moveValidator}
}

func (s *TeamService) ListTeams(ctx context.Context, workspaceID *uuid.UUID) ([]db.Team, error) {
	filter := uuid.Nil
	if workspaceID != nil {
		filter = *workspaceID
	}
	return s.repo.ListTeamsFiltered(ctx, filter)
}

func (s *TeamService) CreateTeam(ctx context.Context, workspaceID uuid.UUID, name, description string, supportTier *string) (db.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return db.Team{}, apperrors.Validation("team name is required", nil)
	}
	if _, err := s.repo.GetWorkspace(ctx, workspaceID); err != nil {
		return db.Team{}, mapWorkspaceError(err)
	}
	tier, err := normalizeSupportTier(supportTier)
	if err != nil {
		return db.Team{}, err
	}
	team, err := s.repo.CreateTeam(ctx, workspaceID, name, strings.TrimSpace(description), tier)
	if err != nil {
		return db.Team{}, mapTeamError(err)
	}
	return team, nil
}

func (s *TeamService) GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error) {
	team, err := s.repo.GetTeam(ctx, id)
	if err != nil {
		return db.Team{}, mapTeamError(err)
	}
	return team, nil
}

func (s *TeamService) UpdateTeam(ctx context.Context, id uuid.UUID, name, description string, supportTier *string, workspaceID *uuid.UUID) (db.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return db.Team{}, apperrors.Validation("team name is required", nil)
	}
	current, err := s.repo.GetTeam(ctx, id)
	if err != nil {
		return db.Team{}, mapTeamError(err)
	}
	tier, err := normalizeSupportTier(supportTier)
	if err != nil {
		return db.Team{}, err
	}
	if workspaceID != nil && *workspaceID != current.WorkspaceID {
		if err := s.moveTeams(ctx, *workspaceID, []uuid.UUID{id}); err != nil {
			return db.Team{}, err
		}
	}
	team, err := s.repo.UpdateTeam(ctx, id, name, strings.TrimSpace(description), tier)
	if err != nil {
		return db.Team{}, mapTeamError(err)
	}
	return team, nil
}

func (s *TeamService) MoveTeamsToWorkspace(ctx context.Context, workspaceID uuid.UUID, teamIDs []uuid.UUID) ([]db.Team, error) {
	if err := s.moveTeams(ctx, workspaceID, teamIDs); err != nil {
		return nil, err
	}
	unique := dedupeTeamIDs(teamIDs)
	teams := make([]db.Team, 0, len(unique))
	for _, teamID := range unique {
		team, err := s.repo.GetTeam(ctx, teamID)
		if err != nil {
			return nil, mapTeamError(err)
		}
		teams = append(teams, team)
	}
	return teams, nil
}

func (s *TeamService) moveTeams(ctx context.Context, workspaceID uuid.UUID, teamIDs []uuid.UUID) error {
	if _, err := s.repo.GetWorkspace(ctx, workspaceID); err != nil {
		return mapWorkspaceError(err)
	}
	unique := dedupeTeamIDs(teamIDs)
	if len(unique) == 0 {
		return apperrors.Validation("team_ids must not be empty", nil)
	}
	if s.moveValidator != nil {
		blocked, err := s.moveValidator.BlockedTeamsForWorkspaceMove(ctx, workspaceID, unique)
		if err != nil {
			return err
		}
		if len(blocked) > 0 {
			return apperrors.ConflictWithDetails("team workspace move blocked by escalation paths", map[string]any{
				"blocked_teams": BlockedTeamsJSON(blocked),
			})
		}
	}
	if err := s.repo.MoveTeamsToWorkspace(ctx, workspaceID, unique); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.NotFound("team")
		}
		return err
	}
	return nil
}

func dedupeTeamIDs(teamIDs []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(teamIDs))
	out := make([]uuid.UUID, 0, len(teamIDs))
	for _, id := range teamIDs {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *TeamService) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteTeam(ctx, id); err != nil {
		return mapTeamError(err)
	}
	return nil
}

func (s *TeamService) ListMembers(ctx context.Context, teamID uuid.UUID) ([]db.TeamMember, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return nil, mapTeamError(err)
	}
	members, err := s.repo.ListTeamMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if members == nil {
		members = []db.TeamMember{}
	}
	return members, nil
}

func (s *TeamService) AddMember(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMember, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return db.TeamMember{}, mapTeamError(err)
	}
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.TeamMember{}, apperrors.NotFound("user")
		}
		return db.TeamMember{}, err
	}
	role, err := normalizeTeamRole(teamRole)
	if err != nil {
		return db.TeamMember{}, err
	}
	membership, err := s.repo.AddTeamMember(ctx, teamID, userID, role)
	if err != nil {
		return db.TeamMember{}, mapTeamError(err)
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return db.TeamMember{}, err
	}
	return membershipToMember(membership, user), nil
}

func (s *TeamService) UpdateMember(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMember, error) {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return db.TeamMember{}, mapTeamError(err)
	}
	role, err := normalizeTeamRole(teamRole)
	if err != nil {
		return db.TeamMember{}, err
	}
	membership, err := s.repo.UpdateTeamMemberRole(ctx, teamID, userID, role)
	if err != nil {
		return db.TeamMember{}, mapTeamError(err)
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return db.TeamMember{}, err
	}
	return membershipToMember(membership, user), nil
}

func (s *TeamService) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	if _, err := s.repo.GetTeam(ctx, teamID); err != nil {
		return mapTeamError(err)
	}
	if err := s.repo.RemoveTeamMember(ctx, teamID, userID); err != nil {
		return mapTeamError(err)
	}
	return nil
}

func normalizeSupportTier(tier *string) (*string, error) {
	if tier == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*tier)
	if value == "" {
		return nil, nil
	}
	switch value {
	case "l1", "l2", "l3", "noc":
		return &value, nil
	default:
		return nil, apperrors.Validation("support_tier must be l1, l2, l3, noc, or empty", nil)
	}
}

func normalizeTeamRole(teamRole string) (string, error) {
	if teamRole == "" {
		return "member", nil
	}
	switch teamRole {
	case "member", "lead":
		return teamRole, nil
	default:
		return "", apperrors.Validation("team_role must be member or lead", nil)
	}
}

func membershipToMember(membership db.TeamMembership, user db.User) db.TeamMember {
	return db.TeamMember{
		ID:          membership.ID,
		TeamID:      membership.TeamID,
		UserID:      membership.UserID,
		TeamRole:    membership.TeamRole,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		CreatedAt:   membership.CreatedAt,
	}
}

func mapTeamError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("team or membership")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if strings.Contains(pgErr.ConstraintName, "teams_name") {
			return apperrors.Conflict("team name already exists")
		}
		return apperrors.Conflict("user is already a team member")
	}
	return err
}

func TeamJSON(team db.Team) map[string]any {
	out := map[string]any{
		"id":           team.ID.String(),
		"workspace_id": team.WorkspaceID.String(),
		"name":         team.Name,
		"description":  team.Description,
		"created_at":   team.CreatedAt,
		"updated_at":   team.UpdatedAt,
	}
	if team.SupportTier != nil {
		out["support_tier"] = *team.SupportTier
	}
	return out
}

func TeamMemberJSON(member db.TeamMember) map[string]any {
	return map[string]any{
		"id":           member.ID.String(),
		"team_id":      member.TeamID.String(),
		"user_id":      member.UserID.String(),
		"team_role":    member.TeamRole,
		"email":        member.Email,
		"display_name": member.DisplayName,
		"created_at":   member.CreatedAt,
	}
}
