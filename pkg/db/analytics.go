package db

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) MTTASeries(ctx context.Context, from, to time.Time) (MetricTimeSeries, error) {
	return s.metricSeries(ctx, from, to, "acknowledged_at")
}

func (s *Store) MTTRSeries(ctx context.Context, from, to time.Time) (MetricTimeSeries, error) {
	return s.metricSeries(ctx, from, to, "resolved_at")
}

func (s *Store) metricSeries(ctx context.Context, from, to time.Time, eventColumn string) (MetricTimeSeries, error) {
	if eventColumn != "acknowledged_at" && eventColumn != "resolved_at" {
		return MetricTimeSeries{}, fmt.Errorf("unsupported event column %q", eventColumn)
	}

	aggregateQuery := fmt.Sprintf(`
SELECT
    COALESCE(AVG(EXTRACT(EPOCH FROM (%[1]s - created_at))), 0) AS mean_seconds,
    COUNT(*)::int AS count
FROM incidents
WHERE %[1]s IS NOT NULL
  AND %[1]s >= $1 AND %[1]s < $2`, eventColumn)

	bucketQuery := fmt.Sprintf(`
SELECT
    date_trunc('day', %[1]s AT TIME ZONE 'UTC') AS bucket_start,
    AVG(EXTRACT(EPOCH FROM (%[1]s - created_at))) AS mean_seconds,
    COUNT(*)::int AS count
FROM incidents
WHERE %[1]s IS NOT NULL
  AND %[1]s >= $1 AND %[1]s < $2
GROUP BY 1
ORDER BY 1`, eventColumn)

	var series MetricTimeSeries
	if err := s.pool.QueryRow(ctx, aggregateQuery, from, to).Scan(&series.MeanSeconds, &series.Count); err != nil {
		return MetricTimeSeries{}, err
	}

	rows, err := s.pool.Query(ctx, bucketQuery, from, to)
	if err != nil {
		return MetricTimeSeries{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var bucket MetricBucket
		if err := rows.Scan(&bucket.BucketStart, &bucket.MeanSeconds, &bucket.Count); err != nil {
			return MetricTimeSeries{}, err
		}
		series.Series = append(series.Series, bucket)
	}
	if err := rows.Err(); err != nil {
		return MetricTimeSeries{}, err
	}
	if series.Series == nil {
		series.Series = []MetricBucket{}
	}
	return series, nil
}

func (s *Store) TopNoise(ctx context.Context, from, to time.Time, limit int) (NoiseStats, error) {
	if limit <= 0 {
		limit = 10
	}
	const q = `
SELECT fingerprint, MAX(title) AS title, COUNT(*)::int AS count
FROM alerts
WHERE received_at >= $1 AND received_at < $2
GROUP BY fingerprint
ORDER BY count DESC
LIMIT $3`

	rows, err := s.pool.Query(ctx, q, from, to, limit)
	if err != nil {
		return NoiseStats{}, err
	}
	defer rows.Close()

	stats := NoiseStats{Items: []NoiseItem{}}
	for rows.Next() {
		var item NoiseItem
		if err := rows.Scan(&item.Fingerprint, &item.Title, &item.Count); err != nil {
			return NoiseStats{}, err
		}
		stats.Items = append(stats.Items, item)
	}
	return stats, rows.Err()
}

func (s *Store) OnCallLoad(ctx context.Context, from, to time.Time) (OnCallLoadStats, error) {
	const q = `
SELECT u.id, u.display_name, u.email, COUNT(*)::int AS page_count
FROM timeline_events te
JOIN incidents i ON i.id = te.incident_id
JOIN users u ON u.id = i.assignee_id
WHERE te.kind IN ('paged', 'escalated')
  AND te.created_at >= $1 AND te.created_at < $2
  AND i.assignee_id IS NOT NULL
GROUP BY u.id, u.display_name, u.email
ORDER BY page_count DESC`

	rows, err := s.pool.Query(ctx, q, from, to)
	if err != nil {
		return OnCallLoadStats{}, err
	}
	defer rows.Close()

	stats := OnCallLoadStats{Items: []OnCallLoadItem{}}
	for rows.Next() {
		var item OnCallLoadItem
		if err := rows.Scan(&item.UserID, &item.DisplayName, &item.Email, &item.PageCount); err != nil {
			return OnCallLoadStats{}, err
		}
		stats.Items = append(stats.Items, item)
	}
	return stats, rows.Err()
}

func (s *Store) EscalationStats(ctx context.Context, from, to time.Time) (EscalationStats, error) {
	const q = `
WITH incidents_in_range AS (
    SELECT id, created_at
    FROM incidents
    WHERE created_at >= $1 AND created_at < $2
),
escalated AS (
    SELECT iir.id, MIN(te.created_at) AS escalated_at
    FROM incidents_in_range iir
    JOIN timeline_events te ON te.incident_id = iir.id AND te.kind = 'escalated'
    GROUP BY iir.id
)
SELECT
    (SELECT COUNT(*)::int FROM incidents_in_range) AS total_incidents,
    (SELECT COUNT(*)::int FROM escalated) AS escalated_count,
    COALESCE((
        SELECT AVG(EXTRACT(EPOCH FROM (e.escalated_at - i.created_at)))
        FROM escalated e
        JOIN incidents i ON i.id = e.id
    ), 0) AS mean_seconds`

	var stats EscalationStats
	if err := s.pool.QueryRow(ctx, q, from, to).Scan(
		&stats.TotalIncidents,
		&stats.EscalatedCount,
		&stats.MeanSecondsToEscalate,
	); err != nil {
		return EscalationStats{}, err
	}
	if stats.TotalIncidents > 0 {
		stats.EscalatedPercent = float64(stats.EscalatedCount) / float64(stats.TotalIncidents) * 100
	}
	return stats, nil
}
