package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ChannelSnapshot struct {
	Incident         Incident `json:"incident"`
	TeamName         string   `json:"team_name"`
	SlackUserGroupID string   `json:"slack_user_group_id"`
	ActorName        string   `json:"actor_name"`
	Locale           string   `json:"locale"`
}

type ChannelDelivery struct {
	ID, EventID, IntegrationID  uuid.UUID
	Provider, Destination, Kind string
	Attempts                    int
	OccurredAt                  time.Time
	Snapshot                    ChannelSnapshot
}

type ChatAckRequest struct {
	ID                                        uuid.UUID
	Provider, DedupKey                        string
	IncidentID, IntegrationID                 uuid.UUID
	WorkspaceID                               *uuid.UUID
	UserIdentity, ChatID, ResponseURL, Locale string
	Outcome                                   json.RawMessage
	Attempts                                  int
}

func (s *Store) EnqueueChatAck(ctx context.Context, r ChatAckRequest) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO chat_ack_requests(provider,dedup_key,incident_id,integration_id,user_identity,chat_id,response_url,locale,workspace_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(provider,dedup_key) DO NOTHING`, r.Provider, r.DedupKey, r.IncidentID, r.IntegrationID, r.UserIdentity, r.ChatID, r.ResponseURL, r.Locale, r.WorkspaceID)
	return err
}

func (s *Store) QueueChannelEscalation(ctx context.Context, incidentID uuid.UUID, jobID string) error {
	// A separate key per job avoids the provider-specific escalated timeline events.
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("channel-escalation:"+jobID))
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	if err = tx.QueryRow(ctx, `SELECT status FROM incidents WHERE id=$1 FOR UPDATE`, incidentID).Scan(&status); err != nil {
		return err
	}
	if status != "open" {
		return nil
	}
	if _, err = tx.Exec(ctx, `SELECT queue_incident_channel_event($1,$2,'escalated',NULL,clock_timestamp())`, id, incidentID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ClaimChannelDelivery(ctx context.Context) (ChannelDelivery, error) {
	var d ChannelDelivery
	var raw []byte
	err := s.pool.QueryRow(ctx, `WITH candidate AS (
 SELECT d.id FROM incident_channel_deliveries d JOIN incident_channel_events e ON e.id=d.event_id
 WHERE d.status='pending' AND d.next_at<=now() AND (d.leased_until IS NULL OR d.leased_until<now())
 AND NOT EXISTS(SELECT 1 FROM incident_channel_deliveries older JOIN incident_channel_events oe ON oe.id=older.event_id
 WHERE oe.incident_id=e.incident_id AND oe.sequence<e.sequence AND older.provider=d.provider
 AND older.destination=d.destination AND older.status='pending')
 ORDER BY e.sequence FOR UPDATE OF d SKIP LOCKED LIMIT 1
 ), claimed AS (
 UPDATE incident_channel_deliveries SET attempts=attempts+1,leased_until=now()+interval '60 seconds'
 WHERE id IN(SELECT id FROM candidate) RETURNING *
 ) SELECT d.id,d.event_id,d.integration_id,d.provider,d.destination,d.attempts,e.kind,e.occurred_at,e.snapshot
 FROM claimed d JOIN incident_channel_events e ON e.id=d.event_id`).Scan(&d.ID, &d.EventID, &d.IntegrationID, &d.Provider, &d.Destination, &d.Attempts, &d.Kind, &d.OccurredAt, &raw)
	if err == nil {
		err = json.Unmarshal(raw, &d.Snapshot)
	}
	return d, err
}

func (s *Store) ClaimChatAckAction(ctx context.Context) (ChatAckRequest, error) {
	return s.claimChatAck(ctx, true)
}
func (s *Store) ClaimChatAckFeedback(ctx context.Context) (ChatAckRequest, error) {
	return s.claimChatAck(ctx, false)
}
func (s *Store) claimChatAck(ctx context.Context, processing bool) (ChatAckRequest, error) {
	var r ChatAckRequest
	err := s.pool.QueryRow(ctx, `UPDATE chat_ack_requests SET attempts=attempts+1,leased_until=now()+interval '60 seconds'
 WHERE id=(SELECT id FROM chat_ack_requests WHERE status='pending' AND next_at<=now()
 AND (outcome IS NULL)=$1 AND (leased_until IS NULL OR leased_until<now()) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1)
 RETURNING id,provider,dedup_key,incident_id,integration_id,user_identity,chat_id,response_url,locale,outcome,attempts,workspace_id`, processing).Scan(
		&r.ID, &r.Provider, &r.DedupKey, &r.IncidentID, &r.IntegrationID, &r.UserIdentity, &r.ChatID, &r.ResponseURL, &r.Locale, &r.Outcome, &r.Attempts, &r.WorkspaceID)
	return r, err
}

func (s *Store) SaveChatAckOutcome(ctx context.Context, id uuid.UUID, outcome []byte) error {
	_, err := s.pool.Exec(ctx, `UPDATE chat_ack_requests SET outcome=$2 WHERE id=$1 AND outcome IS NULL`, id, outcome)
	return err
}

func (s *Store) FinishChannelDelivery(ctx context.Context, id uuid.UUID, status, ref, message string, next time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE incident_channel_deliveries SET status=$2,external_ref=NULLIF($3,''),last_error=NULLIF($4,''),next_at=$5,leased_until=NULL WHERE id=$1`, id, status, ref, message, next)
	return err
}
func (s *Store) FinishChatAck(ctx context.Context, id uuid.UUID, status, message string, next time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE chat_ack_requests SET status=$2,last_error=NULLIF($3,''),next_at=$4,leased_until=NULL WHERE id=$1`, id, status, message, next)
	return err
}

func (s *Store) ReadyChatAckFeedback(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE chat_ack_requests SET attempts=0,next_at=now(),leased_until=NULL WHERE id=$1`, id)
	return err
}
