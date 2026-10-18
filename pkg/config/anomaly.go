package config

import (
	"fmt"
	"net/url"
	"strings"
)

// AnomalyConfig configures the optional module, independently of core services.
type AnomalyConfig struct {
	URL            string
	Enabled        bool
	InstallationID string
	IdentitySecret string
	ServiceToken   string
}

func (c AnomalyConfig) Configured() bool {
	return c.URL != "" && c.InstallationID != "" && len(c.IdentitySecret) >= 32 && len(c.ServiceToken) >= 32
}

func anomalyConfig(values map[string]string) (AnomalyConfig, error) {
	c := AnomalyConfig{URL: strings.TrimRight(values["AEGIS_ANOMALY_URL"], "/"), Enabled: values["AEGIS_ANOMALY_ENABLED"] == "true", InstallationID: values["AEGIS_INSTALLATION_ID"], IdentitySecret: values["AEGIS_ANOMALY_IDENTITY_SECRET"], ServiceToken: values["AEGIS_ANOMALY_SERVICE_TOKEN"]}
	if value := values["AEGIS_ANOMALY_ENABLED"]; value != "" && value != "true" && value != "false" {
		return c, fmt.Errorf("invalid setting AEGIS_ANOMALY_ENABLED")
	}
	if c.URL != "" {
		u, err := url.Parse(c.URL)
		if err != nil || !ValidURL(c.URL) || strings.ContainsAny(c.URL, "?#") || (u.Path != "" && u.Path != "/") {
			return c, fmt.Errorf("invalid setting AEGIS_ANOMALY_URL")
		}
	}
	if len(c.InstallationID) > 128 || strings.TrimSpace(c.InstallationID) != c.InstallationID || strings.ContainsRune(c.InstallationID, 0) {
		return c, fmt.Errorf("invalid setting AEGIS_INSTALLATION_ID")
	}
	for _, key := range []string{"AEGIS_ANOMALY_IDENTITY_SECRET", "AEGIS_ANOMALY_SERVICE_TOKEN"} {
		if value := values[key]; value != "" && len(value) < 32 {
			return c, fmt.Errorf("invalid setting %s: at least 32 bytes required", key)
		}
	}
	if c.IdentitySecret != "" && c.IdentitySecret == c.ServiceToken {
		return c, fmt.Errorf("module identity and service credentials must differ")
	}
	if (c.URL != "" || c.Enabled) && !c.Configured() {
		return c, fmt.Errorf("complete anomaly module connection settings before configuring its URL or enabling access")
	}
	return c, nil
}
