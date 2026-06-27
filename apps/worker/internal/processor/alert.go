package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	intjira "github.com/aegis/aegis/pkg/integrations/jira"
	intslack "github.com/aegis/aegis/pkg/integrations/slack"
	"github.com/aegis/aegis/pkg/routing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AlertStore interface {
	GetAlertByID(ctx context.Context, id uuid.UUID) (db.Alert, error)
	GetIncidentForAlert(ctx context.Context, alertID uuid.UUID) (db.Incident, error)
	FindOpenIncidentByFingerprint(ctx context.Context, fingerprint string, since time.Time) (db.Incident, error)
	CreateIncidentWithAlert(ctx context.Context, input db.CreateIncidentInput) (db.Incident, error)
	LinkAlertToIncident(ctx context.Context, incidentID, alertID uuid.UUID) error
	ListRoutingRules(ctx context.Context) ([]db.RoutingRule, error)
	CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error)
	UpdateIncidentJiraKey(ctx context.Context, incidentID uuid.UUID, issueKey string) error
	AppendTimelineEvent(ctx context.Context, incidentID uuid.UUID, kind string, actorID *uuid.UUID, payload []byte) error
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
	CreateNotification(ctx context.Context, incidentID, integrationID uuid.UUID, status, externalRef string) (db.Notification, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	EnqueueEscalation(ctx context.Context, incidentID uuid.UUID, runAt time.Time) error
	ListEnabledIntegrations(ctx context.Context) ([]integrations.IntegrationRow, error)
}

type AlertProcessor struct {
	log               *slog.Logger
	store             AlertStore
	dedupWindow       time.Duration
	escalationTimeout time.Duration
	publicURL         string
}

func NewAlertProcessor(log *slog.Logger, store AlertStore, dedupWindow, escalationTimeout time.Duration, publicURL string) *AlertProcessor {
	if log == nil {
		log = slog.Default()
	}
	return &AlertProcessor{
		log:               log,
		store:             store,
		dedupWindow:       dedupWindow,
		escalationTimeout: escalationTimeout,
		publicURL:         publicURL,
	}
}

func (p *AlertProcessor) Handle(ctx context.Context, job Job) error {
	var payload map[string]any
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	alertIDRaw, _ := payload["alert_id"].(string)
	alertID, err := uuid.Parse(alertIDRaw)
	if err != nil {
		return fmt.Errorf("invalid alert_id: %w", err)
	}

	if _, err := p.store.GetIncidentForAlert(ctx, alertID); err == nil {
		return nil
	} else if err != pgx.ErrNoRows {
		return err
	}

	alert, err := p.store.GetAlertByID(ctx, alertID)
	if err != nil {
		return err
	}
	if alert.Status != "firing" {
		return nil
	}

	labels, err := decodeAlertLabels(alert.Labels)
	if err != nil {
		return err
	}

	teamID, err := p.matchTeam(ctx, labels)
	if err != nil {
		return err
	}

	since := time.Now().UTC().Add(-p.dedupWindow)
	existing, err := p.store.FindOpenIncidentByFingerprint(ctx, alert.Fingerprint, since)
	if err == nil {
		if err := p.store.LinkAlertToIncident(ctx, existing.ID, alertID); err != nil {
			return err
		}
		p.log.Info("linked alert to existing incident", "alert_id", alertID, "incident_id", existing.ID)
		return nil
	}
	if err != pgx.ErrNoRows {
		return err
	}

	var assigneeID *uuid.UUID
	onCall, err := p.store.CurrentOnCallUsers(ctx, teamID, time.Now().UTC())
	if err != nil {
		return err
	}
	if len(onCall) > 0 {
		id := onCall[0].UserID
		assigneeID = &id
	}

	incident, err := p.store.CreateIncidentWithAlert(ctx, db.CreateIncidentInput{
		TeamID:      teamID,
		AssigneeID:  assigneeID,
		Severity:    alert.Severity,
		Title:       alert.Title,
		Fingerprint: alert.Fingerprint,
		AlertID:     alertID,
	})
	if err != nil {
		return err
	}

	if err := p.store.EnqueueEscalation(ctx, incident.ID, time.Now().UTC().Add(p.escalationTimeout)); err != nil {
		return err
	}

	return p.notifyIntegrations(ctx, incident, assigneeID)
}

func (p *AlertProcessor) matchTeam(ctx context.Context, labels map[string]string) (uuid.UUID, error) {
	rules, err := p.store.ListRoutingRules(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	parsed := make([]routing.Rule, 0, len(rules))
	for _, rule := range rules {
		matchLabels, err := routing.ParseMatchLabels(rule.MatchLabels)
		if err != nil {
			continue
		}
		parsed = append(parsed, routing.Rule{
			TeamID:      rule.TeamID.String(),
			MatchLabels: matchLabels,
			Priority:    int(rule.Priority),
		})
	}
	teamRaw, ok := routing.MatchTeam(parsed, labels)
	if !ok {
		return uuid.Nil, fmt.Errorf("no routing rule matched alert labels")
	}
	return uuid.Parse(teamRaw)
}

func (p *AlertProcessor) notifyIntegrations(ctx context.Context, incident db.Incident, assigneeID *uuid.UUID) error {
	reg, err := p.loadRegistry(ctx)
	if err != nil {
		return err
	}
	ref := toIncidentRef(incident)

	integrations.ForEachTicket(reg, func(provider integrations.TicketProvider) error {
		key, err := provider.CreateTicket(ctx, ref)
		status := "sent"
		externalRef := key
		if err != nil {
			p.log.Error("ticket provider failed", "kind", provider.Kind(), "error", err)
			status = "failed"
			externalRef = ""
		} else if err := p.store.UpdateIncidentJiraKey(ctx, incident.ID, key); err != nil {
			p.log.Error("update jira key failed", "error", err)
		} else {
			payload, _ := json.Marshal(map[string]any{"jira_issue_key": key})
			_ = p.store.AppendTimelineEvent(ctx, incident.ID, "jira_linked", nil, payload)
		}
		integration, err := p.store.GetIntegrationByKind(ctx, provider.Kind())
		if err == nil {
			_, _ = p.store.CreateNotification(ctx, incident.ID, integration.ID, status, externalRef)
		}
		return nil
	})

	if assigneeID == nil {
		return nil
	}
	user, err := p.store.GetUserByID(ctx, *assigneeID)
	if err != nil {
		return nil
	}
	recipient := integrations.PageRecipient{
		UserID:      user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Locale:      user.Locale,
		SlackUserID: user.SlackUserID,
	}

	integrations.ForEachChat(reg, func(provider integrations.ChatProvider) error {
		ref, err := provider.SendPage(ctx, toIncidentRef(incident), recipient)
		status := "sent"
		externalRef := ref
		if err != nil {
			p.log.Error("chat provider failed", "kind", provider.Kind(), "error", err)
			status = "failed"
			externalRef = ""
		} else {
			payload, _ := json.Marshal(map[string]any{"provider": provider.Kind(), "ref": ref})
			_ = p.store.AppendTimelineEvent(ctx, incident.ID, "paged", nil, payload)
		}
		integration, err := p.store.GetIntegrationByKind(ctx, provider.Kind())
		if err == nil {
			_, _ = p.store.CreateNotification(ctx, incident.ID, integration.ID, status, externalRef)
		}
		return nil
	})
	return nil
}

func (p *AlertProcessor) loadRegistry(ctx context.Context) (*integrations.Registry, error) {
	rows, err := p.store.ListEnabledIntegrations(ctx)
	if err != nil {
		return nil, err
	}
	reg := integrations.NewRegistry()
	for _, row := range rows {
		switch row.Kind {
		case "jira":
			provider, err := intjira.NewFromJSON(row.Config)
			if err != nil {
				continue
			}
			reg.RegisterTicket(provider)
		case "slack":
			provider, err := intslack.NewFromJSON(row.Config, p.publicURL)
			if err != nil {
				continue
			}
			reg.RegisterChat(provider)
		}
	}
	return reg, nil
}

func decodeAlertLabels(raw []byte) (map[string]string, error) {
	var labels map[string]string
	if err := json.Unmarshal(raw, &labels); err != nil {
		return nil, err
	}
	if labels == nil {
		labels = map[string]string{}
	}
	return labels, nil
}

func toIncidentRef(incident db.Incident) integrations.IncidentRef {
	return integrations.IncidentRef{
		ID:           incident.ID,
		TeamID:       incident.TeamID,
		AssigneeID:   incident.AssigneeID,
		Status:       incident.Status,
		Severity:     incident.Severity,
		Title:        incident.Title,
		Fingerprint:  incident.Fingerprint,
		JiraIssueKey: incident.JiraIssueKey,
		CreatedAt:    incident.CreatedAt,
	}
}
