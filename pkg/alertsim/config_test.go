package alertsim

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfigCustomEnv(t *testing.T) {
	t.Setenv("AEGIS_WEBHOOK_URL", "http://aegis.test/webhook")
	t.Setenv("WEBHOOK_SECRET", "secret")
	t.Setenv("ALERT_SIM_TEAM", "data")
	t.Setenv("ALERT_SIM_PROJECT", "data")
	t.Setenv("ALERT_SIM_INTERVAL", "2m")

	cfg := LoadConfig()
	require.Equal(t, "http://aegis.test/webhook", cfg.WebhookURL)
	require.Equal(t, "data", cfg.Team)
	require.Equal(t, "data", cfg.Project)
	require.Equal(t, 2*time.Minute, cfg.Interval)
}

func TestLoadConfigIgnoresInvalidInterval(t *testing.T) {
	t.Setenv("ALERT_SIM_INTERVAL", "not-a-duration")
	t.Setenv("WEBHOOK_SECRET", "s")

	cfg := LoadConfig()
	require.Equal(t, 30*time.Second, cfg.Interval)
}

func TestBuildPayloadDefaultTeamProject(t *testing.T) {
	scenario, ok := ScenarioByID("dns_failures")
	require.True(t, ok)
	payload := BuildPayload(scenario, LabelDefaults{}, "")
	require.Equal(t, "platform", payload.Labels["team"])
	require.Equal(t, "platform", payload.Labels["project"])
}

func TestClientSendMissingURL(t *testing.T) {
	client := NewClient("", "secret")
	_, err := client.Send(t.Context(), []byte(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "URL")
}
