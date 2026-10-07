package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/oidc"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type pagingHandlerRepo struct {
	users   *authMockUsers
	items   map[string]db.Integration
	attempt db.PagingAuthorization
	claimed bool
	err     error
}

func (r *pagingHandlerRepo) GetIntegrationByKind(_ context.Context, provider string) (db.Integration, error) {
	return r.items[provider], r.err
}
func (r *pagingHandlerRepo) CreatePagingAuthorization(_ context.Context, a db.PagingAuthorization) error {
	r.attempt = a
	r.claimed = false
	return r.err
}
func (r *pagingHandlerRepo) ClaimPagingAuthorization(_ context.Context, state, session, provider string) (db.PagingAuthorization, error) {
	if r.claimed || r.attempt.StateHash != state || r.attempt.SessionHash != session || r.attempt.Provider != provider {
		return db.PagingAuthorization{}, errors.New("invalid")
	}
	r.claimed = true
	return r.attempt, nil
}
func (r *pagingHandlerRepo) DeletePagingAuthorization(_ context.Context, _ string) error { return nil }
func (r *pagingHandlerRepo) CompletePagingAuthorization(_ context.Context, a db.PagingAuthorization, identity string) error {
	if r.err != nil {
		return r.err
	}
	user := r.users.users[a.UserID]
	if a.Provider == "slack" {
		user.SlackUserID = &identity
	} else {
		user.ExpressUserHuid = db.ExpressHuidToPg(uuid.MustParse(identity))
	}
	r.users.users[user.ID] = user
	return nil
}
func (r *pagingHandlerRepo) DisconnectPagingIdentity(_ context.Context, id uuid.UUID, provider string) error {
	if r.err != nil {
		return r.err
	}
	user := r.users.users[id]
	if provider == "slack" {
		user.SlackUserID = nil
	} else {
		user.ExpressUserHuid.Valid = false
	}
	r.users.users[id] = user
	r.claimed = true
	return nil
}

type pagingHandlerExchange struct{ info oidc.PagingIdentity }

func (x pagingHandlerExchange) AuthorizationURL(_ context.Context, provider, state, nonce, _ string) (string, error) {
	return "https://provider.test/authorize?" + url.Values{"state": {state}, "nonce": {nonce}, "provider": {provider}}.Encode(), nil
}
func (x pagingHandlerExchange) ExchangePaging(_ context.Context, _, _, _ string) (oidc.PagingIdentity, error) {
	return x.info, nil
}

func pagingRouter(t *testing.T) (*gin.Engine, *pagingHandlerRepo, string, uuid.UUID) {
	t.Helper()
	bot := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"team_id":"T123"}`))
		case "/api/v2/botx/bots/bot/token":
			_, _ = w.Write([]byte(`{"status":"ok","result":"token"}`))
		case "/api/v3/botx/users/by_email":
			_, _ = w.Write([]byte(`{"status":"ok","result":[{"active":true,"emails":["authorized@example.com"],"user_huid":"00000000-0000-4000-8000-000000000123"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(bot.Close)
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	cfg := &config.Config{PublicURL: "https://aegis.test", SessionTTL: time.Hour, OIDC: map[string]config.OIDCProvider{
		"slack":   {ClientID: "client", ClientSecret: "secret", RedirectURL: "https://aegis.test/auth/slack/callback", Issuer: "https://slack.com"},
		"express": {ClientID: "client", ClientSecret: "secret", RedirectURL: "https://aegis.test/auth/express/callback", Issuer: "https://sso.test"},
	}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	token, user, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)
	slackConfig, _ := json.Marshal(map[string]string{"bot_token": "bot", "signing_secret": "secret", "api_base_url": bot.URL})
	expressConfig, _ := json.Marshal(map[string]string{"bot_id": "bot", "secret_key": "secret", "host": bot.URL})
	repo := &pagingHandlerRepo{users: users, items: map[string]db.Integration{
		"slack": {Kind: "slack", Enabled: true, Config: slackConfig}, "express": {Kind: "express", Enabled: true, Config: expressConfig},
	}}
	exchanger := pagingHandlerExchange{info: oidc.PagingIdentity{Subject: "U123", SlackUserID: "U123", SlackTeamID: "T123", Email: "authorized@example.com", EmailVerified: true}}
	paging := NewPagingHandler(service.NewPagingService(cfg, repo, exchanger), auth, cfg.PublicURL)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAuthHandler(auth, cfg.PublicURL, paging.Callback).Register(router)
	paging.Register(router)
	return router, repo, token, user.ID
}

func pagingRequest(router *gin.Engine, method, path, token, origin string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	if token != "" {
		request.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	}
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestPagingHTTPFlow(t *testing.T) {
	for _, provider := range []string{"slack", "express"} {
		t.Run(provider, func(t *testing.T) {
			router, repo, session, userID := pagingRouter(t)
			start := pagingRequest(router, http.MethodPost, "/api/v1/users/me/paging-connections/"+provider+"/authorize", session, "https://aegis.test")
			require.Equal(t, http.StatusOK, start.Code, start.Body.String())
			var response struct {
				AuthorizationURL string `json:"authorization_url"`
			}
			require.NoError(t, json.Unmarshal(start.Body.Bytes(), &response))
			authorization, err := url.Parse(response.AuthorizationURL)
			require.NoError(t, err)
			state := authorization.Query().Get("state")
			require.Equal(t, sessiontoken.Hash(state), repo.attempt.StateHash)
			callbackPath := "/auth/" + provider + "/callback?" + url.Values{"state": {state}, "code": {"code"}}.Encode()
			callback := pagingRequest(router, http.MethodGet, callbackPath, session, "")
			require.Equal(t, http.StatusFound, callback.Code)
			require.Contains(t, callback.Header().Get("Location"), "paging_result=connected")
			require.Empty(t, callback.Result().Cookies(), "must not replace the Aegis session")
			require.Len(t, repo.users.users, 1, "must not create another user")
			identities, err := repo.users.ListUserIdentities(context.Background(), userID)
			require.NoError(t, err)
			require.Len(t, identities, 1)
			require.Equal(t, "google", identities[0].Provider)
			connected := pagingRequest(router, http.MethodGet, "/api/v1/users/me/paging-connections", session, "")
			require.Equal(t, http.StatusOK, connected.Code)
			require.Contains(t, connected.Body.String(), "\"available\":true")
			replay := pagingRequest(router, http.MethodGet, callbackPath, session, "")
			require.Contains(t, replay.Header().Get("Location"), "paging_error=invalid_authorization")
			disconnected := pagingRequest(router, http.MethodDelete, "/api/v1/users/me/paging-connections/"+provider, session, "https://aegis.test")
			require.Equal(t, http.StatusNoContent, disconnected.Code)
			user := repo.users.users[userID]
			if provider == "slack" {
				require.Nil(t, user.SlackUserID)
			} else {
				require.False(t, user.ExpressUserHuid.Valid)
			}
		})
	}
}

func TestPagingHTTPFailures(t *testing.T) {
	router, repo, session, _ := pagingRouter(t)
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		path := "/api/v1/users/me/paging-connections"
		if method == http.MethodPost {
			path += "/slack/authorize"
		}
		if method == http.MethodDelete {
			path += "/slack"
		}
		require.Equal(t, http.StatusUnauthorized, pagingRequest(router, method, path, "", "https://aegis.test").Code)
		if method != http.MethodGet {
			require.Equal(t, http.StatusForbidden, pagingRequest(router, method, path, session, "https://evil.test").Code)
			require.Equal(t, http.StatusForbidden, pagingRequest(router, method, path, session, "").Code)
		}
	}
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		path := "/api/v1/users/me/paging-connections/google"
		if method == http.MethodPost {
			path += "/authorize"
		}
		require.Equal(t, http.StatusBadRequest, pagingRequest(router, method, path, session, "https://aegis.test").Code)
	}
	invalid := pagingRequest(router, http.MethodGet, "/auth/slack/callback?state=paging.invalid&code=x", session, "")
	require.Contains(t, invalid.Header().Get("Location"), "/account?")
	require.Contains(t, invalid.Header().Get("Location"), "invalid_authorization")
	start := pagingRequest(router, http.MethodPost, "/api/v1/users/me/paging-connections/slack/authorize", session, "https://aegis.test")
	require.Equal(t, http.StatusOK, start.Code)
	var response struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	require.NoError(t, json.Unmarshal(start.Body.Bytes(), &response))
	parsed, err := url.Parse(response.AuthorizationURL)
	require.NoError(t, err)
	cancelled := pagingRequest(router, http.MethodGet, "/auth/slack/callback?"+url.Values{"state": {parsed.Query().Get("state")}, "error": {"access_denied"}}.Encode(), session, "")
	require.Contains(t, cancelled.Header().Get("Location"), "paging_error=cancelled")
	expiredSession := pagingRequest(router, http.MethodGet, "/auth/slack/callback?state=paging.expired&code=x", "", "")
	require.Contains(t, expiredSession.Header().Get("Location"), "invalid_authorization")
	repo.err = errors.New("db failed")
	require.Equal(t, http.StatusInternalServerError, pagingRequest(router, http.MethodGet, "/api/v1/users/me/paging-connections", session, "").Code)
	require.Equal(t, http.StatusInternalServerError, pagingRequest(router, http.MethodPost, "/api/v1/users/me/paging-connections/slack/authorize", session, "https://aegis.test").Code)
	require.Equal(t, http.StatusInternalServerError, pagingRequest(router, http.MethodDelete, "/api/v1/users/me/paging-connections/slack", session, "https://aegis.test").Code)
}
