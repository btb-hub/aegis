package config

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errors.New("private read error") }

func TestSettingsEnvParser(t *testing.T) {
	values, err := ParseEnv(strings.NewReader("\ufeff# comment\nexport A=one # comment\nB='$(must-not-run) # x'\nC=\"a\\nb\" # note\nEMPTY=\nD=hash#inside\n"))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"A": "one", "B": "$(must-not-run) # x", "C": "a\nb", "EMPTY": "", "D": "hash#inside"}, values)
	for _, raw := range []string{"no equals", "1A=x", "A-B=x", "A=x\nA=y", "A='unterminated", "A=\"bad\\x\"", "A='ok' trailing"} {
		_, err = ParseEnv(strings.NewReader(raw))
		require.Error(t, err)
		require.NotContains(t, err.Error(), "trailing")
	}
	_, err = ParseEnv(brokenReader{})
	require.EqualError(t, err, "cannot read env file")
}

func TestSettingsRegistryAndValidation(t *testing.T) {
	file, err := os.Open("../../deploy/.env.legacy.example")
	if os.IsNotExist(err) {
		file, err = os.Open("../../deploy/.env.example")
	}
	require.NoError(t, err)
	defer file.Close()
	values, err := ParseEnv(file)
	require.NoError(t, err)
	for key := range values {
		require.True(t, KnownKey(key), key)
	}
	require.False(t, KnownKey("UNKNOWN"))
	require.True(t, DeploymentKey("DATABASE_URL"))
	require.False(t, DeploymentKey("PUBLIC_URL"))
	for _, key := range []string{"SESSION_SECRET", "WEBHOOK_SECRET", "GOOGLE_OIDC_CLIENT_SECRET", "JIRA_API_TOKEN", "SLACK_SIGNING_SECRET", "EXPRESS_BOT_SECRET", "DATABASE_URL", "AEGIS_BOOTSTRAP_TOKEN", "SLACK_BOT_TOKEN"} {
		require.True(t, SecretKey(key), key)
	}
	require.False(t, SecretKey("GOOGLE_OIDC_CLIENT_ID"))
	require.Len(t, ProviderKeys("express"), 4)
	for _, values := range []map[string]string{
		{"SESSION_TTL": "0"}, {"SESSION_TTL": "wrong"}, {"INCIDENT_DEDUP_WINDOW": "-1h"}, {"ESCALATION_TIMEOUT": "0s"},
		{"PUBLIC_URL": "https://user:pass@host"}, {"HTTP_ADDR": ":zero"}, {"HTTP_ADDR": ":0"}, {"HTTP_ADDR": ":65536"}, {"HTTP_ADDR": "not-address"},
		{"ADMIN_EMAILS": "bad"}, {"ALERT_FINGERPRINT_LABELS": ""}, {"ALERT_FINGERPRINT_LABELS": "a,a"}, {"ALERT_FINGERPRINT_LABELS": "a,,b"}, {"ALERT_FINGERPRINT_LABELS": "a b"},
		{"UNKNOWN": "private"}, {"GOOGLE_OIDC_REDIRECT_URL": "garbage"}, {"EXPRESS_OIDC_ISSUER": "garbage"},
	} {
		err := ValidateValues(values)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "private")
	}
	require.NoError(t, ValidateValues(map[string]string{"HTTP_ADDR": "127.0.0.1:8080", "PUBLIC_URL": "https://aegis.example", "ADMIN_EMAILS": "A@company.com, B@company.com", "ALERT_FINGERPRINT_LABELS": "a, b", "SESSION_TTL": "1h", "GOOGLE_OIDC_REDIRECT_URL": ""}))
	for _, raw := range []string{"https://a.example", "http://localhost:3000"} {
		require.True(t, ValidURL(raw))
	}
	for _, raw := range []string{"ftp://a", "https://a?secret=x", "https://a#x", "https://u:p@a", "/path", "https://%"} {
		require.False(t, ValidURL(raw))
	}
}

func TestSettingsConfigAndRuntime(t *testing.T) {
	s := DefaultSettings()
	s.Values["WEBHOOK_SECRET"] = "webhook"
	s.Values["SESSION_SECRET"] = "compatibility"
	s.Values["ADMIN_EMAILS"] = " ADMIN@company.com "
	for _, p := range []string{"google", "slack", "express"} {
		for _, key := range ProviderKeys(p) {
			s.Values[key] = "value"
		}
		s.Values[strings.ToUpper(p)+"_OIDC_REDIRECT_URL"] = "https://aegis.example/auth/" + p + "/callback"
		s.Enabled[p] = true
	}
	s.Values["EXPRESS_OIDC_ISSUER"] = "https://sso.example"
	boot := &Config{DatabaseURL: "postgres://test", DevAuthDefaultRole: "admin"}
	cfg, err := s.Config(boot)
	require.NoError(t, err)
	require.Equal(t, "postgres://test", cfg.DatabaseURL)
	require.True(t, cfg.IsAdminEmail("admin@company.com"))
	require.Len(t, cfg.ConfiguredProviders(), 3)
	p, _ := cfg.Provider("google")
	require.Equal(t, "https://accounts.google.com", p.Issuer)
	p, _ = cfg.Provider("slack")
	require.Equal(t, "https://slack.com", p.Issuer)
	s.Enabled["google"] = false
	cfg, err = s.Config(boot)
	require.NoError(t, err)
	require.Len(t, cfg.ConfiguredProviders(), 2)
	ctx := WithRuntime(context.Background(), cfg)
	require.Same(t, cfg, Runtime(ctx, boot))
	require.Same(t, boot, Runtime(context.Background(), boot))
	require.Equal(t, cfg.PublicURL, PublicURL(ctx, "fallback"))
	require.Equal(t, "fallback", PublicURL(context.Background(), "fallback"))
	require.Equal(t, 24*time.Hour, DedupWindow(ctx, time.Hour))
	require.Equal(t, time.Hour, DedupWindow(context.Background(), time.Hour))
	require.Equal(t, 15*time.Minute, EscalationDelay(ctx, time.Hour))
	require.Equal(t, time.Hour, EscalationDelay(context.Background(), time.Hour))
	s.Values["SESSION_TTL"] = "invalid"
	_, err = s.Config(boot)
	require.Error(t, err)
	s.Values["SESSION_TTL"] = "1h"
	s.Values["PUBLIC_URL"] = "https://production.example"
	boot.DevAuthEnabled = true
	_, err = s.Config(boot)
	require.Error(t, err)
}

func TestLoadBootstrapOnly(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := LoadBootstrap()
	require.ErrorContains(t, err, "DATABASE_URL")
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("AEGIS_BOOTSTRAP_TOKEN", "short")
	_, err = LoadBootstrap()
	require.ErrorContains(t, err, "32")
	t.Setenv("AEGIS_BOOTSTRAP_TOKEN", strings.Repeat("x", 32))
	t.Setenv("DEV_AUTH_DEFAULT_ROLE", "wrong")
	_, err = LoadBootstrap()
	require.Error(t, err)
	t.Setenv("DEV_AUTH_DEFAULT_ROLE", "admin")
	t.Setenv("WEBHOOK_SECRET", "must-not-load")
	t.Setenv("PUBLIC_URL", "must-not-load")
	cfg, err := LoadBootstrap()
	require.NoError(t, err)
	require.Empty(t, cfg.PublicURL)
	require.Empty(t, cfg.WebhookSecret)
}

func TestSettingsSnapshot(t *testing.T) {
	_, ok := SettingsSnapshot(context.Background())
	require.False(t, ok)
	doc := DefaultSettings()
	doc.Values["WEBHOOK_SECRET"] = "fixture-secret"
	cfg, err := doc.Config(&Config{})
	require.NoError(t, err)
	ctx := WithSettings(context.Background(), doc, cfg)
	snapshot, ok := SettingsSnapshot(ctx)
	require.True(t, ok)
	require.Equal(t, doc.Revision, snapshot.Revision)
	require.Equal(t, cfg, Runtime(ctx, nil))
	require.ErrorContains(t, ValidateValues(map[string]string{"WEBHOOK_SECRET": ""}), "WEBHOOK_SECRET")
	require.ErrorContains(t, ValidateValues(map[string]string{"JIRA_BASE_URL": "bad"}), "JIRA_BASE_URL")
}
