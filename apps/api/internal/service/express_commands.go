package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/integrations"
	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/aegis/aegis/pkg/rbac"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ExpressCommandStore interface {
	RunExpressCommand(context.Context, uuid.UUID, string, func(db.ExpressCommandRepository) (string, error)) error
	HandleExpressNotificationResult(context.Context, uuid.UUID, string, string, string) error
}
type ExpressCommandService struct{ store ExpressCommandStore }

func NewExpressCommandService(store ExpressCommandStore) *ExpressCommandService {
	return &ExpressCommandService{store: store}
}
func (s *ExpressCommandService) Process(ctx context.Context, connector db.Integration, event intexpress.CommandEvent) error {
	if event.SyncID == "" {
		event.SyncID = uuid.New().String()
		event.Command.Body = "/invalid"
		event.Command.Data = nil
	}
	return s.store.RunExpressCommand(ctx, connector.ID, event.SyncID, func(repo db.ExpressCommandRepository) (string, error) {
		if event.Command.CommandType == "system" || strings.HasPrefix(event.Command.Body, "system:") {
			return "system", nil
		}
		locale := event.From.Locale
		if locale != "ru" {
			locale = "en"
		}
		outcome, body, err := processExpressCommand(ctx, repo, event, locale)
		if err != nil {
			return "", err
		}
		if event.From.ChatID != "" {
			if err := repo.EnqueueExpressReply(ctx, connector.ID, event.SyncID, event.From.ChatID, body, locale); err != nil {
				return "", err
			}
		}
		return outcome, nil
	})
}
func (s *ExpressCommandService) NotificationResult(ctx context.Context, connector db.Integration, body []byte) error {
	var result struct {
		SyncID string `json:"sync_id"`
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.SyncID == "" || result.Status == "" {
		return fmt.Errorf("invalid notification result")
	}
	return s.store.HandleExpressNotificationResult(ctx, connector.ID, result.SyncID, result.Status, result.Reason)
}
func processExpressCommand(ctx context.Context, repo db.ExpressCommandRepository, event intexpress.CommandEvent, locale string) (string, string, error) {
	reply := func(key string) (string, string, error) { return key, i18n.T(locale, "bot."+key, nil), nil }
	parts := strings.Fields(event.Command.Body)
	command := ""
	if len(parts) > 0 {
		command = parts[0]
	}
	switch command {
	case "/link":
		code, huid, err := intexpress.ParseLinkCommand(event)
		if err != nil {
			return reply("link_invalid")
		}
		id, err := uuid.Parse(huid)
		if err != nil {
			return reply("link_invalid")
		}
		_, err = repo.RedeemExpressLinkCode(ctx, code, id)
		if errors.Is(err, db.ErrLinkCodeInvalid) {
			return reply("link_invalid")
		}
		if err != nil {
			return "", "", err
		}
		return reply("linked")
	case "/oncall":
		teams, err := repo.ListTeams(ctx)
		if err != nil {
			return "", "", err
		}
		var lines []string
		now := time.Now().UTC()
		for _, team := range teams {
			users, err := repo.CurrentOnCallUsers(ctx, team.ID, now)
			if err != nil {
				return "", "", err
			}
			var names []string
			for _, user := range users {
				names = append(names, integrations.OnCallName(integrations.OnCallPerson{DisplayName: user.DisplayName, StartAt: user.StartAt, EndAt: user.EndAt}, locale))
			}
			if len(names) == 0 {
				lines = append(lines, i18n.T(locale, "oncall.announce_empty", map[string]string{"team": team.Name}))
			} else {
				lines = append(lines, i18n.T(locale, "oncall.announce", map[string]string{"team": team.Name, "people": strings.Join(names, ", ")}))
			}
		}
		if len(lines) == 0 {
			return reply("oncall_empty")
		}
		return "oncall", strings.Join(lines, "\n"), nil
	case "/ack_incident":
		incidentRaw, huid, err := intexpress.ParseAckCommand(event)
		if err != nil {
			return reply("incident_missing")
		}
		incidentID, err := uuid.Parse(incidentRaw)
		if err != nil {
			return reply("incident_missing")
		}
		userID, err := uuid.Parse(huid)
		if err != nil {
			return reply("unlinked")
		}
		user, err := repo.GetUserByExpressHuid(ctx, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return reply("unlinked")
		}
		if err != nil {
			return "", "", err
		}
		if !rbac.CanMutate(rbac.Role(user.Role)) {
			return reply("not_allowed")
		}
		incident, err := repo.GetIncidentByID(ctx, incidentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return reply("incident_missing")
		}
		if err != nil {
			return "", "", err
		}
		if incident.Status == "acknowledged" {
			return reply("already_acknowledged")
		}
		if incident.Status == "resolved" {
			return reply("incident_resolved")
		}
		if _, err := repo.AcknowledgeIncident(ctx, incidentID, user.ID); errors.Is(err, pgx.ErrNoRows) {
			return reply("already_acknowledged")
		} else if err != nil {
			return "", "", err
		}
		if err := repo.CancelEscalationJobs(ctx, incidentID); err != nil {
			return "", "", err
		}
		return reply("acknowledged")
	default:
		// Some clients put the incident ID in button data without a body.
		if _, ok := event.Command.Data["incident_id"]; ok {
			event.Command.Body = "/ack_incident"
			return processExpressCommand(ctx, repo, event, locale)
		}
		return reply("unsupported")
	}
}
