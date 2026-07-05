package alertsim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCatalogHasScenarios(t *testing.T) {
	items := Catalog()
	require.GreaterOrEqual(t, len(items), 10)
	for _, s := range items {
		require.NotEmpty(t, s.ID)
		require.NotEmpty(t, s.AlertName)
		require.NotEmpty(t, s.Severity)
		require.NotEmpty(t, s.Summary)
	}
}

func TestBuildPayloadAppliesDefaults(t *testing.T) {
	scenario, ok := ScenarioByID("high_cpu")
	require.True(t, ok)

	payload := BuildPayload(scenario, LabelDefaults{Team: "platform", Project: "data"}, "abc")
	require.Equal(t, "firing", payload.Status)
	require.Equal(t, "HighCPUUsage", payload.Labels["alertname"])
	require.Equal(t, "platform", payload.Labels["team"])
	require.Equal(t, "data", payload.Labels["project"])
	require.Equal(t, "critical", payload.Labels["severity"])
	require.Contains(t, payload.Labels["instance"], "high_cpu-abc")
	require.Equal(t, scenario.Summary, payload.Annotations["summary"])
}

func TestScenarioByIDUnknown(t *testing.T) {
	_, ok := ScenarioByID("does-not-exist")
	require.False(t, ok)
}

func TestMarshalPayloadValidJSON(t *testing.T) {
	scenario, _ := ScenarioByID("http_5xx")
	raw, err := MarshalPayload(BuildPayload(scenario, LabelDefaults{Team: "platform"}, "1"))
	require.NoError(t, err)
	require.Contains(t, string(raw), `"alertname":"HighHTTP5xxRate"`)
}
