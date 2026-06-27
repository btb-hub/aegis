package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/integrations"
)

type Config struct {
	BaseURL    string `json:"base_url"`
	Email      string `json:"email"`
	APIToken   string `json:"api_token"`
	ProjectKey string `json:"project_key"`
	IssueType  string `json:"issue_type"`
}

type Provider struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Provider {
	if cfg.IssueType == "" {
		cfg.IssueType = "Task"
	}
	return &Provider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func NewFromJSON(raw []byte) (*Provider, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Email) == "" || strings.TrimSpace(cfg.APIToken) == "" {
		return nil, fmt.Errorf("jira config incomplete")
	}
	if cfg.ProjectKey == "" {
		return nil, fmt.Errorf("jira project_key is required")
	}
	return New(cfg), nil
}

func (p *Provider) Kind() string { return "jira" }

func (p *Provider) CreateTicket(ctx context.Context, incident integrations.IncidentRef) (string, error) {
	body := map[string]any{
		"fields": map[string]any{
			"project":     map[string]string{"key": p.cfg.ProjectKey},
			"summary":     incident.Title,
			"description": fmt.Sprintf("Aegis incident %s\nSeverity: %s", incident.ID, incident.Severity),
			"issuetype":   map[string]string{"name": p.cfg.IssueType},
			"labels":      []string{"aegis"},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.cfg.BaseURL, "/")+"/rest/api/3/issue", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(p.cfg.Email, p.cfg.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("jira create issue: status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if parsed.Key == "" {
		return "", fmt.Errorf("jira response missing issue key")
	}
	return parsed.Key, nil
}

func (p *Provider) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(p.cfg.BaseURL, "/")+"/rest/api/3/myself", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.cfg.Email, p.cfg.APIToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira auth failed: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
