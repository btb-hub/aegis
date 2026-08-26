package service

import (
	"context"
	"encoding/json"
	"errors"
	"bytes"
	"io"
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

func (m *mockAlertRepo) GroupAlerts(ctx context.Context, filters db.ListAlertsParams, groupBy db.AlertGroupBy) ([]db.AlertGroupBucket, error) {
	key := "critical"
	if groupBy.LabelKey != "" {
		key = "platform"
	}
	sample := db.Alert{ID: m.id, Title: "CPU", Status: "firing", Labels: []byte(`{"team":"platform"}`)}
	return []db.AlertGroupBucket{{Key: key, Count: 3, Sample: &sample}}, nil
}

func (m *mockAlertRepo) AlertAnalytics(ctx context.Context, params db.ListAlertsParams, labelKey string) (db.AlertAnalytics, error) {
	return db.AlertAnalytics{
		BySeverity: map[string]int{"critical": 1},
		ByStatus:   map[string]int{"firing": 1},
	}, nil
}

func (m *mockAlertRepo) StreamAlertsCSV(ctx context.Context, params db.ListAlertsParams, w io.Writer) error {
	_, err := w.Write([]byte("id,fingerprint,status,severity,title,body,labels,received_at\n"))
	return err
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

func TestSendTestAlert(t *testing.T) {
	repo := &mockAlertRepo{}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	id, err := svc.SendTestAlert(context.Background())
	require.NoError(t, err)
	require.Equal(t, repo.id, id)
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

func (f *failAlertRepo) GroupAlerts(ctx context.Context, filters db.ListAlertsParams, groupBy db.AlertGroupBy) ([]db.AlertGroupBucket, error) {
	return nil, errors.New("db down")
}

func (f *failAlertRepo) AlertAnalytics(ctx context.Context, params db.ListAlertsParams, labelKey string) (db.AlertAnalytics, error) {
	return db.AlertAnalytics{}, errors.New("db down")
}

func (f *failAlertRepo) StreamAlertsCSV(ctx context.Context, params db.ListAlertsParams, w io.Writer) error {
	return errors.New("db down")
}

func TestAlertList(t *testing.T) {
	repo := &mockAlertRepo{id: uuid.New()}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	result, err := svc.List(context.Background(), db.ListAlertsParams{Query: "cpu"})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "CPU", result.Items[0].Title)
	require.Equal(t, 1, result.Total)
	require.Equal(t, 1, result.Page)
	require.Equal(t, 100, result.PageSize)
}

func TestAlertListCountError(t *testing.T) {
	svc := NewAlertService("secret", []string{"alertname", "team"}, &failCountAlertRepo{})
	_, err := svc.List(context.Background(), db.ListAlertsParams{})
	require.Error(t, err)
}

func TestAlertListQueryError(t *testing.T) {
	svc := NewAlertService("secret", []string{"alertname", "team"}, &failListOnlyAlertRepo{})
	_, err := svc.List(context.Background(), db.ListAlertsParams{})
	require.Error(t, err)
}

func TestAlertListPageFromOffset(t *testing.T) {
	repo := &mockAlertRepo{id: uuid.New()}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	result, err := svc.List(context.Background(), db.ListAlertsParams{Limit: 25, Offset: 50})
	require.NoError(t, err)
	require.Equal(t, 3, result.Page)
	require.Equal(t, 25, result.PageSize)
}

func TestAlertListDefaultPageSizeWhenUnset(t *testing.T) {
	repo := &mockAlertRepo{id: uuid.New()}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	result, err := svc.List(context.Background(), db.ListAlertsParams{Limit: 0, Offset: 0})
	require.NoError(t, err)
	require.Equal(t, db.DefaultAlertListLimit, result.PageSize)
}

func TestAlertGroupBySeverity(t *testing.T) {
	repo := &mockAlertRepo{id: uuid.New()}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	result, err := svc.Group(context.Background(), db.ListAlertsParams{}, db.AlertGroupBy{Severity: true})
	require.NoError(t, err)
	require.Equal(t, "severity", result.GroupBy)
	require.Len(t, result.Groups, 1)
	require.Equal(t, "critical", result.Groups[0].Key)
	require.Equal(t, 3, result.Groups[0].Count)
	require.NotNil(t, result.Groups[0].Sample)
	require.Equal(t, 1, result.Total)
}

func TestAlertGroupByLabel(t *testing.T) {
	repo := &mockAlertRepo{id: uuid.New()}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	result, err := svc.Group(context.Background(), db.ListAlertsParams{}, db.AlertGroupBy{LabelKey: "team"})
	require.NoError(t, err)
	require.Equal(t, "label:team", result.GroupBy)
	require.Equal(t, "platform", result.Groups[0].Key)
}

func TestAlertGroupCountError(t *testing.T) {
	svc := NewAlertService("secret", []string{"alertname", "team"}, &failCountAlertRepo{})
	_, err := svc.Group(context.Background(), db.ListAlertsParams{}, db.AlertGroupBy{Severity: true})
	require.Error(t, err)
}

func TestAlertGroupQueryError(t *testing.T) {
	svc := NewAlertService("secret", []string{"alertname", "team"}, &failGroupOnlyAlertRepo{})
	_, err := svc.Group(context.Background(), db.ListAlertsParams{}, db.AlertGroupBy{Severity: true})
	require.Error(t, err)
}

type failGroupOnlyAlertRepo struct {
	mockAlertRepo
}

func (f *failGroupOnlyAlertRepo) GroupAlerts(ctx context.Context, filters db.ListAlertsParams, groupBy db.AlertGroupBy) ([]db.AlertGroupBucket, error) {
	return nil, errors.New("group failed")
}

type failCountAlertRepo struct {
	mockAlertRepo
}

func (f *failCountAlertRepo) CountAlerts(ctx context.Context, params db.ListAlertsParams) (int, error) {
	return 0, errors.New("count failed")
}

type failListOnlyAlertRepo struct {
	mockAlertRepo
}

func (f *failListOnlyAlertRepo) ListAlerts(ctx context.Context, params db.ListAlertsParams) ([]db.Alert, error) {
	return nil, errors.New("list failed")
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
	require.Nil(t, out["incident_id"])
}

func TestAlertJSONIncidentID(t *testing.T) {
	t.Run("nil incident id", func(t *testing.T) {
		alert := db.Alert{ID: uuid.New(), Labels: []byte(`{}`)}
		out := AlertJSON(alert)
		require.Contains(t, out, "incident_id")
		require.Nil(t, out["incident_id"])
	})

	t.Run("set incident id", func(t *testing.T) {
		incidentID := uuid.New()
		alert := db.Alert{ID: uuid.New(), Labels: []byte(`{}`), IncidentID: &incidentID}
		out := AlertJSON(alert)
		require.Equal(t, incidentID.String(), out["incident_id"])
	})
}

func TestAlertAnalytics(t *testing.T) {
	repo := &mockAlertRepo{id: uuid.New()}
	svc := NewAlertService("secret", []string{"alertname"}, repo)
	analytics, err := svc.Analytics(context.Background(), db.ListAlertsParams{}, "team")
	require.NoError(t, err)
	require.Equal(t, 1, analytics.BySeverity["critical"])
}

func TestAlertExportCSV(t *testing.T) {
	repo := &mockAlertRepo{id: uuid.New()}
	svc := NewAlertService("secret", []string{"alertname"}, repo)
	var buf bytes.Buffer
	require.NoError(t, svc.ExportCSV(context.Background(), db.ListAlertsParams{}, &buf))
	require.Contains(t, buf.String(), "id,fingerprint")
}

func TestAnalyticsJSON(t *testing.T) {
	out := AnalyticsJSON(db.AlertAnalytics{
		BySeverity: map[string]int{"critical": 2},
		ByStatus:   map[string]int{"firing": 2},
		TopLabels:  []db.LabelCount{{Key: "team", Value: "platform", Count: 2}},
	})
	require.EqualValues(t, 2, out["by_severity"].(map[string]int)["critical"])
	top := out["top_labels"].([]map[string]any)
	require.Len(t, top, 1)
	require.Equal(t, "platform", top[0]["value"])
}
