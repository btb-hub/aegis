package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) GetAlertByID(ctx context.Context, id uuid.UUID) (Alert, error) {
	const q = `
SELECT id, fingerprint, status, severity, title, body, labels, search_tsv, raw_payload, received_at
FROM alerts
WHERE id = $1`
	var alert Alert
	var body *string
	var searchTsv *string
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&alert.ID, &alert.Fingerprint, &alert.Status, &alert.Severity, &alert.Title,
		&body, &alert.Labels, &searchTsv, &alert.RawPayload, &alert.ReceivedAt,
	)
	alert.Body = body
	alert.SearchTsv = searchTsv
	return alert, err
}

func (s *Store) GetIncidentForAlert(ctx context.Context, alertID uuid.UUID) (Incident, error) {
	const q = `
SELECT i.id, i.team_id, i.assignee_id, i.status, i.severity, i.title, i.fingerprint,
       i.jira_issue_key, i.acknowledged_at, i.resolved_at, i.created_at
FROM incidents i
JOIN incident_alerts ia ON ia.incident_id = i.id
WHERE ia.alert_id = $1`
	var incident Incident
	err := s.pool.QueryRow(ctx, q, alertID).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	return incident, err
}

func (s *Store) FindOpenIncidentByFingerprint(ctx context.Context, fingerprint string, since time.Time) (Incident, error) {
	const q = `
SELECT id, team_id, assignee_id, status, severity, title, fingerprint,
       jira_issue_key, acknowledged_at, resolved_at, created_at
FROM incidents
WHERE fingerprint = $1 AND status != 'resolved' AND created_at >= $2
ORDER BY created_at DESC
LIMIT 1`
	var incident Incident
	err := s.pool.QueryRow(ctx, q, fingerprint, since).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	return incident, err
}

type CreateIncidentInput struct {
	TeamID      uuid.UUID
	AssigneeID  *uuid.UUID
	Severity    string
	Title       string
	Fingerprint string
	AlertID     uuid.UUID
	ActorID     *uuid.UUID
}

func (s *Store) CreateIncidentWithAlert(ctx context.Context, input CreateIncidentInput) (Incident, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Incident{}, err
	}
	defer tx.Rollback(ctx)

	const incidentQ = `
INSERT INTO incidents (team_id, assignee_id, severity, title, fingerprint)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, team_id, assignee_id, status, severity, title, fingerprint, jira_issue_key, acknowledged_at, resolved_at, created_at`
	var incident Incident
	err = tx.QueryRow(ctx, incidentQ, input.TeamID, input.AssigneeID, input.Severity, input.Title, input.Fingerprint).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	if err != nil {
		return Incident{}, err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO incident_alerts (incident_id, alert_id) VALUES ($1, $2)`, incident.ID, input.AlertID); err != nil {
		return Incident{}, err
	}

	payload, _ := json.Marshal(map[string]any{"alert_id": input.AlertID.String()})
	if err := insertTimelineEventTx(ctx, tx, incident.ID, "created", input.ActorID, payload); err != nil {
		return Incident{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Incident{}, err
	}
	return incident, nil
}

func (s *Store) LinkAlertToIncident(ctx context.Context, incidentID, alertID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO incident_alerts (incident_id, alert_id) VALUES ($1, $2)
ON CONFLICT (incident_id, alert_id) DO NOTHING`, incidentID, alertID)
	return err
}

func (s *Store) GetIncidentByID(ctx context.Context, id uuid.UUID) (Incident, error) {
	const q = `
SELECT id, team_id, assignee_id, status, severity, title, fingerprint,
       jira_issue_key, acknowledged_at, resolved_at, created_at
FROM incidents
WHERE id = $1`
	var incident Incident
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	return incident, err
}

func (s *Store) ListIncidents(ctx context.Context, status string) ([]Incident, error) {
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = s.pool.Query(ctx, `
SELECT id, team_id, assignee_id, status, severity, title, fingerprint,
       jira_issue_key, acknowledged_at, resolved_at, created_at
FROM incidents
ORDER BY created_at DESC`)
	} else {
		rows, err = s.pool.Query(ctx, `
SELECT id, team_id, assignee_id, status, severity, title, fingerprint,
       jira_issue_key, acknowledged_at, resolved_at, created_at
FROM incidents
WHERE status = $1
ORDER BY created_at DESC`, status)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var incident Incident
		if err := rows.Scan(
			&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
			&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
			&incident.ResolvedAt, &incident.CreatedAt,
		); err != nil {
			return nil, err
		}
		incidents = append(incidents, incident)
	}
	return incidents, rows.Err()
}

func (s *Store) UpdateIncidentJiraKey(ctx context.Context, incidentID uuid.UUID, issueKey string) error {
	_, err := s.pool.Exec(ctx, `UPDATE incidents SET jira_issue_key = $2 WHERE id = $1`, incidentID, issueKey)
	return err
}

func (s *Store) AcknowledgeIncident(ctx context.Context, incidentID, actorID uuid.UUID) (Incident, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Incident{}, err
	}
	defer tx.Rollback(ctx)

	const q = `
UPDATE incidents
SET status = 'acknowledged', acknowledged_at = now()
WHERE id = $1 AND status = 'open'
RETURNING id, team_id, assignee_id, status, severity, title, fingerprint, jira_issue_key, acknowledged_at, resolved_at, created_at`
	var incident Incident
	err = tx.QueryRow(ctx, q, incidentID).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	if err != nil {
		return Incident{}, err
	}

	payload, _ := json.Marshal(map[string]any{"actor_id": actorID.String()})
	if err := insertTimelineEventTx(ctx, tx, incidentID, "acknowledged", &actorID, payload); err != nil {
		return Incident{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Incident{}, err
	}
	return incident, nil
}

func (s *Store) ResolveIncident(ctx context.Context, incidentID, actorID uuid.UUID) (Incident, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Incident{}, err
	}
	defer tx.Rollback(ctx)

	const q = `
UPDATE incidents
SET status = 'resolved', resolved_at = now()
WHERE id = $1 AND status IN ('open', 'acknowledged')
RETURNING id, team_id, assignee_id, status, severity, title, fingerprint, jira_issue_key, acknowledged_at, resolved_at, created_at`
	var incident Incident
	err = tx.QueryRow(ctx, q, incidentID).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	if err != nil {
		return Incident{}, err
	}

	payload, _ := json.Marshal(map[string]any{"actor_id": actorID.String()})
	if err := insertTimelineEventTx(ctx, tx, incidentID, "resolved", &actorID, payload); err != nil {
		return Incident{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Incident{}, err
	}
	return incident, nil
}

func (s *Store) ListTimelineEvents(ctx context.Context, incidentID uuid.UUID) ([]TimelineEvent, error) {
	const q = `
SELECT id, incident_id, kind, actor_id, payload, created_at
FROM timeline_events
WHERE incident_id = $1
ORDER BY created_at ASC`
	rows, err := s.pool.Query(ctx, q, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var event TimelineEvent
		if err := rows.Scan(&event.ID, &event.IncidentID, &event.Kind, &event.ActorID, &event.Payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) AppendTimelineEvent(ctx context.Context, incidentID uuid.UUID, kind string, actorID *uuid.UUID, payload []byte) error {
	if payload == nil {
		payload = []byte(`{}`)
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO timeline_events (incident_id, kind, actor_id, payload)
VALUES ($1, $2, $3, $4)`, incidentID, kind, actorID, payload)
	return err
}

func insertTimelineEventTx(ctx context.Context, tx pgx.Tx, incidentID uuid.UUID, kind string, actorID *uuid.UUID, payload []byte) error {
	if payload == nil {
		payload = []byte(`{}`)
	}
	_, err := tx.Exec(ctx, `
INSERT INTO timeline_events (incident_id, kind, actor_id, payload)
VALUES ($1, $2, $3, $4)`, incidentID, kind, actorID, payload)
	return err
}

func (s *Store) CreateNotification(ctx context.Context, incidentID, integrationID uuid.UUID, status, externalRef string) (Notification, error) {
	const q = `
INSERT INTO notifications (incident_id, integration_id, status, external_ref, sent_at)
VALUES ($1, $2, $3, NULLIF($4, ''), CASE WHEN $3 = 'sent' THEN now() ELSE NULL END)
RETURNING id, incident_id, integration_id, status, external_ref, sent_at, created_at`
	var notification Notification
	var external *string
	if externalRef != "" {
		external = &externalRef
	}
	err := s.pool.QueryRow(ctx, q, incidentID, integrationID, status, externalRef).Scan(
		&notification.ID, &notification.IncidentID, &notification.IntegrationID, &notification.Status,
		&external, &notification.SentAt, &notification.CreatedAt,
	)
	notification.ExternalRef = external
	return notification, err
}

func (s *Store) CancelEscalationJobs(ctx context.Context, incidentID uuid.UUID) error {
	payload := `"incident_id":"` + incidentID.String() + `"`
	_, err := s.pool.Exec(ctx, `
UPDATE jobs
SET status = 'done', updated_at = now()
WHERE kind = 'escalate_incident' AND status = 'pending' AND payload::text LIKE $1`, "%"+payload+"%")
	return err
}

func (s *Store) EnqueueEscalation(ctx context.Context, incidentID uuid.UUID, runAt time.Time) error {
	payload := []byte(`{"incident_id":"` + incidentID.String() + `"}`)
	_, err := s.EnqueueJob(ctx, "escalate_incident", payload, runAt)
	return err
}

func (s *Store) ListAlertsForIncident(ctx context.Context, incidentID uuid.UUID) ([]Alert, error) {
	const q = `
SELECT a.id, a.fingerprint, a.status, a.severity, a.title, a.body, a.labels, a.search_tsv, a.raw_payload, a.received_at
FROM alerts a
JOIN incident_alerts ia ON ia.alert_id = a.id
WHERE ia.incident_id = $1
ORDER BY a.received_at ASC`
	rows, err := s.pool.Query(ctx, q, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var alert Alert
		if err := rows.Scan(
			&alert.ID, &alert.Fingerprint, &alert.Status, &alert.Severity, &alert.Title,
			&alert.Body, &alert.Labels, &alert.SearchTsv, &alert.RawPayload, &alert.ReceivedAt,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}
