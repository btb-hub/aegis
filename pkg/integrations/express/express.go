package express

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
)

const ackCommand = "/ack_incident"

type Config struct {
	BotID     string `json:"bot_id"`
	Host      string `json:"host"`
	SecretKey string `json:"secret_key"`
}

type Provider struct {
	cfg    Config
	client *http.Client
	mu     sync.Mutex
	token  string
	tokenExp time.Time
}

func New(cfg Config) *Provider {
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
	if strings.TrimSpace(cfg.BotID) == "" || strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("express config incomplete")
	}
	return New(cfg), nil
}

func (p *Provider) Kind() string { return "express" }

func (p *Provider) SendPage(ctx context.Context, incident integrations.IncidentRef, recipient integrations.PageRecipient) (string, error) {
	if recipient.ExpressUserHuid == nil || strings.TrimSpace(*recipient.ExpressUserHuid) == "" {
		return "", fmt.Errorf("recipient has no express_user_huid")
	}

	locale := recipient.Locale
	if locale == "" {
		locale = "en"
	}
	ackLabel := i18n.T(locale, "page.acknowledge_button", nil)
	title := i18n.T(locale, "page.incident_title", map[string]string{"id": incident.ID.String()[:8]})
	body := fmt.Sprintf("%s: %s\n%s", incident.Severity, incident.Title, title)

	payload := map[string]any{
		"group_chat_id": nil,
		"notification": map[string]any{
			"status": "ok",
			"body":   body,
			"bubble": [][]map[string]any{
				{
					{
						"command": ackCommand,
						"label":   ackLabel,
						"data": map[string]string{
							"incident_id": incident.ID.String(),
						},
						"opts": map[string]any{"silent": true, "align": "center"},
					},
				},
			},
			"keyboard": []any{},
			"mentions": []any{},
		},
		"file": nil,
		"opts": map[string]any{
			"stealth_mode": false,
			"notification_opts": map[string]any{
				"send":      true,
				"force_dnd": true,
			},
		},
		"recipients": []string{*recipient.ExpressUserHuid},
	}

	var respBody []byte
	err := p.withRetry(ctx, func() error {
		token, err := p.ensureToken(ctx)
		if err != nil {
			return err
		}
		respBody, err = p.postJSON(ctx, "/api/v4/botx/notifications/direct", payload, token)
		return err
	})
	if err != nil {
		return "", err
	}

	var parsed struct {
		Status string `json:"status"`
		Result struct {
			SyncID string `json:"sync_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if parsed.Status != "ok" {
		return "", fmt.Errorf("express notification failed: %s", string(respBody))
	}
	if parsed.Result.SyncID != "" {
		return parsed.Result.SyncID, nil
	}
	return uuid.New().String(), nil
}

func (p *Provider) TestConnection(ctx context.Context) error {
	return p.withRetry(ctx, func() error {
		_, err := p.ensureToken(ctx)
		return err
	})
}

func (p *Provider) ensureToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token != "" && time.Now().Before(p.tokenExp) {
		return p.token, nil
	}

	signature := signBotID(p.cfg.BotID, p.cfg.SecretKey)
	url := fmt.Sprintf("%s/api/v2/botx/bots/%s/token?signature=%s", p.baseURL(), p.cfg.BotID, signature)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("express token request failed: status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Status string `json:"status"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.Status != "ok" || strings.TrimSpace(parsed.Result) == "" {
		return "", fmt.Errorf("express token response invalid: %s", string(body))
	}
	p.token = parsed.Result
	p.tokenExp = time.Now().Add(50 * time.Minute)
	return p.token, nil
}

func (p *Provider) postJSON(ctx context.Context, path string, payload any, token string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL()+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("express request failed: status %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func (p *Provider) baseURL() string {
	return strings.TrimRight(p.cfg.Host, "/")
}

func (p *Provider) withRetry(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		if attempt == 0 {
			p.mu.Lock()
			p.token = ""
			p.mu.Unlock()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
	return err
}
