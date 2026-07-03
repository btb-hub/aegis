package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) UpsertUser(ctx context.Context, provider, providerSub, email, displayName, role, locale string) (User, error) {
	const q = `
INSERT INTO users (provider, provider_sub, email, display_name, role, locale)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (provider, provider_sub) DO UPDATE
SET email = EXCLUDED.email, display_name = EXCLUDED.display_name
RETURNING ` + userSelectColumns
	return scanUser(s.pool.QueryRow(ctx, q, provider, providerSub, email, displayName, role, locale))
}

func (s *Store) UpsertDevUser(ctx context.Context, email, displayName, role, locale string) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	const q = `
INSERT INTO users (provider, provider_sub, email, display_name, role, locale)
VALUES ('dev', 'dev-local', $1, $2, $3, $4)
ON CONFLICT (provider, provider_sub) DO UPDATE
SET email = EXCLUDED.email, display_name = EXCLUDED.display_name, role = EXCLUDED.role
RETURNING ` + userSelectColumns
	user, err := scanUser(tx.QueryRow(ctx, q, email, displayName, role, locale))
	if err != nil {
		return User{}, err
	}
	if err := insertIdentityTx(ctx, tx, user.ID, "dev", "dev-local"); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	q := `SELECT ` + userSelectColumns + ` FROM users WHERE id = $1`
	return scanUser(s.pool.QueryRow(ctx, q, id))
}

func (s *Store) GetUserBySlackID(ctx context.Context, slackUserID string) (User, error) {
	q := `SELECT ` + userSelectColumns + ` FROM users WHERE slack_user_id = $1`
	return scanUser(s.pool.QueryRow(ctx, q, slackUserID))
}

func (s *Store) UpdateUserLocale(ctx context.Context, id uuid.UUID, locale string) (User, error) {
	q := `UPDATE users SET locale = $2 WHERE id = $1 RETURNING ` + userSelectColumns
	return scanUser(s.pool.QueryRow(ctx, q, id, locale))
}

func (s *Store) UpdateUserProfile(ctx context.Context, id uuid.UUID, displayName, locale string) (User, error) {
	q := `UPDATE users SET display_name = $2, locale = $3 WHERE id = $1 RETURNING ` + userSelectColumns
	return scanUser(s.pool.QueryRow(ctx, q, id, displayName, locale))
}

func (s *Store) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (Session, error) {
	const q = `INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING id, user_id, token_hash, expires_at, created_at`
	var session Session
	err := s.pool.QueryRow(ctx, q, userID, tokenHash, expiresAt).Scan(
		&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt,
	)
	return session, err
}

func (s *Store) GetSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	const q = `SELECT id, user_id, token_hash, expires_at, created_at FROM sessions WHERE token_hash = $1 AND expires_at > now()`
	var session Session
	err := s.pool.QueryRow(ctx, q, tokenHash).Scan(
		&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt,
	)
	return session, err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

type CreateAlertJobInput struct {
	Fingerprint string
	Status      string
	Severity    string
	Title       string
	Body        string
	Labels      map[string]string
	RawPayload  json.RawMessage
	JobKind     string
	JobPayload  map[string]any
}

type CreateAlertJobResult struct {
	AlertID uuid.UUID
	JobID   uuid.UUID
}

func (s *Store) CreateAlertAndJob(ctx context.Context, input CreateAlertJobInput) (CreateAlertJobResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CreateAlertJobResult{}, err
	}
	defer tx.Rollback(ctx)

	labelsJSON, err := json.Marshal(input.Labels)
	if err != nil {
		return CreateAlertJobResult{}, err
	}

	var alertID uuid.UUID
	const alertQ = `
INSERT INTO alerts (fingerprint, status, severity, title, body, labels, raw_payload, search_tsv)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, to_tsvector('english', $4 || ' ' || coalesce($5, '')))
RETURNING id`
	err = tx.QueryRow(ctx, alertQ, input.Fingerprint, input.Status, input.Severity, input.Title, input.Body, labelsJSON, input.RawPayload).Scan(&alertID)
	if err != nil {
		return CreateAlertJobResult{}, err
	}

	jobPayload := input.JobPayload
	if jobPayload == nil {
		jobPayload = map[string]any{"alert_id": alertID.String()}
	} else {
		jobPayload["alert_id"] = alertID.String()
	}
	jobJSON, err := json.Marshal(jobPayload)
	if err != nil {
		return CreateAlertJobResult{}, err
	}

	var jobID uuid.UUID
	const jobQ = `INSERT INTO jobs (kind, payload, status, run_at) VALUES ($1, $2, 'pending', now()) RETURNING id`
	err = tx.QueryRow(ctx, jobQ, input.JobKind, jobJSON).Scan(&jobID)
	if err != nil {
		return CreateAlertJobResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CreateAlertJobResult{}, err
	}
	return CreateAlertJobResult{AlertID: alertID, JobID: jobID}, nil
}

func (s *Store) ClaimNextJob(ctx context.Context) (Job, error) {
	const q = `
UPDATE jobs
SET status = 'running', updated_at = now(), attempts = attempts + 1
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'pending' AND run_at <= now()
    ORDER BY run_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, kind, payload, status, run_at, attempts, last_error, created_at, updated_at`
	var job Job
	var payload []byte
	var lastError *string
	err := s.pool.QueryRow(ctx, q).Scan(
		&job.ID, &job.Kind, &payload, &job.Status, &job.RunAt, &job.Attempts, &lastError, &job.CreatedAt, &job.UpdatedAt,
	)
	job.Payload = payload
	job.LastError = lastError
	return job, err
}

func (s *Store) CompleteJob(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status = 'done', updated_at = now(), last_error = NULL WHERE id = $1`, id)
	return err
}

func (s *Store) FailJob(ctx context.Context, id uuid.UUID, lastError string) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status = 'failed', updated_at = now(), last_error = $2 WHERE id = $1`, id, lastError)
	return err
}

func isNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
