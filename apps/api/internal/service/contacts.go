package service

import (
	"net/url"
	"strings"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func ContactLinks(email string, slackUserID *string, expressHuid pgtype.UUID) map[string]string {
	contacts := map[string]string{}
	email = strings.TrimSpace(email)
	if email != "" {
		contacts["email"] = "mailto:" + email
	}
	if slackUserID != nil {
		id := strings.TrimSpace(*slackUserID)
		if id != "" {
			contacts["slack"] = "https://slack.com/app_redirect?channel=" + url.QueryEscape(id)
		}
	}
	if expressHuid.Valid {
		contacts["express"] = "https://xlnk.ms/open/profile/" + uuid.UUID(expressHuid.Bytes).String()
	}
	return contacts
}

func PersonJSON(userID uuid.UUID, email, displayName string, slackUserID *string, expressHuid pgtype.UUID) map[string]any {
	return map[string]any{
		"user_id":      userID.String(),
		"email":        email,
		"display_name": displayName,
		"contacts":     ContactLinks(email, slackUserID, expressHuid),
	}
}

func AssigneeJSON(user db.User) map[string]any {
	return PersonJSON(user.ID, user.Email, user.DisplayName, user.SlackUserID, user.ExpressUserHuid)
}
