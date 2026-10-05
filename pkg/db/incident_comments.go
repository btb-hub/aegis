package db

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

func (s *Store) AddIncidentComment(ctx context.Context, incidentID, actorID uuid.UUID, body string) (TimelineEvent, error) {
	var event TimelineEvent
	err := s.InTransaction(ctx, func(tx *Store) error {
		// Serialize comments with resolution and synchronization for this incident.
		var id uuid.UUID
		if err := tx.pool.QueryRow(ctx, "SELECT id FROM incidents WHERE id=$1 FOR UPDATE", incidentID).Scan(&id); err != nil {
			return err
		}
		user, err := tx.GetUserByID(ctx, actorID)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]string{"body": body, "author_name": user.DisplayName})
		event = TimelineEvent{ID: uuid.New(), IncidentID: incidentID, Kind: "comment_added", ActorID: &actorID, Payload: payload}
		err = tx.pool.QueryRow(ctx, `INSERT INTO timeline_events(id,incident_id,kind,actor_id,payload,created_at) VALUES($1,$2,$3,$4,$5,clock_timestamp()) RETURNING created_at`, event.ID, incidentID, event.Kind, actorID, payload).Scan(&event.CreatedAt)
		if err != nil {
			return err
		}
		if _, err := tx.pool.Exec(ctx, `INSERT INTO jira_comment_sync(event_id,incident_id) VALUES($1,$2)`, event.ID, incidentID); err != nil {
			return err
		}
		return tx.EnqueueJiraSync(ctx, incidentID)
	})
	return event, err
}
func (s *Store) ResolveIncidentWithComment(ctx context.Context, incidentID, actorID uuid.UUID, comment string) (Incident, error) {
	var incident Incident
	err := s.InTransaction(ctx, func(tx *Store) error {
		var err error
		incident, err = tx.ResolveIncident(ctx, incidentID, actorID)
		if err != nil {
			return err
		}
		if comment != "" {
			_, err = tx.AddIncidentComment(ctx, incidentID, actorID, comment)
		} else {
			err = tx.EnqueueJiraSync(ctx, incidentID)
		}
		return err
	})
	return incident, err
}
func (s *Store) EnqueueJiraSync(ctx context.Context, id uuid.UUID) error {
	raw, _ := json.Marshal(map[string]string{"incident_id": id.String()})
	_, err := s.EnqueueJob(ctx, "sync_jira", raw, time.Now().UTC())
	return err
}
