package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/aegis/aegis/pkg/anomalymodule"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AnomalyEventResult struct {
	AlertID   uuid.UUID `json:"alert_id"`
	Duplicate bool      `json:"duplicate"`
	Ignored   bool      `json:"ignored"`
}

// AcceptAnomalyEvent commits the receipt and every lifecycle effect together.
// Lock order is receipt, episode, alert, matching the worker's alert lock before link.
func (s *Store) AcceptAnomalyEvent(ctx context.Context, event anomalymodule.Event) (AnomalyEventResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AnomalyEventResult{}, err
	}
	defer tx.Rollback(ctx)
	result, err := acceptAnomalyEventTx(ctx, tx, event)
	if err != nil {
		return result, err
	}
	err = tx.Commit(ctx)
	return result, err
}

func acceptAnomalyEventTx(ctx context.Context, tx pgx.Tx, e anomalymodule.Event) (AnomalyEventResult, error) {
	var result AnomalyEventResult
	// Different event IDs for one episode serialize after the event replay check.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "anomaly-event:"+e.InstallationID+":"+e.EventID); err != nil {
		return result, err
	}
	var hash string
	err := tx.QueryRow(ctx, `SELECT payload_hash,alert_id,ignored FROM anomaly_module_receipts WHERE installation_id=$1 AND event_id=$2`, e.InstallationID, e.EventID).Scan(&hash, &result.AlertID, &result.Ignored)
	if err == nil {
		if hash != e.Hash() {
			return result, apperrors.Conflict("event ID has already been used for another payload")
		}
		result.Duplicate = true
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return result, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "anomaly-episode:"+e.InstallationID+":"+e.EpisodeID); err != nil {
		return result, err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE id=$1)`, e.WorkspaceID).Scan(&exists); err != nil {
		return result, err
	}
	if !exists {
		return result, apperrors.Validation("workspace does not exist", nil)
	}
	var workspace, metric, series, state string
	var sequence int64
	err = tx.QueryRow(ctx, `SELECT workspace_id::text,metric_id::text,series_id,state,sequence,alert_id FROM anomaly_module_episodes WHERE installation_id=$1 AND episode_id=$2 FOR UPDATE`, e.InstallationID, e.EpisodeID).Scan(&workspace, &metric, &series, &state, &sequence, &result.AlertID)
	switch {
	case err == nil:
		if workspace != e.WorkspaceID || metric != e.MetricID || series != e.SeriesID {
			return result, apperrors.Conflict("episode identity cannot change")
		}
		if e.Sequence <= sequence || state == "resolved" {
			result.Ignored = true
			break
		}
		// Serialize recovery with ManualCreateFromAlert's FOR UPDATE lock.
		if _, err = tx.Exec(ctx, `SELECT id FROM alerts WHERE id=$1 FOR UPDATE`, result.AlertID); err != nil {
			return result, err
		}
		raw, _ := json.Marshal(e)
		labels, _ := json.Marshal(e.AlertLabels())
		if _, err = tx.Exec(ctx, `UPDATE alerts SET status=$2,severity=$3,title=$4,body=NULLIF($5,''),labels=$6,raw_payload=$7,search_tsv=to_tsvector('english',$4||' '||$5) WHERE id=$1`, result.AlertID, e.State, e.Severity, e.Title, e.Reason, labels, raw); err != nil {
			return result, err
		}
		if e.State == "resolved" {
			payload, _ := json.Marshal(map[string]any{"alert_id": result.AlertID, "episode_id": e.EpisodeID, "occurred_at": e.OccurredAt, "reason": e.Reason})
			if _, err = tx.Exec(ctx, `INSERT INTO timeline_events(incident_id,kind,payload) SELECT DISTINCT incident_id,'anomaly_recovered',$2::jsonb FROM incident_alerts WHERE alert_id=$1`, result.AlertID, payload); err != nil {
				return result, err
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE anomaly_module_episodes SET state=$3,sequence=$4,occurred_at=$5 WHERE installation_id=$1 AND episode_id=$2`, e.InstallationID, e.EpisodeID, e.State, e.Sequence, e.OccurredAt); err != nil {
			return result, err
		}
	case errors.Is(err, pgx.ErrNoRows):
		raw, _ := json.Marshal(e)
		labels, _ := json.Marshal(e.AlertLabels())
		if err = tx.QueryRow(ctx, `INSERT INTO alerts(fingerprint,status,severity,title,body,labels,raw_payload,search_tsv) VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,to_tsvector('english',$4||' '||$5)) RETURNING id`, e.Fingerprint(), e.State, e.Severity, e.Title, e.Reason, labels, raw).Scan(&result.AlertID); err != nil {
			return result, err
		}
		if e.State == "firing" {
			payload, _ := json.Marshal(map[string]string{"alert_id": result.AlertID.String()})
			if err = enqueueJobTx(ctx, tx, "process_alert", payload, time.Now().UTC()); err != nil {
				return result, err
			}
		}
		if _, err = tx.Exec(ctx, `INSERT INTO anomaly_module_episodes(installation_id,episode_id,workspace_id,metric_id,series_id,sequence,state,alert_id,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, e.InstallationID, e.EpisodeID, e.WorkspaceID, e.MetricID, e.SeriesID, e.Sequence, e.State, result.AlertID, e.OccurredAt); err != nil {
			return result, err
		}
	default:
		return result, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO anomaly_module_receipts(installation_id,event_id,payload_hash,alert_id,ignored) VALUES($1,$2,$3,$4,$5)`, e.InstallationID, e.EventID, e.Hash(), result.AlertID, result.Ignored)
	return result, err
}
