package db

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProcessChatAck commits the transition, cancellation, channel event, and saved
// response together. A restart cannot turn a successful first click into a
// different response or duplicate its state transition.
func (s *Store) ProcessChatAck(ctx context.Context, request ChatAckRequest) (json.RawMessage, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var outcome json.RawMessage
	if err = tx.QueryRow(ctx, `SELECT outcome FROM chat_ack_requests WHERE id=$1 FOR UPDATE`, request.ID).Scan(&outcome); err != nil {
		return nil, err
	}
	if len(outcome) == 0 {
		result, err := AcknowledgeChat(ctx, chatAckTransaction{tx}, request.IncidentID, request.Provider, request.UserIdentity)
		if err != nil {
			return nil, err
		}
		if result.Locale == "" {
			result.Locale = request.Locale
		}
		outcome, err = json.Marshal(result)
		if err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `UPDATE chat_ack_requests SET outcome=$2,attempts=0 WHERE id=$1`, request.ID, outcome); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return outcome, nil
}

type chatAckTransaction struct{ tx pgx.Tx }

func (s chatAckTransaction) GetUserBySlackID(ctx context.Context, id string) (User, error) {
	return scanUser(s.tx.QueryRow(ctx, `SELECT `+userSelectColumns+` FROM users WHERE slack_user_id=$1`, id))
}
func (s chatAckTransaction) GetUserByExpressHuid(ctx context.Context, id uuid.UUID) (User, error) {
	return scanUser(s.tx.QueryRow(ctx, `SELECT `+userSelectColumns+` FROM users WHERE express_user_huid=$1`, id))
}
func (s chatAckTransaction) GetIncidentByID(ctx context.Context, id uuid.UUID) (Incident, error) {
	var i Incident
	err := s.tx.QueryRow(ctx, `SELECT id,team_id,assignee_id,status,severity,title,fingerprint,jira_issue_key,acknowledged_at,resolved_at,created_at FROM incidents WHERE id=$1 FOR UPDATE`, id).Scan(
		&i.ID, &i.TeamID, &i.AssigneeID, &i.Status, &i.Severity, &i.Title, &i.Fingerprint, &i.JiraIssueKey, &i.AcknowledgedAt, &i.ResolvedAt, &i.CreatedAt)
	return i, err
}
func (s chatAckTransaction) AcknowledgeIncident(ctx context.Context, id, actor uuid.UUID) (Incident, error) {
	return acknowledgeIncidentTx(ctx, s.tx, id, actor)
}
func (s chatAckTransaction) CancelEscalationJobs(ctx context.Context, id uuid.UUID) error {
	_, err := s.tx.Exec(ctx, `UPDATE jobs SET status='done',updated_at=now() WHERE kind='escalate_incident' AND status='pending' AND payload->>'incident_id'=$1`, id.String())
	return err
}
