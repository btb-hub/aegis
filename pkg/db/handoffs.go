package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type HandoffIncidentInput struct {
	IncidentID uuid.UUID
	ActorID    uuid.UUID
	ToTeamID   uuid.UUID
	ToUserID   uuid.UUID
	Note       string
}

type BounceIncidentInput struct {
	IncidentID uuid.UUID
	ActorID    uuid.UUID
	Note       string
}

func (s *Store) HandoffIncident(ctx context.Context, input HandoffIncidentInput) (Incident, Handoff, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Incident{}, Handoff{}, err
	}
	defer tx.Rollback(ctx)

	var incident Incident
	err = tx.QueryRow(ctx, `
SELECT id, team_id, assignee_id, status, severity, title, fingerprint,
       jira_issue_key, acknowledged_at, resolved_at, created_at
FROM incidents
WHERE id = $1
FOR UPDATE`, input.IncidentID).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	if err != nil {
		return Incident{}, Handoff{}, err
	}
	if incident.Status == "resolved" {
		return Incident{}, Handoff{}, pgx.ErrNoRows
	}

	fromUserID := input.ActorID
	if incident.AssigneeID != nil {
		fromUserID = *incident.AssigneeID
	}

	var reason *string
	if input.Note != "" {
		reason = &input.Note
	}

	var handoff Handoff
	err = tx.QueryRow(ctx, `
INSERT INTO handoffs (incident_id, from_user_id, to_user_id, from_team_id, to_team_id, reason)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, incident_id, from_user_id, to_user_id, from_team_id, to_team_id, reason, bounced_at, created_at`,
		input.IncidentID, fromUserID, input.ToUserID, incident.TeamID, input.ToTeamID, reason,
	).Scan(
		&handoff.ID, &handoff.IncidentID, &handoff.FromUserID, &handoff.ToUserID,
		&handoff.FromTeamID, &handoff.ToTeamID, &handoff.Reason, &handoff.BouncedAt, &handoff.CreatedAt,
	)
	if err != nil {
		return Incident{}, Handoff{}, err
	}

	toUserID := input.ToUserID
	err = tx.QueryRow(ctx, `
UPDATE incidents
SET assignee_id = $2
WHERE id = $1
RETURNING id, team_id, assignee_id, status, severity, title, fingerprint,
          jira_issue_key, acknowledged_at, resolved_at, created_at`,
		input.IncidentID, toUserID,
	).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	if err != nil {
		return Incident{}, Handoff{}, err
	}

	payload, _ := json.Marshal(map[string]any{
		"handoff_id":   handoff.ID.String(),
		"to_team_id":   input.ToTeamID.String(),
		"to_user_id":   input.ToUserID.String(),
		"from_user_id": fromUserID.String(),
		"note":         input.Note,
	})
	if err := insertTimelineEventTx(ctx, tx, input.IncidentID, "handoff", &input.ActorID, payload); err != nil {
		return Incident{}, Handoff{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Incident{}, Handoff{}, err
	}
	return incident, handoff, nil
}

func (s *Store) BounceIncident(ctx context.Context, input BounceIncidentInput) (Incident, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Incident{}, err
	}
	defer tx.Rollback(ctx)

	var incident Incident
	err = tx.QueryRow(ctx, `
SELECT id, team_id, assignee_id, status, severity, title, fingerprint,
       jira_issue_key, acknowledged_at, resolved_at, created_at
FROM incidents
WHERE id = $1
FOR UPDATE`, input.IncidentID).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	if err != nil {
		return Incident{}, err
	}
	if incident.Status == "resolved" {
		return Incident{}, pgx.ErrNoRows
	}

	var handoff Handoff
	err = tx.QueryRow(ctx, `
SELECT id, incident_id, from_user_id, to_user_id, from_team_id, to_team_id, reason, bounced_at, created_at
FROM handoffs
WHERE incident_id = $1 AND bounced_at IS NULL
ORDER BY created_at DESC
LIMIT 1`, input.IncidentID).Scan(
		&handoff.ID, &handoff.IncidentID, &handoff.FromUserID, &handoff.ToUserID,
		&handoff.FromTeamID, &handoff.ToTeamID, &handoff.Reason, &handoff.BouncedAt, &handoff.CreatedAt,
	)
	if err != nil {
		return Incident{}, err
	}
	if handoff.FromUserID == nil {
		return Incident{}, pgx.ErrNoRows
	}

	_, err = tx.Exec(ctx, `UPDATE handoffs SET bounced_at = now() WHERE id = $1`, handoff.ID)
	if err != nil {
		return Incident{}, err
	}

	err = tx.QueryRow(ctx, `
UPDATE incidents
SET assignee_id = $2
WHERE id = $1
RETURNING id, team_id, assignee_id, status, severity, title, fingerprint,
          jira_issue_key, acknowledged_at, resolved_at, created_at`,
		input.IncidentID, *handoff.FromUserID,
	).Scan(
		&incident.ID, &incident.TeamID, &incident.AssigneeID, &incident.Status, &incident.Severity,
		&incident.Title, &incident.Fingerprint, &incident.JiraIssueKey, &incident.AcknowledgedAt,
		&incident.ResolvedAt, &incident.CreatedAt,
	)
	if err != nil {
		return Incident{}, err
	}

	payload, _ := json.Marshal(map[string]any{
		"handoff_id":   handoff.ID.String(),
		"to_user_id":   handoff.ToUserID,
		"from_user_id": handoff.FromUserID,
		"note":         input.Note,
	})
	if err := insertTimelineEventTx(ctx, tx, input.IncidentID, "bounced", &input.ActorID, payload); err != nil {
		return Incident{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Incident{}, err
	}
	return incident, nil
}

func (s *Store) EnqueueHandoffNotify(ctx context.Context, incidentID uuid.UUID) error {
	payload, err := json.Marshal(map[string]string{"incident_id": incidentID.String()})
	if err != nil {
		return err
	}
	_, err = s.EnqueueJob(ctx, "notify_handoff", payload, time.Now().UTC())
	return err
}

func (s *Store) HandoffStats(ctx context.Context, from, to time.Time) (HandoffStats, error) {
	const q = `
WITH handoffs_in_range AS (
    SELECT h.id, h.incident_id, h.to_user_id, h.created_at
    FROM handoffs h
    WHERE h.created_at >= $1 AND h.created_at < $2
),
first_l3_ack AS (
    SELECT hir.id AS handoff_id,
           MIN(te.created_at) AS ack_at
    FROM handoffs_in_range hir
    JOIN timeline_events te ON te.incident_id = hir.incident_id
    WHERE te.kind = 'acknowledged'
      AND te.created_at > hir.created_at
      AND (te.actor_id = hir.to_user_id OR hir.to_user_id IS NULL)
    GROUP BY hir.id
),
durations AS (
    SELECT EXTRACT(EPOCH FROM (fa.ack_at - hir.created_at)) AS seconds
    FROM handoffs_in_range hir
    JOIN first_l3_ack fa ON fa.handoff_id = hir.id
)
SELECT
    (SELECT COUNT(*) FROM handoffs_in_range) AS count,
    (SELECT COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY seconds), 0) FROM durations) AS median_seconds`

	var stats HandoffStats
	err := s.pool.QueryRow(ctx, q, from, to).Scan(&stats.Count, &stats.MedianResponseSeconds)
	return stats, err
}

func (s *Store) ListTimelineEventsForIncident(ctx context.Context, incidentID uuid.UUID) ([]TimelineEvent, error) {
	return s.ListTimelineEvents(ctx, incidentID)
}
