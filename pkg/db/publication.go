package db

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/aegis/aegis/pkg/oncall"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

type PublicationSettings struct {
	Time      string    `json:"time"`
	Timezone  string    `json:"timezone"`
	NextRunAt time.Time `json:"next_run_at"`
}

func (s *Store) GetPublicationSettings(ctx context.Context) (PublicationSettings, error) {
	var out PublicationSettings
	err := s.pool.QueryRow(ctx, `SELECT time,timezone,next_run_at FROM oncall_publication_settings WHERE id=true`).Scan(&out.Time, &out.Timezone, &out.NextRunAt)
	return out, err
}
func (s *Store) UpdatePublicationSettings(ctx context.Context, clock, zone string, now time.Time) (PublicationSettings, error) {
	next, err := oncall.NextPublication(now, clock, zone)
	if err != nil {
		return PublicationSettings{}, err
	}
	_, err = s.pool.Exec(ctx, `UPDATE oncall_publication_settings SET time=$1,timezone=$2,next_run_at=$3,updated_at=now() WHERE id=true`, clock, zone, next)
	return PublicationSettings{Time: clock, Timezone: zone, NextRunAt: next}, err
}
func (s *Store) EnqueueScheduledPublication(ctx context.Context, now time.Time) error {
	return s.InTransaction(ctx, func(tx *Store) error {
		var settings PublicationSettings
		if err := tx.pool.QueryRow(ctx, `SELECT time,timezone,next_run_at FROM oncall_publication_settings WHERE id=true FOR UPDATE`).Scan(&settings.Time, &settings.Timezone, &settings.NextRunAt); err != nil {
			return err
		}
		if now.Before(settings.NextRunAt) {
			return nil
		}
		loc, err := time.LoadLocation(settings.Timezone)
		if err != nil {
			return err
		}
		if settings.NextRunAt.In(loc).Format("2006-01-02") == now.In(loc).Format("2006-01-02") {
			date := now.In(loc).Format("2006-01-02")
			tag, err := tx.pool.Exec(ctx, `INSERT INTO oncall_publication_runs(local_date) VALUES($1) ON CONFLICT DO NOTHING`, date)
			if err != nil {
				return err
			}
			if tag.RowsAffected() > 0 {
				payload, _ := json.Marshal(map[string]string{"publication_key": "daily:" + date})
				if _, err := tx.pool.Exec(ctx, `INSERT INTO jobs(kind,payload,status,run_at,dedup_key) VALUES('publish_oncall',$1,'pending',$2,$3) ON CONFLICT(dedup_key) WHERE dedup_key IS NOT NULL DO NOTHING`, payload, now, "daily-oncall:"+date); err != nil {
					return err
				}
			}
		}
		next, err := oncall.NextPublication(now, settings.Time, settings.Timezone)
		if err != nil {
			return err
		}
		_, err = tx.pool.Exec(ctx, `UPDATE oncall_publication_settings SET next_run_at=$1 WHERE id=true`, next)
		return err
	})
}

type PublicationAttempt struct {
	TeamID                                                            uuid.UUID
	Provider, Destination, Fingerprint, ConfigVersion, PublicationKey string
}

func (s *Store) NeedsOnCallPublication(ctx context.Context, a PublicationAttempt, now time.Time) (bool, error) {
	var fp, version, status string
	var retry *time.Time
	err := s.pool.QueryRow(ctx, `SELECT fingerprint,config_version,status,retry_at FROM oncall_deliveries WHERE team_id=$1 AND provider=$2 AND destination=$3`, a.TeamID, a.Provider, a.Destination).Scan(&fp, &version, &status, &retry)
	if errors.Is(err, pgx.ErrNoRows) {
		return a.Fingerprint != "", nil
	}
	if err != nil {
		return false, err
	}
	return fp != a.Fingerprint || version != a.ConfigVersion || (status != "sent" && retry != nil && !now.Before(*retry)), nil
}
func (s *Store) RunOnCallDelivery(ctx context.Context, a PublicationAttempt, send func() error) error {
	return s.runOnCallDelivery(ctx, a, func(*Store) (string, error) { return "", send() })
}
func (s *Store) RunOnCallDeliveryWithRef(ctx context.Context, a PublicationAttempt, connector uuid.UUID, send func() (string, error)) error {
	return s.runOnCallDelivery(ctx, a, func(tx *Store) (string, error) {
		ref, err := send()
		if err != nil {
			return ref, err
		}
		if err := tx.RecordOnCallExpressRef(ctx, connector, a.TeamID, a.Destination, a.PublicationKey, ref); err != nil {
			return ref, err
		}
		var status string
		var reason *string
		if err := tx.pool.QueryRow(ctx, `SELECT status,delivery_error FROM express_outbound WHERE integration_id=$1 AND sync_id=$2`, connector, ref).Scan(&status, &reason); err != nil {
			return ref, err
		}
		if status == "failed" {
			message := "asynchronous delivery failed"
			if reason != nil {
				message = *reason
			}
			return ref, &integrations.HTTPError{Provider: "express", Operation: "oncall delivery result", Status: 422, Message: message}
		}
		return ref, nil
	})
}
func (s *Store) runOnCallDelivery(ctx context.Context, a PublicationAttempt, send func(*Store) (string, error)) error {
	var sendErr error
	err := s.InTransaction(ctx, func(tx *Store) error {
		if _, err := tx.pool.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,2))", a.TeamID.String()+a.Provider+a.Destination); err != nil {
			return err
		}
		var fp, version, key, status string
		var lastError *string
		var retry *time.Time
		var attempts int
		err := tx.pool.QueryRow(ctx, `SELECT fingerprint,config_version,publication_key,status,retry_at,attempts,last_error FROM oncall_deliveries WHERE team_id=$1 AND provider=$2 AND destination=$3`, a.TeamID, a.Provider, a.Destination).Scan(&fp, &version, &key, &status, &retry, &attempts, &lastError)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		same := err == nil && fp == a.Fingerprint && version == a.ConfigVersion && (a.PublicationKey == "" || key == a.PublicationKey)
		if same {
			if status == "sent" {
				return nil
			}
			if retry == nil || time.Now().Before(*retry) {
				message := "publication failed"
				if lastError != nil {
					message = *lastError
				}
				statusCode := 400
				if retry != nil {
					statusCode = 503
				}
				sendErr = &integrations.HTTPError{Provider: a.Provider, Operation: "oncall publication", Status: statusCode, Message: message}
				return nil
			}
		} else {
			attempts = 0
		}
		ref, err := send(tx)
		sendErr = err
		attempts++
		status = "sent"
		message := ""
		retry = nil
		if sendErr != nil {
			status = "failed"
			message = sendErr.Error()
			type retryable interface{ Retryable() bool }
			var e retryable
			transient := !errors.As(sendErr, &e) || e.Retryable()
			if transient && attempts < 4 {
				delay := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}[attempts-1]
				at := time.Now().Add(delay)
				retry = &at
			}
		}
		_, err = tx.pool.Exec(ctx, `INSERT INTO oncall_deliveries(team_id,provider,destination,fingerprint,config_version,publication_key,status,attempts,retry_at,last_error,external_ref) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),NULLIF($11,'')) ON CONFLICT(team_id,provider,destination) DO UPDATE SET fingerprint=$4,config_version=$5,publication_key=$6,status=$7,attempts=$8,retry_at=$9,last_error=NULLIF($10,''),external_ref=NULLIF($11,'')`, a.TeamID, a.Provider, a.Destination, a.Fingerprint, a.ConfigVersion, a.PublicationKey, status, attempts, retry, message, ref)
		return err
	})
	if err != nil {
		return err
	}
	return sendErr
}

func (s *Store) EnqueueManualPublishOnCall(ctx context.Context, id uuid.UUID) error {
	raw, _ := json.Marshal(map[string]string{"team_id": id.String()})
	_, err := s.EnqueueJob(ctx, "publish_oncall", raw, time.Now())
	return err
}
