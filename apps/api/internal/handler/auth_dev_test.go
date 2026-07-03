package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupDevAuthRouter(t *testing.T, enabled bool) (*gin.Engine, *service.AuthService) {
	t.Helper()
	cfg := &config.Config{
		SessionTTL:         time.Hour,
		PublicURL:          "http://localhost:3000",
		DevAuthEnabled:     enabled,
		DevAuthDefaultRole: "admin",
		DevAuthEmail:       "dev@localhost",
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
	return setupRouterWithConfig(t, cfg)
}

func setupRouterWithConfig(t *testing.T, cfg *config.Config) (*gin.Engine, *service.AuthService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &authMockAlertRepo{id: uuid.New()})
	teams := service.NewTeamService(&emptyTeamRepo{})
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth, cfg.PublicURL).Register(r)
	NewAlertHandler(alerts, teams, auth).Register(r)
	return r, auth
}

func TestDevAuthStatusDisabled(t *testing.T) {
	r, _ := setupDevAuthRouter(t, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/dev/status", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"enabled":false}`, w.Body.String())
}

func TestDevAuthStatusEnabled(t *testing.T) {
	r, _ := setupDevAuthRouter(t, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/dev/status", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"enabled":true}`, w.Body.String())
}

func TestDevAuthLoginDisabledReturns404(t *testing.T) {
	r, _ := setupDevAuthRouter(t, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/dev/login", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDevAuthLoginEnabledSetsCookieAndRedirects(t *testing.T) {
	r, auth := setupDevAuthRouter(t, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/dev/login?role=admin&redirect=/dashboard", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000/dashboard", w.Header().Get("Location"))

	cookie := findCookie(w, sessionCookie)
	require.NotNil(t, cookie)
	require.NotEmpty(t, cookie.Value)

	user, err := auth.CurrentUser(t.Context(), cookie.Value)
	require.NoError(t, err)
	require.Equal(t, "admin", user.Role)
}

func TestDevAuthLoginUsesDefaultRoleAndPublicURL(t *testing.T) {
	r, auth := setupDevAuthRouter(t, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/dev/login", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000", w.Header().Get("Location"))

	cookie := findCookie(w, sessionCookie)
	require.NotNil(t, cookie)

	user, err := auth.CurrentUser(t.Context(), cookie.Value)
	require.NoError(t, err)
	require.Equal(t, "admin", user.Role)
}

func TestDevAuthLoginInvalidRole(t *testing.T) {
	r, _ := setupDevAuthRouter(t, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/dev/login?role=superuser", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000/login?dev_auth_error=1", w.Header().Get("Location"))
}

func TestDevAuthLoginRejectsUnsafeRedirect(t *testing.T) {
	r, _ := setupDevAuthRouter(t, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/dev/login?redirect=//evil.test", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000", w.Header().Get("Location"))
}

func findCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func TestDevAuthAdminCanPostTestAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newPhase2HandlerRepo()
	cfg := &config.Config{
		SessionTTL:         time.Hour,
		PublicURL:          "http://localhost:3000",
		DevAuthEnabled:     true,
		DevAuthDefaultRole: "admin",
		DevAuthEmail:       "dev@localhost",
	}
	auth := service.NewAuthService(cfg, repo, repo, &authMockOIDC{})
	analytics := service.NewAnalyticsService(repo)
	alertsRepo := &authMockAlertRepo{id: uuid.New()}
	repo.alertRepo = alertsRepo
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, alertsRepo)
	handoffs := service.NewHandoffService(repo)

	r := gin.New()
	NewAuthHandler(auth, cfg.PublicURL).Register(r)
	NewAnalyticsHandler(analytics, alerts, handoffs, auth).Register(r)

	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/dev/login?role=admin", nil)
	r.ServeHTTP(login, loginReq)
	cookie := findCookie(login, sessionCookie)
	require.NotNil(t, cookie)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/test-alert", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)
}
