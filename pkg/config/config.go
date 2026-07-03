package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/rbac"
)

type OIDCProvider struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Issuer       string
}

type Config struct {
	DatabaseURL            string
	SessionSecret          string
	WebhookSecret          string
	PublicURL              string
	SessionTTL             time.Duration
	HTTPAddr               string
	AlertFingerprintLabels []string
	IncidentDedupWindow    time.Duration
	EscalationTimeout      time.Duration
	OIDC                   map[string]OIDCProvider
	DevAuthEnabled         bool
	DevAuthDefaultRole     string
	DevAuthEmail           string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		SessionSecret:          os.Getenv("SESSION_SECRET"),
		WebhookSecret:          os.Getenv("WEBHOOK_SECRET"),
		PublicURL:              strings.TrimRight(os.Getenv("PUBLIC_URL"), "/"),
		HTTPAddr:               envOr("HTTP_ADDR", ":8080"),
		SessionTTL:             parseDuration(envOr("SESSION_TTL", "168h")),
		AlertFingerprintLabels: parseCSV(envOr("ALERT_FINGERPRINT_LABELS", "alertname,team")),
		IncidentDedupWindow:    parseDuration(envOr("INCIDENT_DEDUP_WINDOW", "24h")),
		EscalationTimeout:      parseDuration(envOr("ESCALATION_TIMEOUT", "15m")),
		OIDC: map[string]OIDCProvider{
			"google": {
				ClientID:     os.Getenv("GOOGLE_OIDC_CLIENT_ID"),
				ClientSecret: os.Getenv("GOOGLE_OIDC_CLIENT_SECRET"),
				RedirectURL:  os.Getenv("GOOGLE_OIDC_REDIRECT_URL"),
				Issuer:       "https://accounts.google.com",
			},
			"slack": {
				ClientID:     os.Getenv("SLACK_OIDC_CLIENT_ID"),
				ClientSecret: os.Getenv("SLACK_OIDC_CLIENT_SECRET"),
				RedirectURL:  os.Getenv("SLACK_OIDC_REDIRECT_URL"),
				Issuer:       "https://slack.com",
			},
			"express": {
				ClientID:     os.Getenv("EXPRESS_OIDC_CLIENT_ID"),
				ClientSecret: os.Getenv("EXPRESS_OIDC_CLIENT_SECRET"),
				RedirectURL:  os.Getenv("EXPRESS_OIDC_REDIRECT_URL"),
				Issuer:       os.Getenv("EXPRESS_OIDC_ISSUER"),
			},
		},
		DevAuthEnabled:     parseBoolEnv("DEV_AUTH_ENABLED"),
		DevAuthDefaultRole: envOr("DEV_AUTH_DEFAULT_ROLE", "admin"),
		DevAuthEmail:       envOr("DEV_AUTH_EMAIL", "dev@localhost"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.SessionSecret == "" {
		missing = append(missing, "SESSION_SECRET")
	}
	if c.WebhookSecret == "" {
		missing = append(missing, "WEBHOOK_SECRET")
	}
	if c.PublicURL == "" {
		missing = append(missing, "PUBLIC_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	if c.DevAuthEnabled {
		if err := validateDevAuthHost(c.PublicURL); err != nil {
			return err
		}
		if _, err := rbac.Parse(c.DevAuthDefaultRole); err != nil {
			return fmt.Errorf("invalid DEV_AUTH_DEFAULT_ROLE: %w", err)
		}
	}
	return nil
}

func parseBoolEnv(key string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return value == "1" || value == "true" || value == "yes"
}

func validateDevAuthHost(publicURL string) error {
	parsed, err := url.Parse(publicURL)
	if err != nil {
		return fmt.Errorf("invalid PUBLIC_URL: %w", err)
	}
	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return nil
	default:
		return fmt.Errorf(
			"DEV_AUTH_ENABLED requires PUBLIC_URL host to be localhost, 127.0.0.1, or [::1]; got %q",
			parsed.Hostname(),
		)
	}
}

func (c *Config) Provider(name string) (OIDCProvider, error) {
	provider, ok := c.OIDC[name]
	if !ok {
		return OIDCProvider{}, errors.New("unknown provider")
	}
	if provider.ClientID == "" || provider.ClientSecret == "" || provider.RedirectURL == "" {
		return OIDCProvider{}, fmt.Errorf("provider %s is not configured", name)
	}
	if name == "express" && provider.Issuer == "" {
		return OIDCProvider{}, fmt.Errorf("provider express missing EXPRESS_OIDC_ISSUER")
	}
	return provider, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(raw string) time.Duration {
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 168 * time.Hour
	}
	return d
}

func parseCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func ParsePort(addr string) (int, error) {
	addr = strings.TrimPrefix(addr, ":")
	port, err := strconv.Atoi(addr)
	if err != nil {
		return 0, err
	}
	return port, nil
}

func (c *Config) SlackSigningSecret() string {
	return os.Getenv("SLACK_SIGNING_SECRET")
}
