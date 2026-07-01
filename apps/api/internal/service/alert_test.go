package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockAlertRepo struct {
	last db.CreateAlertJobInput
	id   uuid.UUID
}

func (m *mockAlertRepo) CreateAlertAndJob(ctx context.Context, input db.CreateAlertJobInput) (db.CreateAlertJobResult, error) {
	m.last = input
	if m.id == uuid.Nil {
		m.id = uuid.New()
	}
	return db.CreateAlertJobResult{AlertID: m.id, JobID: uuid.New()}, nil
}

func (m *mockAlertRepo) ListAlerts(ctx context.Context, params db.ListAlertsParams) ([]db.Alert, error) {
	return []db.Alert{{ID: m.id, Title: "CPU", Status: "firing", Labels: []byte(`{"team":"platform"}`)}}, nil
}

func (m *mockAlertRepo) CountAlerts(ctx context.Context, params db.ListAlertsParams) (int, error) {
	return 1, nil
}

func TestAlertIngestSuccess(t *testing.T) {
	repo := &mockAlertRepo{}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	raw := json.RawMessage(`{"status":"firing","labels":{"alertname":"HighCPU"},"annotations":{"summary":"CPU"}}`)

	id, err := svc.Ingest(context.Background(), "secret", raw)
	require.NoError(t, err)
	require.Equal(t, repo.id, id)
	require.Equal(t, "process_alert", repo.last.JobKind)
	require.Equal(t, "CPU", repo.last.Title)
}

func TestAlertIngestBadSecret(t *testing.T) {
	svc := NewAlertService("secret", []string{"alertname", "team"}, &mockAlertRepo{})
	_, err := svc.Ingest(context.Background(), "wrong", json.RawMessage(`{}`))
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "INVALID_WEBHOOK_SECRET", appErr.Code)
}

func TestAlertIngestInvalidPayload(t *testing.T) {
	svc := NewAlertService("secret", []string{"alertname", "team"}, &mockAlertRepo{})
	_, err := svc.Ingest(context.Background(), "secret", json.RawMessage(`{"status":"nope"}`))
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestAlertIngestRepoFailure(t *testing.T) {
	repo := &failAlertRepo{}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	_, err := svc.Ingest(context.Background(), "secret", json.RawMessage(`{"status":"firing","labels":{"alertname":"X"}}`))
	require.Error(t, err)
}

type failAlertRepo struct{}

func (f *failAlertRepo) CreateAlertAndJob(ctx context.Context, input db.CreateAlertJobInput) (db.CreateAlertJobResult, error) {
	return db.CreateAlertJobResult{}, errors.New("db down")
}

func (f *failAlertRepo) ListAlerts(ctx context.Context, params db.ListAlertsParams) ([]db.Alert, error) {
	return nil, errors.New("db down")
}

func (f *failAlertRepo) CountAlerts(ctx context.Context, params db.ListAlertsParams) (int, error) {
	return 0, errors.New("db down")
}

func TestAlertList(t *testing.T) {
	repo := &mockAlertRepo{id: uuid.New()}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	result, err := svc.List(context.Background(), db.ListAlertsParams{Query: "cpu"})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "CPU", result.Items[0].Title)
	require.Equal(t, 1, result.Total)
}

func TestAlertJSON(t *testing.T) {
	body := "high usage"
	alert := db.Alert{
		ID:       uuid.New(),
		Status:   "firing",
		Severity: "critical",
		Title:    "CPU",
		Body:     &body,
		Labels:   []byte(`{"team":"platform"}`),
	}
	out := AlertJSON(alert)
	require.Equal(t, "CPU", out["title"])
	require.Equal(t, "high usage", out["body"])
	labels, ok := out["labels"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, "platform", labels["team"])
}

func TestAlertJSONEmptyLabels(t *testing.T) {
	alert := db.Alert{ID: uuid.New(), Labels: []byte(`invalid`)}
	out := AlertJSON(alert)
	labels, ok := out["labels"].(map[string]string)
	require.True(t, ok)
	require.Empty(t, labels)
}
