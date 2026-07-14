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

type EscalateStore interface {
	GetIncidentByID(ctx context.Context, id uuid.UUID) (db.Incident, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	AppendTimelineEvent(ctx context.Context, incidentID uuid.UUID, kind string, actorID *uuid.UUID, payload []byte) error
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
	CreateNotification(ctx context.Context, incidentID, integrationID uuid.UUID, status, externalRef string) (db.Notification, error)
	GetTeamWorkspaceID(ctx context.Context, teamID uuid.UUID) (uuid.UUID, error)
	GetWorkspaceIntegration(ctx context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error)
}

type EscalateProcessor struct {
	log       *slog.Logger
	store     EscalateStore
	publicURL string
}

func NewEscalateProcessor(log *slog.Logger, store EscalateStore, publicURL string) *EscalateProcessor {
	if log == nil {
		log = slog.Default()
	}
	return &EscalateProcessor{log: log, store: store, publicURL: publicURL}
}

func (p *EscalateProcessor) Handle(ctx context.Context, job Job) error {
	var payload map[string]any
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	incidentIDRaw, _ := payload["incident_id"].(string)
	incidentID, err := uuid.Parse(incidentIDRaw)
	if err != nil {
		return fmt.Errorf("invalid incident_id: %w", err)
	}

	incident, err := p.store.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return err
	}
	if incident.Status != "open" {
		return nil
	}
	if incident.AssigneeID == nil {
		return nil
	}

	user, err := p.store.GetUserByID(ctx, *incident.AssigneeID)
	if err != nil {
		return err
	}

	reg, notices, err := loadWorkspaceRegistry(ctx, p.store, incident.TeamID, p.publicURL)
	if err != nil {
		return err
	}
	for _, notice := range notices {
		eventPayload, _ := json.Marshal(map[string]string{
			"kind":    notice.Kind,
			"reason":  notice.Reason,
			"message": skipMessage(notice),
		})
		_ = p.store.AppendTimelineEvent(ctx, incident.ID, "integration_skipped", nil, eventPayload)
	}

	ref := integrations.IncidentRef{
		ID:          incident.ID,
		TeamID:      incident.TeamID,
		AssigneeID:  incident.AssigneeID,
		Status:      incident.Status,
		Severity:    incident.Severity,
		Title:       incident.Title,
		Fingerprint: incident.Fingerprint,
		CreatedAt:   incident.CreatedAt,
	}
	recipient := integrations.PageRecipient{
		UserID:          user.ID,
		Email:           user.Email,
		DisplayName:     user.DisplayName,
		Locale:          user.Locale,
		SlackUserID:     user.SlackUserID,
		ExpressUserHuid: db.ExpressHuidString(user),
	}

	integrations.ForEachChat(reg, func(provider integrations.ChatProvider) error {
		messageRef, err := provider.SendPage(ctx, ref, recipient)
		status := "sent"
		externalRef := messageRef
		if err != nil {
			p.log.Error("escalation page failed", "kind", provider.Kind(), "error", err)
			status = "failed"
			externalRef = ""
		} else {
			eventPayload, _ := json.Marshal(map[string]any{"provider": provider.Kind(), "ref": messageRef, "escalation": true})
			_ = p.store.AppendTimelineEvent(ctx, incident.ID, "escalated", nil, eventPayload)
		}
		integration, err := p.store.GetIntegrationByKind(ctx, provider.Kind())
		if err == nil {
			_, _ = p.store.CreateNotification(ctx, incident.ID, integration.ID, status, externalRef)
		}
		return nil
	})
	return nil
}
