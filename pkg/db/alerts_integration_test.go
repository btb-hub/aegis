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
