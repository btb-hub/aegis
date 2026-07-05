package alertsim

import (
	"os"
	"strings"
	"time"
)

// Config holds simulator settings from environment and defaults.
type Config struct {
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

	url := strings.TrimSpace(os.Getenv("AEGIS_WEBHOOK_URL"))
	if url == "" {
		base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_URL")), "/")
		if base == "" {
			base = "http://localhost:8080"
		}
		url = base + "/api/v1/alerts/webhook"
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
		WebhookURL: url,
		Secret:     strings.TrimSpace(os.Getenv("WEBHOOK_SECRET")),
		Team:       team,
		Project:    project,
		Interval:   interval,
	}
}

func (c Config) LabelDefaults() LabelDefaults {
	return LabelDefaults{Team: c.Team, Project: c.Project}
}
