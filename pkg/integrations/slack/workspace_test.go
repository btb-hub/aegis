package slack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceID(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		valid      bool
	}{
		{name: "workspace", body: `{"ok":true,"team_id":"T123"}`, valid: true},
		{name: "invalid token", body: `{"ok":false,"error":"invalid_auth"}`},
		{name: "missing team", body: `{"ok":true}`},
		{name: "invalid JSON", body: `{`},
		{name: "http failure", status: http.StatusTooManyRequests},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/auth.test", r.URL.Path)
				require.Equal(t, "Bearer bot", r.Header.Get("Authorization"))
				if test.status != 0 {
					w.WriteHeader(test.status)
				}
				_, _ = w.Write([]byte(test.body))
			}))
			provider := New(Config{BotToken: "bot", APIBaseURL: server.URL})
			team, err := provider.WorkspaceID(context.Background())
			if test.valid {
				require.NoError(t, err)
				require.Equal(t, "T123", team)
			} else {
				require.Error(t, err)
			}
			server.Close()
			_, err = provider.WorkspaceID(context.Background())
			require.Error(t, err)
		})
	}
}
