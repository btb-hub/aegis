package express

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLookupPagingHUID(t *testing.T) {
	huid := uuid.NewString()
	user := map[string]any{"user_huid": huid, "active": true, "emails": []string{"ALICE@example.com"}}
	for _, test := range []struct {
		name   string
		result any
		status int
		valid  bool
	}{
		{name: "match", result: []any{user}, valid: true},
		{name: "same user repeated", result: []any{user, user}, valid: true},
		{name: "missing", result: []any{}},
		{name: "ambiguous", result: []any{user, map[string]any{"user_huid": uuid.NewString(), "active": true, "emails": []string{"alice@example.com"}}}},
		{name: "inactive", result: []any{map[string]any{"user_huid": huid, "active": false, "emails": []string{"alice@example.com"}}}},
		{name: "partial match", result: []any{map[string]any{"user_huid": huid, "active": true, "emails": []string{"otheralice@example.com"}}}},
		{name: "bad huid", result: []any{map[string]any{"user_huid": "bad", "active": true, "emails": []string{"alice@example.com"}}}},
		{name: "zero huid", result: []any{map[string]any{"user_huid": uuid.Nil.String(), "active": true, "emails": []string{"alice@example.com"}}}},
		{name: "permission", status: http.StatusForbidden},
		{name: "bad shape", result: "not an array"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/api/v2/botx/bots/bot/token" {
					_, _ = w.Write([]byte(`{"status":"ok","result":"bot-token"}`))
					return
				}
				require.Equal(t, "/api/v3/botx/users/by_email", r.URL.Path)
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "Bearer bot-token", r.Header.Get("Authorization"))
				var request struct {
					Emails []string `json:"emails"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, []string{"alice@example.com"}, request.Emails)
				if test.status != 0 {
					w.WriteHeader(test.status)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": test.result})
			}))
			defer server.Close()
			provider := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"})
			result, err := provider.LookupPagingHUID(context.Background(), "alice@example.com")
			if test.valid {
				require.NoError(t, err)
				require.Equal(t, huid, result)
			} else {
				require.Error(t, err)
			}
			_, err = provider.LookupPagingHUID(context.Background(), "")
			require.Error(t, err)
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer server.Close()
	_, err := New(Config{BotID: "bot", Host: server.URL, SecretKey: "secret"}).LookupPagingHUID(context.Background(), "alice@example.com")
	require.Error(t, err)
}
