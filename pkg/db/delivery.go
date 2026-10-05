package db

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Store) HasDelivery(ctx context.Context, id, connector uuid.UUID, key string) (bool, error) {
	var sent bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notifications WHERE incident_id=$1 AND integration_id=$2 AND delivery_key=$3 AND (status='sent' OR external_ref IS NOT NULL OR failure_permanent))`, id, connector, key).Scan(&sent)
	return sent, err
}
func (s *Store) lockExpressResult(ctx context.Context, id uuid.UUID, ref string) error {
	_, err := s.pool.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,4))", id.String()+ref)
	return err
}
func (s *Store) expressFailureEvent(ctx context.Context, id uuid.UUID, key, ref, reason string) error {
	payload, _ := json.Marshal(map[string]string{"provider": "express", "message": "eXpress delivery failed: " + reason, "destination": key, "ref": ref})
	return s.AppendTimelineEvent(ctx, id, "integration_failed", nil, payload)
}

// The result inbox and per-reference lock reconcile callbacks arriving before a send commits.
func (s *Store) RecordDelivery(ctx context.Context, id, connector uuid.UUID, key, status, ref, message string) error {
	permanent := status == "failed_permanent"
	if permanent {
		status = "failed"
	}
	return s.InTransaction(ctx, func(tx *Store) error {
		if ref != "" {
			if err := tx.lockExpressResult(ctx, connector, ref); err != nil {
				return err
			}
			var result, reason string
			err := tx.pool.QueryRow(ctx, `SELECT status,reason FROM express_notification_results WHERE integration_id=$1 AND sync_id=$2`, connector, ref).Scan(&result, &reason)
			if err == nil && result != "ok" {
				status = "failed"
				message = reason
			} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		_, err := tx.pool.Exec(ctx, `INSERT INTO notifications(incident_id,integration_id,delivery_key,status,external_ref,delivery_error,sent_at,failure_permanent) VALUES($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),CASE WHEN $4='sent' THEN now() END,$7)`, id, connector, key, status, ref, message, permanent)
		if err != nil {
			return err
		}
		if ref != "" && status == "failed" {
			return tx.expressFailureEvent(ctx, id, key, ref, message)
		}
		return nil
	})
}
func (s *Store) HandleExpressNotificationResult(ctx context.Context, connector uuid.UUID, ref, status, reason string) error {
	return s.InTransaction(ctx, func(tx *Store) error {
		if err := tx.lockExpressResult(ctx, connector, ref); err != nil {
			return err
		}
		tag, err := tx.pool.Exec(ctx, `INSERT INTO express_notification_results(integration_id,sync_id,status,reason) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, connector, ref, status, reason)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		rows, err := tx.pool.Query(ctx, `UPDATE notifications SET status=CASE WHEN $3='ok' THEN 'sent' ELSE 'failed' END,delivery_error=NULLIF($4,'') WHERE integration_id=$1 AND external_ref=$2 RETURNING incident_id,delivery_key`, connector, ref, status, reason)
		if err != nil {
			return err
		}
		type affected struct {
			id  uuid.UUID
			key string
		}
		var notices []affected
		for rows.Next() {
			var a affected
			if err := rows.Scan(&a.id, &a.key); err != nil {
				rows.Close()
				return err
			}
			notices = append(notices, a)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if status != "ok" {
			for _, a := range notices {
				if err := tx.expressFailureEvent(ctx, a.id, a.key, ref, reason); err != nil {
					return err
				}
			}
		}
		if _, err := tx.pool.Exec(ctx, `UPDATE express_outbound SET status=CASE WHEN $3='ok' THEN 'sent' ELSE 'failed' END,delivery_error=NULLIF($4,'') WHERE integration_id=$1 AND sync_id=$2`, connector, ref, status, reason); err != nil {
			return err
		}
		if status != "ok" {
			_, err = tx.pool.Exec(ctx, `UPDATE oncall_deliveries d SET status='failed',retry_at=NULL,last_error=$3 FROM express_outbound o WHERE o.integration_id=$1 AND o.sync_id=$2 AND o.team_id=d.team_id AND d.provider='express' AND o.destination=d.destination AND o.publication_key=d.publication_key AND o.sync_id=d.external_ref`, connector, ref, "eXpress delivery failed: "+reason)
		}
		return err
	})
}
func (s *Store) HasExpressOutbound(ctx context.Context, id uuid.UUID, key string) (bool, error) {
	var sent bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM express_outbound WHERE integration_id=$1 AND delivery_key=$2)`, id, key).Scan(&sent)
	return sent, err
}
func (s *Store) RecordExpressOutbound(ctx context.Context, id uuid.UUID, key, ref string) error {
	return s.InTransaction(ctx, func(tx *Store) error {
		if err := tx.lockExpressResult(ctx, id, ref); err != nil {
			return err
		}
		_, err := tx.pool.Exec(ctx, `INSERT INTO express_outbound(integration_id,delivery_key,sync_id,status,delivery_error) SELECT $1,$2,$3,CASE WHEN r.status IS NOT NULL AND r.status<>'ok' THEN 'failed' ELSE 'sent' END,NULLIF(r.reason,'') FROM (SELECT 1) dummy LEFT JOIN express_notification_results r ON r.integration_id=$1 AND r.sync_id=$3 ON CONFLICT(integration_id,delivery_key) DO NOTHING`, id, key, ref)
		return err
	})
}
func (s *Store) RecordOnCallExpressRef(ctx context.Context, connector, team uuid.UUID, destination, key, ref string) error {
	return s.InTransaction(ctx, func(tx *Store) error {
		deliveryKey := "oncall:" + team.String() + ":" + destination + ":" + key + ":" + ref
		if err := tx.RecordExpressOutbound(ctx, connector, deliveryKey, ref); err != nil {
			return err
		}
		if _, err := tx.pool.Exec(ctx, `UPDATE express_outbound SET team_id=$3,destination=$4,publication_key=$5 WHERE integration_id=$1 AND delivery_key=$2`, connector, deliveryKey, team, destination, key); err != nil {
			return err
		}
		_, err := tx.pool.Exec(ctx, `UPDATE oncall_deliveries d SET status='failed',retry_at=NULL,last_error=o.delivery_error FROM express_outbound o WHERE o.integration_id=$1 AND o.delivery_key=$2 AND o.status='failed' AND d.team_id=$3 AND d.provider='express' AND d.destination=$4 AND d.publication_key=$5 AND d.external_ref=o.sync_id`, connector, deliveryKey, team, destination, key)
		return err
	})
}
func (s *Store) EnqueueExpressReply(ctx context.Context, id uuid.UUID, command, chat, body, locale string) error {
	raw, _ := json.Marshal(map[string]string{"integration_id": id.String(), "command_id": command, "chat_id": chat, "body": body, "locale": locale})
	_, err := s.pool.Exec(ctx, `INSERT INTO jobs(kind,payload,status,run_at,dedup_key) VALUES('express_reply',$1,'pending',$2,$3) ON CONFLICT(dedup_key) WHERE dedup_key IS NOT NULL DO NOTHING`, raw, time.Now().UTC(), "express-reply:"+id.String()+":"+command)
	return err
}
func (s *Store) RunDelivery(ctx context.Context, id uuid.UUID, key string, fn func(*Store) error) error {
	return s.InTransaction(ctx, func(tx *Store) error {
		if _, err := tx.pool.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,3))", id.String()+key); err != nil {
			return err
		}
		return fn(tx)
	})
}
