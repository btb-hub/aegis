package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	intjira "github.com/aegis/aegis/pkg/integrations/jira"
	"github.com/aegis/aegis/pkg/integrations/loader"
	"github.com/aegis/aegis/pkg/integrations/resolve"
	intslack "github.com/aegis/aegis/pkg/integrations/slack"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// SecretRedacted is the sentinel returned for secret config values in API responses.
const SecretRedacted = "***"

var integrationSecretKeys = []string{"api_token", "bot_token", "signing_secret", "secret_key"}

type IntegrationRepository interface {
	ListIntegrations(ctx context.Context) ([]db.Integration, error)
	GetIntegration(ctx context.Context, id uuid.UUID) (db.Integration, error)
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
	GetWorkspaceIntegration(ctx context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error)
	UpsertIntegration(ctx context.Context, kind, name string, config json.RawMessage, enabled bool, workspaceID *uuid.UUID, mode *string) (db.Integration, error)
	UpdateIntegration(ctx context.Context, id uuid.UUID, name string, config json.RawMessage, enabled bool, mode *string) (db.Integration, error)
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

func (s *IntegrationService) Get(ctx context.Context, id uuid.UUID) (db.Integration, error) {
	item, err := s.repo.GetIntegration(ctx, id)
	if err != nil {
		return db.Integration{}, mapIntegrationError(err)
	}
	return item, nil
}

func (s *IntegrationService) JSON(ctx context.Context, item db.Integration) (map[string]any, error) {
	if item.WorkspaceID == nil {
		return IntegrationJSON(item), nil
	}
	global, err := s.repo.GetIntegrationByKind(ctx, item.Kind)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return IntegrationJSON(item, false), nil
		}
		return nil, err
	}
	return IntegrationJSON(item, global.Enabled), nil
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
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	if workspaceID != nil {
		return db.Integration{}, apperrors.Conflict("workspace integration slots must be updated with PATCH")
	}
	if err := validateGlobalIntegrationConfig(kind, config, s.publicURL); err != nil {
		return db.Integration{}, err
	}
	item, err := s.repo.UpsertIntegration(ctx, kind, name, config, enabled, nil, nil)
	if err != nil {
		return db.Integration{}, err
	}
	return item, nil
}

// Update patches an integration by id. Omitted or blank secret fields keep stored values.
func (s *IntegrationService) Update(ctx context.Context, id uuid.UUID, name *string, enabled *bool, config json.RawMessage, mode *string) (db.Integration, error) {
	existing, err := s.repo.GetIntegration(ctx, id)
	if err != nil {
		return db.Integration{}, mapIntegrationError(err)
	}
	nextName := existing.Name
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return db.Integration{}, apperrors.Validation("name is required", nil)
		}
		nextName = trimmed
	}
	nextEnabled := existing.Enabled
	if enabled != nil {
		nextEnabled = *enabled
	}
	nextConfig := existing.Config
	if len(config) > 0 {
		merged, mergeErr := mergeIntegrationPatchConfig(existing.Config, config)
		if mergeErr != nil {
			return db.Integration{}, apperrors.Validation("invalid integration config", nil)
		}
		nextConfig = merged
	}
	if existing.WorkspaceID != nil {
		nextMode, modeErr := workspaceIntegrationMode(existing.Mode, mode)
		if modeErr != nil {
			return db.Integration{}, modeErr
		}
		if nextMode == "inherit" {
			nextConfig, err = stripWorkspaceSecrets(existing.Kind, nextConfig)
			if err != nil {
				return db.Integration{}, apperrors.Validation("invalid integration config", nil)
			}
		} else if err := validateGlobalIntegrationConfig(existing.Kind, nextConfig, s.publicURL); err != nil {
			return db.Integration{}, err
		}
		mode = &nextMode
	} else {
		if mode != nil {
			return db.Integration{}, apperrors.Validation("mode is only valid for workspace integration slots", nil)
		}
		if err := validateGlobalIntegrationConfig(existing.Kind, nextConfig, s.publicURL); err != nil {
			return db.Integration{}, err
		}
	}
	item, err := s.repo.UpdateIntegration(ctx, id, nextName, nextConfig, nextEnabled, mode)
	if err != nil {
		return db.Integration{}, mapIntegrationError(err)
	}
	return item, nil
}

func workspaceIntegrationMode(existing, patch *string) (string, error) {
	mode := "inherit"
	if existing != nil {
		mode = *existing
	}
	if patch != nil {
		mode = strings.TrimSpace(*patch)
	}
	if mode != "inherit" && mode != "custom" {
		return "", apperrors.Validation("mode must be inherit or custom", nil)
	}
	return mode, nil
}

func stripWorkspaceSecrets(kind string, config json.RawMessage) (json.RawMessage, error) {
	var current map[string]any
	if err := json.Unmarshal(config, &current); err != nil {
		return nil, err
	}
	overlay := map[string]any{}
	if kind == "jira" {
		if projectKey, ok := current["project_key"]; ok {
			overlay["project_key"] = projectKey
		}
	}
	if kind == "slack" {
		if channelID, ok := current["channel_id"]; ok {
			overlay["channel_id"] = channelID
		}
	}
	return json.Marshal(overlay)
}

func validateWorkspaceIntegrationConfig(kind string, config json.RawMessage) error {
	if kind != "jira" {
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

func validateGlobalIntegrationConfig(kind string, config json.RawMessage, publicURL string) error {
	if err := parseProviderConfig(kind, config, publicURL); err != nil {
		return apperrors.Validation(err.Error(), map[string]any{"kind": kind})
	}
	return nil
}

func parseProviderConfig(kind string, config json.RawMessage, publicURL string) error {
	switch kind {
	case "jira":
		_, err := intjira.NewFromJSON(config)
		return err
	case "slack":
		_, err := intslack.NewFromJSON(config, publicURL)
		return err
	case "express":
		_, err := intexpress.NewFromJSON(config)
		return err
	default:
		return errors.New("unknown integration kind")
	}
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
	cfg, err := s.resolveConfigForTest(ctx, item)
	if err != nil {
		return err
	}
	switch item.Kind {
	case "jira":
		provider, err := intjira.NewFromJSON(cfg)
		if err != nil {
			return apperrors.Validation(err.Error(), map[string]any{"kind": item.Kind})
		}
		if err := provider.TestConnection(ctx); err != nil {
			return apperrors.Validation(err.Error(), map[string]any{"kind": item.Kind})
		}
		return nil
	case "slack":
		provider, err := intslack.NewFromJSON(cfg, s.publicURL)
		if err != nil {
			return apperrors.Validation(err.Error(), map[string]any{"kind": item.Kind})
		}
		if err := provider.TestConnection(ctx); err != nil {
			return apperrors.Validation(err.Error(), map[string]any{"kind": item.Kind})
		}
		return nil
	case "express":
		provider, err := intexpress.NewFromJSON(cfg)
		if err != nil {
			return apperrors.Validation(err.Error(), map[string]any{"kind": item.Kind})
		}
		if err := provider.TestConnection(ctx); err != nil {
			return apperrors.Validation(err.Error(), map[string]any{"kind": item.Kind})
		}
		return nil
	}
	return apperrors.Validation("unsupported integration kind", map[string]any{"kind": item.Kind})
}

func (s *IntegrationService) resolveConfigForTest(ctx context.Context, item db.Integration) (json.RawMessage, error) {
	if item.WorkspaceID == nil {
		return item.Config, nil
	}
	mode, err := workspaceIntegrationMode(item.Mode, nil)
	if err != nil {
		return nil, err
	}
	global, err := s.repo.GetIntegrationByKind(ctx, item.Kind)
	var globalSlot *resolve.Slot
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	} else {
		globalSlot = &resolve.Slot{Enabled: global.Enabled, Config: global.Config}
	}
	result := resolve.Resolve(resolve.Input{
		Kind:   item.Kind,
		Slot:   &resolve.Slot{Mode: mode, Enabled: item.Enabled, Config: item.Config},
		Global: globalSlot,
	})
	if !result.OK {
		return nil, integrationResolveError(item.Kind, result.Reason)
	}
	return result.Config, nil
}

func integrationResolveError(kind, reason string) error {
	var message string
	switch reason {
	case resolve.ReasonSlotDisabled:
		message = kind + " integration slot is disabled"
	case resolve.ReasonSlotMissing:
		message = kind + " integration slot is missing"
	case resolve.ReasonNoGlobal:
		message = "no global " + kind + " integration; add a global integration or switch the slot to custom"
	case resolve.ReasonGlobalDisabled:
		message = "global " + kind + " integration is disabled"
	case resolve.ReasonCustomIncomplete:
		message = kind + " config incomplete"
	default:
		message = kind + " integration is unavailable"
	}
	return apperrors.Validation(message, map[string]any{"kind": kind, "reason": reason})
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

func IntegrationJSON(item db.Integration, globalEnabled ...bool) map[string]any {
	var config map[string]any
	_ = json.Unmarshal(item.Config, &config)
	if config == nil {
		config = map[string]any{}
	}
	redacted := redactIntegrationConfig(config)
	out := map[string]any{
		"id":              item.ID.String(),
		"kind":            item.Kind,
		"name":            item.Name,
		"config":          redacted,
		"enabled":         item.Enabled,
		"config_complete": integrationConfigComplete(item),
		"created_at":      item.CreatedAt,
		"updated_at":      item.UpdatedAt,
	}
	if item.WorkspaceID != nil {
		out["workspace_id"] = item.WorkspaceID.String()
		mode := "inherit"
		if item.Mode != nil {
			mode = *item.Mode
		}
		out["mode"] = mode
		hasGlobal := len(globalEnabled) > 0 && globalEnabled[0]
		out["slot_status"] = integrationSlotStatus(item, mode, hasGlobal)
	}
	return out
}

func integrationSlotStatus(item db.Integration, mode string, globalEnabled bool) string {
	if !item.Enabled {
		return "disabled"
	}
	if mode == "custom" {
		if resolve.ConfigComplete(item.Kind, item.Config) {
			return "ready"
		}
		return "needs_setup"
	}
	if globalEnabled {
		return "using_global"
	}
	return "missing"
}

func integrationConfigComplete(item db.Integration) bool {
	if item.WorkspaceID != nil {
		return validateWorkspaceIntegrationConfig(item.Kind, item.Config) == nil
	}
	return parseProviderConfig(item.Kind, item.Config, "") == nil
}

func redactIntegrationConfig(config map[string]any) map[string]any {
	out := make(map[string]any, len(config))
	for key, value := range config {
		out[key] = value
	}
	for _, key := range integrationSecretKeys {
		if value, ok := out[key]; ok {
			if str, isStr := value.(string); isStr && strings.TrimSpace(str) == "" {
				continue
			}
			if value == nil {
				continue
			}
			out[key] = SecretRedacted
		}
	}
	return out
}

func mergeIntegrationPatchConfig(existing, patch json.RawMessage) (json.RawMessage, error) {
	var base map[string]any
	if err := json.Unmarshal(existing, &base); err != nil {
		return nil, err
	}
	if base == nil {
		base = map[string]any{}
	}
	var next map[string]any
	if err := json.Unmarshal(patch, &next); err != nil {
		return nil, err
	}
	secretSet := make(map[string]struct{}, len(integrationSecretKeys))
	for _, key := range integrationSecretKeys {
		secretSet[key] = struct{}{}
	}
	for key, value := range next {
		if _, isSecret := secretSet[key]; isSecret {
			if shouldKeepExistingSecret(value) {
				continue
			}
		}
		base[key] = value
	}
	return json.Marshal(base)
}

func shouldKeepExistingSecret(value any) bool {
	if value == nil {
		return true
	}
	str, ok := value.(string)
	if !ok {
		return false
	}
	trimmed := strings.TrimSpace(str)
	return trimmed == "" || trimmed == SecretRedacted
}

func mergeIntegrationConfigMaps(global, override json.RawMessage) (json.RawMessage, error) {
	var globalMap map[string]any
	if err := json.Unmarshal(global, &globalMap); err != nil {
		return nil, err
	}
	var overrideMap map[string]any
	if err := json.Unmarshal(override, &overrideMap); err != nil {
		return nil, err
	}
	if globalMap == nil {
		globalMap = map[string]any{}
	}
	for key, value := range overrideMap {
		globalMap[key] = value
	}
	return json.Marshal(globalMap)
}

func mapIntegrationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound("integration not found")
	}
	return err
}
