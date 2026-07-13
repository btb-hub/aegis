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

// TierDemo describes a demo team and routing rule for one support tier.
type TierDemo struct {
	Label    string // routing label value (team=noc, team=ops, …)
	Name     string // team display name
	Tier     string // support_tier (noc, l1, l2, l3)
	Priority int    // routing rule priority
}

// TierDemos returns the full tier ladder used by bootstrap and scenario routing.
func TierDemos() []TierDemo {
	return []TierDemo{
		{Label: "noc", Name: "NOC", Tier: "noc", Priority: 40},
		{Label: "l1", Name: "Helpdesk", Tier: "l1", Priority: 30},
		{Label: "ops", Name: "Ops", Tier: "l2", Priority: 20},
		{Label: "platform", Name: "Platform", Tier: "l3", Priority: 10},
	}
}

// BootstrapOptions configure demo routing setup via the Aegis HTTP API.
type BootstrapOptions struct {
	APIBaseURL   string
	TeamLabelKey string
}

// BootstrappedTeam describes one ensured team and routing rule.
type BootstrappedTeam struct {
	TeamID        string
	TeamName      string
	TeamLabel     string
	Tier          string
	RoutingRuleID string
	CreatedTeam   bool
	CreatedRule   bool
}

// BootstrapResult describes what was ensured during bootstrap.
type BootstrapResult struct {
	Teams        []BootstrappedTeam
	CreatedPaths int
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

// EnsureDemoRouting ensures demo teams for every support tier, routing rules, and escalation paths.
func EnsureDemoRouting(ctx context.Context, api *AegisAPI, opts BootstrapOptions) (BootstrapResult, error) {
	if opts.TeamLabelKey == "" {
		opts.TeamLabelKey = "team"
	}

	if err := api.DevLogin(ctx); err != nil {
		return BootstrapResult{}, fmt.Errorf("dev login: %w", err)
	}

	rules, err := api.listRoutingRules(ctx)
	if err != nil {
		return BootstrapResult{}, err
	}

	result := BootstrapResult{}
	teamByLabel := map[string]string{}

	for _, demo := range TierDemos() {
		bt, err := api.ensureTierDemo(ctx, demo, opts.TeamLabelKey, rules)
		if err != nil {
			return BootstrapResult{}, err
		}
		result.Teams = append(result.Teams, bt)
		teamByLabel[demo.Label] = bt.TeamID
	}

	paths, err := api.listEscalationPaths(ctx)
	if err != nil {
		return BootstrapResult{}, err
	}
	existingPaths := map[string]struct{}{}
	for _, p := range paths {
		existingPaths[p.FromTeamID+"->"+p.ToTeamID] = struct{}{}
	}

	chain := []struct{ from, to string }{
		{"noc", "l1"},
		{"l1", "ops"},
		{"ops", "platform"},
	}
	for _, link := range chain {
		fromID := teamByLabel[link.from]
		toID := teamByLabel[link.to]
		if fromID == "" || toID == "" {
			continue
		}
		key := fromID + "->" + toID
		if _, ok := existingPaths[key]; ok {
			continue
		}
		if err := api.createEscalationPath(ctx, fromID, toID); err != nil {
			return BootstrapResult{}, err
		}
		result.CreatedPaths++
		existingPaths[key] = struct{}{}
	}

	return result, nil
}

func (a *AegisAPI) ensureTierDemo(
	ctx context.Context,
	demo TierDemo,
	labelKey string,
	rules []routingRuleItem,
) (BootstrappedTeam, error) {
	for _, rule := range rules {
		if rule.MatchLabels[labelKey] == demo.Label {
			return BootstrappedTeam{
				TeamID:        rule.TeamID,
				TeamName:      demo.Name,
				TeamLabel:     demo.Label,
				Tier:          demo.Tier,
				RoutingRuleID: rule.ID,
			}, nil
		}
	}

	teamID, createdTeam, err := a.findOrCreateTeam(ctx, demo.Name, demo.Tier)
	if err != nil {
		return BootstrappedTeam{}, err
	}

	ruleID, err := a.createRoutingRule(ctx, teamID, map[string]string{
		labelKey: demo.Label,
	}, demo.Priority)
	if err != nil {
		return BootstrappedTeam{}, err
	}

	return BootstrappedTeam{
		TeamID:        teamID,
		TeamName:      demo.Name,
		TeamLabel:     demo.Label,
		Tier:          demo.Tier,
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

type escalationPathItem struct {
	ID         string `json:"id"`
	FromTeamID string `json:"from_team_id"`
	ToTeamID   string `json:"to_team_id"`
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

func (a *AegisAPI) listEscalationPaths(ctx context.Context) ([]escalationPathItem, error) {
	var payload struct {
		Items []escalationPathItem `json:"items"`
	}
	path := fmt.Sprintf("/api/v1/workspaces/%s/escalation-paths", defaultWorkspaceID)
	if err := a.getJSON(ctx, path, &payload); err != nil {
		return nil, fmt.Errorf("list escalation paths: %w", err)
	}
	return payload.Items, nil
}

func (a *AegisAPI) findOrCreateTeam(ctx context.Context, name, tier string) (string, bool, error) {
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

func (a *AegisAPI) createEscalationPath(ctx context.Context, fromTeamID, toTeamID string) error {
	body, _ := json.Marshal(map[string]any{
		"from_team_id": fromTeamID,
		"to_team_id":   toTeamID,
	})
	path := fmt.Sprintf("/api/v1/workspaces/%s/escalation-paths", defaultWorkspaceID)
	return a.postJSON(ctx, path, body, nil)
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
