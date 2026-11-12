package db

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// INC-NUM-AC1, contract: Incident.DisplayID. Pure formatting evidence, including
// growth past two digits and fallback for old snapshots without a number.
func TestIncidentDisplayID(t *testing.T) {
	id := uuid.New()
	for _, tc := range []struct {
		number int64
		want   string
	}{
		{1, "INC-01"}, {2, "INC-02"}, {99, "INC-99"}, {100, "INC-100"}, {0, id.String()},
	} {
		require.Equal(t, tc.want, (Incident{ID: id, Number: tc.number}).DisplayID())
	}
}

// INC-NUM-DB, contract: migration 24. Real PostgreSQL in a disposable schema
// proves chronological backfill, unchanged UUIDs, concurrent allocation and
// persistence across store reads/transitions. Does not access a deployed DB.
func TestIncidentNumberMigrationAndStore(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set (isolated PostgreSQL required)")
	}
	ctx := t.Context()
	admin, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(admin.Close)
	schema := "incident_number_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(ctx, "CREATE SCHEMA "+schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, dropErr := admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		require.NoError(t, dropErr)
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	files, err := filepath.Glob("../../db/migrations/*.up.sql")
	require.NoError(t, err)
	apply := func(path string) {
		raw, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		_, applyErr := pool.Exec(ctx, string(raw))
		require.NoError(t, applyErr, path)
	}
	for _, file := range files {
		if filepath.Base(file) < "000024_" {
			apply(file)
		}
	}
	var team uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO teams(name,workspace_id) SELECT 'Number test',id FROM workspaces LIMIT 1 RETURNING id`).Scan(&team))
	older := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	newer := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	_, err = pool.Exec(ctx, `INSERT INTO incidents(id,team_id,severity,title,fingerprint,created_at) VALUES($1,$3,'critical','Older','old','2026-01-01'),($2,$3,'critical','Newer','new','2026-01-02')`, older, newer, team)
	require.NoError(t, err)
	apply("../../db/migrations/000024_incident_numbers.up.sql")
	var oldNumber, newNumber int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT number FROM incidents WHERE id=$1`, older).Scan(&oldNumber))
	require.NoError(t, pool.QueryRow(ctx, `SELECT number FROM incidents WHERE id=$1`, newer).Scan(&newNumber))
	require.EqualValues(t, 1, oldNumber)
	require.EqualValues(t, 2, newNumber)
	store := NewStore(pool)
	got, err := store.GetIncidentByID(ctx, older)
	require.NoError(t, err)
	require.Equal(t, "INC-01", got.DisplayID())
	listed, err := store.ListIncidents(ctx, "")
	require.NoError(t, err)
	require.Len(t, listed, 2)
	require.EqualValues(t, 2, listed[0].Number)
	listed, err = store.ListIncidents(ctx, "open")
	require.NoError(t, err)
	require.Len(t, listed, 2)
	got, err = store.FindOpenIncidentByFingerprint(ctx, "old", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.EqualValues(t, 1, got.Number)

	type result struct {
		number int64
		err    error
	}
	results := make(chan result, 12)
	for n := 0; n < 12; n++ {
		go func() {
			var number int64
			err := pool.QueryRow(ctx, `INSERT INTO incidents(team_id,severity,title,fingerprint) VALUES($1,'critical','Concurrent','concurrent') RETURNING number`, team).Scan(&number)
			results <- result{number, err}
		}()
	}
	seen := map[int64]bool{}
	for n := 0; n < 12; n++ {
		r := <-results
		require.NoError(t, r.err)
		require.Greater(t, r.number, int64(2))
		require.False(t, seen[r.number])
		seen[r.number] = true
	}
	var alert uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO alerts(fingerprint,status,title) VALUES('created','firing','Created') RETURNING id`).Scan(&alert))
	created, err := store.CreateIncidentWithAlert(ctx, CreateIncidentInput{TeamID: team, AlertID: alert, Severity: "critical", Title: "Created", Fingerprint: "created"})
	require.NoError(t, err)
	require.Greater(t, created.Number, int64(14))
	for _, read := range []func(context.Context, uuid.UUID) (Incident, error){store.GetIncidentForAlert, store.GetOpenIncidentForAlert} {
		linked, err := read(ctx, alert)
		require.NoError(t, err)
		require.Equal(t, created.Number, linked.Number)
	}
	_, err = store.ManualCreateFromAlert(ctx, ManualCreateFromAlertInput{AlertID: alert, TeamID: team})
	require.ErrorIs(t, err, ErrAlertAlreadyLinked)
	var actor uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO users(provider,provider_sub) VALUES('google','number-test') RETURNING id`).Scan(&actor))
	acked, err := store.AcknowledgeIncident(ctx, created.ID, actor)
	require.NoError(t, err)
	require.Equal(t, created.Number, acked.Number)
	handed, _, err := store.HandoffIncident(ctx, HandoffIncidentInput{IncidentID: created.ID, ActorID: actor, ToTeamID: team, ToUserID: actor})
	require.NoError(t, err)
	require.Equal(t, created.Number, handed.Number)
	bounced, err := store.BounceIncident(ctx, BounceIncidentInput{IncidentID: created.ID, ActorID: actor})
	require.NoError(t, err)
	require.Equal(t, created.Number, bounced.Number)
	locked, err := pool.Begin(ctx)
	require.NoError(t, err)
	read, err := (chatAckTransaction{locked}).GetIncidentByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.Number, read.Number)
	require.NoError(t, locked.Rollback(ctx))
	resolved, err := store.ResolveIncident(ctx, created.ID, actor)
	require.NoError(t, err)
	require.Equal(t, created.Number, resolved.Number)
	apply("../../db/migrations/000024_incident_numbers.down.sql")
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incidents`).Scan(&count))
	require.Equal(t, 15, count)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=$1 AND table_name='incidents' AND column_name='number'`, schema).Scan(&count))
	require.Zero(t, count)

	// Empty installations start at 1; manual creation and fingerprint linking
	// return the same persisted number. Everything remains in this private schema.
	_, err = pool.Exec(ctx, `DELETE FROM incidents`)
	require.NoError(t, err)
	apply("../../db/migrations/000024_incident_numbers.up.sql")
	first, err := store.CreateIncidentWithAlert(ctx, CreateIncidentInput{TeamID: team, AlertID: alert, Severity: "critical", Title: "First", Fingerprint: "first"})
	require.NoError(t, err)
	require.EqualValues(t, 1, first.Number)
	var manualAlert uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO alerts(fingerprint,status,title,severity) VALUES('manual','firing','Manual','critical') RETURNING id`).Scan(&manualAlert))
	manual, err := store.ManualCreateFromAlert(ctx, ManualCreateFromAlertInput{AlertID: manualAlert, TeamID: team, DedupSince: time.Now().Add(-time.Hour)})
	require.NoError(t, err)
	require.True(t, manual.Created)
	require.EqualValues(t, 2, manual.Incident.Number)
	var duplicateAlert uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO alerts(fingerprint,status,title,severity) VALUES('manual','firing','Repeat','critical') RETURNING id`).Scan(&duplicateAlert))
	duplicate, err := store.ManualCreateFromAlert(ctx, ManualCreateFromAlertInput{AlertID: duplicateAlert, TeamID: team, DedupSince: time.Now().Add(-time.Hour)})
	require.NoError(t, err)
	require.False(t, duplicate.Created)
	require.Equal(t, manual.Incident.ID, duplicate.Incident.ID)
	require.Equal(t, manual.Incident.Number, duplicate.Incident.Number)
}
