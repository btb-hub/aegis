package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/oidc"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type settingsExchange struct {
	email    string
	verified bool
}

func (x settingsExchange) AuthorizationURL(_ context.Context, provider, state, nonce, team string) (string, error) {
	return "https://provider.example/authorize?state=" + url.QueryEscape(state), nil
}
func (x settingsExchange) ExchangePaging(_ context.Context, provider, code, nonce string) (oidc.PagingIdentity, error) {
	if code == "bad" {
		return oidc.PagingIdentity{}, errors.New("private upstream response")
	}
	return oidc.PagingIdentity{Subject: "verified-user", Email: x.email, EmailVerified: x.verified, DisplayName: "Admin"}, nil
}

type settingsFixture struct {
	router  *gin.Engine
	store   *db.Store
	service *service.SettingsService
	token   string
	user    db.User
	boot    *config.Config
}

func newSettingsFixture(t *testing.T, fresh bool) settingsFixture {
	t.Helper()
	connection := os.Getenv("DATABASE_URL")
	if connection == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, connection)
	require.NoError(t, err)
	schema := "handler_settings_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = root.Exec(ctx, `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	pc, err := pgxpool.ParseConfig(connection)
	require.NoError(t, err)
	pc.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	require.NoError(t, err)
	files, err := filepath.Glob("../../../../db/migrations/*.up.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		raw, e := os.ReadFile(file)
		require.NoError(t, e)
		_, e = pool.Exec(ctx, string(raw))
		require.NoError(t, e)
	}
	t.Cleanup(func() {
		pool.Close()
		_, e := root.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
		require.NoError(t, e)
		root.Close()
	})
	store := db.NewStore(pool)
	boot := &config.Config{BootstrapToken: strings.Repeat("installation-token-test-", 3)}
	fixture := settingsFixture{store: store, boot: boot}
	if fresh {
		require.NoError(t, store.InitializeSettings(ctx, boot))
	} else {
		_, err = store.ImportEnvironment(ctx, map[string]string{"PUBLIC_URL": "http://aegis.test", "HTTP_ADDR": ":8080", "WEBHOOK_SECRET": "webhook-fixture", "GOOGLE_OIDC_CLIENT_ID": "client", "GOOGLE_OIDC_CLIENT_SECRET": "secret-fixture", "GOOGLE_OIDC_REDIRECT_URL": "http://aegis.test/auth/google/callback"}, true)
		require.NoError(t, err)
		fixture.user, err = store.UpsertUser(ctx, "google", "verified-user", "admin@company.com", "Admin", "admin", "en")
		require.NoError(t, err)
		var hash string
		fixture.token, hash, err = sessiontoken.New()
		require.NoError(t, err)
		_, err = store.CreateSession(ctx, fixture.user.ID, hash, time.Now().Add(time.Hour))
		require.NoError(t, err)
	}
	cfg, err := store.SettingsConfig(ctx, boot)
	require.NoError(t, err)
	auth := service.NewAuthService(cfg, store, store, service.NewOAuthTokenExchanger(cfg))
	settings := service.NewSettingsService(store, boot)
	settings.Exchange = func(*config.Config) service.PagingExchanger { return settingsExchange{"admin@company.com", true} }
	fixture.service = settings
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		snapshot, e := store.SettingsContext(c.Request.Context(), boot)
		require.NoError(t, e)
		c.Request = c.Request.WithContext(snapshot)
	})
	h := NewSettingsHandler(settings, auth)
	h.Register(r)
	NewIntegrationHandler(service.NewIntegrationService(store, cfg.PublicURL), auth).Register(r)
	authHandler := NewAuthHandler(auth, cfg.PublicURL)
	authHandler.SettingsFlows(h.Login, h.Callback)
	authHandler.Register(r)
	fixture.router = r
	return fixture
}

func TestSettingsIntegrationStaleEditsAndOrigin(t *testing.T) {
	f := newSettingsFixture(t, false)
	ctx := context.Background()
	bot, err := f.store.UpsertIntegration(ctx, "slack", "Slack", json.RawMessage(`{"bot_token":"bot-fixture","signing_secret":"signing-fixture"}`), true, nil, nil)
	require.NoError(t, err)
	body, _ := json.Marshal(map[string]any{"name": "Updated Slack", "expected_updated_at": bot.UpdatedAt})
	path := "/api/v1/integrations/" + bot.ID.String()
	require.Equal(t, 200, f.request("PATCH", path, string(body), f.token).Code)
	require.Equal(t, 409, f.request("PATCH", path, string(body), f.token).Code)
	require.Equal(t, 400, f.request("PATCH", path, `{"expected_updated_at":"bad"}`, f.token).Code)
	r := httptest.NewRequest("PATCH", "http://aegis.test"+path, strings.NewReader(string(body)))
	r.Header.Set("Origin", "http://evil.test")
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: f.token})
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	require.Equal(t, 403, w.Code)
}

func (f settingsFixture) request(method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://aegis.test"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://aegis.test")
	if token != "" {
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	return w
}

func TestSettingsAdminEndpointsAndRedaction(t *testing.T) {
	f := newSettingsFixture(t, false)
	w := f.request("GET", "/api/v1/settings", "", f.token)
	require.Equal(t, 200, w.Code)
	require.NotContains(t, w.Body.String(), "secret-fixture")
	require.NotContains(t, w.Body.String(), "webhook-fixture")
	for _, path := range []string{"/api/v1/settings", "/api/v1/settings/providers/google"} {
		require.Equal(t, 401, f.request("GET", path, "", "").Code)
	}
	_, err := f.store.UpdateUserRole(context.Background(), f.user.ID, "member")
	require.NoError(t, err)
	require.Equal(t, 403, f.request("GET", "/api/v1/settings", "", f.token).Code)
	_, err = f.store.UpdateUserRole(context.Background(), f.user.ID, "admin")
	require.NoError(t, err)
	w = f.request("GET", "/api/v1/settings/providers/google", "", f.token)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "GOOGLE_OIDC_REDIRECT_URL")
}

func TestSettingsSaveTestActivateAndLogin(t *testing.T) {
	f := newSettingsFixture(t, false)
	w := f.request("PUT", "/api/v1/settings/providers/google", `{"revision":0,"expected_email":"attacker@company.com","values":{"GOOGLE_OIDC_CLIENT_ID":"draft-client","GOOGLE_OIDC_CLIENT_SECRET":"draft-fixture"}}`, f.token)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "admin@company.com")
	require.NotContains(t, w.Body.String(), "draft-fixture")
	w = f.request("POST", "/api/v1/settings/providers/google/activate", `{"revision":1,"draft_revision":1,"enabled":true}`, f.token)
	require.Equal(t, 409, w.Code)
	w = f.request("POST", "/api/v1/settings/providers/google/test", "", f.token)
	require.Equal(t, 200, w.Code)
	var result struct {
		URL string `json:"authorization_url"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	u, _ := url.Parse(result.URL)
	state := u.Query().Get("state")
	w = f.request("GET", "/auth/google/callback?state="+url.QueryEscape(state)+"&code=good", "", f.token)
	require.Equal(t, 302, w.Code)
	require.Contains(t, w.Header().Get("Location"), "settings_result=tested")
	require.Equal(t, 302, f.request("GET", "/auth/google/callback?state="+url.QueryEscape(state)+"&code=good", "", f.token).Code)
	w = f.request("GET", "/api/v1/settings/providers/google", "", f.token)
	require.Contains(t, w.Body.String(), `"tested":true`)
	w = f.request("POST", "/api/v1/settings/providers/google/activate", `{"revision":1,"draft_revision":1,"enabled":true}`, f.token)
	require.Equal(t, 200, w.Code)
	w = f.request("POST", "/api/v1/settings/providers/google/activate", `{"revision":2,"enabled":false}`, f.token)
	require.Equal(t, 409, w.Code)
	w = f.request("PATCH", "/api/v1/settings/behavior", `{"revision":2,"values":{"SESSION_TTL":"2h"}}`, f.token)
	require.Equal(t, 200, w.Code)
	w = f.request("PATCH", "/api/v1/settings/behavior", `{"revision":2,"values":{"SESSION_TTL":"3h"}}`, f.token)
	require.Equal(t, 409, w.Code)
	w = f.request("GET", "/auth/google/login?redirect=/account", "", "")
	require.Equal(t, 302, w.Code)
	u, _ = url.Parse(w.Header().Get("Location"))
	state = u.Query().Get("state")
	r := httptest.NewRequest("GET", "http://aegis.test/auth/google/callback?state="+url.QueryEscape(state)+"&code=good", nil)
	for _, cookie := range w.Result().Cookies() {
		r.AddCookie(cookie)
	}
	callback := httptest.NewRecorder()
	f.router.ServeHTTP(callback, r)
	require.Equal(t, 302, callback.Code)
	require.Contains(t, callback.Header().Get("Location"), "/account?connected=google")
	require.NotEmpty(t, callback.Result().Cookies())
	doc, _ := f.store.GetSettings(context.Background())
	require.Equal(t, "2h", doc.Values["SESSION_TTL"])
}

func TestSettingsEndpointFailures(t *testing.T) {
	f := newSettingsFixture(t, false)
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"PATCH", "/api/v1/settings/behavior", "{", 400}, {"PUT", "/api/v1/settings/providers/google", "{", 400}, {"POST", "/api/v1/settings/providers/google/activate", "{", 400},
		{"PATCH", "/api/v1/settings/unknown", `{"revision":1,"values":{}}`, 400}, {"GET", "/api/v1/settings/providers/unknown", "", 400},
		{"PUT", "/api/v1/settings/providers/google", `{"revision":0,"values":{"UNSUPPORTED":"value"}}`, 400},
		{"POST", "/api/v1/settings/providers/google/test", "", 400},
		{"PATCH", "/api/v1/settings/deployment", `{"revision":1,"values":{"PUBLIC_URL":"https://new.example"}}`, 409},
	} {
		w := f.request(tc.method, tc.path, tc.body, f.token)
		require.Equal(t, tc.status, w.Code, w.Body.String())
	}
	r := httptest.NewRequest("PATCH", "http://aegis.test/api/v1/settings/behavior", strings.NewReader(`{"revision":1,"values":{}}`))
	r.Header.Set("Origin", "https://evil.example")
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: f.token})
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	require.Equal(t, 403, w.Code)
	w = f.request("GET", "/auth/google/callback?state=login.fake&code=good", "", f.token)
	require.Contains(t, w.Header().Get("Location"), "authorization_failed")
	w = f.request("GET", "/auth/express/login", "", "")
	require.Contains(t, w.Header().Get("Location"), "unconfigured")
	w = f.request("GET", "/api/v1/bootstrap/status", "", "")
	require.Contains(t, w.Body.String(), `"available":false`)
}

func TestSettingsBootstrapBrowserFlow(t *testing.T) {
	f := newSettingsFixture(t, true)
	w := f.request("GET", "/api/v1/bootstrap/status", "", "")
	require.Contains(t, w.Body.String(), `"available":true`)
	w = f.request("POST", "/api/v1/bootstrap/session", `{"token":"wrong","public_url":"http://aegis.test"}`, "")
	require.Equal(t, 401, w.Code)
	require.Equal(t, 400, f.request("POST", "/api/v1/bootstrap/session", "{", "").Code)
	require.Equal(t, 403, f.request("POST", "/api/v1/bootstrap/session", `{"token":"wrong","public_url":"https://evil.example"}`, "").Code)
	body, _ := json.Marshal(map[string]string{"token": f.boot.BootstrapToken, "public_url": "http://aegis.test"})
	w = f.request("POST", "/api/v1/bootstrap/session", string(body), "")
	require.Equal(t, 200, w.Code)
	installation := w.Result().Cookies()[0]
	require.True(t, installation.HttpOnly)
	require.Equal(t, 1800, installation.MaxAge)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://aegis.test"+path, strings.NewReader(body))
		r.Header.Set("Origin", "http://aegis.test")
		r.Header.Set("Content-Type", "application/json")
		r.AddCookie(installation)
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, r)
		return w
	}
	w = request("GET", "/api/v1/bootstrap/status", "")
	require.Contains(t, w.Body.String(), `"session_active":true`)
	w = request("GET", "/api/v1/bootstrap/configuration", "")
	require.Equal(t, 200, w.Code)
	require.NotContains(t, w.Body.String(), "WEBHOOK_SECRET")
	w = request("PUT", "/api/v1/bootstrap/providers/google", `{"revision":0,"expected_email":"admin@company.com","values":{"GOOGLE_OIDC_CLIENT_ID":"client","GOOGLE_OIDC_CLIENT_SECRET":"fixture"}}`)
	require.Equal(t, 200, w.Code)
	w = request("GET", "/api/v1/bootstrap/providers/google", "")
	require.Equal(t, 200, w.Code)
	w = request("POST", "/api/v1/bootstrap/providers/google/test", "")
	require.Equal(t, 200, w.Code)
	var result struct {
		URL string `json:"authorization_url"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	u, _ := url.Parse(result.URL)
	w = request("GET", "/auth/google/callback?state="+url.QueryEscape(u.Query().Get("state"))+"&code=good", "")
	require.Equal(t, 302, w.Code)
	require.Contains(t, w.Header().Get("Location"), "settings_result=installed")
	require.Equal(t, 401, request("GET", "/api/v1/bootstrap/configuration", "").Code)
	w = f.request("GET", "/api/v1/bootstrap/status", "", "")
	require.Contains(t, w.Body.String(), `"available":false`)
}
