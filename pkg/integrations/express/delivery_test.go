package express

import (
	"encoding/json"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageResolvesPersonalChat(t *testing.T) {
	huid := uuid.New().String()
	chat := uuid.New().String()
	sent := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/botx/bots/bot/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "token"})
		case "/api/v1/botx/chats/personal":
			require.Equal(t, huid, r.URL.Query().Get("user_huid"))
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": map[string]string{"group_chat_id": chat}})
		case "/api/v4/botx/notifications/direct":
			var b map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
			require.Equal(t, chat, b["group_chat_id"])
			sent = true
			w.WriteHeader(202)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": map[string]string{"sync_id": "sent"}})
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	p := New(Config{Host: srv.URL, BotID: "bot", SecretKey: "s"})
	_, err := p.SendPage(t.Context(), integrations.IncidentRef{ID: uuid.New()}, integrations.PageRecipient{ExpressUserHuid: &huid})
	require.NoError(t, err)
	require.True(t, sent)
}
