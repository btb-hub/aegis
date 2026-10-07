package express

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	cfg      Config
	client   *http.Client
	mu       sync.Mutex
	token    string
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
		"group_chat_id": "",
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
		chatID, lookupErr := p.personalChat(ctx, *recipient.ExpressUserHuid, token)
		if lookupErr != nil {
			return lookupErr
		}
		payload["group_chat_id"] = chatID
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
	return "", fmt.Errorf("express notification response missing sync_id")
}

func (p *Provider) AnnounceOnCall(ctx context.Context, channelID, group, teamName string, people []integrations.OnCallPerson, locale string) error {
	_, err := p.AnnounceOnCallWithRef(ctx, channelID, group, teamName, people, locale)
	return err
}
func (p *Provider) AnnounceOnCallWithRef(ctx context.Context, channelID, _ string, teamName string, people []integrations.OnCallPerson, locale string) (string, error) {
	if strings.TrimSpace(channelID) == "" {
		return "", fmt.Errorf("express chat id is required")
	}
	if locale == "" {
		locale = "en"
	}

	mentions := make([]map[string]any, 0, len(people))
	names := make([]string, 0, len(people))
	for _, person := range people {
		if person.ExpressUserHuid == nil || strings.TrimSpace(*person.ExpressUserHuid) == "" {
			if person.DisplayName != "" {
				names = append(names, integrations.OnCallName(person, locale))
			}
			continue
		}
		mentionID := uuid.New()
		name := fmt.Sprintf("@{mention:%s}", mentionID.String())
		if shift := integrations.ShiftTimeMSK(person.StartAt, person.EndAt, locale); shift != "" {
			name += " (" + shift + ")"
		}
		names = append(names, name)
		mentions = append(mentions, map[string]any{
			"mention_type": "user",
			"mention_id":   mentionID.String(),
			"mention_data": map[string]any{
				"user_huid": *person.ExpressUserHuid,
			},
		})
	}

	var body string
	if len(names) == 0 {
		body = i18n.T(locale, "oncall.announce_empty", map[string]string{"team": teamName})
	} else {
		body = i18n.T(locale, "oncall.announce", map[string]string{
			"team":   teamName,
			"people": strings.Join(names, ", "),
		})
	}

	payload := map[string]any{
		"group_chat_id": channelID,
		"notification": map[string]any{
			"status":   "ok",
			"body":     body,
			"mentions": mentions,
		},
	}

	return p.sendNotification(ctx, payload)
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
		return "", &integrations.HTTPError{Provider: "express", Operation: "token request", Status: resp.StatusCode, Message: "check bot credentials and server configuration"}
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
		return nil, &integrations.HTTPError{Provider: "express", Operation: "POST " + path, Status: resp.StatusCode, Message: "check bot permissions, destination chat and request configuration"}
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
			var httpErr *integrations.HTTPError
			if errors.As(err, &httpErr) && httpErr.Status != http.StatusUnauthorized && !httpErr.Retryable() {
				return err
			}
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

func (p *Provider) personalChat(ctx context.Context, huid, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL()+"/api/v1/botx/chats/personal?"+url.Values{"user_huid": {huid}}.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", &integrations.HTTPError{Provider: "express", Operation: "personal chat lookup", Status: resp.StatusCode, Message: "open a personal chat with the Aegis bot before receiving private pages"}
	}
	var result struct {
		Status string `json:"status"`
		Result struct {
			ChatID string `json:"group_chat_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Status != "ok" || strings.TrimSpace(result.Result.ChatID) == "" {
		return "", fmt.Errorf("open a personal chat with the Aegis bot before receiving private pages")
	}
	return result.Result.ChatID, nil
}
func (p *Provider) SendMessage(ctx context.Context, chatID, body, locale string) (string, error) {
	if strings.TrimSpace(chatID) == "" {
		return "", fmt.Errorf("express chat id is required")
	}
	payload := map[string]any{"group_chat_id": chatID, "notification": map[string]any{"status": "ok", "body": body}}
	return p.sendNotification(ctx, payload)
}
func (p *Provider) SendIncidentToChannel(ctx context.Context, i integrations.IncidentRef, chatID, locale string) (string, error) {
	payload := map[string]any{"group_chat_id": chatID, "notification": map[string]any{
		"status": "ok", "body": fmt.Sprintf("%s: %s\n%s", i.Severity, i.Title, i.URL),
		"bubble": [][]map[string]any{{{"command": ackCommand, "label": i18n.T(locale, "page.acknowledge_button", nil), "data": map[string]string{"incident_id": i.ID.String()}, "opts": map[string]any{"silent": true}}}},
	}}
	return p.sendNotification(ctx, payload)
}
func (p *Provider) sendNotification(ctx context.Context, payload any) (string, error) {
	var raw []byte
	err := p.withRetry(ctx, func() error {
		token, err := p.ensureToken(ctx)
		if err != nil {
			return err
		}
		raw, err = p.postJSON(ctx, "/api/v4/botx/notifications/direct", payload, token)
		return err
	})
	if err != nil {
		return "", err
	}
	var result struct {
		Status string `json:"status"`
		Result struct {
			SyncID string `json:"sync_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.Status != "ok" || result.Result.SyncID == "" {
		return "", fmt.Errorf("express notification response missing sync_id")
	}
	return result.Result.SyncID, nil
}
