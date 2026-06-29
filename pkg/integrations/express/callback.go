package express

import (
	"encoding/json"
	"fmt"
	"strings"
)

type CommandEvent struct {
	SyncID  string `json:"sync_id"`
	BotID   string `json:"bot_id"`
	Command struct {
		Body        string         `json:"body"`
		CommandType string         `json:"command_type"`
		Data        map[string]any `json:"data"`
	} `json:"command"`
	From struct {
		UserHuid string `json:"user_huid"`
	} `json:"from"`
}

func ParseCommandEvent(body []byte) (CommandEvent, error) {
	var event CommandEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return CommandEvent{}, err
	}
	if strings.TrimSpace(event.From.UserHuid) == "" {
		return CommandEvent{}, fmt.Errorf("missing user_huid")
	}
	return event, nil
}

func ParseAckCommand(event CommandEvent) (incidentID, userHuid string, err error) {
	userHuid = event.From.UserHuid
	if id, ok := event.Command.Data["incident_id"].(string); ok && strings.TrimSpace(id) != "" {
		return id, userHuid, nil
	}
	body := strings.TrimSpace(event.Command.Body)
	if strings.HasPrefix(body, "/ack_incident") {
		parts := strings.Fields(body)
		if len(parts) >= 2 {
			return parts[1], userHuid, nil
		}
	}
	return "", "", fmt.Errorf("ack command not found")
}

func ParseLinkCommand(event CommandEvent) (code, userHuid string, err error) {
	userHuid = event.From.UserHuid
	body := strings.TrimSpace(event.Command.Body)
	if !strings.HasPrefix(body, "/link") {
		return "", "", fmt.Errorf("not a link command")
	}
	parts := strings.Fields(body)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("link code missing")
	}
	return parts[1], userHuid, nil
}
