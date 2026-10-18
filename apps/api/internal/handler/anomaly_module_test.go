package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/anomalymodule"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type anomalyRepo struct {
	workspace    uuid.UUID
	event        anomalymodule.Event
	err          error
	workspaceErr error
}

func (r *anomalyRepo) ListWorkspaces(context.Context) ([]db.Workspace, error) {
	return []db.Workspace{{ID: r.workspace}}, r.workspaceErr
}
func (r *anomalyRepo) AcceptAnomalyEvent(_ context.Context, e anomalymodule.Event) (db.AnomalyEventResult, error) {
	r.event = e
	return db.AnomalyEventResult{AlertID: uuid.New()}, r.err
}

type anomalyAuth struct{ user db.User }

func (a anomalyAuth) CurrentUser(_ context.Context, token string) (db.User, error) {
	if token != "session" {
		return db.User{}, apperrors.Unauthorized("sign in")
	}
	return a.user, nil
}

type anomalyTransport func(*http.Request) (*http.Response, error)

func (f anomalyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func anomalySetup(t *testing.T, role string) (*gin.Engine, *AnomalyHandler, *anomalyRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{PublicURL: "https://aegis.test", Anomaly: config.AnomalyConfig{URL: "http://private-module", InstallationID: "install", IdentitySecret: strings.Repeat("i", 32), ServiceToken: strings.Repeat("s", 32), Enabled: true}}
	repo := &anomalyRepo{workspace: uuid.MustParse("00000000-0000-0000-0000-000000000001")}
	h := NewAnomalyHandler(cfg, repo, anomalyAuth{db.User{ID: uuid.New(), Role: role, Locale: "en"}})
	h.client.Transport = anomalyTransport(func(r *http.Request) (*http.Response, error) {
		return anomalyResponse(`{"contract_version":1,"module":"anomaly-detector","version":"v1","healthy":true,"entitled":false}`), nil
	})
	r := gin.New()
	h.Register(r)
	return r, h, repo
}
func anomalyResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}
}
func anomalyRequest(r *gin.Engine, method, path, body, token, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://aegis.test"+path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func TestAnomalyServiceRoutes(t *testing.T) {
	r, h, repo := anomalySetup(t, "admin")
	service := h.config.Anomaly.ServiceToken
	events := "/api/v1/modules/anomaly-detector/events"
	grant := "/api/v1/modules/anomaly-detector/entitlement"
	for _, path := range []string{grant, events} {
		method := "GET"
		if path == events {
			method = "POST"
		}
		require.Equal(t, 401, anomalyRequest(r, method, path, "{}", "wrong", "").Code)
	}
	require.Equal(t, 401, anomalyRequest(r, "GET", "/api/v1/modules/anomaly-detector/capabilities", "", "wrong", "").Code)
	w := anomalyRequest(r, "GET", grant, "", service, "")
	require.Equal(t, 200, w.Code)
	var got anomalymodule.Grant
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.True(t, got.Enabled)
	require.WithinDuration(t, time.Now().Add(time.Minute), got.ExpiresAt, time.Second)
	e := anomalymodule.Event{ContractVersion: 1, EventID: uuid.NewString(), InstallationID: "install", WorkspaceID: repo.workspace.String(), MetricID: uuid.NewString(), SeriesID: strings.Repeat("a", 64), EpisodeID: uuid.NewString(), Sequence: 1, State: "firing", OccurredAt: time.Now().UTC(), Title: "Spike", Severity: "high"}
	raw, _ := json.Marshal(e)
	w = anomalyRequest(r, "POST", events, string(raw), service, "")
	require.Equal(t, 202, w.Code, w.Body.String())
	require.Equal(t, e.EventID, repo.event.EventID)
	for _, raw := range []string{`{}`, `{`, string(raw) + ` {}`, strings.Replace(string(raw), `"contract_version":1`, `"contract_version":2`, 1), strings.Replace(string(raw), `"installation_id":"install"`, `"installation_id":"other"`, 1), `{"unknown":true}`, `{"title":"` + strings.Repeat("a", (1<<20)+1) + `"}`} {
		w = anomalyRequest(r, "POST", events, raw, service, "")
		require.Contains(t, []int{400, 403}, w.Code, w.Body.String())
	}
	repo.err = apperrors.Conflict("duplicate payload differs")
	require.Equal(t, 409, anomalyRequest(r, "POST", events, string(raw), service, "").Code)
	h.config.Anomaly.Enabled = false
	require.Equal(t, 403, anomalyRequest(r, "POST", events, string(raw), service, "").Code)
	w = anomalyRequest(r, "GET", grant, "", service, "")
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"enabled":false`)
	h.config.Anomaly.URL = ""
	require.Equal(t, 401, anomalyRequest(r, "GET", grant, "", service, "").Code)
}
func TestAnomalyProxyIdentityAndHeaderIsolation(t *testing.T) {
	r, h, repo := anomalySetup(t, "admin")
	var received *http.Request
	h.client.Transport = anomalyTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/capabilities" {
			require.Equal(t, "Bearer "+h.config.Anomaly.ServiceToken, req.Header.Get("Authorization"))
			return anomalyResponse(`{"contract_version":1,"module":"anomaly-detector","version":"v1","healthy":true}`), nil
		}
		received = req.Clone(req.Context())
		response := anomalyResponse("module screen")
		response.Header.Set("Set-Cookie", "session=attack")
		return response, nil
	})
	req := httptest.NewRequest("GET", "https://aegis.test/modules/anomaly-detector/metrics?workspace_id="+repo.workspace.String(), nil)
	req.AddCookie(&http.Cookie{Name: "aegis_session", Value: "session"})
	req.Header.Set("Authorization", "Bearer user-token")
	req.Header.Set("X-Aegis-Identity", "forged")
	req.Header.Set("X-Aegis-Role", "admin")
	req.Header.Set("Forwarded", "host=attacker")
	req.Header.Set("X-Forwarded-Host", "attacker")
	req.Header.Set("X-Forwarded-For", "attacker")
	req.Header.Set("Connection", "X-Aegis-Identity")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, "module screen", w.Body.String())
	require.Empty(t, w.Header().Get("Set-Cookie"))
	require.Equal(t, "aegis.test", received.Host)
	require.Equal(t, "/modules/anomaly-detector/metrics", received.URL.Path)
	for _, name := range []string{"Cookie", "Authorization", "X-Aegis-Role", "Forwarded", "X-Forwarded-For", "X-Forwarded-Host"} {
		require.Empty(t, received.Header.Get(name), name)
	}
	parts := strings.Split(received.Header.Get("X-Aegis-Identity"), ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var identity anomalymodule.Identity
	require.NoError(t, json.Unmarshal(raw, &identity))
	require.Equal(t, []string{repo.workspace.String()}, identity.WorkspaceIDs)
	require.Equal(t, "install", identity.InstallationID)
	require.True(t, identity.Entitled)
	require.Equal(t, int64(60), identity.ExpiresAt-identity.IssuedAt)
	h.config.Anomaly.Enabled = false
	w = anomalyRequest(r, "GET", "/modules/anomaly-detector/", "", "session", "")
	require.Equal(t, 200, w.Code)
	raw, _ = base64.RawURLEncoding.DecodeString(strings.Split(received.Header.Get("X-Aegis-Identity"), ".")[1])
	require.NoError(t, json.Unmarshal(raw, &identity))
	require.False(t, identity.Entitled)
}
func TestAnomalyProxyBoundsSlowOperations(t *testing.T) {
	for _, test := range []struct {
		method, path string
		timeout      time.Duration
	}{
		{http.MethodGet, "/api/v1/metrics", 10 * time.Second},
		{http.MethodPost, "/api/v1/metrics/" + uuid.NewString() + "/preview", 150 * time.Second},
		{http.MethodPost, "/api/v1/sources/" + uuid.NewString() + "/test", 150 * time.Second},
		{http.MethodPost, "/api/v1/history/" + uuid.NewString() + "/feedback", 10 * time.Second},
	} {
		t.Run(test.path, func(t *testing.T) {
			r, h, _ := anomalySetup(t, "admin")
			called := false
			h.client.Transport = anomalyTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/api/v1/capabilities" {
					return anomalyResponse(`{"contract_version":1,"module":"anomaly-detector","version":"v1","healthy":true}`), nil
				}
				called = true
				deadline, bounded := req.Context().Deadline()
				require.True(t, bounded)
				require.WithinDuration(t, time.Now().Add(test.timeout), deadline, time.Second)
				return anomalyResponse(`{"ok":true}`), nil
			})
			response := anomalyRequest(r, test.method, anomalymodule.Prefix+test.path, "{}", "session", "https://aegis.test")
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.True(t, called)
		})
	}
}

func TestAnomalyProxyPermissions(t *testing.T) {
	feedback := "/modules/anomaly-detector/api/v1/history/" + uuid.NewString() + "/feedback?workspace_id=" + uuid.NewString()
	for _, role := range []string{"admin", "member", "viewer"} {
		r, _, _ := anomalySetup(t, role)
		for _, path := range []string{feedback, "/modules/anomaly-detector/api/v1/sources", "/modules/anomaly-detector/api/v1/history/not-uuid/feedback"} {
			want := 403
			if role == "admin" || role == "member" && path == feedback {
				want = 200
			}
			w := anomalyRequest(r, "POST", path, `{}`, "session", "https://aegis.test")
			require.Equal(t, want, w.Code, w.Body.String())
		}
		require.Equal(t, 403, anomalyRequest(r, "POST", feedback, `{}`, "session", "").Code)
		require.Equal(t, 403, anomalyRequest(r, "POST", feedback, `{}`, "session", "https://attacker.test").Code)
	}
	r, h, _ := anomalySetup(t, "admin")
	h.config.Anomaly.Enabled = false
	require.Equal(t, 403, anomalyRequest(r, "POST", feedback, `{}`, "session", "https://aegis.test").Code)
	require.Equal(t, 401, anomalyRequest(r, "GET", "/modules/anomaly-detector/", "", "wrong", "").Code)
}
func TestAnomalyFallbacksAndCapabilityStatus(t *testing.T) {
	r, h, repo := anomalySetup(t, "admin")
	w := anomalyRequest(r, "GET", "/api/v1/modules/anomaly-detector/capabilities", "", "session", "")
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"compatible":true`)
	require.NotContains(t, w.Body.String(), h.config.Anomaly.ServiceToken)
	repo.workspaceErr = errors.New("db unavailable")
	require.Equal(t, 500, anomalyRequest(r, "GET", "/modules/anomaly-detector/", "", "session", "").Code)
	repo.workspaceErr = nil
	h.client.Transport = anomalyTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("http://secret:password@private") })
	w = anomalyRequest(r, "GET", "/modules/anomaly-detector/api/v1/sources", "", "session", "")
	require.Equal(t, 503, w.Code)
	require.NotContains(t, w.Body.String(), "password")
	for _, locale := range []string{"en", "ru"} {
		h.auth = anomalyAuth{db.User{ID: uuid.New(), Role: "admin", Locale: locale}}
		r = gin.New()
		h.Register(r)
		for _, configured := range []bool{true, false} {
			if !configured {
				h.config.Anomaly.URL = ""
			}
			req := httptest.NewRequest("GET", "https://aegis.test/modules/anomaly-detector/", nil)
			req.Header.Set("Authorization", "Bearer session")
			req.Header.Set("Accept", "text/html")
			w = httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, 503, w.Code)
			require.Contains(t, w.Body.String(), "/settings?section=anomaly")
			require.Contains(t, w.Body.String(), "/shifts")
			if locale == "ru" {
				require.Contains(t, w.Body.String(), "Обнаружение")
			}
		}
		h.config.Anomaly.URL = "http://private-module"
	}
	h.client.Transport = anomalyTransport(func(*http.Request) (*http.Response, error) {
		return anomalyResponse(`{"contract_version":99,"module":"anomaly-detector","version":"v1","healthy":true}`), nil
	})
	w = anomalyRequest(r, "GET", "/modules/anomaly-detector/api/v1/sources", "", "session", "")
	require.Contains(t, w.Body.String(), "ANOMALY_MODULE_INCOMPATIBLE")
	h.client.Transport = anomalyTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/capabilities" {
			return anomalyResponse(`{"contract_version":1,"module":"anomaly-detector","version":"v1","healthy":true}`), nil
		}
		return nil, errors.New("upstream failed")
	})
	require.Equal(t, 503, anomalyRequest(r, "GET", "/modules/anomaly-detector/", "", "session", "").Code)
}

type deniedAnomalyEntitlement struct{}

func (deniedAnomalyEntitlement) Grant(c config.AnomalyConfig, now time.Time) anomalymodule.Grant {
	return anomalymodule.Grant{ContractVersion: 1, InstallationID: c.InstallationID, Enabled: false, ExpiresAt: now.Add(time.Minute)}
}
func TestAnomalyCustomEntitlementAndBrowserSignIn(t *testing.T) {
	r, h, repo := anomalySetup(t, "viewer")
	req := httptest.NewRequest("GET", "https://aegis.test/modules/anomaly-detector/metrics", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 302, w.Code)
	require.Equal(t, "/login", w.Header().Get("Location"))
	req = httptest.NewRequest("GET", "https://aegis.test/modules/anomaly-detector/api/v1/history", nil)
	req.Header.Set("Accept", "text/html")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 401, w.Code)
	h.config.Anomaly.URL = ""
	req = httptest.NewRequest("GET", "https://aegis.test/modules/anomaly-detector/", nil)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Authorization", "Bearer session")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 503, w.Code)
	require.NotContains(t, w.Body.String(), "/settings?section=anomaly")
	require.Contains(t, w.Body.String(), "Ask an Aegis administrator")
	h.auth = anomalyAuth{db.User{ID: uuid.New(), Role: "viewer", Locale: "ru"}}
	r = gin.New()
	h.Register(r)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "Попросите администратора")
	cfg := *h.config
	cfg.Anomaly.URL = "http://module"
	h = NewAnomalyHandler(&cfg, repo, h.auth, deniedAnomalyEntitlement{})
	r = gin.New()
	h.Register(r)
	w = anomalyRequest(r, "GET", "/api/v1/modules/anomaly-detector/entitlement", "", cfg.Anomaly.ServiceToken, "")
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"enabled":false`)
	runtime := config.Config{Anomaly: cfg.Anomaly}
	runtime.Anomaly.Enabled = false
	r = gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(config.WithRuntime(c.Request.Context(), &runtime))
	})
	h.Register(r)
	w = anomalyRequest(r, "GET", "/api/v1/modules/anomaly-detector/entitlement", "", cfg.Anomaly.ServiceToken, "")
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"enabled":false`)
}

func TestAnomalyProxyBoundsMutationBeforeUpstream(t *testing.T) {
	r, _, _ := anomalySetup(t, "admin")
	w := anomalyRequest(r, "POST", "/modules/anomaly-detector/api/v1/sources", strings.Repeat("x", (1<<20)+1), "session", "https://aegis.test")
	require.Equal(t, 400, w.Code)
	require.Contains(t, w.Body.String(), "1 MiB")
}
