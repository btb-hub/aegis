package simulator

import (
	"os"
	"strings"
	"time"
)

// Config holds simulator settings from environment and defaults.
type Config struct {
	APIBaseURL string
	WebhookURL string
	Secret     string
	Team       string
	Project    string
	Interval   time.Duration
}

// LoadConfig reads simulator settings from environment variables.
func LoadConfig() Config {
	interval := 30 * time.Second
	if raw := strings.TrimSpace(os.Getenv("ALERT_SIM_INTERVAL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			interval = d
		}
	}

	apiBase := strings.TrimRight(strings.TrimSpace(os.Getenv("AEGIS_API_URL")), "/")
	if apiBase == "" {
		apiBase = strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_URL")), "/")
		if apiBase == "" {
			apiBase = "http://localhost:8080"
		}
	}

	webhookURL := strings.TrimSpace(os.Getenv("AEGIS_WEBHOOK_URL"))
	if webhookURL == "" {
		webhookURL = apiBase + "/api/v1/alerts/webhook"
	}

	team := strings.TrimSpace(os.Getenv("ALERT_SIM_TEAM"))
	if team == "" {
		team = "platform"
	}
	project := strings.TrimSpace(os.Getenv("ALERT_SIM_PROJECT"))
	if project == "" {
		project = team
	}

	return Config{
		APIBaseURL: apiBase,
		WebhookURL: webhookURL,
		Secret:     strings.TrimSpace(os.Getenv("WEBHOOK_SECRET")),
		Team:       team,
		Project:    project,
		Interval:   interval,
	}
}

func (c Config) LabelDefaults() LabelDefaults {
	return LabelDefaults{Team: c.Team, Project: c.Project}
}
