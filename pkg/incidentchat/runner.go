// Package incidentchat delivers durable channel events and chat action results.
package incidentchat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/incidentack"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/aegis/aegis/pkg/integrations/express"
	"github.com/aegis/aegis/pkg/integrations/resolve"
	"github.com/aegis/aegis/pkg/integrations/slack"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	incidentack.Store
	ClaimChannelDelivery(context.Context) (db.ChannelDelivery, error)
	ClaimChatAckAction(context.Context) (db.ChatAckRequest, error)
	ClaimChatAckFeedback(context.Context) (db.ChatAckRequest, error)
	ReadyChatAckFeedback(context.Context, uuid.UUID) error
	SaveChatAckOutcome(context.Context, uuid.UUID, []byte) error
	ProcessChatAck(context.Context, db.ChatAckRequest) (json.RawMessage, error)
	FinishChannelDelivery(context.Context, uuid.UUID, string, string, string, time.Time) error
	FinishChatAck(context.Context, uuid.UUID, string, string, time.Time) error
	GetIntegration(context.Context, uuid.UUID) (db.Integration, error)
	GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error)
	GetTeam(context.Context, uuid.UUID) (db.Team, error)
	CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error)
	GetUserByID(context.Context, uuid.UUID) (db.User, error)
}

type Runner struct {
	store     Store
	publicURL string
	log       *slog.Logger
}

func New(store Store, publicURL string) *Runner {
	return &Runner{store: store, publicURL: publicURL, log: slog.Default()}
}

// Independent loops keep slow chat sends from holding up callback processing
// or the existing incident worker. Leases recover work after a process restart.
func (r *Runner) Run(ctx context.Context) {
	go r.loop(ctx, r.DeliverChannel)
	go r.loop(ctx, r.ProcessAck)
	r.loop(ctx, r.DeliverFeedback)
}
func (r *Runner) loop(ctx context.Context, fn func(context.Context) error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			taskCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			err := fn(taskCtx)
			cancel()
			if err != nil && !errors.Is(err, pgx.ErrNoRows) && !errors.Is(err, context.Canceled) {
				r.log.Error("incident chat processing failed", "error", err)
			}
		}
	}
}

func permanent(message string) error {
	return &integrations.HTTPError{Provider: "incident chat", Operation: "configuration", Status: 400, Message: message}
}

func (r *Runner) integration(ctx context.Context, id uuid.UUID) (db.Integration, error) {
	row, err := r.store.GetIntegration(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return row, permanent("integration removed")
	}
	if err != nil {
		return row, err
	}
	if !row.Enabled {
		return row, permanent("integration disabled")
	}
	return row, nil
}

func RetryDisposition(attempt int, err error) (string, time.Time) {
	delays := []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 120 * time.Second}
	if attempt >= 5 || !integrations.RetryableError(err) {
		return "failed", time.Now()
	}
	if attempt < 1 {
		attempt = 1
	}
	delay := delays[attempt-1]
	var httpErr *integrations.HTTPError
	if errors.As(err, &httpErr) && httpErr.RetryAfter > delay {
		delay = httpErr.RetryAfter
	}
	return "pending", time.Now().Add(delay)
}

func (r *Runner) DeliverChannel(ctx context.Context) error {
	d, err := r.store.ClaimChannelDelivery(ctx)
	if err != nil {
		return err
	}
	var ref string
	var sendErr error
	if d.Attempts > 5 {
		sendErr = permanent("delivery attempt limit exceeded after lease expiration")
	} else {
		ref, sendErr = r.sendChannel(ctx, d)
	}
	status, next, message := "sent", time.Now(), ""
	if sendErr != nil {
		status, next = RetryDisposition(d.Attempts, sendErr)
		message = sendErr.Error()
		r.log.Error("incident channel delivery failed", "incident_id", d.Snapshot.Incident.ID, "event_id", d.EventID, "provider", d.Provider, "destination", d.Destination, "attempt", d.Attempts, "error", sendErr)
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	return r.store.FinishChannelDelivery(finishCtx, d.ID, status, ref, message, next)
}

func (r *Runner) sendChannel(ctx context.Context, d db.ChannelDelivery) (string, error) {
	cfg, err := r.integration(ctx, d.IntegrationID)
	if err != nil {
		return "", err
	}
	current, err := r.store.GetIncidentByID(ctx, d.Snapshot.Incident.ID)
	if err != nil {
		return "", err
	}
	i := d.Snapshot.Incident
	post := integrations.IncidentChannelPost{
		Incident: integrations.IncidentRef{ID: i.ID, TeamID: i.TeamID, AssigneeID: i.AssigneeID, Status: i.Status, Severity: i.Severity, Title: i.Title, URL: integrations.IncidentURL(r.publicURL, i.ID.String())},
		Kind:     d.Kind, TeamName: d.Snapshot.TeamName, ActorName: d.Snapshot.ActorName, Locale: d.Snapshot.Locale, SlackUserGroupID: d.Snapshot.SlackUserGroupID, OccurredAt: d.OccurredAt,
		Actionable: (d.Kind == "created" || d.Kind == "escalated") && current.Status == "open",
	}
	var provider integrations.IncidentChannelProvider
	switch d.Provider {
	case "slack":
		team, err := r.store.GetTeam(ctx, i.TeamID)
		if err != nil {
			return "", err
		}
		slot, err := r.store.GetWorkspaceIntegration(ctx, team.WorkspaceID, "slack")
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && !slot.Enabled) {
			return "", permanent("slack workspace slot unavailable")
		}
		if err != nil {
			return "", err
		}
		// Never send a queued event using credentials from a different slot mode.
		if slot.Mode == nil || (*slot.Mode == "custom" && slot.ID != cfg.ID) || (*slot.Mode == "inherit" && cfg.WorkspaceID != nil) {
			return "", permanent("slack workspace credentials changed")
		}
		config := cfg.Config
		if *slot.Mode == "inherit" && len(slot.Config) > 0 {
			config, err = resolve.MergeConfig(config, slot.Config)
			if err != nil {
				return "", permanent("invalid slack workspace configuration")
			}
		}
		provider, err = slack.NewFromJSON(config, r.publicURL)
		if err != nil {
			return "", permanent(err.Error())
		}
	case "express":
		provider, err = express.NewFromJSON(cfg.Config)
		if err != nil {
			return "", permanent(err.Error())
		}
		if post.Actionable {
			users, err := r.store.CurrentOnCallUsers(ctx, i.TeamID, time.Now().UTC())
			if err != nil {
				return "", err
			}
			for _, u := range users {
				full, err := r.store.GetUserByID(ctx, u.UserID)
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				if err != nil {
					return "", err
				}
				post.OnCall = append(post.OnCall, integrations.OnCallPerson{DisplayName: full.DisplayName, ExpressUserHuid: db.ExpressHuidString(full)})
			}
		}
	default:
		return "", permanent("unknown provider")
	}
	return provider.SendIncidentEvent(ctx, post, d.Destination)
}

// DeliverAck performs one action and feedback pass for deterministic checks.
func (r *Runner) DeliverAck(ctx context.Context) error {
	if err := r.ProcessAck(ctx); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return r.DeliverFeedback(ctx)
}

func (r *Runner) ProcessAck(ctx context.Context) error {
	req, err := r.store.ClaimChatAckAction(ctx)
	if err != nil {
		return err
	}
	var processErr error
	if req.Attempts > 5 {
		processErr = permanent("processing attempt limit exceeded after lease expiration")
	} else {
		_, processErr = r.store.ProcessChatAck(ctx, req)
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if processErr == nil {
		return r.store.ReadyChatAckFeedback(finishCtx, req.ID)
	}
	status, next := RetryDisposition(req.Attempts, processErr)
	r.log.Error("incident acknowledgement processing failed", "incident_id", req.IncidentID, "provider", req.Provider, "request_id", req.ID, "error", processErr)
	if status == "failed" {
		// An uncertain commit must never be reported as a confirmed success.
		// SaveChatAckOutcome won't overwrite an outcome if commit actually succeeded.
		locale := req.Locale
		var user db.User
		var lookupErr error
		if req.Provider == "slack" {
			user, lookupErr = r.store.GetUserBySlackID(finishCtx, req.UserIdentity)
		} else {
			huid, parseErr := db.ParseExpressHuid(req.UserIdentity)
			lookupErr = parseErr
			if parseErr == nil {
				user, lookupErr = r.store.GetUserByExpressHuid(finishCtx, huid)
			}
		}
		if lookupErr == nil && user.Locale != "" {
			locale = user.Locale
		}
		raw, _ := json.Marshal(incidentack.Result{Code: "not_confirmed", Locale: locale})
		if err = r.store.SaveChatAckOutcome(finishCtx, req.ID, raw); err == nil {
			return r.store.ReadyChatAckFeedback(finishCtx, req.ID)
		}
	}
	return r.store.FinishChatAck(finishCtx, req.ID, status, processErr.Error(), next)
}

func (r *Runner) DeliverFeedback(ctx context.Context) error {
	req, err := r.store.ClaimChatAckFeedback(ctx)
	if err != nil {
		return err
	}
	var outcome incidentack.Result
	if req.Attempts > 5 {
		err = permanent("feedback attempt limit exceeded after lease expiration")
	} else {
		err = json.Unmarshal(req.Outcome, &outcome)
	}
	if err == nil {
		err = r.sendFeedback(ctx, req, outcome)
	}
	status, next, message := "sent", time.Now(), ""
	if err != nil {
		status, next = RetryDisposition(req.Attempts, err)
		message = err.Error()
		r.log.Error("incident acknowledgement feedback failed", "incident_id", req.IncidentID, "provider", req.Provider, "request_id", req.ID, "error", err)
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	return r.store.FinishChatAck(finishCtx, req.ID, status, message, next)
}

func (r *Runner) sendFeedback(ctx context.Context, req db.ChatAckRequest, result incidentack.Result) error {
	cfg, err := r.integration(ctx, req.IntegrationID)
	if err != nil {
		return err
	}
	body := i18n.T(result.Locale, "incident_chat.feedback_"+result.Code, map[string]string{"id": req.IncidentID.String()})
	switch req.Provider {
	case "slack":
		workspaceID := req.WorkspaceID
		if workspaceID == nil {
			incident, err := r.store.GetIncidentByID(ctx, req.IncidentID)
			if err != nil {
				return err
			}
			team, err := r.store.GetTeam(ctx, incident.TeamID)
			if err != nil {
				return err
			}
			workspaceID = &team.WorkspaceID
		}
		slot, err := r.store.GetWorkspaceIntegration(ctx, *workspaceID, "slack")
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && !slot.Enabled) {
			return permanent("slack workspace slot unavailable")
		}
		if err != nil {
			return err
		}
		if slot.Mode == nil || (*slot.Mode == "custom" && slot.ID != cfg.ID) || (*slot.Mode == "inherit" && cfg.WorkspaceID != nil) {
			return permanent("slack workspace credentials changed")
		}
		config := cfg.Config
		if *slot.Mode == "inherit" && len(slot.Config) > 0 {
			config, err = resolve.MergeConfig(config, slot.Config)
			if err != nil {
				return permanent("invalid slack workspace configuration")
			}
		}
		p, err := slack.NewFromJSON(config, r.publicURL)
		if err != nil {
			return permanent(err.Error())
		}
		return p.SendAckFeedback(ctx, req.ResponseURL, req.ChatID, req.UserIdentity, body)
	case "express":
		p, err := express.NewFromJSON(cfg.Config)
		if err != nil {
			return permanent(err.Error())
		}
		return p.SendAckFeedback(ctx, req.ChatID, req.UserIdentity, body)
	}
	return fmt.Errorf("unsupported chat provider")
}
