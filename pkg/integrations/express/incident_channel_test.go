package express

import (
	"context"
	"encoding/json"
	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIncidentChannelMentionsAndPrivateFeedback(t *testing.T) {
	require.NoError(t, i18n.LoadMessages("../../i18n/messages"))
	var sent []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"status":"ok","result":"token"}`))
			return
		}
		require.Equal(t, "/api/v4/botx/notifications/direct/sync", r.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		sent = append(sent, payload)
		_, _ = w.Write([]byte(`{"status":"ok","result":{"sync_id":"message"}}`))
	}))
	defer server.Close()
	p := New(Config{Host: server.URL, BotID: "bot", SecretKey: "secret"})
	huid := uuid.New().String()
	post := integrations.IncidentChannelPost{Incident: integrations.IncidentRef{ID: uuid.New(), Severity: "critical", Title: "CPU", URL: "https://aegis.local/incidents"}, Kind: "escalated", TeamName: "Platform", Locale: "ru", OccurredAt: time.Now(), Actionable: true, OnCall: []integrations.OnCallPerson{{ExpressUserHuid: &huid}, {DisplayName: "Missing identity"}}}
	ref, err := p.SendIncidentEvent(context.Background(), post, "GROUP")
	require.NoError(t, err)
	require.Equal(t, "message", ref)
	require.Equal(t, "GROUP", sent[0]["group_chat_id"])
	notification := sent[0]["notification"].(map[string]any)
	require.Len(t, notification["mentions"], 1)
	require.Contains(t, notification["body"], "@{mention:")
	raw, _ := json.Marshal(notification)
	require.Contains(t, string(raw), "ack_incident")
	post.Actionable = false
	_, err = p.SendIncidentEvent(context.Background(), post, "GROUP")
	require.NoError(t, err)
	raw, _ = json.Marshal(sent[1])
	require.NotContains(t, string(raw), "ack_incident")
	require.NotContains(t, string(raw), huid)
	require.NoError(t, p.SendAckFeedback(context.Background(), "GROUP", huid, "Acknowledged"))
	require.Equal(t, []any{huid}, sent[2]["recipients"])
	require.Equal(t, "GROUP", sent[2]["group_chat_id"])
}

func TestSyncDeliveryFailuresAndLegacyFeedbackChatLookup(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"transient", "", 503}, {"permanent", "", 403}, {"bad JSON", "bad", 200}, {"missing reference", `{"status":"ok","result":{}}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					_, _ = w.Write([]byte(`{"status":"ok","result":"token"}`))
					return
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			p := New(Config{Host: server.URL, BotID: "bot", SecretKey: "secret"})
			_, err := p.SendIncidentEvent(t.Context(), integrations.IncidentChannelPost{}, "GROUP")
			require.Error(t, err)
			require.Equal(t, tc.status != 403, integrations.RetryableError(err))
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/botx/bots/bot/token":
			_, _ = w.Write([]byte(`{"status":"ok","result":"token"}`))
		case "/api/v1/botx/chats/personal":
			_, _ = w.Write([]byte(`{"status":"ok","result":{"group_chat_id":"PERSONAL"}}`))
		default:
			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Equal(t, "PERSONAL", payload["group_chat_id"])
			require.Equal(t, []any{"HUID"}, payload["recipients"])
			_, _ = w.Write([]byte(`{"status":"ok","result":{"sync_id":"message"}}`))
		}
	}))
	defer server.Close()
	p := New(Config{Host: server.URL, BotID: "bot", SecretKey: "secret"})
	require.NoError(t, p.SendAckFeedback(t.Context(), "", "HUID", "Acknowledged"))
}

// CHAT-FEEDBACK-EMPTY-PROVIDER (incident_channel.go): Blank feedback must have
// no network side effects with or without a chat ID; nonblank text must arrive
// unchanged. A local HTTP server exercises the provider's real request path,
// not a real BotX service or downstream message delivery.
func TestSendAckFeedbackBlankAndOriginalBody(t *testing.T) {
	for _, tc := range []struct {
		name, chat, body string
		wantSend         bool
	}{
		{"empty existing chat", "GROUP", "", false},
		{"empty missing chat", "", "", false},
		{"whitespace existing chat", "GROUP", " \t\n\r", false},
		{"whitespace missing chat", "", " \t\n\r", false},
		{"unicode existing chat", "GROUP", "\u2003\u00a0", false},
		{"unicode missing chat", "", "\u2003\u00a0", false},
		{"original text existing chat", "GROUP", " \tAcknowledged\u2003", true},
		{"original text missing chat", "", " \tAcknowledged\u2003", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			var sent map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				switch r.URL.Path {
				case "/api/v2/botx/bots/bot/token":
					_, _ = w.Write([]byte(`{"status":"ok","result":"token"}`))
				case "/api/v1/botx/chats/personal":
					_, _ = w.Write([]byte(`{"status":"ok","result":{"group_chat_id":"PERSONAL"}}`))
				case "/api/v4/botx/notifications/direct/sync":
					require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
					_, _ = w.Write([]byte(`{"status":"ok","result":{"sync_id":"message"}}`))
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			p := New(Config{Host: server.URL, BotID: "bot", SecretKey: "secret"})
			require.NoError(t, p.SendAckFeedback(t.Context(), tc.chat, "HUID", tc.body))
			if !tc.wantSend {
				require.Empty(t, paths)
				return
			}
			wantChat := tc.chat
			if wantChat == "" {
				wantChat = "PERSONAL"
				require.Contains(t, paths, "/api/v1/botx/chats/personal")
			}
			require.Equal(t, wantChat, sent["group_chat_id"])
			require.Equal(t, []any{"HUID"}, sent["recipients"])
			require.Equal(t, tc.body, sent["notification"].(map[string]any)["body"])
		})
	}
}
