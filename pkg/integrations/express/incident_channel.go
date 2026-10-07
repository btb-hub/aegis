package express

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"strings"
)

func (p *Provider) SendIncidentEvent(ctx context.Context, event integrations.IncidentChannelPost, chat string) (string, error) {
	body := integrations.IncidentEventBody(event)
	mentions := []map[string]any{}
	if event.Actionable {
		for _, person := range event.OnCall {
			if person.ExpressUserHuid == nil || strings.TrimSpace(*person.ExpressUserHuid) == "" {
				continue
			}
			id := uuid.New().String()
			body = "@{mention:" + id + "} " + body
			mentions = append(mentions, map[string]any{"mention_type": "user", "mention_id": id, "mention_data": map[string]string{"user_huid": *person.ExpressUserHuid}})
		}
	}
	notification := map[string]any{"status": "ok", "body": body, "mentions": mentions}
	if event.Actionable {
		notification["bubble"] = [][]map[string]any{{{"command": ackCommand, "label": i18n.T(event.Locale, "page.acknowledge_button", nil), "data": map[string]string{"incident_id": event.Incident.ID.String()}, "opts": map[string]any{"silent": true}}}}
	}
	return p.sendSyncNotification(ctx, map[string]any{"group_chat_id": chat, "notification": notification})
}

func (p *Provider) SendAckFeedback(ctx context.Context, chat, user, body string) error {
	if strings.TrimSpace(chat) == "" {
		token, err := p.ensureToken(ctx)
		if err != nil {
			return err
		}
		chat, err = p.personalChat(ctx, user, token)
		if err != nil {
			return err
		}
	}
	_, err := p.sendSyncNotification(ctx, map[string]any{"group_chat_id": chat, "recipients": []string{user}, "notification": map[string]string{"status": "ok", "body": body}})
	return err
}

// New channel and feedback deliveries use synchronous sends so "sent" means
// BotX delivered the message. Existing DM paging keeps its original protocol.
func (p *Provider) sendSyncNotification(ctx context.Context, payload any) (string, error) {
	token, err := p.ensureToken(ctx)
	if err != nil {
		return "", err
	}
	raw, err := p.postJSON(ctx, "/api/v4/botx/notifications/direct/sync", payload, token)
	if err != nil {
		p.mu.Lock()
		p.token = ""
		p.mu.Unlock()
		return "", err
	}
	var result struct {
		Status string `json:"status"`
		Result struct {
			SyncID string `json:"sync_id"`
		} `json:"result"`
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.Status != "ok" || result.Result.SyncID == "" {
		return "", fmt.Errorf("express notification response missing sync_id")
	}
	return result.Result.SyncID, nil
}
