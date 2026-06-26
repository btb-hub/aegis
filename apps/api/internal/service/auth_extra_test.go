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

func TestAuthLoginStateFailure(t *testing.T) {
	svc, _, _ := testAuthService(t)
	svc.newState = func() (string, error) { return "", errors.New("state failed") }
	_, _, err := svc.LoginURL("google")
	require.Error(t, err)
}

func TestCompleteLoginExchangeFailure(t *testing.T) {
	cfg := testOAuthConfig(t)
	users := &mockUsers{users: map[uuid.UUID]db.User{}}
	sessions := &mockSessions{byHash: map[string]db.Session{}}
	svc := NewAuthService(cfg, users, sessions, failOIDC{})

	_, _, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.Error(t, err)
}

func TestCompleteLoginSessionFailure(t *testing.T) {
	cfg := testOAuthConfig(t)
	users := &mockUsers{users: map[uuid.UUID]db.User{}}
	sessions := &failSessions{}
	svc := NewAuthService(cfg, users, sessions, &mockOIDC{})

	_, _, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.Error(t, err)
}

type failOIDC struct{}

func (f failOIDC) AuthCodeURL(string, string) (string, error) {
	return "", errors.New("auth url failed")
}

func (f failOIDC) Exchange(context.Context, string, string) (*OIDCUserInfo, error) {
	return nil, errors.New("exchange failed")
}

type failSessions struct{ mockSessions }

func (f *failSessions) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.Session, error) {
	return db.Session{}, errors.New("session create failed")
}

func TestLoginURLOIDCAuthFailure(t *testing.T) {
	cfg := testOAuthConfig(t)
	users := &mockUsers{users: map[uuid.UUID]db.User{}}
	sessions := &mockSessions{byHash: map[string]db.Session{}}
	svc := NewAuthService(cfg, users, sessions, failOIDC{})

	_, _, err := svc.LoginURL("google")
	require.Error(t, err)
}

func TestAuthUpdateLocaleUnauthorized(t *testing.T) {
	svc, _, _ := testAuthService(t)
	_, err := svc.UpdateLocale(context.Background(), "", "ru")
	require.Error(t, err)
}

func testOAuthConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		SessionTTL: 24 * time.Hour,
		OIDC: map[string]config.OIDCProvider{
			"google": {
				ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb",
				Issuer: "https://accounts.google.com",
			},
		},
	}
}
