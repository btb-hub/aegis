package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestResolveOIDCLoginLinksByEmailWithoutOverwrite(t *testing.T) {
	users := newIdentityMockUsers()
	svc := NewAuthService(testAuthConfig(t), users, &mockSessions{byHash: map[string]db.Session{}}, &sequenceOIDC{
		responses: []*OIDCUserInfo{
			{Sub: "google-1", Email: "person@example.com", DisplayName: "Google Name", AvatarURL: "https://google/avatar.png"},
			{Sub: "slack-1", Email: "person@example.com", DisplayName: "Slack Name", SlackUserID: "U99"},
		},
	})

	token1, googleUser, err := svc.CompleteLogin(context.Background(), "google", "code-1")
	require.NoError(t, err)
	require.Equal(t, "Google Name", googleUser.DisplayName)
	require.NotNil(t, googleUser.AvatarURL)

	token2, slackUser, err := svc.CompleteLogin(context.Background(), "slack", "code-2")
	require.NoError(t, err)
	require.Equal(t, googleUser.ID, slackUser.ID)
	require.Equal(t, "Google Name", slackUser.DisplayName, "display_name must not be overwritten")
	require.Equal(t, "https://google/avatar.png", *slackUser.AvatarURL)
	require.NotNil(t, slackUser.SlackUserID)
	require.Equal(t, "U99", *slackUser.SlackUserID)

	profile, err := svc.CurrentUserProfile(context.Background(), token2)
	require.NoError(t, err)
	require.Len(t, profile.Identities, 2)

	_, err = svc.CurrentUserProfile(context.Background(), token1)
	require.NoError(t, err)
}

func TestResolveOIDCLoginBackfillsEmptyDisplayName(t *testing.T) {
	users := newIdentityMockUsers()
	userID := uuid.New()
	users.users[userID] = db.User{ID: userID, Provider: "google", Email: "empty@example.com", Role: "member", Locale: "en"}
	users.byEmail["empty@example.com"] = userID
	users.identities[identityKey("google", "google-1")] = db.UserIdentity{
		ID: uuid.New(), UserID: userID, Provider: "google", ProviderSub: "google-1", LinkedAt: time.Now(),
	}

	svc := NewAuthService(testAuthConfig(t), users, &mockSessions{byHash: map[string]db.Session{}}, &sequenceOIDC{
		responses: []*OIDCUserInfo{{Sub: "google-1", Email: "empty@example.com", DisplayName: "Filled Later"}},
	})
	_, user, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)
	require.Equal(t, "Filled Later", user.DisplayName)
}

func TestCurrentUserProfileIncludesIdentities(t *testing.T) {
	svc, _, _ := testAuthService(t)
	token, _, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	profile, err := svc.CurrentUserProfile(context.Background(), token)
	require.NoError(t, err)
	require.NotEmpty(t, profile.Identities)
}

func TestUserJSONIncludesLinkedProviders(t *testing.T) {
	avatar := "https://example/a.png"
	slackID := "U1"
	out := UserJSON(db.User{
		ID: uuid.New(), Email: "a@b.c", DisplayName: "A", Role: "member", Locale: "en", Provider: "google",
		AvatarURL: &avatar, SlackUserID: &slackID,
	}, []db.UserIdentity{{Provider: "google", ProviderSub: "g1", LinkedAt: time.Now()}})
	require.Equal(t, avatar, out["avatar_url"])
	require.Equal(t, slackID, out["slack_user_id"])
	require.NotNil(t, out["identities"])
}

type sequenceOIDC struct {
	responses []*OIDCUserInfo
	index     int
}

func (s *sequenceOIDC) AuthCodeURL(string, string) (string, error) {
	return "https://idp/authorize", nil
}

func (s *sequenceOIDC) Exchange(context.Context, string, string) (*OIDCUserInfo, error) {
	if s.index >= len(s.responses) {
		return &OIDCUserInfo{Sub: "fallback"}, nil
	}
	info := s.responses[s.index]
	s.index++
	return info, nil
}

func TestUserJSONIncludesExpressHuid(t *testing.T) {
	huid := uuid.New()
	out := UserJSON(db.User{
		ID: uuid.New(), Email: "a@b.c", DisplayName: "A", Role: "member", Locale: "en", Provider: "express",
		ExpressUserHuid: db.ExpressHuidToPg(huid),
	}, nil)
	require.Equal(t, huid.String(), out["express_user_huid"])
}

func TestCurrentUserProfileUnauthorized(t *testing.T) {
	svc, _, _ := testAuthService(t)
	_, err := svc.CurrentUserProfile(context.Background(), "")
	require.Error(t, err)
}

func TestIdentitiesJSONEmptySlice(t *testing.T) {
	items := IdentitiesJSON([]db.UserIdentity{})
	require.Empty(t, items)
}

func TestCompleteLoginResolveFailure(t *testing.T) {
	users := newIdentityMockUsers()
	users.resolveErr = errors.New("resolve failed")
	svc := NewAuthService(testAuthConfig(t), users, &mockSessions{byHash: map[string]db.Session{}}, &mockOIDC{})
	_, _, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.Error(t, err)
}

func TestCurrentUserProfileInvalidSession(t *testing.T) {
	svc, _, _ := testAuthService(t)
	_, err := svc.CurrentUserProfile(context.Background(), "not-a-real-token")
	require.Error(t, err)
}

func testAuthConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		SessionTTL: 24 * time.Hour,
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
			"slack":  {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
}
