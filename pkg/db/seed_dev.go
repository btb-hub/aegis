package db

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// SeedDevUser describes one idempotent local dev directory user.
type SeedDevUser struct {
	Provider        string
	ProviderSub     string
	Email           string
	DisplayName     string
	Role            string
	Locale          string
	AvatarURL       string
	SlackUserID     string
	ExpressUserHuid uuid.UUID
}

// DevSeedUsers is the canonical set of SSO-like users for local testing.
func DevSeedUsers() []SeedDevUser {
	return []SeedDevUser{
		{
			Provider:    "google",
			ProviderSub: "seed-google-alice",
			Email:       "alice@seed.local",
			DisplayName: "Alice Google",
			Role:        "member",
			Locale:      "en",
			AvatarURL:   "https://ui-avatars.com/api/?name=Alice+Google&size=128",
		},
		{
			Provider:    "slack",
			ProviderSub: "seed-slack-bob",
			Email:       "bob@seed.local",
			DisplayName: "Bob Slack",
			Role:        "member",
			Locale:      "en",
			AvatarURL:   "https://ui-avatars.com/api/?name=Bob+Slack&size=128",
			SlackUserID: "U0SEEDBOB",
		},
		{
			Provider:        "express",
			ProviderSub:     "seed-express-carol",
			Email:           "carol@seed.local",
			DisplayName:     "Carol eXpress",
			Role:            "member",
			Locale:          "ru",
			AvatarURL:       "https://ui-avatars.com/api/?name=Carol+Express&size=128",
			ExpressUserHuid: uuid.MustParse("00000000-0000-4000-8000-000000000003"),
		},
		{
			Provider:    "dev",
			ProviderSub: "dev-local",
			Email:       "dev@localhost",
			DisplayName: "Local Admin",
			Role:        "admin",
			Locale:      "en",
			AvatarURL:   "https://ui-avatars.com/api/?name=Local+Admin&size=128",
		},
	}
}

func (s *Store) SeedDevUsers(ctx context.Context) ([]User, error) {
	seeds := DevSeedUsers()
	users := make([]User, 0, len(seeds))
	for _, seed := range seeds {
		user, err := s.UpsertSeedDevUser(ctx, seed)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *Store) UpsertSeedDevUser(ctx context.Context, input SeedDevUser) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	user, err := upsertSeedDevUserTx(ctx, tx, input)
	if err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func upsertSeedDevUserTx(ctx context.Context, tx pgx.Tx, input SeedDevUser) (User, error) {
	avatarURL := optionalString(input.AvatarURL)
	slackUserID := optionalString(input.SlackUserID)
	expressHuid := ExpressHuidToPg(input.ExpressUserHuid)

	userID, err := findUserIDByIdentityTx(ctx, tx, input.Provider, input.ProviderSub)
	if err != nil {
		return User{}, err
	}

	var user User
	if userID != uuid.Nil {
		const q = `
UPDATE users SET
    email = $2,
    display_name = $3,
    role = $4,
    locale = $5,
    avatar_url = $6,
    slack_user_id = $7,
    express_user_huid = CASE WHEN $8::uuid IS NULL THEN express_user_huid ELSE $8 END
WHERE id = $1
RETURNING ` + userSelectColumns
		user, err = scanUser(tx.QueryRow(
			ctx, q, userID, strings.TrimSpace(input.Email), strings.TrimSpace(input.DisplayName),
			input.Role, input.Locale, avatarURL, slackUserID, expressHuid,
		))
	} else {
		const q = `
INSERT INTO users (provider, provider_sub, email, display_name, role, locale, avatar_url, slack_user_id, express_user_huid)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING ` + userSelectColumns
		user, err = scanUser(tx.QueryRow(
			ctx, q, input.Provider, input.ProviderSub, strings.TrimSpace(input.Email),
			strings.TrimSpace(input.DisplayName), input.Role, input.Locale,
			avatarURL, slackUserID, expressHuid,
		))
		if err != nil {
			return User{}, err
		}
		if err := insertIdentityTx(ctx, tx, user.ID, input.Provider, input.ProviderSub); err != nil {
			return User{}, err
		}
		return user, nil
	}
	if err != nil {
		return User{}, err
	}

	_, err = tx.Exec(ctx, `
INSERT INTO user_identities (user_id, provider, provider_sub)
VALUES ($1, $2, $3)
ON CONFLICT (provider, provider_sub) DO NOTHING`, user.ID, input.Provider, input.ProviderSub)
	return user, err
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
