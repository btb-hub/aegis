package db

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestListAlertsSearchP95At10k(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx))

	_, err = pool.Exec(ctx, `DELETE FROM incident_alerts`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `DELETE FROM alerts`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
INSERT INTO alerts (fingerprint, status, severity, title, body, labels, raw_payload, search_tsv, received_at)
SELECT
    'fp-' || g::text,
    CASE WHEN g % 3 = 0 THEN 'resolved' ELSE 'firing' END,
    CASE WHEN g % 4 = 0 THEN 'critical' WHEN g % 4 = 1 THEN 'warning' ELSE 'info' END,
    'Alert title ' || g::text,
    'Body text with searchable token omega-' || (g % 100)::text,
    jsonb_build_object('team', 'platform', 'alertname', 'HighCPU'),
    '{}'::jsonb,
    to_tsvector('english', 'Alert title ' || g::text || ' Body text with searchable token omega-' || (g % 100)::text),
    now() - (g || ' seconds')::interval
FROM generate_series(1, 10000) AS g`)
	require.NoError(t, err)

	store := NewStore(pool)
	iterations := 100
	durations := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		alerts, err := store.ListAlerts(ctx, ListAlertsParams{Query: "omega-42"})
		require.NoError(t, err)
		require.NotEmpty(t, alerts)
		durations = append(durations, time.Since(start))
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95Index := int(float64(len(durations)) * 0.95)
	if p95Index >= len(durations) {
		p95Index = len(durations) - 1
	}
	p95 := durations[p95Index]
	t.Logf("list alerts search p95 over %d iterations: %s", iterations, p95)
	require.Less(t, p95, 500*time.Millisecond, "NFR-2: alert list search p95 must stay under 500ms at 10k rows")
}

func TestListAlertsFilters(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx))

	_, err = pool.Exec(ctx, `DELETE FROM incident_alerts`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `DELETE FROM alerts`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
INSERT INTO alerts (fingerprint, status, severity, title, body, labels, raw_payload, search_tsv, received_at)
VALUES
  ('fp-1', 'firing', 'critical', 'CPU high', 'body', '{"team":"platform","env":"prod"}', '{}', to_tsvector('english', 'CPU high body'), now() - interval '2 hours'),
  ('fp-2', 'resolved', 'warning', 'Disk low', 'body', '{"team":"data","env":"prod"}', '{}', to_tsvector('english', 'Disk low body'), now() - interval '1 hour'),
  ('fp-3', 'firing', 'info', 'CPU ok', 'body', '{"team":"platform","env":"dev"}', '{}', to_tsvector('english', 'CPU ok body'), now())`)
	require.NoError(t, err)

	store := NewStore(pool)

	severityCount, err := store.CountAlerts(ctx, ListAlertsParams{Severity: "critical"})
	require.NoError(t, err)
	require.Equal(t, 1, severityCount)

	statusAlerts, err := store.ListAlerts(ctx, ListAlertsParams{Status: "firing", Limit: 10})
	require.NoError(t, err)
	require.Len(t, statusAlerts, 2)

	labelAlerts, err := store.ListAlerts(ctx, ListAlertsParams{
		LabelFilters: map[string]string{"team": "platform", "env": "prod"},
		Limit:        10,
	})
	require.NoError(t, err)
	require.Len(t, labelAlerts, 1)
	require.Equal(t, "CPU high", labelAlerts[0].Title)

	from := time.Now().Add(-90 * time.Minute)
	to := time.Now()
	rangeCount, err := store.CountAlerts(ctx, ListAlertsParams{From: &from, To: &to})
	require.NoError(t, err)
	require.Equal(t, 2, rangeCount)

	pageOne, err := store.ListAlerts(ctx, ListAlertsParams{Limit: 1, Offset: 0})
	require.NoError(t, err)
	require.Len(t, pageOne, 1)

	pageTwo, err := store.ListAlerts(ctx, ListAlertsParams{Limit: 1, Offset: 1})
	require.NoError(t, err)
	require.Len(t, pageTwo, 1)
	require.NotEqual(t, pageOne[0].ID, pageTwo[0].ID)
}

func TestGroupAlertsBySeverity(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx))

	_, err = pool.Exec(ctx, `DELETE FROM incident_alerts`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `DELETE FROM alerts`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
INSERT INTO alerts (fingerprint, status, severity, title, body, labels, raw_payload, search_tsv, received_at)
VALUES
  ('fp-1', 'firing', 'critical', 'CPU high', 'body', '{"team":"platform"}', '{}', to_tsvector('english', 'CPU high'), now() - interval '2 hours'),
  ('fp-2', 'firing', 'critical', 'Disk full', 'body', '{"team":"data"}', '{}', to_tsvector('english', 'Disk full'), now() - interval '1 hour'),
  ('fp-3', 'firing', 'warning', 'CPU ok', 'body', '{"team":"platform"}', '{}', to_tsvector('english', 'CPU ok'), now())`)
	require.NoError(t, err)

	store := NewStore(pool)

	groups, err := store.GroupAlerts(ctx, ListAlertsParams{}, AlertGroupBy{Severity: true})
	require.NoError(t, err)
	require.Len(t, groups, 2)

	counts := map[string]int{}
	for _, g := range groups {
		counts[g.Key] = g.Count
		require.NotNil(t, g.Sample)
	}
	require.Equal(t, 2, counts["critical"])
	require.Equal(t, 1, counts["warning"])
}

func TestGroupAlertsByLabel(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx))

	_, err = pool.Exec(ctx, `DELETE FROM incident_alerts`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `DELETE FROM alerts`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
INSERT INTO alerts (fingerprint, status, severity, title, body, labels, raw_payload, search_tsv, received_at)
VALUES
  ('fp-1', 'firing', 'critical', 'CPU high', 'body', '{"team":"platform","env":"prod"}', '{}', to_tsvector('english', 'CPU high'), now()),
  ('fp-2', 'firing', 'warning', 'Disk low', 'body', '{"team":"data","env":"prod"}', '{}', to_tsvector('english', 'Disk low'), now()),
  ('fp-3', 'firing', 'info', 'CPU ok', 'body', '{"team":"platform","env":"dev"}', '{}', to_tsvector('english', 'CPU ok'), now())`)
	require.NoError(t, err)

	store := NewStore(pool)

	groups, err := store.GroupAlerts(ctx, ListAlertsParams{LabelFilters: map[string]string{"env": "prod"}}, AlertGroupBy{LabelKey: "team"})
	require.NoError(t, err)
	require.Len(t, groups, 2)

	counts := map[string]int{}
	for _, g := range groups {
		counts[g.Key] = g.Count
	}
	require.Equal(t, 1, counts["platform"])
	require.Equal(t, 1, counts["data"])
}
