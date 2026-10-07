package service

import (
	"context"
	"encoding/json"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations/express"
	"github.com/google/uuid"
)

func (s *IntegrationService) ExpressCallbackIntegration(ctx context.Context) (db.Integration, express.Config, error) {
	row, err := s.repo.GetIntegrationByKind(ctx, "express")
	if err != nil {
		return row, express.Config{}, apperrors.Validation("express integration is not configured", nil)
	}
	if !row.Enabled {
		return row, express.Config{}, apperrors.Validation("express integration disabled", nil)
	}
	var cfg express.Config
	if err = json.Unmarshal(row.Config, &cfg); err != nil {
		return row, cfg, err
	}
	if _, err = express.NewFromJSON(row.Config); err != nil {
		return row, cfg, apperrors.Validation(err.Error(), nil)
	}
	return row, cfg, nil
}

func (s *IntegrationService) SlackCallbackIntegration(ctx context.Context, workspaceID uuid.UUID) (db.Integration, error) {
	row, err := s.repo.GetWorkspaceIntegration(ctx, workspaceID, "slack")
	if err != nil {
		return row, err
	}
	cfg, err := s.resolveConfigForTest(ctx, row)
	if err != nil {
		return row, err
	}
	if row.Mode != nil && *row.Mode == "inherit" {
		row, err = s.repo.GetIntegrationByKind(ctx, "slack")
		if err != nil {
			return row, err
		}
	}
	row.Config = cfg
	return row, nil
}
