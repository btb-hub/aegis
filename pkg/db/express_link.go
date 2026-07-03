package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) GetUserByExpressHuid(ctx context.Context, expressHuid uuid.UUID) (User, error) {
	q := `SELECT ` + userSelectColumns + ` FROM users WHERE express_user_huid = $1`
	return scanUser(s.pool.QueryRow(ctx, q, expressHuid))
}

func (s *Store) UpdateUserExpressHuid(ctx context.Context, userID, expressHuid uuid.UUID) (User, error) {
	q := `
UPDATE users SET express_user_huid = $2
WHERE id = $1
RETURNING ` + userSelectColumns
	return scanUser(s.pool.QueryRow(ctx, q, userID, expressHuid))
}

func (s *Store) CreateExpressLinkCode(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	code, err := randomLinkCode()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(ttl)
	_, err = s.pool.Exec(ctx, `
INSERT INTO express_link_codes (code, user_id, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (code) DO UPDATE SET user_id = EXCLUDED.user_id, expires_at = EXCLUDED.expires_at`,
		code, userID, expiresAt,
	)
	return code, err
}

func (s *Store) RedeemExpressLinkCode(ctx context.Context, code string, expressHuid uuid.UUID) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	err = tx.QueryRow(ctx, `
SELECT user_id FROM express_link_codes
WHERE code = $1 AND expires_at > now()
FOR UPDATE`, code).Scan(&userID)
	if err != nil {
		if isNoRows(err) {
			return User{}, fmt.Errorf("link code invalid or expired")
		}
		return User{}, err
	}

	var user User
	err = tx.QueryRow(ctx, `
UPDATE users SET express_user_huid = $2
WHERE id = $1
RETURNING `+userSelectColumns,
		userID, expressHuid,
	).Scan(
		&user.ID, &user.Provider, &user.ProviderSub, &user.Email, &user.DisplayName,
		&user.Role, &user.Locale, &user.AvatarURL, &user.SlackUserID, &user.ExpressUserHuid, &user.CreatedAt,
	)
	if err != nil {
		return User{}, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM express_link_codes WHERE code = $1`, code); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func ExpressHuidString(user User) *string {
	if !user.ExpressUserHuid.Valid {
		return nil
	}
	id := uuid.UUID(user.ExpressUserHuid.Bytes)
	s := id.String()
	return &s
}

func ParseExpressHuid(raw string) (uuid.UUID, error) {
	return uuid.Parse(raw)
}

func ExpressHuidToPg(huid uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: huid, Valid: true}
}

func randomLinkCode() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
