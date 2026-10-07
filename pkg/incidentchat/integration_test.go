package incidentchat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/incidentack"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Each test has its own schema, including migrations, and never touches the
// caller's application tables. The database must be a disposable test database.
func chatDatabase(t *testing.T) (context.Context, *pgxpool.Pool, *db.Store) {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	schema := "chat_test_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err = admin.Exec(ctx, "CREATE SCHEMA "+schema)
	require.NoError(t, err)
	cfg, err := pgxpool.ParseConfig(databaseURL)
	require.NoError(t, err)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close(); _, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	paths, err := filepath.Glob(filepath.Join("..", "..", "db", "migrations", "*.up.sql"))
	require.NoError(t, err)
	sort.Strings(paths)
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, string(raw))
		require.NoError(t, err, path)
	}
	require.NoError(t, i18n.LoadMessages(filepath.Join("..", "i18n", "messages")))
	return ctx, pool, db.NewStore(pool)
}

func seedChatIncident(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slackURL string) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	workspace, team, user, incident, global, express := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO workspaces(id,name,slug) VALUES($1,'Chat test','chat-test')`, workspace)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams(id,workspace_id,name,slack_channel_id,slack_user_group_id) VALUES($1,$2,'Platform','C-TEAM','S-ONCALL')`, team, workspace)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users(id,provider,provider_sub,email,display_name,role,locale,slack_user_id) VALUES($1,'slack','chat-user','chat@test.local','Alice','member','ru','U-ALICE')`, user)
	require.NoError(t, err)
	cfg, _ := json.Marshal(map[string]string{"bot_token": "test", "signing_secret": "secret", "api_base_url": slackURL})
	_, err = pool.Exec(ctx, `INSERT INTO integrations(id,kind,name,enabled,config) VALUES($1,'slack','test',true,$2),($3,'express','test',true,'{"bot_id":"test","host":"https://cts.example.com","secret_key":"test","oncall_group_chat_id":"EXPRESS-GROUP"}')`, global, cfg, express)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO integrations(kind,name,workspace_id,mode,enabled) VALUES('slack','slot',$1,'inherit',true)`, workspace)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO incidents(id,team_id,assignee_id,severity,title,fingerprint) VALUES($1,$2,$3,'critical','CPU high','fp-chat')`, incident, team, user)
	require.NoError(t, err)
	return incident, user, global
}

func event(t *testing.T, ctx context.Context, pool *pgxpool.Pool, incident uuid.UUID, kind string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO timeline_events(id,incident_id,kind,payload) VALUES($1,$2,$3,'{}')`, id, incident, kind)
	require.NoError(t, err)
	return id
}

func TestChannelTransactionsOrderingAndMigration(t *testing.T) {
	ctx, pool, store := chatDatabase(t)
	incident, user, global := seedChatIncident(t, ctx, pool, "https://slack.example.com")
	// Rolled-back timeline events cannot leave an outbox entry.
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `INSERT INTO timeline_events(incident_id,kind) VALUES($1,'created')`, incident)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incident_channel_events`).Scan(&count))
	require.Zero(t, count)

	created := event(t, ctx, pool, incident, "created")
	acknowledged, err := store.AcknowledgeIncident(ctx, incident, user)
	require.NoError(t, err)
	require.Equal(t, "acknowledged", acknowledged.Status)
	_, err = store.ResolveIncident(ctx, incident, user)
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incident_channel_events`).Scan(&count))
	require.Equal(t, 3, count)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incident_channel_deliveries`).Scan(&count))
	require.Equal(t, 6, count)
	d, err := store.ClaimChannelDelivery(ctx)
	require.NoError(t, err)
	require.Equal(t, created, d.EventID)
	require.Equal(t, "open", d.Snapshot.Incident.Status)
	require.Equal(t, "ru", d.Snapshot.Locale)
	require.NoError(t, store.FinishChannelDelivery(ctx, d.ID, "pending", "", "temporary", time.Now().Add(time.Hour)))
	other, err := store.ClaimChannelDelivery(ctx)
	require.NoError(t, err)
	require.Equal(t, created, other.EventID)
	require.NotEqual(t, d.Provider, other.Provider)
	require.NoError(t, store.FinishChannelDelivery(ctx, other.ID, "sent", "message", "", time.Now()))
	later, err := store.ClaimChannelDelivery(ctx)
	require.NoError(t, err)
	require.Equal(t, "acknowledged", later.Kind)
	require.Equal(t, other.Provider, later.Provider)
	require.Equal(t, "Alice", later.Snapshot.ActorName)
	// Channel delivery records never suppress existing personal pages.
	sent, err := store.HasNotification(ctx, incident, global)
	require.NoError(t, err)
	require.False(t, sent)
	down, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", "000020_incident_chat_outboxes.down.sql"))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(down))
	require.NoError(t, err)
	up, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", "000020_incident_chat_outboxes.up.sql"))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(up))
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incident_channel_events`).Scan(&count))
	require.Zero(t, count, "no historical backfill")
}

func TestAckFeedbackRetryDedupAndConcurrentClicks(t *testing.T) {
	var sends int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/chat.postEphemeral", r.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "U-ALICE", payload["user"])
		require.Equal(t, "C-TEAM", payload["channel"])
		require.Contains(t, payload["text"], "подтверждён")
		sends++
		if sends == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"message_ts":"feedback"}`))
	}))
	defer server.Close()
	ctx, pool, store := chatDatabase(t)
	incident, _, integration := seedChatIncident(t, ctx, pool, server.URL)
	request := db.ChatAckRequest{Provider: "slack", DedupKey: "click-1", IncidentID: incident, IntegrationID: integration, UserIdentity: "U-ALICE", ChatID: "C-TEAM", Locale: "en"}
	require.NoError(t, store.EnqueueEscalation(ctx, incident, time.Now().Add(time.Hour)))
	require.NoError(t, store.EnqueueChatAck(ctx, request))
	require.NoError(t, store.EnqueueChatAck(ctx, request))
	runner := New(store, "https://aegis.example.com")
	require.NoError(t, runner.DeliverAck(ctx))
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM chat_ack_requests`).Scan(&count))
	require.Equal(t, 1, count)
	var outcome []byte
	require.NoError(t, pool.QueryRow(ctx, `SELECT outcome FROM chat_ack_requests`).Scan(&outcome))
	require.Contains(t, string(outcome), `"code": "acknowledged"`)
	_, err := pool.Exec(ctx, `UPDATE chat_ack_requests SET next_at=now()`)
	require.NoError(t, err)
	require.NoError(t, runner.DeliverAck(ctx))
	require.Equal(t, 2, sends)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM timeline_events WHERE kind='acknowledged'`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE kind='escalate_incident' AND status='pending'`).Scan(&count))
	require.Zero(t, count)
	// Concurrent distinct clicks observe the current state without duplicating it.
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := incidentack.Acknowledge(ctx, store, incident, "slack", "U-ALICE")
			if err != nil {
				t.Error(err)
				return
			}
			if result.Code != "already_acknowledged" {
				t.Errorf("unexpected result %s", result.Code)
			}
		}()
	}
	wg.Wait()
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM timeline_events WHERE kind='acknowledged'`).Scan(&count))
	require.Equal(t, 1, count)
	unknown, err := incidentack.Acknowledge(ctx, store, incident, "slack", "U-UNKNOWN")
	require.NoError(t, err)
	require.Equal(t, "link_required", unknown.Code)
}

func TestAcknowledgementOutcomeFailureRollsBackStateAndChannelEvent(t *testing.T) {
	ctx, pool, store := chatDatabase(t)
	incident, _, integration := seedChatIncident(t, ctx, pool, "https://slack.example.com")
	request := db.ChatAckRequest{Provider: "slack", DedupKey: "atomic-click", IncidentID: incident, IntegrationID: integration, UserIdentity: "U-ALICE", ChatID: "C-TEAM", Locale: "en"}
	require.NoError(t, store.EnqueueChatAck(ctx, request))
	claimed, err := store.ClaimChatAckAction(ctx)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `CREATE FUNCTION reject_outcome() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'injected persistence failure'; END$$;
 CREATE TRIGGER reject_outcome BEFORE UPDATE ON chat_ack_requests FOR EACH ROW EXECUTE FUNCTION reject_outcome()`)
	require.NoError(t, err)
	_, err = store.ProcessChatAck(ctx, claimed)
	require.ErrorContains(t, err, "injected persistence failure")
	current, err := store.GetIncidentByID(ctx, incident)
	require.NoError(t, err)
	require.Equal(t, "open", current.Status)
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incident_channel_events`).Scan(&count))
	require.Zero(t, count)
	_, err = pool.Exec(ctx, `DROP TRIGGER reject_outcome ON chat_ack_requests`)
	require.NoError(t, err)
	outcome, err := store.ProcessChatAck(ctx, claimed)
	require.NoError(t, err)
	require.Contains(t, string(outcome), `"code":"acknowledged"`)
	// The saved response survives later state changes and a worker restart.
	user, err := store.GetUserBySlackID(ctx, "U-ALICE")
	require.NoError(t, err)
	_, err = store.ResolveIncident(ctx, incident, user.ID)
	require.NoError(t, err)
	replayed, err := store.ProcessChatAck(ctx, claimed)
	require.NoError(t, err)
	require.JSONEq(t, string(outcome), string(replayed))
}

func TestUnassignedEscalationIsOncePerJob(t *testing.T) {
	ctx, pool, store := chatDatabase(t)
	incident, _, _ := seedChatIncident(t, ctx, pool, "https://slack.example.com")
	_, err := pool.Exec(ctx, `UPDATE incidents SET assignee_id=NULL WHERE id=$1`, incident)
	require.NoError(t, err)
	event(t, ctx, pool, incident, "created")
	for range 2 {
		require.NoError(t, store.QueueChannelEscalation(ctx, incident, "job-1"))
	}
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incident_channel_events WHERE kind='escalated'`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incident_channel_deliveries`).Scan(&count))
	require.Equal(t, 4, count)
	d, err := store.ClaimChannelDelivery(ctx)
	require.NoError(t, err)
	require.Equal(t, "en", d.Snapshot.Locale)
	// Leases expire after restart and are reclaimed, while live leases aren't.
	_, err = pool.Exec(ctx, `UPDATE incident_channel_deliveries SET leased_until=now()-interval '1 second' WHERE id=$1`, d.ID)
	require.NoError(t, err)
	reclaimed, err := store.ClaimChannelDelivery(ctx)
	require.NoError(t, err)
	require.Equal(t, d.ID, reclaimed.ID)
	require.NoError(t, store.FinishChannelDelivery(ctx, d.ID, "failed", "", "forbidden", time.Now()))
}

func TestRetryDisposition(t *testing.T) {
	for attempt, delay := range []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 120 * time.Second} {
		before := time.Now()
		status, next := RetryDisposition(attempt+1, fmt.Errorf("network unavailable"))
		require.Equal(t, "pending", status)
		require.WithinDuration(t, before.Add(delay), next, 100*time.Millisecond)
	}
	status, _ := RetryDisposition(5, fmt.Errorf("network unavailable"))
	require.Equal(t, "failed", status)
}

func TestConcurrentOpenAcknowledgementRequestsCommitOneTransition(t *testing.T) {
	ctx, pool, store := chatDatabase(t)
	incident, _, integration := seedChatIncident(t, ctx, pool, "https://slack.example.com")
	var requests []db.ChatAckRequest
	for n := range 8 {
		require.NoError(t, store.EnqueueChatAck(ctx, db.ChatAckRequest{Provider: "slack", DedupKey: fmt.Sprint(n), IncidentID: incident, IntegrationID: integration, UserIdentity: "U-ALICE", Locale: "en"}))
		r, err := store.ClaimChatAckAction(ctx)
		require.NoError(t, err)
		requests = append(requests, r)
	}
	start := make(chan struct{})
	results := make(chan incidentack.Result, 8)
	errs := make(chan error, 8)
	for _, req := range requests {
		go func(req db.ChatAckRequest) {
			<-start
			raw, err := store.ProcessChatAck(ctx, req)
			var result incidentack.Result
			if err == nil {
				err = json.Unmarshal(raw, &result)
			}
			errs <- err
			results <- result
		}(req)
	}
	close(start)
	confirmed := 0
	for range 8 {
		require.NoError(t, <-errs)
		result := <-results
		if result.Code == "acknowledged" {
			confirmed++
		} else {
			require.Equal(t, "already_acknowledged", result.Code)
		}
	}
	require.Equal(t, 1, confirmed)
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM timeline_events WHERE kind='acknowledged'`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incident_channel_events WHERE kind='acknowledged'`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM chat_ack_requests WHERE outcome IS NOT NULL`).Scan(&count))
	require.Equal(t, 8, count)
}

func TestDeletedIncidentFeedbackAndDisabledWorkspace(t *testing.T) {
	var sends int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "U-ALICE", payload["user"])
		require.Contains(t, payload["text"], "не найден")
		sends++
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	ctx, pool, store := chatDatabase(t)
	incident, _, integration := seedChatIncident(t, ctx, pool, server.URL)
	var workspace uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `SELECT workspace_id FROM teams`).Scan(&workspace))
	req := db.ChatAckRequest{Provider: "slack", DedupKey: "deleted", IncidentID: incident, IntegrationID: integration, WorkspaceID: &workspace, UserIdentity: "U-ALICE", ChatID: "C-TEAM", Locale: "en"}
	require.NoError(t, store.EnqueueChatAck(ctx, req))
	_, err := pool.Exec(ctx, `DELETE FROM incidents WHERE id=$1`, incident)
	require.NoError(t, err)
	runner := New(store, "https://aegis.local")
	require.NoError(t, runner.DeliverAck(ctx))
	require.Equal(t, 1, sends)
	req.DedupKey = "disabled"
	require.NoError(t, store.EnqueueChatAck(ctx, req))
	_, err = pool.Exec(ctx, `UPDATE integrations SET enabled=false WHERE workspace_id=$1`, workspace)
	require.NoError(t, err)
	require.NoError(t, runner.DeliverAck(ctx))
	require.Equal(t, 1, sends)
	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM chat_ack_requests WHERE dedup_key='disabled'`).Scan(&status))
	require.Equal(t, "failed", status)
}

func TestUnassignedIncidentPostsDespiteFailedPersonalPage(t *testing.T) {
	sends := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/chat.postMessage", r.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "C-TEAM", payload["channel"])
		raw, _ := json.Marshal(payload)
		require.Contains(t, string(raw), "ack_incident")
		require.Contains(t, string(raw), "S-ONCALL")
		sends++
		_, _ = w.Write([]byte(`{"ok":true,"ts":"posted"}`))
	}))
	defer server.Close()
	ctx, pool, store := chatDatabase(t)
	incident, _, integration := seedChatIncident(t, ctx, pool, server.URL)
	_, err := pool.Exec(ctx, `UPDATE incidents SET assignee_id=NULL WHERE id=$1`, incident)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE integrations SET enabled=false WHERE kind='express'`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO notifications(incident_id,integration_id,status) VALUES($1,$2,'failed')`, incident, integration)
	require.NoError(t, err)
	event(t, ctx, pool, incident, "created")
	require.NoError(t, New(store, "https://aegis.local").DeliverChannel(ctx))
	require.Equal(t, 1, sends)
	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM notifications`).Scan(&status))
	require.Equal(t, "failed", status)
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM incident_channel_deliveries`).Scan(&status))
	require.Equal(t, "sent", status)
	_, err = pool.Exec(ctx, `UPDATE integrations SET enabled=false WHERE workspace_id IS NOT NULL`)
	require.NoError(t, err)
	event(t, ctx, pool, incident, "created")
	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM incident_channel_deliveries`).Scan(&count))
	require.Equal(t, 1, count, "disabled destinations aren't queued")
}
