package config

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Settings is the database document. Values never belong in logs or public responses.
type Settings struct {
	Values          map[string]string `json:"values"`
	Enabled         map[string]bool   `json:"enabled"`
	Revision        int64             `json:"revision"`
	Imported        bool              `json:"imported"`
	BootstrapClosed bool              `json:"bootstrap_closed"`
}

var SettingSections = map[string][]string{
	"behavior":      {"SESSION_TTL", "ALERT_FINGERPRINT_LABELS", "INCIDENT_DEDUP_WINDOW", "ESCALATION_TIMEOUT"},
	"access":        {"ADMIN_EMAILS"},
	"deployment":    {"PUBLIC_URL", "HTTP_ADDR", "WEBHOOK_SECRET"},
	"compatibility": {"SESSION_SECRET"},
}

var ConnectorKeys = map[string]map[string]string{
	"jira":    {"JIRA_BASE_URL": "base_url", "JIRA_EMAIL": "email", "JIRA_API_TOKEN": "api_token", "JIRA_PROJECT_KEY": "project_key"},
	"slack":   {"SLACK_BOT_TOKEN": "bot_token", "SLACK_SIGNING_SECRET": "signing_secret"},
	"express": {"EXPRESS_BOT_ID": "bot_id", "EXPRESS_BOT_HOST": "host", "EXPRESS_BOT_SECRET": "secret_key"},
}

var deploymentKeys = map[string]bool{
	"DATABASE_URL": true, "AEGIS_BOOTSTRAP_TOKEN": true, "DEV_AUTH_ENABLED": true,
	"DEV_AUTH_DEFAULT_ROLE": true, "DEV_AUTH_EMAIL": true, "SEED_DEV": true,
	"AEGIS_API_BIN": true, "AEGIS_WORKER_BIN": true, "AEGIS_NGINX_BIN": true,
	"AEGIS_NGINX_CONF": true, "AEGIS_MIGRATIONS_PATH": true,
	"AEGIS_API_URL": true, "AEGIS_WEBHOOK_URL": true, "ALERT_SIM_INTERVAL": true,
	"ALERT_SIM_TEAM": true, "ALERT_SIM_PROJECT": true,
}

func ProviderKeys(provider string) []string {
	prefix := strings.ToUpper(provider) + "_OIDC_"
	keys := []string{prefix + "CLIENT_ID", prefix + "CLIENT_SECRET", prefix + "REDIRECT_URL"}
	if provider == "express" {
		keys = append(keys, prefix+"ISSUER")
	}
	return keys
}

func KnownKey(key string) bool {
	if deploymentKeys[key] {
		return true
	}
	for _, keys := range SettingSections {
		for _, k := range keys {
			if key == k {
				return true
			}
		}
	}
	for _, p := range []string{"google", "slack", "express"} {
		for _, k := range ProviderKeys(p) {
			if key == k {
				return true
			}
		}
	}
	for _, fields := range ConnectorKeys {
		if _, ok := fields[key]; ok {
			return true
		}
	}
	return false
}

func DeploymentKey(key string) bool { return deploymentKeys[key] }
func SecretKey(key string) bool {
	return key == "SESSION_SECRET" || key == "WEBHOOK_SECRET" || key == "AEGIS_BOOTSTRAP_TOKEN" || key == "DATABASE_URL" || strings.HasSuffix(key, "CLIENT_SECRET") || key == "JIRA_API_TOKEN" || key == "SLACK_BOT_TOKEN" || key == "SLACK_SIGNING_SECRET" || key == "EXPRESS_BOT_SECRET"
}

func DefaultSettings() Settings {
	return Settings{Values: map[string]string{
		"PUBLIC_URL": "http://localhost:3000", "HTTP_ADDR": ":8080", "SESSION_TTL": "168h",
		"INCIDENT_DEDUP_WINDOW": "24h", "ESCALATION_TIMEOUT": "15m", "ALERT_FINGERPRINT_LABELS": "alertname,team", "ADMIN_EMAILS": "",
	}, Enabled: map[string]bool{}, Revision: 1}
}

// ParseEnv reads dotenv as data. It never expands variables or executes commands.
func ParseEnv(r io.Reader) (map[string]string, error) {
	values := map[string]string{}
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 4096), 1024*1024)
	lineNumber := 0
	for s.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.TrimPrefix(s.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || !validEnvKey(key) {
			return nil, fmt.Errorf("invalid env key at line %d", lineNumber)
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate key %s", key)
		}
		if len(value) > 0 && (value[0] == '\'' || value[0] == '"') {
			quote := value[0]
			end := -1
			for i := 1; i < len(value); i++ {
				if quote == '"' && value[i] == '\\' {
					i++
					continue
				}
				if value[i] == quote {
					end = i
					break
				}
			}
			if end < 0 || (strings.TrimSpace(value[end+1:]) != "" && !strings.HasPrefix(strings.TrimSpace(value[end+1:]), "#")) {
				return nil, fmt.Errorf("invalid quoted value for %s", key)
			}
			if quote == '"' {
				decoded, err := strconv.Unquote(value[:end+1])
				if err != nil {
					return nil, fmt.Errorf("invalid quoted value for %s", key)
				}
				value = decoded
			} else {
				value = value[1:end]
			}
		} else {
			if i := strings.Index(value, " #"); i >= 0 {
				value = strings.TrimSpace(value[:i])
			}
		}
		values[key] = value
	}
	if s.Err() != nil {
		return nil, fmt.Errorf("cannot read env file")
	}
	return values, nil
}

func validEnvKey(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if c != '_' && !(c >= 'A' && c <= 'Z') && !(c >= 'a' && c <= 'z') && !(i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func ValidateValues(values map[string]string) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := values[key]
		if !KnownKey(key) {
			return fmt.Errorf("unsupported key %s", key)
		}
		switch key {
		case "WEBHOOK_SECRET":
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("invalid setting %s", key)
			}
		case "SESSION_TTL", "INCIDENT_DEDUP_WINDOW", "ESCALATION_TIMEOUT":
			d, err := time.ParseDuration(value)
			if err != nil || d <= 0 {
				return fmt.Errorf("invalid setting %s", key)
			}
		case "PUBLIC_URL":
			if !ValidURL(value) {
				return fmt.Errorf("invalid setting %s", key)
			}
		case "HTTP_ADDR":
			host, port, err := net.SplitHostPort(value)
			_ = host
			n, e := strconv.Atoi(port)
			if err != nil || e != nil || n < 1 || n > 65535 {
				return fmt.Errorf("invalid setting %s", key)
			}
		case "ADMIN_EMAILS":
			if _, err := parseAdminEmails(value); err != nil {
				return fmt.Errorf("invalid setting %s", key)
			}
		case "ALERT_FINGERPRINT_LABELS":
			seen := map[string]bool{}
			for _, label := range strings.Split(value, ",") {
				label = strings.TrimSpace(label)
				if label == "" || strings.ContainsAny(label, " \t\r\n") || seen[label] {
					return fmt.Errorf("invalid setting %s", key)
				}
				seen[label] = true
			}
		default:
			if strings.HasSuffix(key, "_REDIRECT_URL") || key == "EXPRESS_OIDC_ISSUER" || key == "JIRA_BASE_URL" || key == "EXPRESS_BOT_HOST" {
				if value != "" && !ValidURL(value) {
					return fmt.Errorf("invalid setting %s", key)
				}
			}
		}
	}
	return nil
}

func ValidURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

func (s Settings) Config(boot *Config) (*Config, error) {
	if err := ValidateValues(s.Values); err != nil {
		return nil, err
	}
	c := *boot
	c.OIDC = map[string]OIDCProvider{}
	c.PublicURL = strings.TrimRight(s.Values["PUBLIC_URL"], "/")
	c.HTTPAddr = s.Values["HTTP_ADDR"]
	c.SessionSecret = s.Values["SESSION_SECRET"]
	c.WebhookSecret = s.Values["WEBHOOK_SECRET"]
	c.SessionTTL, _ = time.ParseDuration(s.Values["SESSION_TTL"])
	c.IncidentDedupWindow, _ = time.ParseDuration(s.Values["INCIDENT_DEDUP_WINDOW"])
	c.EscalationTimeout, _ = time.ParseDuration(s.Values["ESCALATION_TIMEOUT"])
	c.AlertFingerprintLabels = parseCSV(s.Values["ALERT_FINGERPRINT_LABELS"])
	c.AdminEmails, _ = parseAdminEmails(s.Values["ADMIN_EMAILS"])
	for _, p := range []string{"google", "slack", "express"} {
		if !s.Enabled[p] {
			continue
		}
		prefix := strings.ToUpper(p) + "_OIDC_"
		issuer := s.Values[prefix+"ISSUER"]
		if p == "google" {
			issuer = "https://accounts.google.com"
		}
		if p == "slack" {
			issuer = "https://slack.com"
		}
		c.OIDC[p] = OIDCProvider{ClientID: s.Values[prefix+"CLIENT_ID"], ClientSecret: s.Values[prefix+"CLIENT_SECRET"], RedirectURL: s.Values[prefix+"REDIRECT_URL"], Issuer: issuer}
	}
	if c.DevAuthEnabled {
		if err := validateDevAuthHost(c.PublicURL); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

type runtimeKey struct{}
type settingsKey struct{}

func WithSettings(ctx context.Context, settings Settings, cfg *Config) context.Context {
	return context.WithValue(WithRuntime(ctx, cfg), settingsKey{}, settings)
}

func SettingsSnapshot(ctx context.Context) (Settings, bool) {
	snapshot, ok := ctx.Value(settingsKey{}).(Settings)
	return snapshot, ok
}

func WithRuntime(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, runtimeKey{}, cfg)
}
func Runtime(ctx context.Context, fallback *Config) *Config {
	if c, ok := ctx.Value(runtimeKey{}).(*Config); ok {
		return c
	}
	return fallback
}
func PublicURL(ctx context.Context, fallback string) string {
	if c := Runtime(ctx, nil); c != nil {
		return c.PublicURL
	}
	return fallback
}
func DedupWindow(ctx context.Context, fallback time.Duration) time.Duration {
	if c := Runtime(ctx, nil); c != nil {
		return c.IncidentDedupWindow
	}
	return fallback
}
func EscalationDelay(ctx context.Context, fallback time.Duration) time.Duration {
	if c := Runtime(ctx, nil); c != nil {
		return c.EscalationTimeout
	}
	return fallback
}
