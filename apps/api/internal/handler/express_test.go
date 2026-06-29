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
	repo.integrations[uuid.New()] = db.Integration{
		Kind: "express", Enabled: true,
		Config: expressConfigJSON(map[string]string{"bot_id": "bot", "host": "https://cts.example.com", "secret_key": "secret"}),
	}

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
	repo.integrations[uuid.New()] = db.Integration{
		Kind: "express", Enabled: true,
		Config: expressConfigJSON(map[string]string{"bot_id": "bot", "host": "https://cts.example.com", "secret_key": "secret"}),
	}
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
