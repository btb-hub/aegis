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
	ListAlerts(ctx context.Context, params db.ListAlertsParams) ([]db.Alert, error)
	CountAlerts(ctx context.Context, params db.ListAlertsParams) (int, error)
}

type AlertService struct {
	secret          string
	fingerprintKeys []string
	repo            AlertRepository
}

func NewAlertService(secret string, fingerprintKeys []string, repo AlertRepository) *AlertService {
	return &AlertService{secret: secret, fingerprintKeys: fingerprintKeys, repo: repo}
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
		Fingerprint: alertparse.FingerprintFromKeys(parsed.Labels, s.fingerprintKeys),
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

func (s *AlertService) List(ctx context.Context, params db.ListAlertsParams) (AlertListResult, error) {
	total, err := s.repo.CountAlerts(ctx, params)
	if err != nil {
		return AlertListResult{}, err
	}
	alerts, err := s.repo.ListAlerts(ctx, params)
	if err != nil {
		return AlertListResult{}, err
	}
	limit := params.Limit
	if limit <= 0 {
		limit = db.DefaultAlertListLimit
	}
	page := 1
	if limit > 0 && params.Offset > 0 {
		page = params.Offset/limit + 1
	}
	return AlertListResult{
		Items:    alerts,
		Total:    total,
		Page:     page,
		PageSize: limit,
	}, nil
}

type AlertListResult struct {
	Items    []db.Alert
	Total    int
	Page     int
	PageSize int
}

func AlertJSON(alert db.Alert) map[string]any {
	var labels map[string]string
	_ = json.Unmarshal(alert.Labels, &labels)
	if labels == nil {
		labels = map[string]string{}
	}
	out := map[string]any{
		"id":          alert.ID.String(),
		"fingerprint": alert.Fingerprint,
		"status":      alert.Status,
		"severity":    alert.Severity,
		"title":       alert.Title,
		"labels":      labels,
		"received_at": alert.ReceivedAt,
	}
	if alert.Body != nil {
		out["body"] = *alert.Body
	}
	return out
}
