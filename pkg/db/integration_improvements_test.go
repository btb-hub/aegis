package db

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"sort"
	"testing"
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
