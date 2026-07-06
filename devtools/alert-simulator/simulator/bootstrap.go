package simulator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const defaultWorkspaceID = "00000000-0000-0000-0000-000000000001"

// BootstrapOptions configure demo routing setup via the Aegis HTTP API.
type BootstrapOptions struct {
	APIBaseURL   string
	TeamLabelKey string
	TeamLabel    string
	TeamName     string
}

// BootstrapResult describes what was ensured during bootstrap.
type BootstrapResult struct {
	TeamID        string
	TeamName      string
	RoutingRuleID string
	CreatedTeam   bool
	CreatedRule   bool
}

// AegisAPI is a minimal HTTP client for dev bootstrap and health checks.
type AegisAPI struct {
	BaseURL string
	Client  *http.Client
}

// NewAegisAPI builds a client with cookie support for dev auth sessions.
func NewAegisAPI(baseURL string) (*AegisAPI, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &AegisAPI{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}, nil
}

// DevLogin signs in with local dev auth (requires DEV_AUTH_ENABLED on the API).
// The API redirects to PUBLIC_URL (often localhost:3000); we stop after the 302 and keep
// the session cookie so bootstrap works inside Docker where the web URL is not reachable.
func (a *AegisAPI) DevLogin(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		a.BaseURL+"/auth/dev/login?role=admin",
		nil,
	)
	if err != nil {
		return err
	}

	client := *a.Client
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("dev auth is disabled on %s", a.BaseURL)
	}
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dev login returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// EnsureDemoRouting creates a Platform team and routing rule through the public API.
func EnsureDemoRouting(ctx context.Context, api *AegisAPI, opts BootstrapOptions) (BootstrapResult, error) {
	if opts.TeamLabelKey == "" {
		opts.TeamLabelKey = "team"
	}
	if opts.TeamLabel == "" {
		opts.TeamLabel = "platform"
	}
	if opts.TeamName == "" {
		opts.TeamName = "Platform"
	}

	if err := api.DevLogin(ctx); err != nil {
		return BootstrapResult{}, fmt.Errorf("dev login: %w", err)
	}

	rules, err := api.listRoutingRules(ctx)
	if err != nil {
		return BootstrapResult{}, err
	}
	for _, rule := range rules {
		if rule.MatchLabels[opts.TeamLabelKey] == opts.TeamLabel {
			return BootstrapResult{
				TeamID:        rule.TeamID,
				TeamName:      opts.TeamName,
				RoutingRuleID: rule.ID,
			}, nil
		}
	}

	teamID, createdTeam, err := api.findOrCreateTeam(ctx, opts.TeamName)
	if err != nil {
		return BootstrapResult{}, err
	}

	ruleID, err := api.createRoutingRule(ctx, teamID, map[string]string{
		opts.TeamLabelKey: opts.TeamLabel,
	}, 10)
	if err != nil {
		return BootstrapResult{}, err
	}

	return BootstrapResult{
		TeamID:        teamID,
		TeamName:      opts.TeamName,
		RoutingRuleID: ruleID,
		CreatedTeam:   createdTeam,
		CreatedRule:   true,
	}, nil
}

type routingRuleItem struct {
	ID          string            `json:"id"`
	TeamID      string            `json:"team_id"`
	MatchLabels map[string]string `json:"match_labels"`
}

type teamItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (a *AegisAPI) listRoutingRules(ctx context.Context) ([]routingRuleItem, error) {
	var payload struct {
		Items []routingRuleItem `json:"items"`
	}
	if err := a.getJSON(ctx, "/api/v1/routing-rules", &payload); err != nil {
		return nil, fmt.Errorf("list routing rules: %w", err)
	}
	return payload.Items, nil
}

func (a *AegisAPI) findOrCreateTeam(ctx context.Context, name string) (string, bool, error) {
	query := url.Values{}
	query.Set("workspace_id", defaultWorkspaceID)
	var payload struct {
		Items []teamItem `json:"items"`
	}
	if err := a.getJSON(ctx, "/api/v1/teams?"+query.Encode(), &payload); err != nil {
		return "", false, fmt.Errorf("list teams: %w", err)
	}
	for _, team := range payload.Items {
		if team.Name == name {
			return team.ID, false, nil
		}
	}

	tier := "l2"
	body, _ := json.Marshal(map[string]any{
		"workspace_id": defaultWorkspaceID,
		"name":         name,
		"description":  "Demo routing target for alert simulator",
		"support_tier": tier,
	})
	var created teamItem
	if err := a.postJSON(ctx, "/api/v1/teams", body, &created); err != nil {
		return "", false, fmt.Errorf("create team: %w", err)
	}
	return created.ID, true, nil
}

func (a *AegisAPI) createRoutingRule(ctx context.Context, teamID string, labels map[string]string, priority int) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"team_id":      teamID,
		"match_labels": labels,
		"priority":     priority,
	})
	var created routingRuleItem
	if err := a.postJSON(ctx, "/api/v1/routing-rules", body, &created); err != nil {
		return "", err
	}
	return created.ID, nil
}

func (a *AegisAPI) getJSON(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.BaseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (a *AegisAPI) postJSON(ctx context.Context, path string, body []byte, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if dest == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
