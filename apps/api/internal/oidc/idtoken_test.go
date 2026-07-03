package oidc

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIDTokenPayloadGoogle(t *testing.T) {
	claims := map[string]any{
		"sub":     "google-sub-1",
		"email":   "alice@example.com",
		"name":    "Alice Example",
		"picture": "https://cdn.example/avatar.png",
	}
	token := testIDToken(t, claims)

	parsed, err := ParseIDTokenPayload(token)
	require.NoError(t, err)
	info := UserInfoFromClaims("google", parsed)
	require.Equal(t, "google-sub-1", info.Sub)
	require.Equal(t, "alice@example.com", info.Email)
	require.Equal(t, "Alice Example", info.DisplayName)
	require.Equal(t, "https://cdn.example/avatar.png", info.AvatarURL)
}

func TestUserInfoFromClaimsSlack(t *testing.T) {
	claims := map[string]any{
		"sub":                       "slack-sub-1",
		"email":                     "bob@example.com",
		"name":                      "Bob Slack",
		"https://slack.com/user_id": "U01234567",
	}
	info := UserInfoFromClaims("slack", claims)
	require.Equal(t, "slack-sub-1", info.Sub)
	require.Equal(t, "U01234567", info.SlackUserID)
}

func TestUserInfoFromClaimsDoesNotOverwriteWithEmptyName(t *testing.T) {
	info := UserInfoFromClaims("google", map[string]any{"sub": "x"})
	require.Equal(t, "x", info.Sub)
	require.Equal(t, "google user", info.DisplayName)
}

func TestParseIDTokenPayloadInvalid(t *testing.T) {
	_, err := ParseIDTokenPayload("not-a-jwt")
	require.Error(t, err)
}

func testIDToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return "header." + encoded + ".signature"
}
