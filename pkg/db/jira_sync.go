package db

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

type JiraSyncState struct {
	IncidentID      uuid.UUID
	IntegrationID   *uuid.UUID
	Config          []byte
	IssueKey        *string
	CreateAttempted bool
	AssigneeID      *uuid.UUID
}
type JiraComment struct {
	EventID      uuid.UUID
	Body, Author string
	CreatedAt    time.Time
	ExternalID   *string
	Attempted    bool
}

// JiraSyncRepository is transaction-scoped; external calls are serialized by the incident lock.
type JiraSyncRepository interface {
	GetIncidentByID(context.Context, uuid.UUID) (Incident, error)
	GetUserByID(context.Context, uuid.UUID) (User, error)
	GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error)
	GetWorkspaceIntegration(context.Context, uuid.UUID, string) (Integration, error)
	GetIntegrationByKind(context.Context, string) (Integration, error)
	GetIntegration(context.Context, uuid.UUID) (Integration, error)
	GetJiraSyncState(context.Context, uuid.UUID) (JiraSyncState, error)
	SaveJiraSyncState(context.Context, JiraSyncState) error
	SaveJiraIssue(context.Context, uuid.UUID, uuid.UUID, string, string) error
	PendingJiraComments(context.Context, uuid.UUID) ([]JiraComment, error)
	SaveJiraComment(context.Context, uuid.UUID, *string, bool) error
	AppendTimelineEvent(context.Context, uuid.UUID, string, *uuid.UUID, []byte) error
}

func (s *Store) RunJiraSync(ctx context.Context, id uuid.UUID, fn func(JiraSyncRepository) error) error {
	return s.InTransaction(ctx, func(tx *Store) error {
		if _, err := tx.pool.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", id.String()); err != nil {
			return err
		}
		// The lock transaction serializes workers; writes use the pool and commit
		// before external requests, so attempt markers survive process crashes.
		return fn(s)
	})
}
func (s *Store) GetJiraSyncState(ctx context.Context, id uuid.UUID) (JiraSyncState, error) {
	state := JiraSyncState{IncidentID: id, Config: []byte(`{}`)}
	err := s.pool.QueryRow(ctx, `SELECT integration_id,config,issue_key,create_attempted,assignee_id FROM jira_sync_state WHERE incident_id=$1`, id).Scan(&state.IntegrationID, &state.Config, &state.IssueKey, &state.CreateAttempted, &state.AssigneeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, nil
	}
	return state, err
}
func (s *Store) SaveJiraSyncState(ctx context.Context, state JiraSyncState) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO jira_sync_state(incident_id,integration_id,config,issue_key,create_attempted,assignee_id) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(incident_id) DO UPDATE SET integration_id=$2,config=$3,issue_key=$4,create_attempted=$5,assignee_id=$6`, state.IncidentID, state.IntegrationID, state.Config, state.IssueKey, state.CreateAttempted, state.AssigneeID)
	return err
}
func (s *Store) SaveJiraIssue(ctx context.Context, id, integrationID uuid.UUID, key, issueURL string) error {
	_, err := s.pool.Exec(ctx, `UPDATE incidents SET jira_issue_key=$2,jira_integration_id=$3,jira_issue_url=$4 WHERE id=$1`, id, key, integrationID, issueURL)
	return err
}
func (s *Store) PendingJiraComments(ctx context.Context, id uuid.UUID) ([]JiraComment, error) {
	rows, err := s.pool.Query(ctx, `SELECT e.id,e.payload,e.created_at,s.external_id,s.attempted FROM jira_comment_sync s JOIN timeline_events e ON e.id=s.event_id WHERE s.incident_id=$1 AND s.external_id IS NULL ORDER BY e.created_at,e.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JiraComment
	for rows.Next() {
		var c JiraComment
		var payload []byte
		if err := rows.Scan(&c.EventID, &payload, &c.CreatedAt, &c.ExternalID, &c.Attempted); err != nil {
			return nil, err
		}
		var p map[string]string
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		c.Body = p["body"]
		c.Author = p["author_name"]
		out = append(out, c)
	}
	return out, rows.Err()
}
func (s *Store) SaveJiraComment(ctx context.Context, id uuid.UUID, external *string, attempted bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE jira_comment_sync SET external_id=$2,attempted=$3 WHERE event_id=$1`, id, external, attempted)
	return err
}
