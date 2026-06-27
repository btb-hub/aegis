package alertparse

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAlertmanagerPayload(t *testing.T) {
	raw := json.RawMessage(`{
		"status": "firing",
		"labels": {"alertname": "HighCPU", "severity": "critical", "team": "platform"},
		"annotations": {"summary": "CPU high on host-1", "description": "95% usage"},
		"startsAt": "2026-06-26T12:00:00Z"
	}`)

	parsed, err := Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "firing", parsed.Status)
	require.Equal(t, "critical", parsed.Severity)
	require.Equal(t, "CPU high on host-1", parsed.Title)
	require.Equal(t, "95% usage", parsed.Body)
	require.NotEmpty(t, parsed.Fingerprint)
}

func TestParseDefaultsStatusToFiring(t *testing.T) {
	raw := json.RawMessage(`{"labels": {"alertname": "DiskFull"}}`)
	parsed, err := Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "firing", parsed.Status)
	require.Equal(t, "DiskFull", parsed.Title)
}

func TestParseInvalidStatus(t *testing.T) {
	raw := json.RawMessage(`{"status": "unknown"}`)
	_, err := Parse(raw)
	require.Error(t, err)
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := Parse(json.RawMessage(`{`))
	require.Error(t, err)
}

func TestFingerprintStable(t *testing.T) {
	labels := map[string]string{"b": "2", "a": "1"}
	fp1 := Fingerprint(labels)
	fp2 := Fingerprint(map[string]string{"a": "1", "b": "2"})
	require.Equal(t, fp1, fp2)
}

func TestFingerprintFromKeysSubset(t *testing.T) {
	labels := map[string]string{"alertname": "HighCPU", "team": "platform", "instance": "host-1"}
	full := Fingerprint(labels)
	subset := FingerprintFromKeys(labels, []string{"alertname", "team"})
	require.NotEqual(t, full, subset)
	require.Equal(t, subset, FingerprintFromKeys(labels, []string{"alertname", "team"}))
	require.NotEqual(t, subset, FingerprintFromKeys(labels, []string{"alertname", "instance"}))
}

func TestValidateWebhookSecret(t *testing.T) {
	require.True(t, ValidateWebhookSecret("secret", "secret"))
	require.False(t, ValidateWebhookSecret("wrong", "secret"))
	require.False(t, ValidateWebhookSecret("", "secret"))
}
