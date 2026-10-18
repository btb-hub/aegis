package db

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/anomalymodule"
	"github.com/aegis/aegis/pkg/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const anomalyDefaultWorkspace = "00000000-0000-0000-0000-000000000001"

func anomalyFixture() anomalymodule.Event {
	return anomalymodule.Event{ContractVersion: 1, EventID: uuid.NewString(), InstallationID: "install", WorkspaceID: anomalyDefaultWorkspace, MetricID: uuid.NewString(), SeriesID: strings.Repeat("a", 64), EpisodeID: uuid.NewString(), Sequence: 1, State: "firing", OccurredAt: time.Now().UTC(), Title: "CPU spike", Severity: "warning", Labels: map[string]string{"team": "platform", "aegis_module": "forged", "aegis_workspace_id": "forged"}}
}
func TestAnomalyLifecycleAndConcurrentReplay(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	e := anomalyFixture()
	require.NoError(t, e.Validate(time.Now()))
	results := make([]AnomalyEventResult, 12)
	errs := make([]error, 12)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) { defer wg.Done(); results[i], errs[i] = s.AcceptAnomalyEvent(ctx, e) }(i)
	}
	wg.Wait()
	for i := range results {
		require.NoError(t, errs[i])
		require.Equal(t, results[0].AlertID, results[i].AlertID)
	}
	var count int
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE kind='process_alert'`).Scan(&count))
	require.Equal(t, 1, count)
	alert, err := s.GetAlertByID(ctx, results[0].AlertID)
	require.NoError(t, err)
	var labels map[string]string
	require.NoError(t, json.Unmarshal(alert.Labels, &labels))
	require.Equal(t, anomalyDefaultWorkspace, labels["aegis_workspace_id"])
	require.Equal(t, "anomaly-detector", labels["aegis_module"])
	conflicting := e
	conflicting.Title = "different"
	_, err = s.AcceptAnomalyEvent(ctx, conflicting)
	require.ErrorContains(t, err, "already been used")
	team := uuid.New()
	_, err = s.pool.Exec(ctx, `INSERT INTO teams(id,workspace_id,name)VALUES($1,$2,'platform')`, team, anomalyDefaultWorkspace)
	require.NoError(t, err)
	incident, err := s.ManualCreateFromAlert(ctx, ManualCreateFromAlertInput{AlertID: alert.ID, TeamID: team, DedupSince: time.Now().Add(-time.Hour)})
	require.NoError(t, err)
	require.True(t, incident.Created)
	update := e
	update.EventID = uuid.NewString()
	update.Sequence = 2
	update.Title = "Still high"
	v := 42.1
	update.Value = &v
	result, err := s.AcceptAnomalyEvent(ctx, update)
	require.NoError(t, err)
	require.Equal(t, alert.ID, result.AlertID)
	recover := update
	recover.EventID = uuid.NewString()
	recover.Sequence = 3
	recover.State = "resolved"
	result, err = s.AcceptAnomalyEvent(ctx, recover)
	require.NoError(t, err)
	require.False(t, result.Ignored)
	result, err = s.AcceptAnomalyEvent(ctx, recover)
	require.NoError(t, err)
	require.True(t, result.Duplicate)
	stale := e
	stale.EventID = uuid.NewString()
	result, err = s.AcceptAnomalyEvent(ctx, stale)
	require.NoError(t, err)
	require.True(t, result.Ignored)
	delayed := update
	delayed.EventID = uuid.NewString()
	delayed.Sequence = 99
	result, err = s.AcceptAnomalyEvent(ctx, delayed)
	require.NoError(t, err)
	require.True(t, result.Ignored)
	alert, err = s.GetAlertByID(ctx, alert.ID)
	require.NoError(t, err)
	require.Equal(t, "resolved", alert.Status)
	events, err := s.ListTimelineEvents(ctx, incident.Incident.ID)
	require.NoError(t, err)
	recovered := 0
	for _, event := range events {
		if event.Kind == "anomaly_recovered" {
			recovered++
		}
	}
	require.Equal(t, 1, recovered)
	existing, err := s.GetIncidentByID(ctx, incident.Incident.ID)
	require.NoError(t, err)
	require.Equal(t, "open", existing.Status)
	require.Nil(t, existing.ResolvedAt)
	next := e
	next.EventID = uuid.NewString()
	next.EpisodeID = uuid.NewString()
	result, err = s.AcceptAnomalyEvent(ctx, next)
	require.NoError(t, err)
	require.NotEqual(t, alert.ID, result.AlertID)
	linked, err := s.ManualCreateFromAlert(ctx, ManualCreateFromAlertInput{AlertID: result.AlertID, TeamID: team, DedupSince: time.Now().Add(-time.Hour)})
	require.NoError(t, err)
	require.False(t, linked.Created)
	require.Equal(t, incident.Incident.ID, linked.Incident.ID)
	changed := next
	changed.EventID = uuid.NewString()
	changed.SeriesID = strings.Repeat("b", 64)
	_, err = s.AcceptAnomalyEvent(ctx, changed)
	require.ErrorContains(t, err, "episode identity")
	absent := e
	absent.EventID = uuid.NewString()
	absent.EpisodeID = uuid.NewString()
	absent.WorkspaceID = uuid.NewString()
	_, err = s.AcceptAnomalyEvent(ctx, absent)
	require.ErrorContains(t, err, "workspace does not exist")
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM anomaly_module_receipts WHERE event_id=$1`, absent.EventID).Scan(&count))
	require.Zero(t, count)
}
func TestAnomalyRecoveryBeforeProcessing(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	e := anomalyFixture()
	e.State = "resolved"
	result, err := s.AcceptAnomalyEvent(ctx, e)
	require.NoError(t, err)
	alert, err := s.GetAlertByID(ctx, result.AlertID)
	require.NoError(t, err)
	require.Equal(t, "resolved", alert.Status)
	var count int
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM jobs`).Scan(&count))
	require.Zero(t, count)
	firing := e
	firing.State = "firing"
	firing.Sequence = 2
	firing.EventID = uuid.NewString()
	result, err = s.AcceptAnomalyEvent(ctx, firing)
	require.NoError(t, err)
	require.True(t, result.Ignored)
	e = anomalyFixture()
	first, err := s.AcceptAnomalyEvent(ctx, e)
	require.NoError(t, err)
	e.State = "resolved"
	e.Sequence = 2
	e.EventID = uuid.NewString()
	_, err = s.AcceptAnomalyEvent(ctx, e)
	require.NoError(t, err)
	_, err = s.ManualCreateFromAlert(ctx, ManualCreateFromAlertInput{AlertID: first.AlertID})
	require.ErrorIs(t, err, ErrAlertNotFiring)
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM incidents`).Scan(&count))
	require.Zero(t, count)
}
func TestAnomalyRecoveryWaitsForAlertLink(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	e := anomalyFixture()
	result, err := s.AcceptAnomalyEvent(ctx, e)
	require.NoError(t, err)
	tx, err := s.pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `SELECT id FROM alerts WHERE id=$1 FOR UPDATE`, result.AlertID)
	require.NoError(t, err)
	team := uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO teams(id,workspace_id,name)VALUES($1,$2,'platform')`, team, anomalyDefaultWorkspace)
	require.NoError(t, err)
	incident, err := createIncidentWithAlertTx(ctx, tx, CreateIncidentInput{AlertID: result.AlertID, TeamID: team, Fingerprint: e.Fingerprint(), Severity: e.Severity, Title: e.Title})
	require.NoError(t, err)
	e.State = "resolved"
	e.EventID = uuid.NewString()
	e.Sequence = 2
	done := make(chan error, 1)
	go func() { _, err := s.AcceptAnomalyEvent(ctx, e); done <- err }()
	require.NoError(t, tx.Commit(ctx))
	require.NoError(t, <-done)
	timeline, err := s.ListTimelineEvents(ctx, incident.ID)
	require.NoError(t, err)
	require.Len(t, timeline, 2)
	require.Equal(t, "anomaly_recovered", timeline[1].Kind)
}
func TestAnomalySettingsImportAndAtomicPatch(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	env := legacyEnvironment()
	for key, value := range map[string]string{"AEGIS_ANOMALY_URL": "http://module", "AEGIS_ANOMALY_ENABLED": "true", "AEGIS_INSTALLATION_ID": "customer-installation", "AEGIS_ANOMALY_IDENTITY_SECRET": strings.Repeat("i", 32), "AEGIS_ANOMALY_SERVICE_TOKEN": strings.Repeat("s", 32)} {
		env[key] = value
	}
	_, err := s.ImportEnvironment(ctx, env, true)
	require.NoError(t, err)
	doc, err := s.GetSettings(ctx)
	require.NoError(t, err)
	runtime, err := doc.Config(&config.Config{})
	require.NoError(t, err)
	require.True(t, runtime.Anomaly.Configured())
	require.True(t, runtime.Anomaly.Enabled)
	actor, err := s.UpsertUser(ctx, "google", "anomaly-admin", "admin@example.test", "Admin", "admin", "en")
	require.NoError(t, err)
	doc, err = s.PatchSettings(ctx, actor.ID, doc.Revision, "anomaly", map[string]string{"AEGIS_ANOMALY_ENABLED": "false", "AEGIS_ANOMALY_IDENTITY_SECRET": "", "AEGIS_ANOMALY_SERVICE_TOKEN": ""}, false)
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("i", 32), doc.Values["AEGIS_ANOMALY_IDENTITY_SECRET"])
	_, err = s.PatchSettings(ctx, actor.ID, doc.Revision, "anomaly", map[string]string{"AEGIS_ANOMALY_URL": "https://secret:private@host", "AEGIS_ANOMALY_ENABLED": "true"}, false)
	require.Error(t, err)
	unchanged, err := s.GetSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, doc.Revision, unchanged.Revision)
	require.Equal(t, "false", unchanged.Values["AEGIS_ANOMALY_ENABLED"])
	raw, err := os.ReadFile("../../db/migrations/000023_anomaly_module.down.sql")
	require.NoError(t, err)
	_, err = s.pool.Exec(ctx, string(raw))
	require.NoError(t, err)
	raw, err = os.ReadFile("../../db/migrations/000023_anomaly_module.up.sql")
	require.NoError(t, err)
	_, err = s.pool.Exec(ctx, string(raw))
	require.NoError(t, err)
}
