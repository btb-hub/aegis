package db

import (
	"github.com/jackc/pgx/v5"
)

const userSelectColumns = `id, provider, provider_sub, email, display_name, role, locale, avatar_url, slack_user_id, express_user_huid, created_at`

func scanUser(row pgx.Row) (User, error) {
	var user User
	err := row.Scan(
		&user.ID, &user.Provider, &user.ProviderSub, &user.Email, &user.DisplayName,
		&user.Role, &user.Locale, &user.AvatarURL, &user.SlackUserID, &user.ExpressUserHuid, &user.CreatedAt,
	)
	return user, err
}

func scanUserRow(rows pgx.Rows) (User, error) {
	var user User
	err := rows.Scan(
		&user.ID, &user.Provider, &user.ProviderSub, &user.Email, &user.DisplayName,
		&user.Role, &user.Locale, &user.AvatarURL, &user.SlackUserID, &user.ExpressUserHuid, &user.CreatedAt,
	)
	return user, err
}
