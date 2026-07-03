package db

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OIDCLoginInput struct {
	Provider    string
	ProviderSub string
	Email       string
	DisplayName string
	AvatarURL   string
	SlackUserID string
}

type OIDCLoginResult struct {
	User              User
	Identities        []UserIdentity
	NewIdentityLinked bool
}

func (s *Store) ResolveOIDCLogin(ctx context.Context, input OIDCLoginInput) (OIDCLoginResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OIDCLoginResult{}, err
	}
	defer tx.Rollback(ctx)

	result, err := resolveOIDCLoginTx(ctx, tx, input)
	if err != nil {
		return OIDCLoginResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return OIDCLoginResult{}, err
	}
	return result, nil
}

func resolveOIDCLoginTx(ctx context.Context, tx pgx.Tx, input OIDCLoginInput) (OIDCLoginResult, error) {
	identityUserID, err := findUserIDByIdentityTx(ctx, tx, input.Provider, input.ProviderSub)
	if err != nil {
		return OIDCLoginResult{}, err
	}

	newIdentityLinked := false
	var user User

	if identityUserID != uuid.Nil {
		user, err = getUserByIDTx(ctx, tx, identityUserID)
		if err != nil {
			return OIDCLoginResult{}, err
		}
		user, err = backfillUserProfileTx(ctx, tx, user.ID, input)
		if err != nil {
			return OIDCLoginResult{}, err
		}
	} else {
		email := normalizeEmail(input.Email)
		existingID := uuid.Nil
		if email != "" {
			existingID, err = findUserIDByEmailTx(ctx, tx, email)
			if err != nil {
				return OIDCLoginResult{}, err
			}
		}

		if existingID != uuid.Nil {
			user, err = getUserByIDTx(ctx, tx, existingID)
			if err != nil {
				return OIDCLoginResult{}, err
			}
			if err := insertIdentityTx(ctx, tx, user.ID, input.Provider, input.ProviderSub); err != nil {
				return OIDCLoginResult{}, err
			}
			newIdentityLinked = true
			user, err = backfillUserProfileTx(ctx, tx, user.ID, input)
			if err != nil {
				return OIDCLoginResult{}, err
			}
			if err := writeAuditLogTx(ctx, tx, user.ID, "identity.linked", "user", user.ID, map[string]any{
				"provider":     input.Provider,
				"provider_sub": input.ProviderSub,
				"via":          "email_match",
			}); err != nil {
				return OIDCLoginResult{}, err
			}
		} else {
			user, err = createUserWithIdentityTx(ctx, tx, input)
			if err != nil {
				return OIDCLoginResult{}, err
			}
			newIdentityLinked = true
			if err := writeAuditLogTx(ctx, tx, user.ID, "identity.linked", "user", user.ID, map[string]any{
				"provider":     input.Provider,
				"provider_sub": input.ProviderSub,
				"via":          "signup",
			}); err != nil {
				return OIDCLoginResult{}, err
			}
		}
	}

	identities, err := listUserIdentitiesTx(ctx, tx, user.ID)
	if err != nil {
		return OIDCLoginResult{}, err
	}

	return OIDCLoginResult{
		User:              user,
		Identities:        identities,
		NewIdentityLinked: newIdentityLinked,
	}, nil
}

func (s *Store) ListUserIdentities(ctx context.Context, userID uuid.UUID) ([]UserIdentity, error) {
	const q = `
SELECT id, user_id, provider, provider_sub, linked_at
FROM user_identities
WHERE user_id = $1
ORDER BY linked_at`
	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identities []UserIdentity
	for rows.Next() {
		var identity UserIdentity
		if err := rows.Scan(&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderSub, &identity.LinkedAt); err != nil {
			return nil, err
		}
		identities = append(identities, identity)
	}
	return identities, rows.Err()
}

func (s *Store) EnsureDevIdentity(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO user_identities (user_id, provider, provider_sub)
VALUES ($1, 'dev', 'dev-local')
ON CONFLICT (provider, provider_sub) DO NOTHING`, userID)
	return err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func findUserIDByIdentityTx(ctx context.Context, tx pgx.Tx, provider, providerSub string) (uuid.UUID, error) {
	const q = `
SELECT user_id FROM user_identities
WHERE provider = $1 AND provider_sub = $2`
	var userID uuid.UUID
	err := tx.QueryRow(ctx, q, provider, providerSub).Scan(&userID)
	if isNoRows(err) {
		return uuid.Nil, nil
	}
	return userID, err
}

func findUserIDByEmailTx(ctx context.Context, tx pgx.Tx, email string) (uuid.UUID, error) {
	const q = `SELECT id FROM users WHERE lower(email) = $1 LIMIT 1`
	var userID uuid.UUID
	err := tx.QueryRow(ctx, q, email).Scan(&userID)
	if isNoRows(err) {
		return uuid.Nil, nil
	}
	return userID, err
}

func getUserByIDTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (User, error) {
	q := `SELECT ` + userSelectColumns + ` FROM users WHERE id = $1`
	return scanUser(tx.QueryRow(ctx, q, id))
}

func insertIdentityTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, provider, providerSub string) error {
	_, err := tx.Exec(ctx, `
INSERT INTO user_identities (user_id, provider, provider_sub)
VALUES ($1, $2, $3)`, userID, provider, providerSub)
	return err
}

func createUserWithIdentityTx(ctx context.Context, tx pgx.Tx, input OIDCLoginInput) (User, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	email := strings.TrimSpace(input.Email)
	var avatarURL *string
	if v := strings.TrimSpace(input.AvatarURL); v != "" {
		avatarURL = &v
	}
	var slackUserID *string
	if v := strings.TrimSpace(input.SlackUserID); v != "" {
		slackUserID = &v
	}

	const q = `
INSERT INTO users (provider, provider_sub, email, display_name, role, locale, avatar_url, slack_user_id)
VALUES ($1, $2, $3, $4, 'member', 'en', $5, $6)
RETURNING ` + userSelectColumns
	user, err := scanUser(tx.QueryRow(ctx, q, input.Provider, input.ProviderSub, email, displayName, avatarURL, slackUserID))
	if err != nil {
		return User{}, err
	}
	if err := insertIdentityTx(ctx, tx, user.ID, input.Provider, input.ProviderSub); err != nil {
		return User{}, err
	}
	return user, nil
}

func backfillUserProfileTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, input OIDCLoginInput) (User, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	avatarURL := strings.TrimSpace(input.AvatarURL)
	slackUserID := strings.TrimSpace(input.SlackUserID)

	const q = `
UPDATE users SET
    display_name = CASE
        WHEN display_name = '' AND $2 <> '' THEN $2
        ELSE display_name
    END,
    avatar_url = COALESCE(avatar_url, NULLIF($3, '')),
    slack_user_id = COALESCE(slack_user_id, NULLIF($4, ''))
WHERE id = $1
RETURNING ` + userSelectColumns
	return scanUser(tx.QueryRow(ctx, q, userID, displayName, avatarURL, slackUserID))
}

func listUserIdentitiesTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) ([]UserIdentity, error) {
	const q = `
SELECT id, user_id, provider, provider_sub, linked_at
FROM user_identities
WHERE user_id = $1
ORDER BY linked_at`
	rows, err := tx.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identities []UserIdentity
	for rows.Next() {
		var identity UserIdentity
		if err := rows.Scan(&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderSub, &identity.LinkedAt); err != nil {
			return nil, err
		}
		identities = append(identities, identity)
	}
	return identities, rows.Err()
}

func writeAuditLogTx(ctx context.Context, tx pgx.Tx, actorID uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO audit_log (actor_id, action, resource_type, resource_id, details)
VALUES ($1, $2, $3, $4, $5)`, actorID, action, resourceType, resourceID, payload)
	return err
}
