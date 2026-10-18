package config

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestAnomalySettings(t *testing.T) {
	values := map[string]string{"AEGIS_ANOMALY_URL": "https://module.example/", "AEGIS_INSTALLATION_ID": "customer-installation", "AEGIS_ANOMALY_ENABLED": "true", "AEGIS_ANOMALY_IDENTITY_SECRET": strings.Repeat("i", 32), "AEGIS_ANOMALY_SERVICE_TOKEN": strings.Repeat("s", 32)}
	c, err := anomalyConfig(values)
	require.NoError(t, err)
	require.Equal(t, "https://module.example", c.URL)
	require.True(t, c.Configured())
	require.True(t, c.Enabled)
	for _, key := range SettingSections["anomaly"] {
		require.True(t, KnownKey(key))
	}
	require.True(t, SecretKey("AEGIS_ANOMALY_SERVICE_TOKEN"))
	require.True(t, SecretKey("AEGIS_ANOMALY_IDENTITY_SECRET"))
	cases := map[string]string{"AEGIS_ANOMALY_URL": "https://module/path", "AEGIS_ANOMALY_ENABLED": "yes", "AEGIS_INSTALLATION_ID": strings.Repeat("i", 129), "AEGIS_ANOMALY_IDENTITY_SECRET": "short", "AEGIS_ANOMALY_SERVICE_TOKEN": "short"}
	for key, value := range cases {
		copy := map[string]string{}
		for k, v := range values {
			copy[k] = v
		}
		copy[key] = value
		require.Error(t, ValidateValues(copy))
	}
	copy := map[string]string{}
	for k, v := range values {
		copy[k] = v
	}
	for _, value := range []string{"https://module?secret=private", "http://module?", "http://module#"} {
		copy["AEGIS_ANOMALY_URL"] = value
		require.Error(t, ValidateValues(copy))
	}
	copy["AEGIS_ANOMALY_URL"] = "http://module"
	copy["AEGIS_INSTALLATION_ID"] = "invalid\x00installation"
	require.Error(t, ValidateValues(copy))
	copy["AEGIS_INSTALLATION_ID"] = values["AEGIS_INSTALLATION_ID"]
	copy["AEGIS_ANOMALY_SERVICE_TOKEN"] = copy["AEGIS_ANOMALY_IDENTITY_SECRET"]
	require.Error(t, ValidateValues(copy))
	delete(copy, "AEGIS_ANOMALY_SERVICE_TOKEN")
	require.Error(t, ValidateValues(copy))
	require.NoError(t, ValidateValues(map[string]string{}))
	doc := DefaultSettings()
	for k, v := range values {
		doc.Values[k] = v
	}
	cfg, err := doc.Config(&Config{})
	require.NoError(t, err)
	require.Equal(t, c, cfg.Anomaly)
}
