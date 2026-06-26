package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockUsers struct {
	users map[uuid.UUID]db.User
}

func (m *mockUsers) UpsertUser(ctx context.Context, provider, providerSub, email, displayName, role, locale string) (db.User, error) {
	user := db.User{
		ID:          uuid.New(),
		Provider:    provider,
		ProviderSub: providerSub,
		Email:       email,
		DisplayName: displayName,
		Role:        role,
		Locale:      locale,
	}
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUsers) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	user, ok := m.users[id]
	if !ok {
		return db.User{}, errors.New("not found")
	}
	return user, nil
}

func (m *mockUsers) UpdateUserLocale(ctx context.Context, id uuid.UUID, locale string) (db.User, error) {
	user, ok := m.users[id]
	if !ok {
		return db.User{}, errors.New("not found")
	}
	user.Locale = locale
	m.users[id] = user
	return user, nil
}

type mockSessions struct {
	byHash map[string]db.Session
}

func (m *mockSessions) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.Session, error) {
	session := db.Session{ID: uuid.New(), UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt}
	m.byHash[tokenHash] = session
	return session, nil
}

func (m *mockSessions) GetSessionByTokenHash(ctx context.Context, tokenHash string) (db.Session, error) {
	session, ok := m.byHash[tokenHash]
	if !ok {
		return db.Session{}, errors.New("not found")
	}
	return session, nil
}

func (m *mockSessions) DeleteSession(ctx context.Context, tokenHash string) error {
	delete(m.byHash, tokenHash)
	return nil
}

type mockOIDC struct{}

func (m *mockOIDC) AuthCodeURL(provider, state string) (string, error) {
	return "https://idp.example/authorize?state=" + state, nil
}

func (m *mockOIDC) Exchange(ctx context.Context, provider, code string) (*OIDCUserInfo, error) {
	return &OIDCUserInfo{Sub: "sub-1", Email: "a@example.com", DisplayName: "A"}, nil
}

func testAuthService(t *testing.T) (*AuthService, *mockUsers, *mockSessions) {
	t.Helper()
	cfg := &config.Config{
		SessionTTL: 24 * time.Hour,
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
	users := &mockUsers{users: map[uuid.UUID]db.User{}}
	sessions := &mockSessions{byHash: map[string]db.Session{}}
	svc := NewAuthService(cfg, users, sessions, &mockOIDC{})
	return svc, users, sessions
}

func TestAuthLoginURL(t *testing.T) {
	svc, _, _ := testAuthService(t)
	url, state, err := svc.LoginURL("google")
	require.NoError(t, err)
	require.NotEmpty(t, state)
	require.Contains(t, url, state)
}

func TestAuthLoginUnknownProvider(t *testing.T) {
	svc, _, _ := testAuthService(t)
	_, _, err := svc.LoginURL("unknown")
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestAuthCompleteLoginCreatesSession(t *testing.T) {
	svc, _, sessions := testAuthService(t)
	token, user, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, user.ID)
	require.Len(t, sessions.byHash, 1)
}

func TestAuthCurrentUserAndLogout(t *testing.T) {
	svc, _, _ := testAuthService(t)
	token, _, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	user, err := svc.CurrentUser(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, "a@example.com", user.Email)

	require.NoError(t, svc.Logout(context.Background(), token))
	_, err = svc.CurrentUser(context.Background(), token)
	require.Error(t, err)
}

func TestAuthUpdateLocaleInvalid(t *testing.T) {
	svc, _, _ := testAuthService(t)
	token, _, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	_, err = svc.UpdateLocale(context.Background(), token, "de")
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "INVALID_LOCALE", appErr.Code)
}

func TestAuthUpdateLocaleSuccess(t *testing.T) {
	svc, _, _ := testAuthService(t)
	token, _, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	user, err := svc.UpdateLocale(context.Background(), token, "ru")
	require.NoError(t, err)
	require.Equal(t, "ru", user.Locale)
}

func TestUserJSON(t *testing.T) {
	id := uuid.New()
	data, err := MarshalUser(db.User{ID: id, Email: "x@y.z", Role: "member", Locale: "en", Provider: "google"})
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, id.String(), out["id"])
}
