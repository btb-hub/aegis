package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/google/uuid"
)

type ExpressLinkRepository interface {
	CreateExpressLinkCode(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error)
	RedeemExpressLinkCode(ctx context.Context, code string, expressHuid uuid.UUID) (db.User, error)
	UpdateUserExpressHuid(ctx context.Context, userID, expressHuid uuid.UUID) (db.User, error)
}

type ExpressLinkService struct {
	repo ExpressLinkRepository
}

func NewExpressLinkService(repo ExpressLinkRepository) *ExpressLinkService {
	return &ExpressLinkService{repo: repo}
}

func (s *ExpressLinkService) CreateLinkCode(ctx context.Context, userID uuid.UUID) (string, error) {
	return s.repo.CreateExpressLinkCode(ctx, userID, 15*time.Minute)
}

func (s *ExpressLinkService) RedeemLinkCode(ctx context.Context, code, expressHuidRaw string) (db.User, error) {
	huid, err := db.ParseExpressHuid(expressHuidRaw)
	if err != nil {
		return db.User{}, apperrors.Validation("invalid express_user_huid", nil)
	}
	user, err := s.repo.RedeemExpressLinkCode(ctx, code, huid)
	if err != nil {
		return db.User{}, apperrors.Validation(err.Error(), nil)
	}
	return user, nil
}

func (s *ExpressLinkService) BindExpressHuid(ctx context.Context, userID uuid.UUID, expressHuidRaw string) (db.User, error) {
	huid, err := db.ParseExpressHuid(expressHuidRaw)
	if err != nil {
		return db.User{}, apperrors.Validation("invalid express_user_huid", nil)
	}
	return s.repo.UpdateUserExpressHuid(ctx, userID, huid)
}

func (s *IntegrationService) ExpressSecretKey(ctx context.Context) (string, error) {
	items, err := s.repo.ListIntegrations(ctx)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if item.Kind != "express" {
			continue
		}
		var cfg intexpress.Config
		if err := json.Unmarshal(item.Config, &cfg); err != nil {
			continue
		}
		if cfg.SecretKey != "" {
			return cfg.SecretKey, nil
		}
	}
	return "", apperrors.Validation("express integration is not configured", nil)
}
