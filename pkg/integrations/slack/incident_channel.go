package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/integrations"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (p *Provider) SendIncidentEvent(ctx context.Context, event integrations.IncidentChannelPost, channel string) (string, error) {
	text := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(integrations.IncidentEventBody(event))
	if event.Actionable && strings.TrimSpace(event.SlackUserGroupID) != "" {
		text = "<!subteam^" + event.SlackUserGroupID + "> " + text
	}
	// plain_text prevents incident titles or actor names from injecting mentions.
	blocks := []map[string]any{{"type": "section", "text": map[string]string{"type": "plain_text", "text": integrations.IncidentEventBody(event)}}}
	if event.Actionable && strings.TrimSpace(event.SlackUserGroupID) != "" {
		blocks = append([]map[string]any{{"type": "section", "text": map[string]string{"type": "mrkdwn", "text": "<!subteam^" + event.SlackUserGroupID + ">"}}}, blocks...)
	}
	blocks = append(blocks, map[string]any{"type": "section", "text": map[string]string{"type": "mrkdwn", "text": "<" + event.Incident.URL + "|" + i18n.T(event.Locale, "incident_chat.open", nil) + ">"}})
	if event.Actionable {
		blocks = append(blocks, map[string]any{"type": "actions", "elements": []map[string]any{{"type": "button", "action_id": "ack_incident", "text": map[string]string{"type": "plain_text", "text": i18n.T(event.Locale, "page.acknowledge_button", nil)}, "value": event.Incident.ID.String()}}})
	}
	return p.call(ctx, "chat.postMessage", map[string]any{"channel": channel, "text": text, "blocks": blocks, "parse": "none", "link_names": false})
}

func (p *Provider) SendAckFeedback(ctx context.Context, responseURL, channel, user, text string) error {
	if responseURL != "" {
		u, err := url.Parse(responseURL)
		if err != nil || u.Scheme != "https" || u.User != nil || (u.Host != "hooks.slack.com" && u.Host != "hooks.slack-gov.com") {
			return &integrations.HTTPError{Provider: "slack", Operation: "feedback", Status: 400, Message: "invalid response URL"}
		}
		body, _ := json.Marshal(map[string]any{"text": text, "response_type": "ephemeral", "replace_original": false})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, responseURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := p.client.Do(req)
		if err != nil {
			return fmt.Errorf("slack feedback request failed")
		}
		defer resp.Body.Close()
		if resp.StatusCode == 403 || resp.StatusCode == 404 || resp.StatusCode == 410 {
			_, err = p.call(ctx, "chat.postEphemeral", map[string]any{"channel": channel, "user": user, "text": text})
			return err
		}
		if resp.StatusCode >= 300 {
			return &integrations.HTTPError{Provider: "slack", Operation: "feedback", Status: resp.StatusCode, Message: "response delivery failed", RetryAfter: integrations.ParseRetryAfter(resp.Header.Get("Retry-After"))}
		}
		return nil
	}
	_, err := p.call(ctx, "chat.postEphemeral", map[string]any{"channel": channel, "user": user, "text": text})
	return err
}

func (p *Provider) call(ctx context.Context, method string, payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL("/api/"+method), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.BotToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("slack %s request failed", method)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", &integrations.HTTPError{Provider: "slack", Operation: method, Status: resp.StatusCode, Message: "request failed", RetryAfter: integrations.ParseRetryAfter(resp.Header.Get("Retry-After"))}
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var result struct {
		OK        bool   `json:"ok"`
		Error     string `json:"error"`
		TS        string `json:"ts"`
		MessageTS string `json:"message_ts"`
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if !result.OK {
		status := 400
		switch result.Error {
		case "ratelimited":
			status = 429
		case "internal_error", "service_unavailable", "fatal_error":
			status = 503
		}
		return "", &integrations.HTTPError{Provider: "slack", Operation: method, Status: status, Message: result.Error}
	}
	if result.TS == "" {
		result.TS = result.MessageTS
	}
	return result.TS, nil
}
