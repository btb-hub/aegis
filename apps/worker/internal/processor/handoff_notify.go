package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
)

type HandoffNotifyStore interface {
	GetIncidentByID(ctx context.Context, id uuid.UUID) (db.Incident, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
	GetTeamWorkspaceID(ctx context.Context, teamID uuid.UUID) (uuid.UUID, error)
	GetWorkspaceIntegration(ctx context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error)
	CreateNotification(ctx context.Context, incidentID, integrationID uuid.UUID, status, externalRef string) (db.Notification, error)
	AppendTimelineEvent(ctx context.Context, incidentID uuid.UUID, kind string, actorID *uuid.UUID, payload []byte) error
}

type HandoffNotifyProcessor struct {
	log       *slog.Logger
	store     HandoffNotifyStore
	publicURL string
}

func NewHandoffNotifyProcessor(log *slog.Logger, store HandoffNotifyStore, publicURL string) *HandoffNotifyProcessor {
	if log == nil {
		log = slog.Default()
	}
	return &HandoffNotifyProcessor{log: log, store: store, publicURL: publicURL}
}

func (p *HandoffNotifyProcessor) Handle(ctx context.Context, job Job) error {
	var payload struct {
		IncidentID string `json:"incident_id"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	incidentID, err := uuid.Parse(payload.IncidentID)
	if err != nil {
		return fmt.Errorf("invalid incident_id: %w", err)
	}

	incident, err := p.store.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return err
	}
	if incident.AssigneeID == nil {
		return nil
	}

	user, err := p.store.GetUserByID(ctx, *incident.AssigneeID)
	if err != nil {
		return nil
	}

	reg, notices, err := loadWorkspaceRegistry(ctx, p.store, incident.TeamID, p.publicURL)
	if err != nil {
		return err
	}
	for _, notice := range notices {
		if notice.Kind == "jira" && incident.JiraIssueKey == nil {
			continue
		}
		eventPayload, _ := json.Marshal(map[string]string{
			"kind":    notice.Kind,
			"reason":  notice.Reason,
			"message": skipMessage(notice),
		})
		_ = p.store.AppendTimelineEvent(ctx, incident.ID, "integration_skipped", nil, eventPayload)
	}
	ref := toIncidentRef(incident)

	if incident.JiraIssueKey != nil {
		integrations.ForEachTicket(reg.Registry, func(provider integrations.TicketProvider) error {
			updater, ok := provider.(integrations.AssigneeUpdater)
			if !ok {
				return nil
			}
			if err := updater.UpdateAssignee(ctx, *incident.JiraIssueKey, user.Email); err != nil {
				p.log.Error("assignee update failed", "kind", provider.Kind(), "error", err)
			}
			return nil
		})
	}

	recipient := integrations.PageRecipient{
		UserID:          user.ID,
		Email:           user.Email,
		DisplayName:     user.DisplayName,
		Locale:          user.Locale,
		SlackUserID:     user.SlackUserID,
		ExpressUserHuid: db.ExpressHuidString(user),
	}

	integrations.ForEachChat(reg.Registry, func(provider integrations.ChatProvider) error {
		externalRef, err := provider.SendPage(ctx, ref, recipient)
		status := "sent"
		if err != nil {
			p.log.Error("handoff page failed", "kind", provider.Kind(), "error", err)
			status = "failed"
			externalRef = ""
		} else {
			eventPayload, _ := json.Marshal(map[string]any{"provider": provider.Kind(), "ref": externalRef, "handoff": true})
			_ = p.store.AppendTimelineEvent(ctx, incident.ID, "paged", nil, eventPayload)
		}
		if integrationID, ok := reg.integrationID(provider.Kind()); ok {
			_, _ = p.store.CreateNotification(ctx, incident.ID, integrationID, status, externalRef)
		}
		return nil
	})

	return nil
}
