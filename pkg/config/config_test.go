package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("WEBHOOK_SECRET", "")
	t.Setenv("PUBLIC_URL", "")

	_, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "DATABASE_URL")
}

func TestLoadSuccess(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "postgres://aegis", cfg.DatabaseURL)
	require.Equal(t, 168*time.Hour, cfg.SessionTTL)
}

func TestProviderValidation(t *testing.T) {
	setRequiredEnv(t)
	cfg, err := Load()
	require.NoError(t, err)

	_, err = cfg.Provider("unknown")
	require.Error(t, err)

	t.Setenv("GOOGLE_OIDC_CLIENT_ID", "id")
	t.Setenv("GOOGLE_OIDC_CLIENT_SECRET", "secret")
	t.Setenv("GOOGLE_OIDC_REDIRECT_URL", "http://localhost/cb")
	cfg, err = Load()
	require.NoError(t, err)

	provider, err := cfg.Provider("google")
	require.NoError(t, err)
	require.Equal(t, "id", provider.ClientID)
}

func TestParsePort(t *testing.T) {
	port, err := ParsePort(":8080")
	require.NoError(t, err)
	require.Equal(t, 8080, port)
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://aegis")
	t.Setenv("SESSION_SECRET", "session-secret")
	t.Setenv("WEBHOOK_SECRET", "webhook-secret")
	t.Setenv("PUBLIC_URL", "http://localhost:8080")
}

func TestParseDurationFallback(t *testing.T) {
	require.Equal(t, 168*time.Hour, parseDuration("not-a-duration"))
}

func TestEnvOr(t *testing.T) {
	key := "AEGIS_TEST_ENV_OR"
	os.Unsetenv(key)
	require.Equal(t, "fallback", envOr(key, "fallback"))
	t.Setenv(key, "value")
	require.Equal(t, "value", envOr(key, "fallback"))
}

func TestSlackSigningSecret(t *testing.T) {
	setRequiredEnv(t)
	cfg, err := Load()
	require.NoError(t, err)
	t.Setenv("SLACK_SIGNING_SECRET", "slack-secret")
	require.Equal(t, "slack-secret", cfg.SlackSigningSecret())
}

func TestDevAuthDisabledByDefault(t *testing.T) {
	setRequiredEnv(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.False(t, cfg.DevAuthEnabled)
	require.Equal(t, "admin", cfg.DevAuthDefaultRole)
	require.Equal(t, "dev@localhost", cfg.DevAuthEmail)
}

func TestDevAuthEnabledOnLocalhost(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DEV_AUTH_ENABLED", "true")
	t.Setenv("PUBLIC_URL", "http://127.0.0.1:3000")
	t.Setenv("DEV_AUTH_DEFAULT_ROLE", "member")
	t.Setenv("DEV_AUTH_EMAIL", "local@test")

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.DevAuthEnabled)
	require.Equal(t, "member", cfg.DevAuthDefaultRole)
	require.Equal(t, "local@test", cfg.DevAuthEmail)
}

func TestDevAuthEnabledOnIPv6Localhost(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DEV_AUTH_ENABLED", "true")
	t.Setenv("PUBLIC_URL", "http://[::1]:3000")

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.DevAuthEnabled)
}

func TestDevAuthRejectsNonLocalhostPublicURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DEV_AUTH_ENABLED", "true")
	t.Setenv("PUBLIC_URL", "https://aegis.example.com")

	_, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "DEV_AUTH_ENABLED requires PUBLIC_URL host")
}

func TestDevAuthRejectsInvalidDefaultRole(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DEV_AUTH_ENABLED", "true")
	t.Setenv("DEV_AUTH_DEFAULT_ROLE", "superuser")

	_, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid DEV_AUTH_DEFAULT_ROLE")
}

func TestParseBoolEnvValues(t *testing.T) {
	key := "AEGIS_TEST_BOOL_ENV"
	t.Setenv(key, "true")
	require.True(t, parseBoolEnv(key))
	t.Setenv(key, "1")
	require.True(t, parseBoolEnv(key))
	t.Setenv(key, "false")
	require.False(t, parseBoolEnv(key))
}
