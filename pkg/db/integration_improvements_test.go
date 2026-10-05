package db

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// Each test owns a schema. The optional database is never cleaned globally.
func improvementsStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("AEGIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AEGIS_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	schema := "test_" + uuid.New().String()
	schema = `"` + schema + `"`
	_, err = admin.Exec(ctx, "CREATE SCHEMA "+schema)
	require.NoError(t, err)
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close(); _, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	files, err := filepath.Glob("../../db/migrations/*.up.sql")
	require.NoError(t, err)
	sort.Strings(files)
	for _, file := range files {
		raw, err := os.ReadFile(file)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, string(raw))
		require.NoError(t, err, file)
	}
	return NewStore(pool), pool
}
func improvementsIncident(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	var team, user, incident, connector uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO teams(name,workspace_id) SELECT 'Ops',id FROM workspaces LIMIT 1 RETURNING id`).Scan(&team))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO users(provider,provider_sub,email,display_name,role,locale) VALUES('dev','tester','tester@example.test','Engineer','member','en') RETURNING id`).Scan(&user))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO incidents(team_id,assignee_id,severity,title,fingerprint) VALUES($1,$2,'P1','Smoke','smoke') RETURNING id`, team, user).Scan(&incident))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO integrations(kind,name,config) VALUES('express','Bot','{}') RETURNING id`).Scan(&connector))
	return team, user, incident, connector
}
func TestCommentsAndResolutionAtomic(t *testing.T) {
	s, pool := improvementsStore(t)
	_, actor, id, _ := improvementsIncident(t, pool)
	ctx := context.Background()
	// A failed enqueue must roll back both resolution and its comment.
	_, err := pool.Exec(ctx, `ALTER TABLE jobs ADD CONSTRAINT reject_sync CHECK(kind<>'sync_jira')`)
	require.NoError(t, err)
	_, err = s.ResolveIncidentWithComment(ctx, id, actor, "fixed\nwith detail")
	require.Error(t, err)
	incident, err := s.GetIncidentByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "open", incident.Status)
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM timeline_events WHERE incident_id=$1`, id).Scan(&count))
	require.Zero(t, count)
	_, err = pool.Exec(ctx, `ALTER TABLE jobs DROP CONSTRAINT reject_sync`)
	require.NoError(t, err)
	_, err = s.ResolveIncidentWithComment(ctx, id, actor, "fixed\nwith detail")
	require.NoError(t, err)
	_, err = s.AddIncidentComment(ctx, id, actor, "postmortem")
	require.NoError(t, err)
	comments, err := s.PendingJiraComments(ctx, id)
	require.NoError(t, err)
	require.Len(t, comments, 2)
	require.Equal(t, "Engineer", comments[0].Author)
	require.Equal(t, "fixed\nwith detail", comments[0].Body)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE kind='sync_jira'`).Scan(&count))
	require.Equal(t, 2, count)
	// Empty resolution body remains supported.
	var other uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO incidents(team_id,severity,title,fingerprint) SELECT team_id,'P1','Other','other' FROM incidents WHERE id=$1 RETURNING id`, id).Scan(&other))
	_, err = s.ResolveIncidentWithComment(ctx, other, actor, "")
	require.NoError(t, err)
}
func TestExpressCallbacksReconcileAndDeduplicate(t *testing.T) {
	s, pool := improvementsStore(t)
	_, _, incident, connector := improvementsIncident(t, pool)
	ctx := context.Background()
	require.NoError(t, s.HandleExpressNotificationResult(ctx, connector, "early", "error", "unavailable"))
	require.NoError(t, s.RecordDelivery(ctx, incident, connector, "channel:opened", "sent", "early", ""))
	sent, err := s.HasDelivery(ctx, incident, connector, "channel:opened")
	require.NoError(t, err)
	require.True(t, sent, "accepted messages are never resent on an async failure")
	require.NoError(t, s.HandleExpressNotificationResult(ctx, connector, "early", "error", "unavailable"))
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM timeline_events WHERE incident_id=$1 AND kind='integration_failed'`, incident).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, s.RecordExpressOutbound(ctx, connector, "reply:1", "early"))
	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM express_outbound WHERE integration_id=$1`, connector).Scan(&status))
	require.Equal(t, "failed", status)
	calls := 0
	for i := 0; i < 2; i++ {
		require.NoError(t, s.RunExpressCommand(ctx, connector, "command-1", func(repo ExpressCommandRepository) (string, error) {
			calls++
			return "linked", repo.EnqueueExpressReply(ctx, connector, "command-1", "chat", "linked", "en")
		}))
	}
	require.Equal(t, 1, calls)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE kind='express_reply'`).Scan(&count))
	require.Equal(t, 1, count)
	require.Error(t, s.RunExpressCommand(ctx, connector, "command-2", func(repo ExpressCommandRepository) (string, error) {
		require.NoError(t, repo.EnqueueExpressReply(ctx, connector, "command-2", "chat", "x", "en"))
		return "", errors.New("reject")
	}))
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE kind='express_reply'`).Scan(&count))
	require.Equal(t, 1, count)
}
func TestExpiredJobLeaseAndFencing(t *testing.T) {
	s, pool := improvementsStore(t)
	ctx := context.Background()
	id, err := s.EnqueueJob(ctx, "express_reply", []byte(`{}`), time.Now())
	require.NoError(t, err)
	first, err := s.ClaimNextJob(ctx)
	require.NoError(t, err)
	require.Equal(t, id, first.ID)
	alive, err := s.RenewClaimedJob(ctx, id, first.Attempts)
	require.NoError(t, err)
	require.True(t, alive)
	_, err = pool.Exec(ctx, `UPDATE jobs SET updated_at=now()-interval '3 minutes' WHERE id=$1`, id)
	require.NoError(t, err)
	second, err := s.ClaimNextJob(ctx)
	require.NoError(t, err)
	require.Equal(t, first.Attempts+1, second.Attempts)
	alive, err = s.RenewClaimedJob(ctx, id, first.Attempts)
	require.NoError(t, err)
	require.False(t, alive)
	require.NoError(t, s.FinishClaimedJob(ctx, id, first.Attempts, "", true, false))
	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM jobs WHERE id=$1`, id).Scan(&status))
	require.Equal(t, "running", status)
	require.NoError(t, s.FinishClaimedJob(ctx, id, second.Attempts, "", true, false))
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM jobs WHERE id=$1`, id).Scan(&status))
	require.Equal(t, "done", status)
}
