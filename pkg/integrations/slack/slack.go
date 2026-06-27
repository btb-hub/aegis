package slack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/integrations"
)

type Config struct {
	BotToken      string `json:"bot_token"`
	SigningSecret string `json:"signing_secret"`
	PublicURL     string `json:"public_url"`
	APIBaseURL    string `json:"api_base_url"`
}

type Provider struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Provider {
	return &Provider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func NewFromJSON(raw []byte, publicURL string) (*Provider, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	cfg.PublicURL = publicURL
	if strings.TrimSpace(cfg.BotToken) == "" || strings.TrimSpace(cfg.SigningSecret) == "" {
		return nil, fmt.Errorf("slack config incomplete")
	}
	return New(cfg), nil
}

func (p *Provider) Kind() string { return "slack" }

func (p *Provider) SendPage(ctx context.Context, incident integrations.IncidentRef, recipient integrations.PageRecipient) (string, error) {
	if recipient.SlackUserID == nil || strings.TrimSpace(*recipient.SlackUserID) == "" {
		return "", fmt.Errorf("recipient has no slack_user_id")
	}

	locale := recipient.Locale
	if locale == "" {
		locale = "en"
	}
	ackLabel := i18n.T(locale, "page.acknowledge_button", nil)
	title := i18n.T(locale, "page.incident_title", map[string]string{"id": incident.ID.String()[:8]})

	blocks := []map[string]any{
		{"type": "header", "text": map[string]string{"type": "plain_text", "text": incident.Severity + ": " + incident.Title}},
		{"type": "section", "text": map[string]string{"type": "mrkdwn", "text": title}},
		{
			"type": "actions",
			"elements": []map[string]any{
				{
					"type":      "button",
					"action_id": "ack_incident",
					"text":      map[string]string{"type": "plain_text", "text": ackLabel},
					"value":     incident.ID.String(),
				},
			},
		},
	}

	payload := map[string]any{
		"channel": *recipient.SlackUserID,
		"text":    incident.Title,
		"blocks":  blocks,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL("/api/chat.postMessage"), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.BotToken)
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

	var parsed struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		TS    string `json:"ts"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if !parsed.OK {
		return "", fmt.Errorf("slack chat.postMessage: %s", parsed.Error)
	}
	return parsed.TS, nil
}

func (p *Provider) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL("/api/auth.test"), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.BotToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return err
	}
	if !parsed.OK {
		return fmt.Errorf("slack auth.test: %s", parsed.Error)
	}
	return nil
}

func (p *Provider) apiURL(path string) string {
	base := strings.TrimRight(p.cfg.APIBaseURL, "/")
	if base == "" {
		base = "https://slack.com"
	}
	return base + path
}

func VerifySignature(secret string, timestamp, signature string, body []byte) error {
	if secret == "" || timestamp == "" || signature == "" {
		return fmt.Errorf("missing slack signature headers")
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return fmt.Errorf("slack request too old")
	}
	base := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(base))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid slack signature")
	}
	return nil
}

type InteractivePayload struct {
	Type string `json:"type"`
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	Actions []struct {
		ActionID string `json:"action_id"`
		Value    string `json:"value"`
	} `json:"actions"`
}

func ParseInteractiveAck(form url.Values) (incidentID, slackUserID string, err error) {
	raw := form.Get("payload")
	if raw == "" {
		return "", "", fmt.Errorf("missing payload")
	}
	var payload InteractivePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", "", err
	}
	if payload.Type != "block_actions" {
		return "", "", fmt.Errorf("unsupported interaction type")
	}
	for _, action := range payload.Actions {
		if action.ActionID == "ack_incident" {
			return action.Value, payload.User.ID, nil
		}
	}
	return "", "", fmt.Errorf("ack action not found")
}
