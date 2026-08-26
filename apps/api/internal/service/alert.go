package service

import (
	"context"
	"encoding/json"
	"io"

	"github.com/aegis/aegis/pkg/alertparse"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
)

type AlertRepository interface {
	CreateAlertAndJob(ctx context.Context, input db.CreateAlertJobInput) (db.CreateAlertJobResult, error)
	ListAlerts(ctx context.Context, params db.ListAlertsParams) ([]db.Alert, error)
	CountAlerts(ctx context.Context, params db.ListAlertsParams) (int, error)
	GroupAlerts(ctx context.Context, filters db.ListAlertsParams, groupBy db.AlertGroupBy) ([]db.AlertGroupBucket, error)
	AlertAnalytics(ctx context.Context, params db.ListAlertsParams, labelKey string) (db.AlertAnalytics, error)
	StreamAlertsCSV(ctx context.Context, params db.ListAlertsParams, w io.Writer) error
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

func (s *AlertService) SendTestAlert(ctx context.Context) (uuid.UUID, error) {
	raw := json.RawMessage(`{
  "status": "firing",
  "severity": "warning",
  "title": "Aegis setup test alert",
  "annotations": {"summary": "Sent from the setup wizard"},
  "labels": {"alertname": "AegisSetupTest", "team": "platform"}
}`)
	return s.Ingest(ctx, s.secret, raw)
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

type AlertGroupResult struct {
	GroupBy string
	Groups  []db.AlertGroupBucket
	Total   int
}

func (s *AlertService) Group(ctx context.Context, filters db.ListAlertsParams, groupBy db.AlertGroupBy) (AlertGroupResult, error) {
	total, err := s.repo.CountAlerts(ctx, filters)
	if err != nil {
		return AlertGroupResult{}, err
	}
	groups, err := s.repo.GroupAlerts(ctx, filters, groupBy)
	if err != nil {
		return AlertGroupResult{}, err
	}
	groupByParam := "severity"
	if groupBy.LabelKey != "" {
		groupByParam = "label:" + groupBy.LabelKey
	}
	return AlertGroupResult{
		GroupBy: groupByParam,
		Groups:  groups,
		Total:   total,
	}, nil
}

func (s *AlertService) Analytics(ctx context.Context, filters db.ListAlertsParams, labelKey string) (db.AlertAnalytics, error) {
	return s.repo.AlertAnalytics(ctx, filters, labelKey)
}

func (s *AlertService) ExportCSV(ctx context.Context, filters db.ListAlertsParams, w io.Writer) error {
	return s.repo.StreamAlertsCSV(ctx, filters, w)
}

func AnalyticsJSON(analytics db.AlertAnalytics) map[string]any {
	bySeverity := analytics.BySeverity
	if bySeverity == nil {
		bySeverity = map[string]int{}
	}
	byStatus := analytics.ByStatus
	if byStatus == nil {
		byStatus = map[string]int{}
	}
	topLabels := make([]map[string]any, 0, len(analytics.TopLabels))
	for _, item := range analytics.TopLabels {
		topLabels = append(topLabels, map[string]any{
			"key":   item.Key,
			"value": item.Value,
			"count": item.Count,
		})
	}
	return map[string]any{
		"by_severity": bySeverity,
		"by_status":   byStatus,
		"top_labels":  topLabels,
	}
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
	if alert.IncidentID != nil {
		out["incident_id"] = alert.IncidentID.String()
	} else {
		out["incident_id"] = nil
	}
	if alert.Body != nil {
		out["body"] = *alert.Body
	}
	return out
}
