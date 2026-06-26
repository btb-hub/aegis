package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type OIDCProvider struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Issuer       string
}

type Config struct {
	DatabaseURL   string
	SessionSecret string
	WebhookSecret string
	PublicURL     string
	SessionTTL    time.Duration
	HTTPAddr      string
	OIDC          map[string]OIDCProvider
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),
		PublicURL:     strings.TrimRight(os.Getenv("PUBLIC_URL"), "/"),
		HTTPAddr:      envOr("HTTP_ADDR", ":8080"),
		SessionTTL:    parseDuration(envOr("SESSION_TTL", "168h")),
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
	return nil
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

func ParsePort(addr string) (int, error) {
	addr = strings.TrimPrefix(addr, ":")
	port, err := strconv.Atoi(addr)
	if err != nil {
		return 0, err
	}
	return port, nil
}
