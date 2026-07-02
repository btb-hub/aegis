package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsMTTA(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	bucket := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	repo.mttaSeries = db.MetricTimeSeries{
		MeanSeconds: 180,
		Count:       2,
		Series: []db.MetricBucket{
			{BucketStart: bucket, MeanSeconds: 180, Count: 2},
		},
	}

	from := "2026-06-01T00:00:00Z"
	to := "2026-06-08T00:00:00Z"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/mtta?from="+from+"&to="+to, nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, float64(180), resp["mean_seconds"])
	require.Equal(t, float64(2), resp["count"])
	series, ok := resp["series"].([]any)
	require.True(t, ok)
	require.Len(t, series, 1)
}

func TestAnalyticsMTTRComparePrevious(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.mttrSeries = db.MetricTimeSeries{MeanSeconds: 240, Count: 1, Series: []db.MetricBucket{}}

	from := "2026-06-08T00:00:00Z"
	to := "2026-06-15T00:00:00Z"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/mttr?from="+from+"&to="+to+"&compare_previous=true", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	previous, ok := resp["previous"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(240), previous["mean_seconds"])
}

func TestAnalyticsNoise(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.noiseStats = db.NoiseStats{
		Items: []db.NoiseItem{{Fingerprint: "fp-1", Title: "CPU high", Count: 4}},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/noise?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	items, ok := resp["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
}

func TestAnalyticsOnCallLoad(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.onCallLoad = db.OnCallLoadStats{
		Items: []db.OnCallLoadItem{{UserID: uuid.New(), DisplayName: "Alex", Email: "alex@example.com", PageCount: 3}},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/on-call-load?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAnalyticsOverview(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.mttaSeries = db.MetricTimeSeries{MeanSeconds: 90, Count: 2, Series: []db.MetricBucket{}}
	repo.mttrSeries = db.MetricTimeSeries{MeanSeconds: 200, Count: 1, Series: []db.MetricBucket{}}
	repo.handoffStats = db.HandoffStats{Count: 2, MedianResponseSeconds: 60}
	repo.escalationStats = db.EscalationStats{TotalIncidents: 5, EscalatedCount: 1, EscalatedPercent: 20}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/overview?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	handoffs, ok := resp["handoffs"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), handoffs["count"])
}

func TestSetupTestAlert(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/test-alert", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp["id"])
}

func TestAnalyticsMTTARequiresRange(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/mtta", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnalyticsMTTRInvalidFrom(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/mttr?from=bad&to=2026-06-30T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnalyticsMTTAServiceError(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.mttaSeriesErr = fmt.Errorf("db down")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/mtta?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAnalyticsNoiseInvalidLimit(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/noise?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z&limit=0", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnalyticsOnCallLoadError(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.onCallLoadErr = fmt.Errorf("db down")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/on-call-load?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAnalyticsOverviewError(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.mttaSeriesErr = fmt.Errorf("db down")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/overview?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAnalyticsOverviewComparePrevious(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.mttaSeries = db.MetricTimeSeries{MeanSeconds: 60, Count: 1, Series: []db.MetricBucket{}}
	repo.mttrSeries = db.MetricTimeSeries{MeanSeconds: 120, Count: 1, Series: []db.MetricBucket{}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/overview?from=2026-06-08T00:00:00Z&to=2026-06-15T00:00:00Z&compare_previous=true", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAnalyticsNoiseError(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.noiseErr = fmt.Errorf("db down")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/noise?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAnalyticsNoiseCustomLimit(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.noiseStats = db.NoiseStats{Items: []db.NoiseItem{{Fingerprint: "fp", Title: "CPU", Count: 1}}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/noise?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z&limit=5", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAnalyticsOnCallLoadInvalidTo(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/on-call-load?from=2026-06-01T00:00:00Z&to=bad", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnalyticsOverviewEscalationError(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.mttaSeries = db.MetricTimeSeries{MeanSeconds: 60, Count: 1, Series: []db.MetricBucket{}}
	repo.mttrSeries = db.MetricTimeSeries{MeanSeconds: 120, Count: 1, Series: []db.MetricBucket{}}
	repo.escalationErr = fmt.Errorf("db down")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/overview?from=2026-06-01T00:00:00Z&to=2026-06-08T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSetupTestAlertFailure(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.alertRepo.ingestErr = fmt.Errorf("db down")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/test-alert", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
