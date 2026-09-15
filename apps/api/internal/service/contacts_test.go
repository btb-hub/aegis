package service

import (
	"testing"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestContactLinksAllChannels(t *testing.T) {
	huid := uuid.MustParse("83fbf1c7-f14b-5176-bd32-ca15cf00d4b7")
	slackID := "U123ABC"
	contacts := ContactLinks("alice@example.com", &slackID, db.ExpressHuidToPg(huid))
	require.Equal(t, "mailto:alice@example.com", contacts["email"])
	require.Equal(t, "https://slack.com/app_redirect?channel=U123ABC", contacts["slack"])
	require.Equal(t, "https://xlnk.ms/open/profile/83fbf1c7-f14b-5176-bd32-ca15cf00d4b7", contacts["express"])
}

func TestContactLinksOmitsMissingChannels(t *testing.T) {
	empty := ""
	var unset pgtype.UUID
	contacts := ContactLinks("  bob@example.com  ", &empty, unset)
	require.Equal(t, map[string]string{"email": "mailto:bob@example.com"}, contacts)

	blank := ContactLinks("  ", nil, unset)
	require.Empty(t, blank)
}

func TestContactLinksEscapesSlackID(t *testing.T) {
	slackID := "U12 3"
	var unset pgtype.UUID
	contacts := ContactLinks("", &slackID, unset)
	require.Equal(t, "https://slack.com/app_redirect?channel=U12+3", contacts["slack"])
	_, hasEmail := contacts["email"]
	require.False(t, hasEmail)
}

func TestAssigneeJSON(t *testing.T) {
	userID := uuid.New()
	slackID := "U9"
	huid := uuid.MustParse("6fafda2c-6505-57a5-a088-25ea5d1d0364")
	payload := AssigneeJSON(db.User{
		ID:              userID,
		Email:           "a@example.com",
		DisplayName:     "Alice",
		SlackUserID:     &slackID,
		ExpressUserHuid: db.ExpressHuidToPg(huid),
	})
	require.Equal(t, userID.String(), payload["user_id"])
	require.Equal(t, "Alice", payload["display_name"])
	contacts, ok := payload["contacts"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, "mailto:a@example.com", contacts["email"])
	require.Equal(t, "https://slack.com/app_redirect?channel=U9", contacts["slack"])
	require.Contains(t, contacts["express"], huid.String())
}
