package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func expressConfigJSON(values map[string]string) []byte {
	raw, _ := json.Marshal(values)
	return raw
}

func TestExpressCallbackAcknowledge(t *testing.T) {
	r, repo := setupPhase2Router(t)
	incidentID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.New()
	huid := uuid.MustParse("6fafda2c-6505-57a5-a088-25ea5d1d0364")
	repo.users[userID] = db.User{ID: userID, Role: "member", ExpressUserHuid: db.ExpressHuidToPg(huid)}
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	seedExpressIntegration(t, repo)

	body := readExpressFixture(t, "command_ack.json")
	token := signExpressJWT(t, "secret", map[string]any{"exp": float64(time.Now().Add(time.Hour).Unix())})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/express/bot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestExpressCallbackInvalidSignature(t *testing.T) {
	r, repo := setupPhase2Router(t)
	seedExpressIntegration(t, repo)
	body := readExpressFixture(t, "command_ack.json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/express/bot", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer bad-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestExpressLinkCode(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/express-link-code", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestExpressBindExpressHuid(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	huid := "6fafda2c-6505-57a5-a088-25ea5d1d0364"
	body := bytes.NewBufferString(`{"express_user_huid":"` + huid + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/express-link", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, huid, resp["express_user_huid"])
}

func TestExpressBindExpressHuidInvalidBody(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/express-link", bytes.NewBufferString(`{`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpressBindExpressHuidInvalidHuid(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	body := bytes.NewBufferString(`{"express_user_huid":"not-a-uuid"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/express-link", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpressCallbackLinkCommand(t *testing.T) {
	r, repo := setupPhase2Router(t)
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID, Role: "member"}
	seedExpressIntegration(t, repo)

	body := readExpressFixture(t, "command_link.json")
	token := signExpressJWT(t, "secret", map[string]any{"exp": float64(time.Now().Add(time.Hour).Unix())})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/express/bot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestExpressCallbackInvalidEvent(t *testing.T) {
	r, repo := setupPhase2Router(t)
	seedExpressIntegration(t, repo)
	token := signExpressJWT(t, "secret", map[string]any{"exp": float64(time.Now().Add(time.Hour).Unix())})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/express/bot", bytes.NewBufferString(`{`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpressCallbackNoExpressIntegration(t *testing.T) {
	r, _ := setupPhase2Router(t)
	body := readExpressFixture(t, "command_ack.json")
	token := signExpressJWT(t, "secret", map[string]any{"exp": float64(time.Now().Add(time.Hour).Unix())})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/express/bot", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpressCallbackInvalidIncidentID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	seedExpressIntegration(t, repo)
	huid := uuid.MustParse("6fafda2c-6505-57a5-a088-25ea5d1d0364")
	repo.users[uuid.New()] = db.User{ID: uuid.New(), ExpressUserHuid: db.ExpressHuidToPg(huid)}

	body := []byte(`{"command":{"body":"/ack_incident not-a-uuid","data":{}},"from":{"user_huid":"` + huid.String() + `"}}`)
	token := signExpressJWT(t, "secret", map[string]any{"exp": float64(time.Now().Add(time.Hour).Unix())})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/express/bot", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpressCallbackAckUserNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	incidentID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	seedExpressIntegration(t, repo)

	body := readExpressFixture(t, "command_ack.json")
	token := signExpressJWT(t, "secret", map[string]any{"exp": float64(time.Now().Add(time.Hour).Unix())})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/express/bot", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func seedExpressIntegration(t *testing.T, repo *phase2HandlerRepo) {
	t.Helper()
	repo.integrations[uuid.New()] = db.Integration{
		Kind: "express", Enabled: true,
		Config: expressConfigJSON(map[string]string{"bot_id": "bot", "host": "https://cts.example.com", "secret_key": "secret"}),
	}
}

func readExpressFixture(t *testing.T, name string) []byte {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "..", "..", "worker", "testdata", "express", name),
		filepath.Join("..", "..", "..", "..", "pkg", "integrations", "express", "testdata", name),
	}
	for _, path := range candidates {
		raw, err := os.ReadFile(path)
		if err == nil {
			return raw
		}
	}
	t.Fatalf("fixture not found: %s", name)
	return nil
}

func signExpressJWT(t *testing.T, secret string, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	payloadEnc := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := header + "." + payloadEnc
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + sig
}
