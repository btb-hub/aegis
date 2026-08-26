package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrAlertNotFiring     = errors.New("alert is not firing")
	ErrAlertAlreadyLinked = errors.New("alert already linked to open incident")
)

type FingerprintTeamMismatchError struct {
	IncidentID uuid.UUID
}

func (e *FingerprintTeamMismatchError) Error() string {
	return fmt.Sprintf("fingerprint matches open incident %s on a different team", e.IncidentID)
}

type IncidentPostCreateJobs struct {
	EscalationRunAt time.Time
}

type ManualCreateFromAlertInput struct {
	AlertID                       uuid.UUID
	TeamID                        uuid.UUID
	AssigneeID                    *uuid.UUID
	ActorID                       *uuid.UUID
	DedupSince                    time.Time
	AllowCrossTeamFingerprintLink bool
	PostCreate                    *IncidentPostCreateJobs
}

type ManualCreateFromAlertResult struct {
	Incident Incident
	Created  bool
}

func (s *Store) GetOpenIncidentForAlert(ctx context.Context, alertID uuid.UUID) (Incident, error) {
	const q = `
SELECT i.id, i.team_id, i.assignee_id, i.status, i.severity, i.title, i.fingerprint,
       i.jira_issue_key, i.acknowledged_at, i.resolved_at, i.created_at
FROM incidents i
JOIN incident_alerts ia ON ia.incident_id = i.id
WHERE ia.alert_id = $1 AND i.status IN ('open', 'acknowledged')
ORDER BY ia.created_at DESC
LIMIT 1`
	var incident Incident
	err := s.pool.QueryRow(ctx, q, alertID).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	return incident, err
}

func (s *Store) EnqueueNotifyIncident(ctx context.Context, incidentID uuid.UUID) error {
	payload := []byte(`{"incident_id":"` + incidentID.String() + `"}`)
	_, err := s.EnqueueJob(ctx, "notify_incident", payload, time.Now().UTC())
	return err
}

func enqueueJobTx(ctx context.Context, tx pgx.Tx, kind string, payload []byte, runAt time.Time) error {
	_, err := tx.Exec(ctx, `
INSERT INTO jobs (kind, payload, status, run_at) VALUES ($1, $2, 'pending', $3)`, kind, payload, runAt)
	return err
}

func enqueuePostCreateJobsTx(ctx context.Context, tx pgx.Tx, incidentID uuid.UUID, jobs IncidentPostCreateJobs) error {
	escalationPayload := []byte(`{"incident_id":"` + incidentID.String() + `"}`)
	if err := enqueueJobTx(ctx, tx, "escalate_incident", escalationPayload, jobs.EscalationRunAt); err != nil {
		return err
	}
	notifyPayload := []byte(`{"incident_id":"` + incidentID.String() + `"}`)
	return enqueueJobTx(ctx, tx, "notify_incident", notifyPayload, time.Now().UTC())
}

func (s *Store) HasPendingJob(ctx context.Context, kind string, incidentID uuid.UUID) (bool, error) {
	const q = `
SELECT EXISTS (
  SELECT 1 FROM jobs
  WHERE kind = $1 AND status = 'pending' AND payload::text LIKE $2
)`
 needle := "%" + incidentID.String() + "%"
	var exists bool
	err := s.pool.QueryRow(ctx, q, kind, needle).Scan(&exists)
	return exists, err
}

func (s *Store) EnsureIncidentPostCreateJobs(ctx context.Context, incidentID uuid.UUID, escalationRunAt time.Time) error {
	hasEscalation, err := s.HasPendingJob(ctx, "escalate_incident", incidentID)
	if err != nil {
		return err
	}
	if !hasEscalation {
		if err := s.EnqueueEscalation(ctx, incidentID, escalationRunAt); err != nil {
			return err
		}
	}
	hasNotify, err := s.HasPendingJob(ctx, "notify_incident", incidentID)
	if err != nil {
		return err
	}
	if !hasNotify {
		return s.EnqueueNotifyIncident(ctx, incidentID)
	}
	return nil
}

func (s *Store) HasNotification(ctx context.Context, incidentID, integrationID uuid.UUID) (bool, error) {
	const q = `
SELECT EXISTS (
  SELECT 1 FROM notifications
  WHERE incident_id = $1 AND integration_id = $2 AND status = 'sent'
)`
	var exists bool
	err := s.pool.QueryRow(ctx, q, incidentID, integrationID).Scan(&exists)
	return exists, err
}

func (s *Store) ManualCreateFromAlert(ctx context.Context, input ManualCreateFromAlertInput) (ManualCreateFromAlertResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ManualCreateFromAlertResult{}, err
	}
	defer tx.Rollback(ctx)

	var alert Alert
	err = tx.QueryRow(ctx, `
SELECT id, fingerprint, status, severity, title, body, labels, search_tsv, raw_payload, received_at
FROM alerts
WHERE id = $1
FOR UPDATE`, input.AlertID).Scan(
		&alert.ID, &alert.Fingerprint, &alert.Status, &alert.Severity, &alert.Title,
		&alert.Body, &alert.Labels, &alert.SearchTsv, &alert.RawPayload, &alert.ReceivedAt,
	)
	if err != nil {
		return ManualCreateFromAlertResult{}, err
	}
	if alert.Status != "firing" {
		return ManualCreateFromAlertResult{}, ErrAlertNotFiring
	}

	var openLinked Incident
	err = tx.QueryRow(ctx, `
SELECT i.id, i.team_id, i.assignee_id, i.status, i.severity, i.title, i.fingerprint,
       i.jira_issue_key, i.acknowledged_at, i.resolved_at, i.created_at
FROM incidents i
JOIN incident_alerts ia ON ia.incident_id = i.id
WHERE ia.alert_id = $1 AND i.status IN ('open', 'acknowledged')
ORDER BY ia.created_at DESC
LIMIT 1`, input.AlertID).Scan(
		&openLinked.ID, &openLinked.TeamID, &openLinked.AssigneeID, &openLinked.Status, &openLinked.Severity,
		&openLinked.Title, &openLinked.Fingerprint, &openLinked.JiraIssueKey, &openLinked.AcknowledgedAt,
		&openLinked.ResolvedAt, &openLinked.CreatedAt,
	)
	if err == nil {
		return ManualCreateFromAlertResult{}, ErrAlertAlreadyLinked
	}
	if err != pgx.ErrNoRows {
		return ManualCreateFromAlertResult{}, err
	}

	var existing Incident
	err = tx.QueryRow(ctx, `
SELECT id, team_id, assignee_id, status, severity, title, fingerprint,
       jira_issue_key, acknowledged_at, resolved_at, created_at
FROM incidents
WHERE fingerprint = $1 AND status != 'resolved' AND created_at >= $2
ORDER BY created_at DESC
LIMIT 1
FOR UPDATE`, alert.Fingerprint, input.DedupSince).Scan(
		&existing.ID, &existing.TeamID, &existing.AssigneeID, &existing.Status, &existing.Severity,
		&existing.Title, &existing.Fingerprint, &existing.JiraIssueKey, &existing.AcknowledgedAt,
		&existing.ResolvedAt, &existing.CreatedAt,
	)
	if err == nil {
		if existing.TeamID != input.TeamID && !input.AllowCrossTeamFingerprintLink {
			return ManualCreateFromAlertResult{}, &FingerprintTeamMismatchError{IncidentID: existing.ID}
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO incident_alerts (incident_id, alert_id) VALUES ($1, $2)
ON CONFLICT (incident_id, alert_id) DO NOTHING`, existing.ID, input.AlertID); err != nil {
			return ManualCreateFromAlertResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ManualCreateFromAlertResult{}, err
		}
		return ManualCreateFromAlertResult{Incident: existing, Created: false}, nil
	}
	if err != pgx.ErrNoRows {
		return ManualCreateFromAlertResult{}, err
	}

	incident, err := createIncidentWithAlertTx(ctx, tx, CreateIncidentInput{
		TeamID:      input.TeamID,
		AssigneeID:  input.AssigneeID,
		Severity:    alert.Severity,
		Title:       alert.Title,
		Fingerprint: alert.Fingerprint,
		AlertID:     input.AlertID,
		ActorID:     input.ActorID,
	})
	if err != nil {
		return ManualCreateFromAlertResult{}, err
	}

	if input.PostCreate != nil {
		if err := enqueuePostCreateJobsTx(ctx, tx, incident.ID, *input.PostCreate); err != nil {
			return ManualCreateFromAlertResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ManualCreateFromAlertResult{}, err
	}
	return ManualCreateFromAlertResult{Incident: incident, Created: true}, nil
}
