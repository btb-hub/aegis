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
