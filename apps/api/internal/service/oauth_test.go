package service

import (
	"context"
	"testing"

	"github.com/aegis/aegis/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestOAuthTokenExchangerUnconfigured(t *testing.T) {
	cfg := &config.Config{OIDC: map[string]config.OIDCProvider{}}
	ex := NewOAuthTokenExchanger(cfg)
	_, err := ex.AuthCodeURL("google", "state")
	require.Error(t, err)
}

func TestOAuthAuthCodeURLSuccess(t *testing.T) {
	cfg := testOAuthConfig(t)
	ex := NewOAuthTokenExchanger(cfg)
	url, err := ex.AuthCodeURL("google", "state-123")
	require.NoError(t, err)
	require.Contains(t, url, "state-123")
}

func TestAuthLogoutMissingSession(t *testing.T) {
	svc, _, _ := testAuthService(t)
	err := svc.Logout(context.Background(), "")
	require.Error(t, err)
}

func TestAuthCurrentUserMissingSession(t *testing.T) {
	svc, _, _ := testAuthService(t)
	_, err := svc.CurrentUser(context.Background(), "")
	require.Error(t, err)
}
