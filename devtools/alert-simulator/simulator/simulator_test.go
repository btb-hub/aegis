package simulator

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	result, err := client.SendScenario(context.Background(), scenario, LabelDefaults{}, "x")
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, result.StatusCode)
	require.Equal(t, "test-secret", secret)
	require.Contains(t, body, "DiskSpaceCritical")
	require.Contains(t, body, `"team":"platform"`)
}

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("PUBLIC_URL", "http://localhost:9090")
	t.Setenv("WEBHOOK_SECRET", "s")
	t.Setenv("ALERT_SIM_TEAM", "")
	t.Setenv("ALERT_SIM_PROJECT", "")

	cfg := LoadConfig()
	require.Equal(t, "http://localhost:9090", cfg.APIBaseURL)
	require.Equal(t, "http://localhost:9090/api/v1/alerts/webhook", cfg.WebhookURL)
	require.Empty(t, cfg.Team)
}

func TestCatalogHasScenarios(t *testing.T) {
	items := Catalog()
	require.GreaterOrEqual(t, len(items), 10)
}

func TestCatalogCoversAllTierLabels(t *testing.T) {
	labels := map[string]int{}
	for _, s := range Catalog() {
		require.NotEmpty(t, s.RoutingTeam)
		labels[s.RoutingTeam]++
	}
	for _, demo := range TierDemos() {
		require.Greater(t, labels[demo.Label], 0, "expected scenarios for team=%s", demo.Label)
	}
}

func TestEffectiveRoutingTeamOverride(t *testing.T) {
	scenario, ok := ScenarioByID("cert_expiry")
	require.True(t, ok)
	require.Equal(t, "noc", EffectiveRoutingTeam(scenario, LabelDefaults{}))
	require.Equal(t, "platform", EffectiveRoutingTeam(scenario, LabelDefaults{Team: "platform"}))
}

func TestDevLoginDoesNotFollowRedirect(t *testing.T) {
	var redirectURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/dev/login" {
			http.SetCookie(w, &http.Cookie{Name: "aegis_session", Value: "token", Path: "/"})
			redirectURL = "http://localhost:3000/"
			w.Header().Set("Location", redirectURL)
			w.WriteHeader(http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	api, err := NewAegisAPI(srv.URL)
	require.NoError(t, err)
	require.NoError(t, api.DevLogin(context.Background()))
}

func TestEnsureDemoRoutingViaAPI(t *testing.T) {
	var teamsCreated, rulesCreated, pathsCreated int
	teamIDs := map[string]string{
		"NOC":      "team-noc",
		"Helpdesk": "team-l1",
		"Ops":      "team-ops",
		"Platform": "team-platform",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/auth/dev/login":
			http.SetCookie(w, &http.Cookie{Name: "aegis_session", Value: "token", Path: "/"})
			w.WriteHeader(http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/routing-rules":
			_, _ = w.Write([]byte(`{"items":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/teams":
			_, _ = w.Write([]byte(`{"items":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/"+defaultWorkspaceID+"/escalation-paths":
			_, _ = w.Write([]byte(`{"items":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/teams":
			teamsCreated++
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			id := teamIDs[body.Name]
			_, _ = w.Write([]byte(`{"id":"` + id + `","name":"` + body.Name + `"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/routing-rules":
			rulesCreated++
			_, _ = w.Write([]byte(`{"id":"rule-` + string(rune('0'+rulesCreated)) + `","team_id":"team-1","match_labels":{"team":"platform"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/"+defaultWorkspaceID+"/escalation-paths":
			pathsCreated++
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	api, err := NewAegisAPI(srv.URL)
	require.NoError(t, err)
	result, err := EnsureDemoRouting(context.Background(), api, BootstrapOptions{})
	require.NoError(t, err)
	require.Equal(t, 4, teamsCreated)
	require.Equal(t, 4, rulesCreated)
	require.Equal(t, 3, pathsCreated)
	require.Len(t, result.Teams, 4)
	require.Equal(t, 3, result.CreatedPaths)
}

func TestLoadConfigCustomEnv(t *testing.T) {
	t.Setenv("AEGIS_API_URL", "http://api.test")
	t.Setenv("AEGIS_WEBHOOK_URL", "http://api.test/webhook")
	t.Setenv("WEBHOOK_SECRET", "secret")
	t.Setenv("ALERT_SIM_INTERVAL", "2m")

	cfg := LoadConfig()
	require.Equal(t, "http://api.test", cfg.APIBaseURL)
	require.Equal(t, "http://api.test/webhook", cfg.WebhookURL)
	require.Equal(t, 2*time.Minute, cfg.Interval)
}

func TestBuildPayloadValidJSON(t *testing.T) {
	scenario, _ := ScenarioByID("http_5xx")
	raw, err := MarshalPayload(BuildPayload(scenario, LabelDefaults{}, "1"))
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, "firing", decoded["status"])
	labels := decoded["labels"].(map[string]any)
	require.Equal(t, "ops", labels["team"])
}
