package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBackupConfig(t *testing.T) {
	c, err := backupConfig(map[string]string{})
	require.NoError(t, err)
	require.False(t, c.Configured())
	require.Equal(t, DefaultBackupStaleAfter, c.StaleAfter)

	c, err = backupConfig(map[string]string{"BACKUP_CHECKER_URL": "https://checker.internal/", "BACKUP_CHECKER_TOKEN": "0123456789abcdef", "BACKUP_STALE_AFTER": "6h"})
	require.NoError(t, err)
	require.True(t, c.Configured())
	require.Equal(t, "https://checker.internal", c.CheckerURL)
	require.Equal(t, 6*time.Hour, c.StaleAfter)

	c, err = backupConfig(map[string]string{"BACKUP_CHECKER_URL": "http://backup-checker.backup-checker.svc.cluster.local:8080"})
	require.NoError(t, err)
	require.True(t, c.Configured(), "token is optional")

	for name, values := range map[string]map[string]string{
		"scheme":   {"BACKUP_CHECKER_URL": "ftp://checker"},
		"path":     {"BACKUP_CHECKER_URL": "https://checker/v1"},
		"query":    {"BACKUP_CHECKER_URL": "https://checker?x=1"},
		"short":    {"BACKUP_CHECKER_TOKEN": "short"},
		"spaces":   {"BACKUP_CHECKER_TOKEN": "0123456789 abcdef"},
		"duration": {"BACKUP_STALE_AFTER": "soon"},
		"negative": {"BACKUP_STALE_AFTER": "-1h"},
	} {
		_, err := backupConfig(values)
		require.Error(t, err, name)
	}
}

func TestBackupSettingsWiring(t *testing.T) {
	require.True(t, KnownKey("BACKUP_CHECKER_URL"))
	require.True(t, SecretKey("BACKUP_CHECKER_TOKEN"))
	require.False(t, SecretKey("BACKUP_STALE_AFTER"))
	require.Error(t, ValidateValues(map[string]string{"BACKUP_STALE_AFTER": "0s"}))

	s := DefaultSettings()
	s.Values["BACKUP_CHECKER_URL"] = "http://checker:8080"
	s.Values["BACKUP_CHECKER_TOKEN"] = "0123456789abcdef"
	cfg, err := s.Config(&Config{})
	require.NoError(t, err)
	require.Equal(t, "http://checker:8080", cfg.Backups.CheckerURL)
	require.Equal(t, DefaultBackupStaleAfter, cfg.Backups.StaleAfter)
}
