package anomalymodule

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aegis/aegis/pkg/config"
)

type Capabilities struct {
	ContractVersion int    `json:"contract_version"`
	Module          string `json:"module"`
	Version         string `json:"version"`
	Healthy         bool   `json:"healthy"`
	Entitled        bool   `json:"entitled"`
}

type Status struct {
	Configured bool   `json:"configured"`
	Enabled    bool   `json:"enabled"`
	Available  bool   `json:"available"`
	Compatible bool   `json:"compatible"`
	Version    string `json:"version,omitempty"`
}

func Check(ctx context.Context, c config.AnomalyConfig, client *http.Client) (Status, error) {
	status := Status{Configured: c.Configured(), Enabled: c.Enabled}
	if !status.Configured {
		return status, fmt.Errorf("module is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL+"/api/v1/capabilities", nil)
	if err != nil {
		return status, fmt.Errorf("invalid module connection")
	}
	req.Header.Set("Authorization", "Bearer "+c.ServiceToken)
	resp, err := client.Do(req)
	if err != nil {
		return status, fmt.Errorf("module is unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return status, fmt.Errorf("module is unavailable")
	}
	var caps Capabilities
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 65537))
	if err = decoder.Decode(&caps); err != nil {
		return status, fmt.Errorf("invalid module response")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return status, fmt.Errorf("invalid module response")
	}
	status.Available = caps.Healthy
	status.Compatible = caps.ContractVersion == ContractVersion && caps.Module == Name && caps.Version != "" && len(caps.Version) <= 128
	if !status.Available || !status.Compatible {
		return status, fmt.Errorf("module is unavailable or incompatible")
	}
	status.Version = caps.Version
	return status, nil
}
