package db

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func settingsStore(t *testing.T) *Store {
	t.Helper()
	connection := os.Getenv("DATABASE_URL")
	if connection == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, connection)
	require.NoError(t, err)
	schema := "settings_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = root.Exec(ctx, `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	cfg, err := pgxpool.ParseConfig(connection)
	require.NoError(t, err)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	files, err := filepath.Glob("../../db/migrations/*.up.sql")
	require.NoError(t, err)
	for _, file := range files {
		raw, e := os.ReadFile(file)
		require.NoError(t, e)
		_, e = pool.Exec(ctx, string(raw))
		require.NoError(t, e, file)
	}
	t.Cleanup(func() {
		pool.Close()
		_, e := root.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
		require.NoError(t, e)
		root.Close()
	})
	return NewStore(pool)
}

func legacyEnvironment() map[string]string {
	return map[string]string{"DATABASE_URL": "postgres://ignored", "PUBLIC_URL": "https://aegis.example", "HTTP_ADDR": ":8080", "SESSION_SECRET": "compat-test", "WEBHOOK_SECRET": "webhook-test", "SESSION_TTL": "48h", "INCIDENT_DEDUP_WINDOW": "12h", "ESCALATION_TIMEOUT": "7m", "ALERT_FINGERPRINT_LABELS": "alertname,team", "ADMIN_EMAILS": "admin@company.com", "GOOGLE_OIDC_CLIENT_ID": "client", "GOOGLE_OIDC_CLIENT_SECRET": "secret-test", "GOOGLE_OIDC_REDIRECT_URL": "https://original.example/custom-callback", "SLACK_BOT_TOKEN": "bot-test", "SLACK_SIGNING_SECRET": "signing-test", "EXPRESS_BOT_ID": "bot", "EXPRESS_BOT_HOST": "https://cts.example", "EXPRESS_BOT_SECRET": "express-test", "JIRA_BASE_URL": "https://jira.example", "JIRA_EMAIL": "admin@company.com", "JIRA_API_TOKEN": "jira-test", "JIRA_PROJECT_KEY": "OPS"}
}

func TestSettingsImportPreservesAndIsAtomic(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	env := legacyEnvironment()
	_, err := s.UpsertIntegration(ctx, "slack", "Saved bot", json.RawMessage(`{"bot_token":"saved-test"}`), false, nil, nil)
	require.NoError(t, err)
	report, err := s.ImportEnvironment(ctx, env, false)
	require.NoError(t, err)
	require.NotEmpty(t, report)
	_, err = s.GetSettings(ctx)
	require.Error(t, err)
	slack, err := s.GetIntegrationByKind(ctx, "slack")
	require.NoError(t, err)
	require.NotContains(t, string(slack.Config), "signing-test")
	env["UNSUPPORTED_SECRET"] = "never-output-this"
	report, err = s.ImportEnvironment(ctx, env, true)
	require.Error(t, err)
	raw, _ := json.Marshal(report)
	require.NotContains(t, string(raw), "never-output-this")
	_, err = s.GetSettings(ctx)
	require.Error(t, err)
	delete(env, "UNSUPPORTED_SECRET")
	env["SESSION_TTL"] = "invalid"
	_, err = s.ImportEnvironment(ctx, env, true)
	require.ErrorContains(t, err, "SESSION_TTL")
	_, err = s.GetSettings(ctx)
	require.Error(t, err)
	env["SESSION_TTL"] = "48h"
	_, err = s.ImportEnvironment(ctx, env, true)
	require.NoError(t, err)
	doc, err := s.GetSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, "https://original.example/custom-callback", doc.Values["GOOGLE_OIDC_REDIRECT_URL"])
	require.True(t, doc.Imported)
	require.True(t, doc.BootstrapClosed)
	slack, err = s.GetIntegrationByKind(ctx, "slack")
	require.NoError(t, err)
	require.Contains(t, string(slack.Config), "saved-test")
	require.Contains(t, string(slack.Config), "signing-test")
	require.False(t, slack.Enabled)
	var count int
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action='settings.imported'`).Scan(&count))
	require.Equal(t, 1, count)
	env["GOOGLE_OIDC_CLIENT_SECRET"] = "replacement-test"
	_, err = s.ImportEnvironment(ctx, env, true)
	require.NoError(t, err)
	next, err := s.GetSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, doc.Revision, next.Revision)
	require.Equal(t, "secret-test", next.Values["GOOGLE_OIDC_CLIENT_SECRET"])
	require.NoError(t, s.InitializeSettings(ctx, &config.Config{}))
	loaded, err := s.SettingsConfig(ctx, &config.Config{})
	require.NoError(t, err)
	require.Equal(t, 48*time.Hour, loaded.SessionTTL)
}

func TestSettingsConcurrentImportsAndPartialConnectors(t *testing.T) {
	s := settingsStore(t)
	env := legacyEnvironment()
	delete(env, "EXPRESS_BOT_SECRET")
	env["EXPRESS_OIDC_CLIENT_ID"] = "partial-client"
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.ImportEnvironment(context.Background(), env, true); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	doc, err := s.GetSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), doc.Revision)
	require.False(t, doc.Enabled["express"])
	bot, err := s.GetIntegrationByKind(context.Background(), "express")
	require.NoError(t, err)
	require.False(t, bot.Enabled)
}

func TestSettingsCompleteImportRollsBackOnAuditFailure(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	env := legacyEnvironment()
	for _, provider := range []string{"slack", "express"} {
		prefix := strings.ToUpper(provider) + "_OIDC_"
		env[prefix+"CLIENT_ID"] = provider + "-client-fixture"
		env[prefix+"CLIENT_SECRET"] = provider + "-secret-fixture"
		env[prefix+"REDIRECT_URL"] = "https://callbacks.example/" + provider
	}
	env["EXPRESS_OIDC_ISSUER"] = "https://sso.example"
	_, err := s.pool.Exec(ctx, `ALTER TABLE audit_log ADD CONSTRAINT reject_import_audit CHECK(action<>'settings.imported')`)
	require.NoError(t, err)
	_, err = s.ImportEnvironment(ctx, env, true)
	require.ErrorContains(t, err, "import audit")
	_, err = s.GetSettings(ctx)
	require.Error(t, err)
	var count int
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM integrations WHERE workspace_id IS NULL`).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log`).Scan(&count))
	require.Zero(t, count)
	_, err = s.pool.Exec(ctx, `ALTER TABLE audit_log DROP CONSTRAINT reject_import_audit`)
	require.NoError(t, err)
	_, err = s.ImportEnvironment(ctx, env, true)
	require.NoError(t, err)
	doc, err := s.GetSettings(ctx)
	require.NoError(t, err)
	for _, provider := range []string{"google", "slack", "express"} {
		require.True(t, doc.Enabled[provider])
		for _, key := range config.ProviderKeys(provider) {
			require.Equal(t, env[key], doc.Values[key], key)
		}
	}
	runtime, err := s.SettingsConfig(ctx, &config.Config{})
	require.NoError(t, err)
	require.Equal(t, []string{"google", "slack", "express"}, runtime.ConfiguredProviders())
	for _, kind := range []string{"jira", "slack", "express"} {
		connector, err := s.GetIntegrationByKind(ctx, kind)
		require.NoError(t, err)
		require.True(t, connector.Enabled)
	}
}

func TestSettingsUpgradeRequiresImportAndPatches(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	user, err := s.UpsertUser(ctx, "google", "subject", "admin@company.com", "Admin", "admin", "en")
	require.NoError(t, err)
	require.ErrorContains(t, s.InitializeSettings(ctx, &config.Config{}), "import required")
	_, err = s.ImportEnvironment(ctx, legacyEnvironment(), true)
	require.NoError(t, err)
	doc, _ := s.GetSettings(ctx)
	_, err = s.PatchSettings(ctx, user.ID, doc.Revision+1, "behavior", map[string]string{"SESSION_TTL": "2h"}, false)
	require.Error(t, err)
	_, err = s.PatchSettings(ctx, user.ID, doc.Revision, "behavior", map[string]string{"DATABASE_URL": "private"}, false)
	require.Error(t, err)
	_, err = s.PatchSettings(ctx, user.ID, doc.Revision, "behavior", map[string]string{"SESSION_TTL": "invalid"}, false)
	require.Error(t, err)
	next, err := s.PatchSettings(ctx, user.ID, doc.Revision, "deployment", map[string]string{"WEBHOOK_SECRET": ""}, false)
	require.NoError(t, err)
	require.Equal(t, "webhook-test", next.Values["WEBHOOK_SECRET"])
	require.Equal(t, doc.Revision, next.Revision)
	_, err = s.PatchSettings(ctx, user.ID, doc.Revision, "deployment", map[string]string{"PUBLIC_URL": "https://new.example"}, false)
	require.Error(t, err)
	next, err = s.PatchSettings(ctx, user.ID, doc.Revision, "deployment", map[string]string{"PUBLIC_URL": "https://new.example"}, true)
	require.NoError(t, err)
	require.Equal(t, doc.Revision+1, next.Revision)
	next, err = s.PatchSettings(ctx, user.ID, next.Revision, "behavior", map[string]string{"SESSION_TTL": "2h", "ESCALATION_TIMEOUT": "2m"}, false)
	require.NoError(t, err)
	runtime, err := next.Config(&config.Config{})
	require.NoError(t, err)
	require.Equal(t, 2*time.Hour, runtime.SessionTTL)
	available, err := s.BootstrapAvailable(ctx)
	require.NoError(t, err)
	require.False(t, available)
}

func settingsDraftValues() map[string]string {
	return map[string]string{"GOOGLE_OIDC_CLIENT_ID": "client", "GOOGLE_OIDC_CLIENT_SECRET": "draft-test", "GOOGLE_OIDC_REDIRECT_URL": "https://aegis.example/auth/google/callback"}
}

func TestSettingsDevLoginURLSafeguardBeforeSaving(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	boot := &config.Config{DevAuthEnabled: true}
	require.NoError(t, s.InitializeSettings(ctx, boot))
	admin, err := s.UpsertUser(ctx, "dev", "local-admin", "dev@localhost", "Admin", "admin", "en")
	require.NoError(t, err)
	snapshot, err := s.SettingsContext(ctx, boot)
	require.NoError(t, err)
	doc, err := s.GetSettings(ctx)
	require.NoError(t, err)
	_, err = s.PatchSettings(snapshot, admin.ID, doc.Revision, "deployment", map[string]string{"PUBLIC_URL": "https://production.example"}, true)
	require.ErrorContains(t, err, "PUBLIC_URL")
	unchanged, err := s.GetSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, doc, unchanged)
	_, err = s.PatchSettings(snapshot, admin.ID, doc.Revision, "deployment", map[string]string{"PUBLIC_URL": "http://localhost:3100"}, true)
	require.NoError(t, err)
	_, err = s.SettingsContext(ctx, boot)
	require.NoError(t, err)
}

func TestSettingsProviderDraftActivationAndAttempts(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	_, err := s.ImportEnvironment(ctx, legacyEnvironment(), true)
	require.NoError(t, err)
	user, err := s.UpsertUser(ctx, "google", "subject", "admin@company.com", "Admin", "admin", "en")
	require.NoError(t, err)
	_, hash, _ := sessiontoken.New()
	_, err = s.CreateSession(ctx, user.ID, hash, time.Now().Add(time.Hour))
	require.NoError(t, err)
	draft, err := s.SaveProviderDraft(ctx, &user.ID, "google", 0, settingsDraftValues(), user.Email)
	require.NoError(t, err)
	doc, _ := s.GetSettings(ctx)
	_, err = s.ActivateProvider(ctx, user.ID, hash, "google", doc.Revision, draft.Revision, true)
	require.Error(t, err)
	a := SettingsAuthorization{StateHash: "test-state", Kind: "provider_test", Provider: "google", Nonce: "nonce", SessionHash: hash, ExpectedEmail: user.Email, DraftRevision: draft.Revision, SettingsRevision: doc.Revision, ExpiresAt: time.Now().Add(time.Minute)}
	require.NoError(t, s.CreateSettingsAuthorization(ctx, a))
	_, err = s.ClaimSettingsAuthorization(ctx, a.StateHash, "wrong", "google")
	require.Error(t, err)
	claimed, err := s.ClaimSettingsAuthorization(ctx, a.StateHash, hash, "google")
	require.NoError(t, err)
	_, err = s.ClaimSettingsAuthorization(ctx, a.StateHash, hash, "google")
	require.Error(t, err)
	require.NoError(t, s.FinishProviderTest(ctx, claimed))
	doc, err = s.ActivateProvider(ctx, user.ID, hash, "google", doc.Revision, draft.Revision, true)
	require.NoError(t, err)
	require.Equal(t, "draft-test", doc.Values["GOOGLE_OIDC_CLIENT_SECRET"])
	_, err = s.ActivateProvider(ctx, user.ID, hash, "google", doc.Revision, 0, false)
	require.ErrorContains(t, err, "last active")
	draft, err = s.SaveProviderDraft(ctx, &user.ID, "google", draft.Revision, map[string]string{"GOOGLE_OIDC_CLIENT_SECRET": ""}, user.Email)
	require.NoError(t, err)
	require.Equal(t, "draft-test", draft.Values["GOOGLE_OIDC_CLIENT_SECRET"])
	require.Nil(t, draft.TestedRevision)
	a.StateHash = "login-state"
	a.Kind = "login"
	a.SettingsRevision = doc.Revision
	a.DraftRevision = 0
	require.NoError(t, s.CreateSettingsAuthorization(ctx, a))
	_, err = s.ClaimSettingsAuthorization(ctx, a.StateHash, hash, "google")
	require.NoError(t, err)
	require.NoError(t, s.ValidateLoginAuthorization(ctx, a))
	doc, err = s.PatchSettings(ctx, user.ID, doc.Revision, "deployment", map[string]string{"PUBLIC_URL": "https://changed.example"}, true)
	require.NoError(t, err)
	require.Error(t, s.ValidateLoginAuthorization(ctx, a))
	require.NoError(t, s.DeleteSettingsAuthorization(ctx, a.StateHash))
}

func TestSettingsBootstrapAtomicAndPermanent(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	require.NoError(t, s.InitializeSettings(ctx, &config.Config{}))
	available, err := s.BootstrapAvailable(ctx)
	require.NoError(t, err)
	require.True(t, available)
	_, hash, _ := sessiontoken.New()
	require.NoError(t, s.CreateBootstrapSession(ctx, hash, "https://aegis.example"))
	valid, err := s.ValidBootstrapSession(ctx, hash)
	require.NoError(t, err)
	require.True(t, valid)
	require.Error(t, s.CreateBootstrapSession(ctx, "second-browser", "https://aegis.example"))
	draft, err := s.SaveProviderDraft(ctx, nil, "google", 0, settingsDraftValues(), "admin@company.com")
	require.NoError(t, err)
	doc, _ := s.GetSettings(ctx)
	a := SettingsAuthorization{StateHash: "bootstrap-state", Kind: "bootstrap", Provider: "google", Nonce: "nonce", SessionHash: hash, ExpectedEmail: "admin@company.com", DraftRevision: draft.Revision, SettingsRevision: doc.Revision, ExpiresAt: time.Now().Add(time.Minute)}
	require.NoError(t, s.CreateSettingsAuthorization(ctx, a))
	a, err = s.ClaimSettingsAuthorization(ctx, a.StateHash, hash, "google")
	require.NoError(t, err)
	info := OIDCLoginInput{Provider: "google", ProviderSub: "subject", Email: "wrong@company.com"}
	_, err = s.CompleteBootstrap(ctx, a, info, "final-session")
	require.Error(t, err)
	var count int
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count))
	require.Zero(t, count)
	info.Email = "admin@company.com"
	info.DisplayName = "Admin"
	user, err := s.CompleteBootstrap(ctx, a, info, "final-session")
	require.NoError(t, err)
	require.Equal(t, "admin", user.Role)
	_, err = s.GetSessionByTokenHash(ctx, "final-session")
	require.NoError(t, err)
	available, err = s.BootstrapAvailable(ctx)
	require.NoError(t, err)
	require.False(t, available)
	valid, err = s.ValidBootstrapSession(ctx, hash)
	require.NoError(t, err)
	require.False(t, valid)
	require.Error(t, s.CreateBootstrapSession(ctx, "new-session", "https://aegis.example"))
	_, err = s.CompleteBootstrap(ctx, a, info, "second-session")
	require.Error(t, err)
	require.NoError(t, s.InitializeSettings(ctx, &config.Config{}))
}

func TestSettingsCallbacksExpiryAndRestartPersistence(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	env := legacyEnvironment()
	env["GOOGLE_OIDC_REDIRECT_URL"] = "https://aegis.example/auth/google/callback"
	_, err := s.ImportEnvironment(ctx, env, true)
	require.NoError(t, err)
	user, err := s.UpsertUser(ctx, "google", "subject", "admin@company.com", "Admin", "admin", "en")
	require.NoError(t, err)
	expiry := time.Now().Add(48 * time.Hour).Truncate(time.Microsecond)
	_, err = s.CreateSession(ctx, user.ID, "existing-session", expiry)
	require.NoError(t, err)
	alert, err := s.CreateAlertAndJob(ctx, CreateAlertJobInput{Fingerprint: "original-fingerprint", Status: "firing", Severity: "warning", Title: "Fixture", Labels: map[string]string{"team": "platform"}, RawPayload: json.RawMessage(`{}`), JobKind: "escalate_incident"})
	require.NoError(t, err)
	runAt := time.Now().Add(3 * time.Hour).Truncate(time.Microsecond)
	_, err = s.pool.Exec(ctx, `UPDATE jobs SET run_at=$2 WHERE id=$1`, alert.JobID, runAt)
	require.NoError(t, err)
	snapshotCtx, err := s.SettingsContext(ctx, &config.Config{})
	require.NoError(t, err)
	_, err = s.SaveProviderDraft(ctx, &user.ID, "google", 0, settingsDraftValues(), user.Email)
	require.NoError(t, err)
	doc, _ := s.GetSettings(ctx)
	a := SettingsAuthorization{StateHash: "expired-attempt", Kind: "provider_test", Provider: "google", SessionHash: "existing-session", SettingsRevision: doc.Revision, ExpiresAt: time.Now().Add(-time.Second)}
	require.NoError(t, s.CreateSettingsAuthorization(ctx, a))
	_, err = s.ClaimSettingsAuthorization(ctx, a.StateHash, a.SessionHash, "google")
	require.Error(t, err)
	a.StateHash = "stale-attempt"
	a.SettingsRevision++
	require.Error(t, s.CreateSettingsAuthorization(ctx, a))
	_, err = s.PatchSettings(ctx, user.ID, doc.Revision, "deployment", map[string]string{"PUBLIC_URL": "https://new.example", "HTTP_ADDR": ":8081"}, true)
	require.NoError(t, err)
	doc, _ = s.GetSettings(ctx)
	require.Equal(t, "https://new.example/auth/google/callback", doc.Values["GOOGLE_OIDC_REDIRECT_URL"])
	draft, err := s.GetProviderDraft(ctx, "google")
	require.NoError(t, err)
	require.Equal(t, "https://new.example/auth/google/callback", draft.Values["GOOGLE_OIDC_REDIRECT_URL"])
	require.Equal(t, int64(2), draft.Revision)
	_, err = s.PatchSettings(ctx, user.ID, doc.Revision, "behavior", map[string]string{"SESSION_TTL": "1h"}, false)
	require.NoError(t, err)
	oldSnapshot, err := s.GetSettings(snapshotCtx)
	require.NoError(t, err)
	require.Equal(t, "https://aegis.example", oldSnapshot.Values["PUBLIC_URL"])
	savedAlert, err := s.GetAlertByID(ctx, alert.AlertID)
	require.NoError(t, err)
	require.Equal(t, "original-fingerprint", savedAlert.Fingerprint)
	var storedRunAt time.Time
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT run_at FROM jobs WHERE id=$1`, alert.JobID).Scan(&storedRunAt))
	require.Equal(t, runAt, storedRunAt)
	session, err := s.GetSessionByTokenHash(ctx, "existing-session")
	require.NoError(t, err)
	require.Equal(t, expiry, session.ExpiresAt)
	t.Setenv("PUBLIC_URL", "https://stale.example")
	t.Setenv("GOOGLE_OIDC_CLIENT_SECRET", "stale-secret")
	require.NoError(t, s.InitializeSettings(ctx, &config.Config{}))
	runtime, err := s.SettingsConfig(ctx, &config.Config{})
	require.NoError(t, err)
	require.Equal(t, "https://new.example", runtime.PublicURL)
	require.Equal(t, ":8081", runtime.HTTPAddr)
	require.Equal(t, "secret-test", runtime.OIDC["google"].ClientSecret)
}

func TestSettingsImportEmptyAndInvalidConnectors(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	env := legacyEnvironment()
	env["EXPRESS_BOT_HOST"] = "not-a-url"
	_, err := s.ImportEnvironment(ctx, env, true)
	require.ErrorContains(t, err, "EXPRESS_BOT_HOST")
	_, err = s.GetSettings(ctx)
	require.Error(t, err)
	env = legacyEnvironment()
	env["EXPRESS_BOT_ID"] = ""
	env["EXPRESS_BOT_HOST"] = ""
	env["EXPRESS_BOT_SECRET"] = ""
	delete(env, "JIRA_EMAIL")
	_, err = s.ImportEnvironment(ctx, env, true)
	require.NoError(t, err)
	bot, err := s.GetIntegrationByKind(ctx, "express")
	require.NoError(t, err)
	require.False(t, bot.Enabled)
	jira, err := s.GetIntegrationByKind(ctx, "jira")
	require.NoError(t, err)
	require.True(t, jira.Enabled)
	_, err = s.ImportEnvironment(ctx, env, true)
	require.NoError(t, err)
	versionCtx := WithIntegrationVersion(ctx, jira.UpdatedAt)
	_, err = s.UpdateIntegration(versionCtx, jira.ID, "Updated Jira", jira.Config, jira.Enabled, jira.Mode)
	require.NoError(t, err)
	_, err = s.UpdateIntegration(versionCtx, jira.ID, "Stale Jira", jira.Config, jira.Enabled, jira.Mode)
	require.ErrorContains(t, err, "reload before saving")
	var audit string
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT details::text FROM audit_log WHERE action='settings.integration_changed' AND resource_id=$1 ORDER BY created_at DESC LIMIT 1`, jira.ID).Scan(&audit))
	require.Contains(t, audit, "name")
	require.NotContains(t, audit, "jira-test")
	var count int
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action='settings.imported'`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestSettingsInstallationExpiresAndMigrationReverses(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	require.NoError(t, s.InitializeSettings(ctx, &config.Config{}))
	require.NoError(t, s.CreateBootstrapSession(ctx, "expired-session", "https://aegis.example"))
	_, err := s.pool.Exec(ctx, `UPDATE settings_bootstrap_sessions SET expires_at=now()-interval '1 second'`)
	require.NoError(t, err)
	valid, err := s.ValidBootstrapSession(ctx, "expired-session")
	require.NoError(t, err)
	require.False(t, valid)
	require.NoError(t, s.CreateBootstrapSession(ctx, "new-session", "https://aegis.example"))
	_, err = s.ImportEnvironment(ctx, legacyEnvironment(), true)
	require.NoError(t, err)
	_, err = s.SaveProviderDraft(ctx, nil, "google", 0, settingsDraftValues(), "admin@company.com")
	require.Error(t, err)
	down, err := os.ReadFile("../../db/migrations/000022_application_settings.down.sql")
	require.NoError(t, err)
	_, err = s.pool.Exec(ctx, string(down))
	require.NoError(t, err)
	_, err = s.GetIntegrationByKind(ctx, "express")
	require.NoError(t, err)
	up, err := os.ReadFile("../../db/migrations/000022_application_settings.up.sql")
	require.NoError(t, err)
	_, err = s.pool.Exec(ctx, string(up))
	require.NoError(t, err)
	require.ErrorContains(t, s.InitializeSettings(ctx, &config.Config{}), "import required")
}
