package processor

import (
	"context"
	"encoding/json"
	"fmt"
	runtimeconfig "github.com/aegis/aegis/pkg/config"
	"log/slog"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
)

type NotifyIncidentStore interface {
	GetIncidentByID(ctx context.Context, id uuid.UUID) (db.Incident, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	UpdateIncidentJiraKey(ctx context.Context, incidentID uuid.UUID, issueKey string) error
	AppendTimelineEvent(ctx context.Context, incidentID uuid.UUID, kind string, actorID *uuid.UUID, payload []byte) error
	CreateNotification(ctx context.Context, incidentID, integrationID uuid.UUID, status, externalRef string) (db.Notification, error)
	HasNotification(ctx context.Context, incidentID, integrationID uuid.UUID) (bool, error)
	GetTeamWorkspaceID(ctx context.Context, teamID uuid.UUID) (uuid.UUID, error)
	GetWorkspaceIntegration(ctx context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error)
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
}

type NotifyIncidentProcessor struct {
	log       *slog.Logger
	store     NotifyIncidentStore
	publicURL string
}

func NewNotifyIncidentProcessor(log *slog.Logger, store NotifyIncidentStore, publicURL string) *NotifyIncidentProcessor {
	if log == nil {
		log = slog.Default()
	}
	return &NotifyIncidentProcessor{log: log, store: store, publicURL: publicURL}
}

func (p *NotifyIncidentProcessor) Handle(ctx context.Context, job Job) error {
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
	if incident.Status == "resolved" {
		return nil
	}
	return notifyIncidentIntegrations(ctx, p.log, p.store, incident, runtimeconfig.PublicURL(ctx, p.publicURL))
}

// JIRA-CREATE-ASSIGN: After ticket creation, attempt assignment to the incident assignee.
// Re-read after linking and after each assignment until the assignee is stable, so a
// handoff during creation or assignment is not overwritten by a stale assignee.
// Missing users and assignment errors do not undo the link or sent notification; retries
// must not create another ticket. Evidence: TestNotifyIncidentAssignsCreatedTicket
// and TestNotifyIncidentReconcilesHandoffDuringCreationOrAssignment in notify_incident_test.go.
func notifyIncidentIntegrations(
	ctx context.Context,
	log *slog.Logger,
	store NotifyIncidentStore,
	incident db.Incident,
	publicURL string,
) error {
	reg, notices, err := loadWorkspaceRegistry(ctx, store, incident.TeamID, publicURL)
	if err != nil {
		return err
	}
	for _, notice := range notices {
		payload, _ := json.Marshal(map[string]string{
			"kind":    notice.Kind,
			"reason":  notice.Reason,
			"message": skipMessage(notice),
		})
		_ = store.AppendTimelineEvent(ctx, incident.ID, "integration_skipped", nil, payload)
	}
	ref := toIncidentRef(incident)

	integrations.ForEachTicket(reg.Registry, func(provider integrations.TicketProvider) error {
		if incident.JiraIssueKey != nil {
			return nil
		}
		integrationID, ok := reg.integrationID(provider.Kind())
		if ok {
			sent, err := store.HasNotification(ctx, incident.ID, integrationID)
			if err != nil {
				return err
			}
			if sent {
				return nil
			}
		}
		key, err := provider.CreateTicket(ctx, ref)
		status := "sent"
		externalRef := key
		linked := false
		if err != nil {
			if log != nil {
				log.Error("ticket provider failed", "kind", provider.Kind(), "error", err)
			}
			status = "failed"
			externalRef = ""
		} else if err := store.UpdateIncidentJiraKey(ctx, incident.ID, key); err != nil {
			if log != nil {
				log.Error("update jira key failed", "error", err)
			}
		} else {
			linked = true
			payload, _ := json.Marshal(map[string]any{"jira_issue_key": key})
			_ = store.AppendTimelineEvent(ctx, incident.ID, "jira_linked", nil, payload)
		}
		if linked {
			if updater, ok := provider.(integrations.AssigneeUpdater); ok {
				if assignErr := reconcileInitialAssignee(ctx, store, updater, incident.ID, key); assignErr != nil && log != nil {
					log.Error("ticket assignee update failed", "kind", provider.Kind(), "error", assignErr)
				}
			}
		}
		if integrationID, ok := reg.integrationID(provider.Kind()); ok {
			_, _ = store.CreateNotification(ctx, incident.ID, integrationID, status, externalRef)
		}
		return nil
	})

	if incident.AssigneeID == nil {
		return nil
	}
	user, err := store.GetUserByID(ctx, *incident.AssigneeID)
	if err != nil {
		return nil
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
		integrationID, ok := reg.integrationID(provider.Kind())
		if ok {
			sent, err := store.HasNotification(ctx, incident.ID, integrationID)
			if err != nil {
				return err
			}
			if sent {
				return nil
			}
		}
		ref, err := provider.SendPage(ctx, toIncidentRef(incident), recipient)
		status := "sent"
		externalRef := ref
		if err != nil {
			if log != nil {
				log.Error("chat provider failed", "kind", provider.Kind(), "error", err)
			}
			status = "failed"
			externalRef = ""
		} else {
			payload, _ := json.Marshal(map[string]any{"provider": provider.Kind(), "ref": ref})
			_ = store.AppendTimelineEvent(ctx, incident.ID, "paged", nil, payload)
		}
		if integrationID, ok := reg.integrationID(provider.Kind()); ok {
			_, _ = store.CreateNotification(ctx, incident.ID, integrationID, status, externalRef)
		}
		return nil
	})
	return nil
}

// reconcileInitialAssignee implements JIRA-CREATE-ASSIGN; see notifyIncidentIntegrations.
func reconcileInitialAssignee(ctx context.Context, store NotifyIncidentStore, updater integrations.AssigneeUpdater, incidentID uuid.UUID, key string) error {
	current, err := store.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return err
	}
	for current.AssigneeID != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
		assignedID := *current.AssigneeID
		user, err := store.GetUserByID(ctx, assignedID)
		if err != nil {
			return err
		}
		if err := updater.UpdateAssignee(ctx, key, user.Email); err != nil {
			return err
		}
		current, err = store.GetIncidentByID(ctx, incidentID)
		if err != nil {
			return err
		}
		if current.AssigneeID != nil && *current.AssigneeID == assignedID {
			return nil
		}
	}
	return nil
}
