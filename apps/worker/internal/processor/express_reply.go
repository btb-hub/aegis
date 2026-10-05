package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aegis/aegis/pkg/db"
	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/google/uuid"
)

type ExpressReplyStore interface {
	GetIntegration(context.Context, uuid.UUID) (db.Integration, error)
	HasExpressOutbound(context.Context, uuid.UUID, string) (bool, error)
	RecordExpressOutbound(context.Context, uuid.UUID, string, string) error
}
type ExpressReplyProcessor struct{ store ExpressReplyStore }

func NewExpressReplyProcessor(store ExpressReplyStore) *ExpressReplyProcessor {
	return &ExpressReplyProcessor{store: store}
}
func (p *ExpressReplyProcessor) Handle(ctx context.Context, job Job) error {
	var payload struct {
		IntegrationID string `json:"integration_id"`
		ChatID        string `json:"chat_id"`
		Body          string `json:"body"`
		Locale        string `json:"locale"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return err
	}
	id, err := uuid.Parse(payload.IntegrationID)
	if err != nil {
		return err
	}
	connector, err := p.store.GetIntegration(ctx, id)
	if err != nil {
		return err
	}
	if !connector.Enabled {
		return fmt.Errorf("eXpress bot is disabled")
	}
	provider, err := intexpress.NewFromJSON(connector.Config)
	if err != nil {
		return err
	}
	// BotX limits each text message to 4096 characters. Split at line boundaries.
	for index, part := range splitChatText(payload.Body, 4000) {
		key := fmt.Sprintf("reply:%s:%d", job.ID, index)
		sent, err := p.store.HasExpressOutbound(ctx, id, key)
		if err != nil {
			return err
		}
		if sent {
			continue
		}
		ref, err := provider.SendMessage(ctx, payload.ChatID, part, payload.Locale)
		if err != nil {
			return err
		}
		if err := p.store.RecordExpressOutbound(ctx, id, key, ref); err != nil {
			return err
		}
	}
	return nil
}
func splitChatText(body string, limit int) []string {
	runes := []rune(body)
	var parts []string
	for len(runes) > limit {
		cut := limit
		for i := limit - 1; i > limit/2; i-- {
			if runes[i] == '\n' {
				cut = i + 1
				break
			}
		}
		parts = append(parts, string(runes[:cut]))
		runes = runes[cut:]
	}
	if len(runes) > 0 {
		parts = append(parts, string(runes))
	}
	return parts
}
