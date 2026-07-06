package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/aegis/aegis/pkg/integrations/loader"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type IntegrationRepository interface {
	ListIntegrations(ctx context.Context) ([]db.Integration, error)
	GetIntegration(ctx context.Context, id uuid.UUID) (db.Integration, error)
	UpsertIntegration(ctx context.Context, kind, name string, config json.RawMessage, enabled bool, workspaceID *uuid.UUID) (db.Integration, error)
	DeleteIntegration(ctx context.Context, id uuid.UUID) error
	ListEnabledIntegrations(ctx context.Context) ([]integrations.IntegrationRow, error)
	GetWorkspace(ctx context.Context, id uuid.UUID) (db.Workspace, error)
}

type IntegrationService struct {
	repo      IntegrationRepository
	publicURL string
}

func NewIntegrationService(repo IntegrationRepository, publicURL string) *IntegrationService {
	return &IntegrationService{repo: repo, publicURL: publicURL}
}

func (s *IntegrationService) List(ctx context.Context) ([]db.Integration, error) {
	return s.repo.ListIntegrations(ctx)
}

func (s *IntegrationService) Upsert(ctx context.Context, kind, name string, config json.RawMessage, enabled bool, workspaceID *uuid.UUID) (db.Integration, error) {
	switch kind {
	case "jira", "slack", "express":
	default:
		return db.Integration{}, apperrors.Validation("kind must be jira, slack, or express", nil)
	}
	if name == "" {
		name = kind
	}
	if workspaceID != nil {
		if _, err := s.repo.GetWorkspace(ctx, *workspaceID); err != nil {
			return db.Integration{}, mapWorkspaceError(err)
		}
		if err := validateWorkspaceIntegrationConfig(kind, config, workspaceID != nil); err != nil {
			return db.Integration{}, err
		}
	}
	item, err := s.repo.UpsertIntegration(ctx, kind, name, config, enabled, workspaceID)
	if err != nil {
		return db.Integration{}, err
	}
	return item, nil
}

func validateWorkspaceIntegrationConfig(kind string, config json.RawMessage, isWorkspace bool) error {
	if !isWorkspace || kind != "jira" {
		return nil
	}
	var cfg map[string]any
	if err := json.Unmarshal(config, &cfg); err != nil {
		return apperrors.Validation("invalid integration config", nil)
	}
	projectKey, _ := cfg["project_key"].(string)
	if strings.TrimSpace(projectKey) == "" {
		return apperrors.Validation("project_key is required for workspace Jira integration", nil)
	}
	return nil
}

func (s *IntegrationService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteIntegration(ctx, id); err != nil {
		return mapIntegrationError(err)
	}
	return nil
}

func (s *IntegrationService) Test(ctx context.Context, id uuid.UUID) error {
	item, err := s.repo.GetIntegration(ctx, id)
	if err != nil {
		return mapIntegrationError(err)
	}
	reg, err := s.buildRegistry(ctx, []db.Integration{item})
	if err != nil {
		return err
	}
	switch item.Kind {
	case "jira":
		if provider, ok := reg.Ticket("jira"); ok {
			return provider.TestConnection(ctx)
		}
	case "slack", "express":
		if provider, ok := reg.Chat(item.Kind); ok {
			return provider.TestConnection(ctx)
		}
	}
	return apperrors.Validation("integration provider is not configured", map[string]any{"kind": item.Kind})
}

func (s *IntegrationService) LoadRegistry(ctx context.Context) (*integrations.Registry, error) {
	rows, err := s.repo.ListEnabledIntegrations(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]db.Integration, 0, len(rows))
	for _, row := range rows {
		items = append(items, db.Integration{
			ID: row.ID, Kind: row.Kind, Name: row.Name, Config: row.Config, Enabled: row.Enabled,
		})
	}
	return s.buildRegistry(ctx, items)
}

func (s *IntegrationService) buildRegistry(_ context.Context, items []db.Integration) (*integrations.Registry, error) {
	reg := integrations.NewRegistry()
	rows := make([]integrations.IntegrationRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, integrations.IntegrationRow{
			ID: item.ID, Kind: item.Kind, Name: item.Name, Config: item.Config, Enabled: item.Enabled,
		})
	}
	loader.RegisterFromRows(reg, rows, s.publicURL)
	return reg, nil
}

func IntegrationJSON(item db.Integration) map[string]any {
	var config map[string]any
	_ = json.Unmarshal(item.Config, &config)
	out := map[string]any{
		"id":         item.ID.String(),
		"kind":       item.Kind,
		"name":       item.Name,
		"config":     config,
		"enabled":    item.Enabled,
		"created_at": item.CreatedAt,
		"updated_at": item.UpdatedAt,
	}
	if item.WorkspaceID != nil {
		out["workspace_id"] = item.WorkspaceID.String()
	}
	return out
}

func mapIntegrationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("integration not found")
	}
	return err
}
