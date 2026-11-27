package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// DefaultBackupStaleAfter marks a daily check stale after one missed run plus slack.
const DefaultBackupStaleAfter = 26 * time.Hour

// BackupConfig connects Aegis to the backup-checker service (AEG-109). The token is optional:
// the in-cluster checker has no authentication of its own.
type BackupConfig struct {
	CheckerURL   string
	CheckerToken string
	StaleAfter   time.Duration
}

func (c BackupConfig) Configured() bool { return c.CheckerURL != "" }

func backupConfig(values map[string]string) (BackupConfig, error) {
	c := BackupConfig{CheckerURL: strings.TrimRight(values["BACKUP_CHECKER_URL"], "/"), CheckerToken: values["BACKUP_CHECKER_TOKEN"], StaleAfter: DefaultBackupStaleAfter}
	if c.CheckerURL != "" {
		u, err := url.Parse(c.CheckerURL)
		if err != nil || !ValidURL(c.CheckerURL) || (u.Path != "" && u.Path != "/") {
			return c, fmt.Errorf("invalid setting BACKUP_CHECKER_URL")
		}
	}
	if c.CheckerToken != "" && (len(c.CheckerToken) < 16 || strings.ContainsAny(c.CheckerToken, " \t\r\n")) {
		return c, fmt.Errorf("invalid setting BACKUP_CHECKER_TOKEN: at least 16 bytes without spaces required")
	}
	if raw := values["BACKUP_STALE_AFTER"]; raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			return c, fmt.Errorf("invalid setting BACKUP_STALE_AFTER")
		}
		c.StaleAfter = d
	}
	return c, nil
}
