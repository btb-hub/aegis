package alertsim

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientSendSuccess(t *testing.T) {
	var secret, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret = r.Header.Get("X-Aegis-Webhook-Secret")
		buf, _ := io.ReadAll(r.Body)
		body = string(buf)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":"alert-1","status":"accepted"}`))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "test-secret")
	scenario, ok := ScenarioByID("disk_full")
	require.True(t, ok)

	result, err := client.SendScenario(context.Background(), scenario, LabelDefaults{Team: "platform"}, "x")
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, result.StatusCode)
	require.Equal(t, "test-secret", secret)
	require.Contains(t, body, "DiskSpaceCritical")
}

func TestClientSendRejectsMissingSecret(t *testing.T) {
	client := NewClient("http://example.com", "")
	_, err := client.Send(context.Background(), []byte(`{}`))
	require.Error(t, err)
}

func TestClientSendHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"INVALID_WEBHOOK_SECRET"}`))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "bad")
	_, err := client.Send(context.Background(), []byte(`{"status":"firing","labels":{}}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("PUBLIC_URL", "http://localhost:9090")
	t.Setenv("WEBHOOK_SECRET", "s")
	t.Setenv("ALERT_SIM_TEAM", "")
	t.Setenv("ALERT_SIM_PROJECT", "")

	cfg := LoadConfig()
	require.Equal(t, "http://localhost:9090/api/v1/alerts/webhook", cfg.WebhookURL)
	require.Equal(t, "platform", cfg.Team)
	require.Equal(t, "platform", cfg.Project)
}
