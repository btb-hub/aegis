package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLoginRedirect(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	require.Contains(t, w.Header().Get("Location"), "authorize")
}

func TestLoginStoresSafeRedirectAndCallbackHonorsIt(t *testing.T) {
	r, _ := setupRouter(t)

	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login?redirect=/account", nil)
	r.ServeHTTP(login, loginReq)
	require.Equal(t, http.StatusFound, login.Code)

	stateCookie := findCookie(login, oauthStateCookie)
	redirectCookie := findCookie(login, oauthRedirectCookie)
	require.NotNil(t, stateCookie)
	require.NotNil(t, redirectCookie)
	decodedRedirect, err := url.QueryUnescape(redirectCookie.Value)
	require.NoError(t, err)
	require.Equal(t, "/account", decodedRedirect)

	w := httptest.NewRecorder()
	cbReq := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+stateCookie.Value+"&code=xyz", nil)
	cbReq.AddCookie(stateCookie)
	cbReq.AddCookie(redirectCookie)
	r.ServeHTTP(w, cbReq)

	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000/account?connected=google", w.Header().Get("Location"))
	cleared := findCookie(w, oauthRedirectCookie)
	require.NotNil(t, cleared)
	require.Less(t, cleared.MaxAge, 1)
}

func TestCallbackHonorsNonAccountRedirect(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "abc"})
	req.AddCookie(&http.Cookie{Name: oauthRedirectCookie, Value: "/dashboard"})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000/dashboard", w.Header().Get("Location"))
}

func TestCallbackAccountRedirectKeepsExistingQuery(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "abc"})
	req.AddCookie(&http.Cookie{Name: oauthRedirectCookie, Value: "/account?tab=sso"})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000/account?tab=sso&connected=google", w.Header().Get("Location"))
}

func TestAppRedirectURLWithoutPublicURL(t *testing.T) {
	h := &AuthHandler{publicURL: ""}
	require.Equal(t, "/account", h.appRedirectURL("/account"))
	require.Equal(t, "/", h.appRedirectURL(""))
	require.Equal(t, "/", h.appRedirectURL("//evil.test"))
}

func TestLoginRejectsUnsafeRedirect(t *testing.T) {
	r, _ := setupRouter(t)

	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodGet, "/auth/google/login?redirect=//evil.test", nil)
	r.ServeHTTP(login, loginReq)
	require.Equal(t, http.StatusFound, login.Code)
	unsafe := findCookie(login, oauthRedirectCookie)
	if unsafe != nil {
		require.NotEqual(t, "//evil.test", unsafe.Value)
	}

	w := httptest.NewRecorder()
	cbReq := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	cbReq.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "abc"})
	cbReq.AddCookie(&http.Cookie{Name: oauthRedirectCookie, Value: "//evil.test"})
	r.ServeHTTP(w, cbReq)

	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000", w.Header().Get("Location"))
}

func TestCallbackInvalidState(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=bad&code=x", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogoutClearsSession(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(t.Context(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestPatchMeSuccess(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(t.Context(), "google", "code")
	require.NoError(t, err)

	payload, _ := json.Marshal(map[string]string{"locale": "ru"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/auth/me", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestReadyzWithoutDB(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestLoginUnknownProviderHTTP(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/unknown/login", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchMeInvalidJSON(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(t.Context(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/auth/me", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchMeDisplayName(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(t.Context(), "google", "code")
	require.NoError(t, err)

	payload, _ := json.Marshal(map[string]string{"display_name": "Updated Name"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/auth/me", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "Updated Name", body["display_name"])
}

func TestPatchMeRequiresField(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(t.Context(), "google", "code")
	require.NoError(t, err)

	payload, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/auth/me", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookReadBodyError(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/webhook", errReader{})
	req.Header.Set("X-Aegis-Webhook-Secret", "secret")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestSessionTokenFromBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer abc")
	require.Equal(t, "abc", SessionToken(c))
}
