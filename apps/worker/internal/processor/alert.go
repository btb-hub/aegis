package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/aegis/aegis/pkg/routing"
	"github.com/google/uuid"
)

type AlertStore interface {
	GetAlertByID(ctx context.Context, id uuid.UUID) (db.Alert, error)
	ManualCreateFromAlert(ctx context.Context, input db.ManualCreateFromAlertInput) (db.ManualCreateFromAlertResult, error)
	EnsureIncidentPostCreateJobs(ctx context.Context, incidentID uuid.UUID, escalationRunAt time.Time) error
	GetOpenIncidentForAlert(ctx context.Context, alertID uuid.UUID) (db.Incident, error)
	ListRoutingRules(ctx context.Context) ([]db.RoutingRule, error)
	CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error)
}

type AlertProcessor struct {
	log               *slog.Logger
	store             AlertStore
	dedupWindow       time.Duration
	escalationTimeout time.Duration
}

func NewAlertProcessor(log *slog.Logger, store AlertStore, dedupWindow, escalationTimeout time.Duration) *AlertProcessor {
	if log == nil {
		log = slog.Default()
	}
	return &AlertProcessor{
		log:               log,
		store:             store,
		dedupWindow:       dedupWindow,
		escalationTimeout: escalationTimeout,
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

	var assigneeID *uuid.UUID
	onCall, err := p.store.CurrentOnCallUsers(ctx, teamID, time.Now().UTC())
	if err != nil {
		return err
	}
	if len(onCall) > 0 {
		id := onCall[0].UserID
		assigneeID = &id
	}

	now := time.Now().UTC()
	result, err := p.store.ManualCreateFromAlert(ctx, db.ManualCreateFromAlertInput{
		AlertID:                       alertID,
		TeamID:                        teamID,
		AssigneeID:                    assigneeID,
		DedupSince:                    now.Add(-p.dedupWindow),
		AllowCrossTeamFingerprintLink: true,
		PostCreate: &db.IncidentPostCreateJobs{
			EscalationRunAt: now.Add(p.escalationTimeout),
		},
	})
	if errors.Is(err, db.ErrAlertAlreadyLinked) {
		incident, lookupErr := p.store.GetOpenIncidentForAlert(ctx, alertID)
		if lookupErr != nil {
			return lookupErr
		}
		return p.store.EnsureIncidentPostCreateJobs(ctx, incident.ID, now.Add(p.escalationTimeout))
	}
	if err != nil {
		return err
	}
	if !result.Created {
		p.log.Info("linked alert to existing incident", "alert_id", alertID, "incident_id", result.Incident.ID)
	}
	return nil
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
