package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/routing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type RoutingRepository interface {
	ListRoutingRules(ctx context.Context) ([]db.RoutingRule, error)
	GetRoutingRule(ctx context.Context, id uuid.UUID) (db.RoutingRule, error)
	CreateRoutingRule(ctx context.Context, workspaceID, teamID uuid.UUID, matchLabels map[string]string, priority int32, crossWorkspace bool) (db.RoutingRule, error)
	UpdateRoutingRule(ctx context.Context, id, workspaceID, teamID uuid.UUID, matchLabels map[string]string, priority int32, crossWorkspace bool) (db.RoutingRule, error)
	DeleteRoutingRule(ctx context.Context, id uuid.UUID) error
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	GetWorkspace(ctx context.Context, id uuid.UUID) (db.Workspace, error)
}

type RoutingService struct {
	repo RoutingRepository
}

func NewRoutingService(repo RoutingRepository) *RoutingService {
	return &RoutingService{repo: repo}
}

type RoutingRuleInput struct {
	WorkspaceID    uuid.UUID
	TeamID         uuid.UUID
	MatchLabels    map[string]string
	Priority       int32
	CrossWorkspace bool
}

func (s *RoutingService) ListRules(ctx context.Context) ([]db.RoutingRule, error) {
	return s.repo.ListRoutingRules(ctx)
}

func (s *RoutingService) CreateRule(ctx context.Context, input RoutingRuleInput) (db.RoutingRule, error) {
	if err := s.validateRule(ctx, input); err != nil {
		return db.RoutingRule{}, err
	}
	rule, err := s.repo.CreateRoutingRule(ctx, input.WorkspaceID, input.TeamID, input.MatchLabels, input.Priority, input.CrossWorkspace)
	if err != nil {
		return db.RoutingRule{}, mapRoutingError(err)
	}
	return rule, nil
}

func (s *RoutingService) UpdateRule(ctx context.Context, id uuid.UUID, input RoutingRuleInput) (db.RoutingRule, error) {
	existing, err := s.repo.GetRoutingRule(ctx, id)
	if err != nil {
		return db.RoutingRule{}, mapRoutingError(err)
	}
	if input.WorkspaceID != existing.WorkspaceID {
		return db.RoutingRule{}, apperrors.Validation("workspace_id cannot be changed on update", nil)
	}
	if err := s.validateRule(ctx, input); err != nil {
		return db.RoutingRule{}, err
	}
	rule, err := s.repo.UpdateRoutingRule(ctx, id, input.WorkspaceID, input.TeamID, input.MatchLabels, input.Priority, input.CrossWorkspace)
	if err != nil {
		return db.RoutingRule{}, mapRoutingError(err)
	}
	return rule, nil
}

func (s *RoutingService) DeleteRule(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteRoutingRule(ctx, id); err != nil {
		return mapRoutingError(err)
	}
	return nil
}

func (s *RoutingService) MatchTeam(ctx context.Context, alertLabels map[string]string) (uuid.UUID, error) {
	rules, err := s.repo.ListRoutingRules(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	parsed := make([]routing.Rule, 0, len(rules))
	for _, rule := range rules {
		labels, err := routing.ParseMatchLabels(rule.MatchLabels)
		if err != nil {
			continue
		}
		parsed = append(parsed, routing.Rule{
			TeamID:      rule.TeamID.String(),
			MatchLabels: labels,
			Priority:    int(rule.Priority),
		})
	}
	teamID, ok := routing.MatchTeam(parsed, alertLabels)
	if !ok {
		return uuid.Nil, apperrors.Validation("no routing rule matched alert labels", nil)
	}
	return uuid.Parse(teamID)
}

func (s *RoutingService) validateRule(ctx context.Context, input RoutingRuleInput) error {
	if input.WorkspaceID == uuid.Nil {
		return apperrors.Validation("workspace_id is required", nil)
	}
	if input.TeamID == uuid.Nil {
		return apperrors.Validation("team_id is required", nil)
	}
	if len(input.MatchLabels) == 0 {
		return apperrors.Validation("match_labels must not be empty", nil)
	}
	for key, value := range input.MatchLabels {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return apperrors.Validation("match_labels keys and values must not be empty", nil)
		}
	}
	if _, err := s.repo.GetWorkspace(ctx, input.WorkspaceID); err != nil {
		return mapRoutingWorkspaceError(err)
	}
	team, err := s.repo.GetTeam(ctx, input.TeamID)
	if err != nil {
		return mapRoutingTeamError(err)
	}
	if !input.CrossWorkspace && team.WorkspaceID != input.WorkspaceID {
		return apperrors.Validation("target team must belong to the rule workspace unless cross_workspace is true", nil)
	}
	return nil
}

func RoutingRuleJSON(rule db.RoutingRule) map[string]any {
	var labels map[string]string
	_ = json.Unmarshal(rule.MatchLabels, &labels)
	return map[string]any{
		"id":              rule.ID.String(),
		"workspace_id":    rule.WorkspaceID.String(),
		"team_id":         rule.TeamID.String(),
		"match_labels":    labels,
		"priority":        rule.Priority,
		"cross_workspace": rule.CrossWorkspace,
		"created_at":      rule.CreatedAt,
		"updated_at":      rule.UpdatedAt,
	}
}

func mapRoutingError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("routing rule not found")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		if strings.Contains(strings.ToLower(pgErr.ConstraintName), "workspace") {
			return apperrors.Validation("workspace not found", nil)
		}
		return apperrors.Validation("team not found", nil)
	}
	return err
}

func mapRoutingTeamError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("team")
	}
	return err
}

func mapRoutingWorkspaceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("workspace")
	}
	return err
}
