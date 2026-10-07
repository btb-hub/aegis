package db

import (
	"context"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PagingAuthorization struct {
	StateHash   string
	UserID      uuid.UUID
	SessionHash string
	Provider    string
	Nonce       string
	ExpiresAt   time.Time
}

// All paging writers take this lock before locking users or authorization rows.
// Legacy duplicate identities are preserved, but new assignments cannot add duplicates.
func lockPagingProvider(ctx context.Context, tx pgx.Tx, provider string) error {
	var key int
	switch provider {
	case "slack":
		key = 1
	case "express":
		key = 2
	default:
		return apperrors.Validation("unknown paging provider", nil)
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(170119, $1)`, key)
	return err
}

func (s *Store) CreatePagingAuthorization(ctx context.Context, attempt PagingAuthorization) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockPagingProvider(ctx, tx, attempt.Provider); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM paging_authorizations WHERE (user_id = $1 AND provider = $2) OR expires_at <= now()`, attempt.UserID, attempt.Provider); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `INSERT INTO paging_authorizations (state_hash, user_id, session_hash, provider, nonce, expires_at)
        SELECT $1, $2, $3, $4, $5, $6 FROM sessions WHERE token_hash = $3 AND user_id = $2 AND expires_at > now()`,
		attempt.StateHash, attempt.UserID, attempt.SessionHash, attempt.Provider, attempt.Nonce, attempt.ExpiresAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return apperrors.Unauthorized("invalid session")
	}
	return tx.Commit(ctx)
}

// Claim commits before the external exchange, so a callback cannot be replayed.
// The row remains until completion, allowing Disconnect to invalidate an in-flight exchange.
func (s *Store) ClaimPagingAuthorization(ctx context.Context, stateHash, sessionHash, provider string) (PagingAuthorization, error) {
	var a PagingAuthorization
	err := s.pool.QueryRow(ctx, `UPDATE paging_authorizations a SET exchanging = true
        FROM sessions s WHERE a.state_hash = $1 AND a.session_hash = $2 AND a.provider = $3
        AND NOT a.exchanging AND a.expires_at > now() AND s.token_hash = a.session_hash
        AND s.user_id = a.user_id AND s.expires_at > now()
        RETURNING a.state_hash, a.user_id, a.session_hash, a.provider, a.nonce, a.expires_at`,
		stateHash, sessionHash, provider).Scan(&a.StateHash, &a.UserID, &a.SessionHash, &a.Provider, &a.Nonce, &a.ExpiresAt)
	if isNoRows(err) {
		return a, apperrors.Validation("paging authorization invalid or expired", nil)
	}
	return a, err
}

func (s *Store) DeletePagingAuthorization(ctx context.Context, stateHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM paging_authorizations WHERE state_hash = $1`, stateHash)
	return err
}

func (s *Store) CompletePagingAuthorization(ctx context.Context, a PagingAuthorization, identity string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockPagingProvider(ctx, tx, a.Provider); err != nil {
		return err
	}
	var userID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT a.user_id FROM paging_authorizations a JOIN sessions s ON s.token_hash = a.session_hash
        WHERE a.state_hash = $1 AND a.session_hash = $2 AND a.provider = $3 AND a.user_id = $4
        AND a.exchanging AND a.expires_at > now() AND s.expires_at > now() AND s.user_id = a.user_id
        FOR UPDATE OF a, s`, a.StateHash, a.SessionHash, a.Provider, a.UserID).Scan(&userID)
	if isNoRows(err) {
		return apperrors.Validation("paging authorization invalid or expired", nil)
	}
	if err != nil {
		return err
	}
	if _, err := setPagingIdentityTx(ctx, tx, userID, a.Provider, identity, true); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) DisconnectPagingIdentity(ctx context.Context, userID uuid.UUID, provider string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockPagingProvider(ctx, tx, provider); err != nil {
		return err
	}
	if _, err := setPagingIdentityTx(ctx, tx, userID, provider, "", true); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func checkPagingOwnerTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, provider, identity string) error {
	if identity == "" {
		return nil
	}
	column := "slack_user_id"
	if provider == "express" {
		column = "express_user_huid"
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE `+column+` = $1 AND id <> $2)`, identity, userID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return apperrors.Conflict("messenger identity is already connected to another account")
	}
	return nil
}

// Caller must hold the provider lock. Explicit writes also invalidate earlier setup flows.
func setPagingIdentityTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, provider, identity string, explicit bool) (User, error) {
	if err := checkPagingOwnerTx(ctx, tx, userID, provider, identity); err != nil {
		return User{}, err
	}
	column := "slack_user_id"
	if provider == "express" {
		column = "express_user_huid"
	}
	user, err := scanUser(tx.QueryRow(ctx, `UPDATE users SET `+column+` = NULLIF($2, '')`+pagingCast(provider)+` WHERE id = $1 RETURNING `+userSelectColumns, userID, identity))
	if err != nil {
		return User{}, err
	}
	if explicit {
		if _, err := tx.Exec(ctx, `INSERT INTO user_paging_settings (user_id, provider) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, provider); err != nil {
			return User{}, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM paging_authorizations WHERE user_id = $1 AND provider = $2`, userID, provider); err != nil {
			return User{}, err
		}
		if provider == "express" {
			if _, err := tx.Exec(ctx, `DELETE FROM express_link_codes WHERE user_id = $1`, userID); err != nil {
				return User{}, err
			}
		}
		action := "paging.connected"
		if identity == "" {
			action = "paging.disconnected"
		}
		if err := writeAuditLogTx(ctx, tx, userID, action, "user", userID, map[string]any{"provider": provider}); err != nil {
			return User{}, err
		}
	}
	return user, nil
}

func pagingCast(provider string) string {
	if provider == "express" {
		return "::uuid"
	}
	return ""
}

func autoBindSlackTx(ctx context.Context, tx pgx.Tx, user User, identity string) (User, error) {
	if identity == "" || (user.SlackUserID != nil && *user.SlackUserID != "") {
		return user, nil
	}
	var managed bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_paging_settings WHERE user_id = $1 AND provider = 'slack')`, user.ID).Scan(&managed); err != nil {
		return User{}, err
	}
	if managed {
		return user, nil
	}
	return setPagingIdentityTx(ctx, tx, user.ID, "slack", identity, false)
}
