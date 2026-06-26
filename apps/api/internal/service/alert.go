package service

import (
	"context"
	"encoding/json"

	"github.com/aegis/aegis/pkg/alertparse"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
)

type AlertRepository interface {
	CreateAlertAndJob(ctx context.Context, input db.CreateAlertJobInput) (db.CreateAlertJobResult, error)
}

type AlertService struct {
	secret string
	repo   AlertRepository
}

func NewAlertService(secret string, repo AlertRepository) *AlertService {
	return &AlertService{secret: secret, repo: repo}
}

func (s *AlertService) Ingest(ctx context.Context, providedSecret string, raw json.RawMessage) (uuid.UUID, error) {
	if !alertparse.ValidateWebhookSecret(providedSecret, s.secret) {
		return uuid.Nil, apperrors.InvalidWebhookSecret()
	}

	parsed, err := alertparse.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.Validation("invalid alert payload", map[string]any{"error": err.Error()})
	}

	result, err := s.repo.CreateAlertAndJob(ctx, db.CreateAlertJobInput{
		Fingerprint: parsed.Fingerprint,
		Status:      parsed.Status,
		Severity:    parsed.Severity,
		Title:       parsed.Title,
		Body:        parsed.Body,
		Labels:      parsed.Labels,
		RawPayload:  parsed.Raw,
		JobKind:     "process_alert",
	})
	if err != nil {
		return uuid.Nil, err
	}
	return result.AlertID, nil
}
