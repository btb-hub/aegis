package express

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSendPageUsesFixture(t *testing.T) {
	responseRaw := readFixture(t, "notification_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/token"):
			require.NotEmpty(t, r.URL.Query().Get("signature"))
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "bot-token"})
		case strings.Contains(r.URL.Path, "/notifications/direct"):
			require.Equal(t, "Bearer bot-token", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(responseRaw)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := New(Config{BotID: "8dada2c8-67a6-4434-9dec-570d244e78ee", Host: server.URL, SecretKey: "secret"})
	provider.client = server.Client()

	huid := "6fafda2c-6505-57a5-a088-25ea5d1d0364"
	ref, err := provider.SendPage(t.Context(), integrations.IncidentRef{
		ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Title: "CPU high", Severity: "critical",
	}, integrations.PageRecipient{Locale: "en", ExpressUserHuid: &huid})
	require.NoError(t, err)
	require.Equal(t, "84a12e71-3efc-5c34-87d5-84e3d9ad64fd", ref)
}

func TestTestConnectionUsesFixture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "bot-token"})
	}))
	defer server.Close()

	provider := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"})
	provider.client = server.Client()
	require.NoError(t, provider.TestConnection(t.Context()))
}

func TestSignBotID(t *testing.T) {
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte("bot_id"))
	expected := strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
	require.Equal(t, expected, signBotID("bot_id", "secret"))
}

func TestVerifyJWT(t *testing.T) {
	token := signTestJWT(t, "secret", map[string]any{"exp": float64(time.Now().Add(time.Hour).Unix())})
	require.NoError(t, VerifyAuthorization("Bearer "+token, "secret"))
	require.Error(t, VerifyAuthorization("Bearer "+token, "wrong"))
}

func TestParseAckFromFixture(t *testing.T) {
	raw := readFixture(t, "command_ack.json")
	event, err := ParseCommandEvent(raw)
	require.NoError(t, err)
	incidentID, userHuid, err := ParseAckCommand(event)
	require.NoError(t, err)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", incidentID)
	require.Equal(t, "6fafda2c-6505-57a5-a088-25ea5d1d0364", userHuid)
}

func TestParseLinkFromFixture(t *testing.T) {
	raw := readFixture(t, "command_link.json")
	event, err := ParseCommandEvent(raw)
	require.NoError(t, err)
	code, userHuid, err := ParseLinkCommand(event)
	require.NoError(t, err)
	require.Equal(t, "ABC123", code)
	require.Equal(t, "6fafda2c-6505-57a5-a088-25ea5d1d0364", userHuid)
}

func TestSendPageMissingHuid(t *testing.T) {
	provider := New(Config{BotID: "bot", Host: "http://example.com", SecretKey: "secret"})
	_, err := provider.SendPage(t.Context(), integrations.IncidentRef{ID: uuid.New()}, integrations.PageRecipient{})
	require.Error(t, err)
}

func TestNewFromJSONValidation(t *testing.T) {
	_, err := NewFromJSON([]byte(`{"bot_id":"x"}`))
	require.Error(t, err)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	candidates := []string{
		filepath.Join("testdata", "express", name),
		filepath.Join("testdata", name),
		filepath.Join("..", "..", "apps", "worker", "testdata", "express", name),
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

func signTestJWT(t *testing.T, secret string, claims map[string]any) string {
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

func TestSignBotIDMatchesPythonExample(t *testing.T) {
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte("bot_id"))
	require.Equal(t, strings.ToUpper(hex.EncodeToString(mac.Sum(nil))), signBotID("bot_id", "secret"))
}
