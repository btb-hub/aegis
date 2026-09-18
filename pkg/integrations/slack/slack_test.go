package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSendPageUsesFixture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/chat.postMessage", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "1234.5678"})
	}))
	defer server.Close()

	provider := New(Config{BotToken: "xoxb-test", SigningSecret: "secret"})
	provider.client = server.Client()
	provider.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = server.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})

	slackID := "U123"
	ref, err := provider.SendPage(t.Context(), integrations.IncidentRef{
		ID:       uuid.New(),
		Title:    "CPU high",
		Severity: "critical",
	}, integrations.PageRecipient{Locale: "en", SlackUserID: &slackID})
	require.NoError(t, err)
	require.Equal(t, "1234.5678", ref)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestVerifySignature(t *testing.T) {
	secret := "secret"
	body := []byte(`payload={"type":"block_actions"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	base := "v0:" + ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(base))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))
	require.NoError(t, VerifySignature(secret, ts, sig, body))
	require.Error(t, VerifySignature(secret, ts, "v0=bad", body))
}

func TestParseInteractiveAck(t *testing.T) {
	incidentID := uuid.New().String()
	payload := map[string]any{
		"type": "block_actions",
		"user": map[string]string{"id": "U999"},
		"actions": []map[string]string{
			{"action_id": "ack_incident", "value": incidentID},
		},
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	form := url.Values{"payload": {string(raw)}}
	gotID, userID, err := ParseInteractiveAck(form)
	require.NoError(t, err)
	require.Equal(t, incidentID, gotID)
	require.Equal(t, "U999", userID)
}

func TestTestConnectionUsesFixture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/auth.test", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	provider := New(Config{BotToken: "xoxb-test", SigningSecret: "secret"})
	provider.client = server.Client()
	provider.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = server.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})
	require.NoError(t, provider.TestConnection(t.Context()))
}

func TestAnnounceOnCallPostsToChannel(t *testing.T) {
	i18n.ResetForTests()
	require.NoError(t, i18n.LoadMessages(filepath.Join("..", "..", "..", "pkg", "i18n", "messages")))

	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/chat.postMessage", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "1.2"})
	}))
	defer server.Close()

	provider := New(Config{BotToken: "xoxb-test", SigningSecret: "secret", APIBaseURL: server.URL})
	provider.client = server.Client()
	slackID := "U123"
	err := provider.AnnounceOnCall(t.Context(), "C999", "Platform", []integrations.OnCallPerson{
		{DisplayName: "Alice", SlackUserID: &slackID},
		{DisplayName: "Bob"},
	}, "en")
	require.NoError(t, err)
	require.Equal(t, "C999", got["channel"])
	require.Contains(t, got["text"], "<@U123>")
	require.Contains(t, got["text"], "Bob")
}

func TestAnnounceOnCallRequiresChannel(t *testing.T) {
	provider := New(Config{BotToken: "xoxb-test", SigningSecret: "secret"})
	require.Error(t, provider.AnnounceOnCall(t.Context(), "", "Platform", nil, "en"))
}

func TestAnnounceOnCallEmptyPeople(t *testing.T) {
	i18n.ResetForTests()
	require.NoError(t, i18n.LoadMessages(filepath.Join("..", "..", "..", "pkg", "i18n", "messages")))
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()
	provider := New(Config{BotToken: "xoxb-test", SigningSecret: "secret", APIBaseURL: server.URL})
	provider.client = server.Client()
	require.NoError(t, provider.AnnounceOnCall(t.Context(), "C1", "Platform", nil, ""))
	require.Contains(t, got["text"], "No one is on call")
}

func TestAnnounceOnCallSlackError(t *testing.T) {
	i18n.ResetForTests()
	require.NoError(t, i18n.LoadMessages(filepath.Join("..", "..", "..", "pkg", "i18n", "messages")))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "channel_not_found"})
	}))
	defer server.Close()
	provider := New(Config{BotToken: "xoxb-test", SigningSecret: "secret", APIBaseURL: server.URL})
	provider.client = server.Client()
	require.Error(t, provider.AnnounceOnCall(t.Context(), "C1", "Platform", nil, "en"))
}
