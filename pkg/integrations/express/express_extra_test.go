package express

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTestConnectionUsesCachedToken(t *testing.T) {
	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "bot-token"})
	}))
	defer server.Close()

	provider := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"})
	provider.client = server.Client()
	require.NoError(t, provider.TestConnection(t.Context()))
	require.NoError(t, provider.TestConnection(t.Context()))
	require.Equal(t, int32(1), tokenCalls.Load())
}

func TestSendPageDefaultsLocale(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "bot-token"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": map[string]string{"sync_id": "sync-1"}})
	}))
	defer server.Close()

	provider := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"})
	provider.client = server.Client()
	huid := "6fafda2c-6505-57a5-a088-25ea5d1d0364"
	_, err := provider.SendPage(t.Context(), integrations.IncidentRef{
		ID: uuid.New(), Title: "CPU", Severity: "warning",
	}, integrations.PageRecipient{ExpressUserHuid: &huid})
	require.NoError(t, err)
}

func TestSendPageNotificationStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "bot-token"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "error"})
	}))
	defer server.Close()

	provider := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"})
	provider.client = server.Client()
	huid := "6fafda2c-6505-57a5-a088-25ea5d1d0364"
	_, err := provider.SendPage(t.Context(), integrations.IncidentRef{
		ID: uuid.New(), Title: "CPU", Severity: "warning",
	}, integrations.PageRecipient{Locale: "en", ExpressUserHuid: &huid})
	require.Error(t, err)
}

func TestSendPageWithoutSyncID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "bot-token"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": map[string]string{}})
	}))
	defer server.Close()

	provider := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"})
	provider.client = server.Client()
	huid := "6fafda2c-6505-57a5-a088-25ea5d1d0364"
	ref, err := provider.SendPage(t.Context(), integrations.IncidentRef{
		ID: uuid.New(), Title: "CPU", Severity: "warning",
	}, integrations.PageRecipient{Locale: "en", ExpressUserHuid: &huid})
	require.NoError(t, err)
	_, err = uuid.Parse(ref)
	require.NoError(t, err)
}

func TestEnsureTokenFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":"error"}`))
	}))
	defer server.Close()

	provider := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"})
	provider.client = server.Client()
	require.Error(t, provider.TestConnection(t.Context()))
}

func TestSendPageRetriesAfterNotificationFailure(t *testing.T) {
	var notifyCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/token") {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "bot-token"})
			return
		}
		if notifyCalls.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`error`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": map[string]string{"sync_id": "sync-2"}})
	}))
	defer server.Close()

	provider := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"})
	provider.client = server.Client()
	huid := "6fafda2c-6505-57a5-a088-25ea5d1d0364"
	ref, err := provider.SendPage(t.Context(), integrations.IncidentRef{
		ID: uuid.New(), Title: "CPU", Severity: "warning",
	}, integrations.PageRecipient{Locale: "en", ExpressUserHuid: &huid})
	require.NoError(t, err)
	require.Equal(t, "sync-2", ref)
	require.Equal(t, int32(2), notifyCalls.Load())
}

func TestWithRetryRespectsCancelledContext(t *testing.T) {
	provider := New(Config{BotID: "bot", Host: "http://example.com", SecretKey: "secret"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := provider.withRetry(ctx, func() error {
		return fmt.Errorf("retry")
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestNewFromJSONInvalidJSON(t *testing.T) {
	_, err := NewFromJSON([]byte(`{`))
	require.Error(t, err)
}

func TestVerifyJWTInvalidFormat(t *testing.T) {
	_, err := verifyJWT("not-a-jwt", "secret")
	require.Error(t, err)
}

func TestParseAckCommandFromBody(t *testing.T) {
	event, err := ParseCommandEvent([]byte(`{"command":{"body":"/ack_incident inc-1"},"from":{"user_huid":"huid-1"}}`))
	require.NoError(t, err)
	incidentID, userHuid, err := ParseAckCommand(event)
	require.NoError(t, err)
	require.Equal(t, "inc-1", incidentID)
	require.Equal(t, "huid-1", userHuid)
}

func TestEnsureTokenInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": ""})
	}))
	defer server.Close()

	provider := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"})
	provider.client = server.Client()
	require.Error(t, provider.TestConnection(t.Context()))
}
