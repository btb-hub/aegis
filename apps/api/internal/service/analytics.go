package service

import (
	"context"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
)

type AnalyticsRepository interface {
	MTTASeries(ctx context.Context, from, to time.Time) (db.MetricTimeSeries, error)
	MTTRSeries(ctx context.Context, from, to time.Time) (db.MetricTimeSeries, error)
	TopNoise(ctx context.Context, from, to time.Time, limit int) (db.NoiseStats, error)
	OnCallLoad(ctx context.Context, from, to time.Time) (db.OnCallLoadStats, error)
	HandoffStats(ctx context.Context, from, to time.Time) (db.HandoffStats, error)
	EscalationStats(ctx context.Context, from, to time.Time) (db.EscalationStats, error)
}

type MetricPeriod struct {
	From        time.Time
	To          time.Time
	MeanSeconds float64
	Count       int
	Series      []db.MetricBucket
}

type MetricAnalytics struct {
	From        time.Time
	To          time.Time
	MeanSeconds float64
	Count       int
	Series      []db.MetricBucket
	Previous    *MetricPeriod
}

type OverviewAnalytics struct {
	From           time.Time
	To             time.Time
	MTTA           MetricAnalytics
	MTTR           MetricAnalytics
	Noise          db.NoiseStats
	OnCallLoad     db.OnCallLoadStats
	Handoffs       db.HandoffStats
	Escalation     db.EscalationStats
}

type AnalyticsService struct {
	repo AnalyticsRepository
}

func NewAnalyticsService(repo AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) validateRange(from, to time.Time) error {
	if !to.After(from) {
		return apperrors.Validation("to must be after from", nil)
	}
	return nil
}

func (s *AnalyticsService) MTTA(ctx context.Context, from, to time.Time, comparePrevious bool) (MetricAnalytics, error) {
	return s.metric(ctx, from, to, comparePrevious, s.repo.MTTASeries)
}

func (s *AnalyticsService) MTTR(ctx context.Context, from, to time.Time, comparePrevious bool) (MetricAnalytics, error) {
	return s.metric(ctx, from, to, comparePrevious, s.repo.MTTRSeries)
}

func (s *AnalyticsService) Noise(ctx context.Context, from, to time.Time, limit int) (db.NoiseStats, error) {
	if err := s.validateRange(from, to); err != nil {
		return db.NoiseStats{}, err
	}
	return s.repo.TopNoise(ctx, from, to, limit)
}

func (s *AnalyticsService) OnCallLoad(ctx context.Context, from, to time.Time) (db.OnCallLoadStats, error) {
	if err := s.validateRange(from, to); err != nil {
		return db.OnCallLoadStats{}, err
	}
	return s.repo.OnCallLoad(ctx, from, to)
}

func (s *AnalyticsService) Escalation(ctx context.Context, from, to time.Time) (db.EscalationStats, error) {
	if err := s.validateRange(from, to); err != nil {
		return db.EscalationStats{}, err
	}
	return s.repo.EscalationStats(ctx, from, to)
}

func (s *AnalyticsService) Overview(ctx context.Context, from, to time.Time, comparePrevious bool) (OverviewAnalytics, error) {
	if err := s.validateRange(from, to); err != nil {
		return OverviewAnalytics{}, err
	}

	mtta, err := s.MTTA(ctx, from, to, comparePrevious)
	if err != nil {
		return OverviewAnalytics{}, err
	}
	mttr, err := s.MTTR(ctx, from, to, comparePrevious)
	if err != nil {
		return OverviewAnalytics{}, err
	}
	noise, err := s.Noise(ctx, from, to, 10)
	if err != nil {
		return OverviewAnalytics{}, err
	}
	onCallLoad, err := s.OnCallLoad(ctx, from, to)
	if err != nil {
		return OverviewAnalytics{}, err
	}
	handoffs, err := s.repo.HandoffStats(ctx, from, to)
	if err != nil {
		return OverviewAnalytics{}, err
	}
	escalation, err := s.Escalation(ctx, from, to)
	if err != nil {
		return OverviewAnalytics{}, err
	}

	return OverviewAnalytics{
		From:       from,
		To:         to,
		MTTA:       mtta,
		MTTR:       mttr,
		Noise:      noise,
		OnCallLoad: onCallLoad,
		Handoffs:   handoffs,
		Escalation: escalation,
	}, nil
}

func (s *AnalyticsService) metric(
	ctx context.Context,
	from, to time.Time,
	comparePrevious bool,
	fetch func(context.Context, time.Time, time.Time) (db.MetricTimeSeries, error),
) (MetricAnalytics, error) {
	if err := s.validateRange(from, to); err != nil {
		return MetricAnalytics{}, err
	}

	current, err := fetch(ctx, from, to)
	if err != nil {
		return MetricAnalytics{}, err
	}

	result := MetricAnalytics{
		From:        from,
		To:          to,
		MeanSeconds: current.MeanSeconds,
		Count:       current.Count,
		Series:      current.Series,
	}

	if comparePrevious {
		duration := to.Sub(from)
		prevTo := from
		prevFrom := from.Add(-duration)
		previous, err := fetch(ctx, prevFrom, prevTo)
		if err != nil {
			return MetricAnalytics{}, err
		}
		result.Previous = &MetricPeriod{
			From:        prevFrom,
			To:          prevTo,
			MeanSeconds: previous.MeanSeconds,
			Count:       previous.Count,
			Series:      previous.Series,
		}
	}

	return result, nil
}

func MetricAnalyticsJSON(m MetricAnalytics) map[string]any {
	out := map[string]any{
		"from":         m.From,
		"to":           m.To,
		"mean_seconds": m.MeanSeconds,
		"count":        m.Count,
		"series":       metricSeriesJSON(m.Series),
	}
	if m.Previous != nil {
		out["previous"] = map[string]any{
			"from":         m.Previous.From,
			"to":           m.Previous.To,
			"mean_seconds": m.Previous.MeanSeconds,
			"count":        m.Previous.Count,
			"series":       metricSeriesJSON(m.Previous.Series),
		}
	}
	return out
}

func NoiseJSON(stats db.NoiseStats) map[string]any {
	items := make([]map[string]any, 0, len(stats.Items))
	for _, item := range stats.Items {
		items = append(items, map[string]any{
			"fingerprint": item.Fingerprint,
			"title":       item.Title,
			"count":       item.Count,
		})
	}
	return map[string]any{"items": items}
}

func OnCallLoadJSON(stats db.OnCallLoadStats) map[string]any {
	items := make([]map[string]any, 0, len(stats.Items))
	for _, item := range stats.Items {
		items = append(items, map[string]any{
			"user_id":      item.UserID,
			"display_name": item.DisplayName,
			"email":        item.Email,
			"page_count":   item.PageCount,
		})
	}
	return map[string]any{"items": items}
}

func EscalationJSON(stats db.EscalationStats) map[string]any {
	return map[string]any{
		"total_incidents":          stats.TotalIncidents,
		"escalated_count":          stats.EscalatedCount,
		"escalated_percent":        stats.EscalatedPercent,
		"mean_seconds_to_escalate": stats.MeanSecondsToEscalate,
	}
}

func OverviewJSON(o OverviewAnalytics) map[string]any {
	return map[string]any{
		"from":         o.From,
		"to":           o.To,
		"mtta":         MetricAnalyticsJSON(o.MTTA),
		"mttr":         MetricAnalyticsJSON(o.MTTR),
		"noise":        NoiseJSON(o.Noise),
		"on_call_load": OnCallLoadJSON(o.OnCallLoad),
		"handoffs": map[string]any{
			"count":                   o.Handoffs.Count,
			"median_response_seconds": o.Handoffs.MedianResponseSeconds,
		},
		"escalation": EscalationJSON(o.Escalation),
	}
}

func metricSeriesJSON(series []db.MetricBucket) []map[string]any {
	items := make([]map[string]any, 0, len(series))
	for _, bucket := range series {
		items = append(items, map[string]any{
			"bucket_start": bucket.BucketStart,
			"mean_seconds": bucket.MeanSeconds,
			"count":        bucket.Count,
		})
	}
	return items
}
