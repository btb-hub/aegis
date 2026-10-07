package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (p *Provider) WorkspaceID(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL("/api/auth.test"), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.BotToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("slack workspace lookup failed")
	}
	var result struct {
		OK     bool   `json:"ok"`
		TeamID string `json:"team_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK || result.TeamID == "" {
		return "", fmt.Errorf("slack bot workspace unavailable")
	}
	return result.TeamID, nil
}
