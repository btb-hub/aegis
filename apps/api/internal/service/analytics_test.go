package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type analyticsMockRepo struct {
	mtta       db.MetricTimeSeries
	mttr       db.MetricTimeSeries
	noise      db.NoiseStats
	onCallLoad db.OnCallLoadStats
	handoffs   db.HandoffStats
	escalation db.EscalationStats
	handoffErr error
	err        error
}

func (m *analyticsMockRepo) MTTASeries(context.Context, time.Time, time.Time) (db.MetricTimeSeries, error) {
	if m.err != nil {
		return db.MetricTimeSeries{}, m.err
	}
	return m.mtta, nil
}

func (m *analyticsMockRepo) MTTRSeries(context.Context, time.Time, time.Time) (db.MetricTimeSeries, error) {
	if m.err != nil {
		return db.MetricTimeSeries{}, m.err
	}
	return m.mttr, nil
}

func (m *analyticsMockRepo) TopNoise(context.Context, time.Time, time.Time, int) (db.NoiseStats, error) {
	if m.err != nil {
		return db.NoiseStats{}, m.err
	}
	return m.noise, nil
}

func (m *analyticsMockRepo) OnCallLoad(context.Context, time.Time, time.Time) (db.OnCallLoadStats, error) {
	if m.err != nil {
		return db.OnCallLoadStats{}, m.err
	}
	return m.onCallLoad, nil
}

func (m *analyticsMockRepo) HandoffStats(context.Context, time.Time, time.Time) (db.HandoffStats, error) {
	if m.handoffErr != nil {
		return db.HandoffStats{}, m.handoffErr
	}
	if m.err != nil {
		return db.HandoffStats{}, m.err
	}
	return m.handoffs, nil
}

func (m *analyticsMockRepo) EscalationStats(context.Context, time.Time, time.Time) (db.EscalationStats, error) {
	if m.err != nil {
		return db.EscalationStats{}, m.err
	}
	return m.escalation, nil
}

func TestAnalyticsServiceMTTA(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	bucket := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	svc := NewAnalyticsService(&analyticsMockRepo{
		mtta: db.MetricTimeSeries{
			MeanSeconds: 120,
			Count:       2,
			Series: []db.MetricBucket{
				{BucketStart: bucket, MeanSeconds: 120, Count: 2},
			},
		},
	})

	result, err := svc.MTTA(context.Background(), from, to, false)
	require.NoError(t, err)
	require.Equal(t, 120.0, result.MeanSeconds)
	require.Equal(t, 2, result.Count)
	require.Len(t, result.Series, 1)
	require.Nil(t, result.Previous)
}

func TestAnalyticsServiceMTTRComparePrevious(t *testing.T) {
	from := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	svc := NewAnalyticsService(&analyticsMockRepo{
		mttr: db.MetricTimeSeries{MeanSeconds: 300, Count: 1, Series: []db.MetricBucket{}},
	})

	result, err := svc.MTTR(context.Background(), from, to, true)
	require.NoError(t, err)
	require.NotNil(t, result.Previous)
	require.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), result.Previous.From)
	require.Equal(t, from, result.Previous.To)
	require.Equal(t, 300.0, result.Previous.MeanSeconds)
}

func TestAnalyticsServiceNoise(t *testing.T) {
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	svc := NewAnalyticsService(&analyticsMockRepo{
		noise: db.NoiseStats{Items: []db.NoiseItem{{Fingerprint: "fp-1", Title: "CPU", Count: 5}}},
	})
	stats, err := svc.Noise(context.Background(), from, to, 10)
	require.NoError(t, err)
	require.Len(t, stats.Items, 1)
}

func TestAnalyticsServiceOnCallLoad(t *testing.T) {
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	svc := NewAnalyticsService(&analyticsMockRepo{
		onCallLoad: db.OnCallLoadStats{Items: []db.OnCallLoadItem{{UserID: uuid.New(), PageCount: 2}}},
	})
	stats, err := svc.OnCallLoad(context.Background(), from, to)
	require.NoError(t, err)
	require.Len(t, stats.Items, 1)
}

func TestAnalyticsServiceEscalation(t *testing.T) {
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	svc := NewAnalyticsService(&analyticsMockRepo{
		escalation: db.EscalationStats{TotalIncidents: 3, EscalatedCount: 1, EscalatedPercent: 33.3},
	})
	stats, err := svc.Escalation(context.Background(), from, to)
	require.NoError(t, err)
	require.Equal(t, 3, stats.TotalIncidents)
}

func TestAnalyticsServiceNoiseInvalidRange(t *testing.T) {
	now := time.Now()
	svc := NewAnalyticsService(&analyticsMockRepo{})
	_, err := svc.Noise(context.Background(), now, now, 10)
	require.Error(t, err)
}

func TestAnalyticsServiceOverview(t *testing.T) {
	from := time.Now().Add(-7 * 24 * time.Hour)
	to := time.Now()
	svc := NewAnalyticsService(&analyticsMockRepo{
		mtta:       db.MetricTimeSeries{MeanSeconds: 60, Count: 1, Series: []db.MetricBucket{}},
		mttr:       db.MetricTimeSeries{MeanSeconds: 120, Count: 1, Series: []db.MetricBucket{}},
		noise:      db.NoiseStats{Items: []db.NoiseItem{}},
		onCallLoad: db.OnCallLoadStats{Items: []db.OnCallLoadItem{{UserID: uuid.New(), PageCount: 2}}},
		handoffs:   db.HandoffStats{Count: 1, MedianResponseSeconds: 90},
		escalation: db.EscalationStats{TotalIncidents: 4, EscalatedCount: 1, EscalatedPercent: 25},
	})
	overview, err := svc.Overview(context.Background(), from, to, false)
	require.NoError(t, err)
	require.Equal(t, 60.0, overview.MTTA.MeanSeconds)
	require.Equal(t, 1, overview.Handoffs.Count)
	require.Equal(t, 25.0, overview.Escalation.EscalatedPercent)
}

func TestAnalyticsServiceOverviewHandoffError(t *testing.T) {
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	svc := NewAnalyticsService(&analyticsMockRepo{
		mtta:       db.MetricTimeSeries{Series: []db.MetricBucket{}},
		mttr:       db.MetricTimeSeries{Series: []db.MetricBucket{}},
		noise:      db.NoiseStats{Items: []db.NoiseItem{}},
		onCallLoad: db.OnCallLoadStats{Items: []db.OnCallLoadItem{}},
		handoffErr: errors.New("db down"),
	})
	_, err := svc.Overview(context.Background(), from, to, false)
	require.Error(t, err)
}

func TestAnalyticsServiceOverviewComparePrevious(t *testing.T) {
	from := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	svc := NewAnalyticsService(&analyticsMockRepo{
		mtta:       db.MetricTimeSeries{MeanSeconds: 60, Count: 1, Series: []db.MetricBucket{}},
		mttr:       db.MetricTimeSeries{MeanSeconds: 120, Count: 1, Series: []db.MetricBucket{}},
		noise:      db.NoiseStats{Items: []db.NoiseItem{}},
		onCallLoad: db.OnCallLoadStats{Items: []db.OnCallLoadItem{}},
		handoffs:   db.HandoffStats{Count: 1},
		escalation: db.EscalationStats{},
	})
	overview, err := svc.Overview(context.Background(), from, to, true)
	require.NoError(t, err)
	require.NotNil(t, overview.MTTA.Previous)
	require.NotNil(t, overview.MTTR.Previous)
}

func TestAnalyticsServiceInvalidRange(t *testing.T) {
	svc := NewAnalyticsService(&analyticsMockRepo{})
	now := time.Now()
	_, err := svc.MTTA(context.Background(), now, now, false)
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestAnalyticsServiceRepoError(t *testing.T) {
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	svc := NewAnalyticsService(&analyticsMockRepo{err: errors.New("db down")})
	_, err := svc.MTTA(context.Background(), from, to, false)
	require.Error(t, err)
}

func TestMetricAnalyticsJSON(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	payload := MetricAnalyticsJSON(MetricAnalytics{
		From:        from,
		To:          to,
		MeanSeconds: 90,
		Count:       3,
		Series:      []db.MetricBucket{},
		Previous: &MetricPeriod{
			From:        time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
			To:          from,
			MeanSeconds: 120,
			Count:       2,
			Series:      []db.MetricBucket{},
		},
	})
	require.Equal(t, 90.0, payload["mean_seconds"])
	previous, ok := payload["previous"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, 120.0, previous["mean_seconds"])
}

func TestOverviewJSON(t *testing.T) {
	payload := OverviewJSON(OverviewAnalytics{
		MTTA: MetricAnalytics{
			Series: []db.MetricBucket{{BucketStart: time.Now(), MeanSeconds: 10, Count: 1}},
		},
		Noise:      db.NoiseStats{Items: []db.NoiseItem{{Count: 2}}},
		OnCallLoad: db.OnCallLoadStats{Items: []db.OnCallLoadItem{{UserID: uuid.New(), PageCount: 1}}},
		Handoffs:   db.HandoffStats{Count: 1},
		Escalation: db.EscalationStats{EscalatedPercent: 10},
	})
	require.NotNil(t, payload["mtta"])
	require.NotNil(t, payload["escalation"])
}

func TestNoiseJSONAndOnCallLoadJSON(t *testing.T) {
	noise := NoiseJSON(db.NoiseStats{Items: []db.NoiseItem{{Fingerprint: "fp", Title: "CPU", Count: 1}}})
	require.Len(t, noise["items"], 1)
	load := OnCallLoadJSON(db.OnCallLoadStats{Items: []db.OnCallLoadItem{{UserID: uuid.New(), Email: "a@b.com", PageCount: 2}}})
	require.Len(t, load["items"], 1)
}
