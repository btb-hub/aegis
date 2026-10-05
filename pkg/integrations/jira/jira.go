package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	AuthType   string `json:"auth_type"`
	Deployment string `json:"deployment"`
}

const (
	authTypeBearer = "bearer"
	authTypeBasic  = "basic"
)

type Provider struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Provider {
	if cfg.IssueType == "" {
		cfg.IssueType = "Task"
	}
	return &Provider{cfg: cfg, client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func NewFromJSON(raw []byte) (*Provider, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.APIToken) == "" {
		return nil, fmt.Errorf("jira config incomplete")
	}
	if strings.TrimSpace(cfg.ProjectKey) == "" {
		return nil, fmt.Errorf("jira project_key is required")
	}
	if cfg.authType() == authTypeBasic && strings.TrimSpace(cfg.Email) == "" {
		return nil, fmt.Errorf("jira email is required for basic auth")
	}
	if cfg.Deployment != "" && cfg.Deployment != "server_dc" && cfg.Deployment != "cloud" {
		return nil, fmt.Errorf("jira deployment must be server_dc or cloud")
	}
	return New(cfg), nil
}
func (c Config) authType() string {
	if strings.EqualFold(strings.TrimSpace(c.AuthType), authTypeBasic) {
		return authTypeBasic
	}
	return authTypeBearer
}
func (p *Provider) cloud() bool {
	return p.cfg.Deployment == "cloud" || (p.cfg.Deployment == "" && p.cfg.authType() == authTypeBasic)
}
func (p *Provider) apiPath() string {
	if p.cloud() {
		return "/rest/api/3"
	}
	return "/rest/api/2"
}
func (p *Provider) applyAuth(r *http.Request) {
	if p.cfg.authType() == authTypeBasic {
		r.SetBasicAuth(p.cfg.Email, p.cfg.APIToken)
	} else {
		r.Header.Set("Authorization", "Bearer "+p.cfg.APIToken)
	}
}
func (p *Provider) Kind() string { return "jira" }
func (p *Provider) IssueURL(key string) string {
	return strings.TrimRight(p.cfg.BaseURL, "/") + "/browse/" + url.PathEscape(key)
}
func (p *Provider) textBody(body string) any {
	if !p.cloud() {
		return body
	}
	paragraphs := make([]any, 0)
	for _, line := range strings.Split(body, "\n") {
		content := []any{}
		if line != "" {
			content = append(content, map[string]any{"type": "text", "text": line})
		}
		paragraphs = append(paragraphs, map[string]any{"type": "paragraph", "content": content})
	}
	return map[string]any{"type": "doc", "version": 1, "content": paragraphs}
}
func (p *Provider) request(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(p.cfg.BaseURL, "/")+p.apiPath()+path, reader)
	if err != nil {
		return nil, fmt.Errorf("jira %s: invalid URL", method)
	}
	p.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira %s %s: transport failed", method, strings.Split(path, "?")[0])
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		hint := "check project permissions and field configuration"
		if resp.StatusCode < 400 {
			hint = "login redirect: check deployment, REST URL and authentication"
		} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
			hint = "check authentication and Jira permissions"
		}
		return nil, &integrations.HTTPError{Provider: "jira", Operation: method + " " + strings.Split(path, "?")[0], Status: resp.StatusCode, Message: hint}
	}
	return raw, nil
}
func (p *Provider) CreateTicket(ctx context.Context, i integrations.IncidentRef) (string, error) {
	description := fmt.Sprintf("Aegis incident %s\nSeverity: %s", i.ID, i.Severity)
	if i.URL != "" {
		description += "\n" + i.URL
	}
	raw, err := p.request(ctx, http.MethodPost, "/issue", map[string]any{"fields": map[string]any{"project": map[string]string{"key": p.cfg.ProjectKey}, "summary": i.Title, "description": p.textBody(description), "issuetype": map[string]string{"name": p.cfg.IssueType}, "labels": []string{"aegis", "aegis-incident-" + i.ID.String()}}})
	if err != nil {
		return "", err
	}
	var result struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.Key == "" {
		return "", fmt.Errorf("jira response missing issue key")
	}
	return result.Key, nil
}
func (p *Provider) TestConnection(ctx context.Context) error {
	_, err := p.request(ctx, http.MethodGet, "/myself", nil)
	return err
}
func (p *Provider) lookupAccountID(ctx context.Context, email string) (string, error) {
	query := url.Values{"maxResults": {"100"}}
	if p.cloud() {
		query.Set("query", email)
	} else {
		query.Set("username", email)
	}
	raw, err := p.request(ctx, http.MethodGet, "/user/search?"+query.Encode(), nil)
	if err != nil {
		return "", err
	}
	var users []struct {
		AccountID string `json:"accountId"`
		Name      string `json:"name"`
		Email     string `json:"emailAddress"`
		Active    *bool  `json:"active"`
	}
	if err := json.Unmarshal(raw, &users); err != nil {
		return "", err
	}
	var ids []string
	for _, u := range users {
		if u.Active != nil && !*u.Active {
			continue
		}
		if len(users) > 1 && !strings.EqualFold(u.Email, email) && !strings.EqualFold(u.Name, email) {
			continue
		}
		id := u.Name
		if p.cloud() {
			id = u.AccountID
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) > 1 {
		return "", fmt.Errorf("jira user search ambiguous; use an exact username or email")
	}
	if len(ids) == 0 {
		if len(users) > 1 {
			return "", fmt.Errorf("jira user search ambiguous; use an exact username or email")
		}
		return "", nil
	}
	return ids[0], nil
}
func (p *Provider) UpdateAssignee(ctx context.Context, key, email string) error {
	email = strings.TrimSpace(email)
	if key == "" || email == "" {
		return nil
	}
	id, err := p.lookupAccountID(ctx, email)
	if err != nil {
		return err
	}
	if id == "" {
		return &integrations.HTTPError{Provider: "jira", Operation: "assignee lookup", Status: 422, Message: "no active exact Jira user matched; check the engineer email or username"}
	}
	field := "name"
	if p.cloud() {
		field = "accountId"
	}
	_, err = p.request(ctx, http.MethodPut, "/issue/"+url.PathEscape(key), map[string]any{"fields": map[string]any{"assignee": map[string]string{field: id}}})
	return err
}
func (p *Provider) AddComment(ctx context.Context, key, body string) (string, error) {
	raw, err := p.request(ctx, http.MethodPost, "/issue/"+url.PathEscape(key)+"/comment", map[string]any{"body": p.textBody(body)})
	if err != nil {
		return "", err
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.ID == "" {
		return "", fmt.Errorf("jira comment response missing id")
	}
	return result.ID, nil
}

// FindTicket reconciles an uncertain create response using an immutable incident label.
func (p *Provider) FindTicket(ctx context.Context, id string) (string, error) {
	query := url.Values{"jql": {"project = \"" + strings.ReplaceAll(p.cfg.ProjectKey, "\"", "\\\"") + "\" AND labels = \"aegis-incident-" + id + "\""}, "maxResults": {"2"}, "fields": {"key"}}
	path := "/search"
	if p.cloud() {
		path = "/search/jql"
	}
	raw, err := p.request(ctx, http.MethodGet, path+"?"+query.Encode(), nil)
	if err != nil {
		return "", err
	}
	var result struct {
		Issues []struct {
			Key string `json:"key"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if len(result.Issues) > 1 {
		return "", fmt.Errorf("multiple Jira tickets exist for Aegis incident")
	}
	if len(result.Issues) == 0 {
		return "", nil
	}
	return result.Issues[0].Key, nil
}
func (p *Provider) FindComment(ctx context.Context, key, marker string) (string, error) {
	for start := 0; ; {
		raw, err := p.request(ctx, http.MethodGet, fmt.Sprintf("/issue/%s/comment?startAt=%d&maxResults=100", url.PathEscape(key), start), nil)
		if err != nil {
			return "", err
		}
		var result struct {
			Total    int `json:"total"`
			Comments []struct {
				ID   string          `json:"id"`
				Body json.RawMessage `json:"body"`
			} `json:"comments"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return "", err
		}
		for _, c := range result.Comments {
			if strings.Contains(string(c.Body), marker) {
				return c.ID, nil
			}
		}
		start += len(result.Comments)
		if len(result.Comments) == 0 || start >= result.Total {
			return "", nil
		}
	}
}
