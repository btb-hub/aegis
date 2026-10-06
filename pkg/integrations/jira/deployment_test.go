package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestServerDCUsesV2AndText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer pat", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/rest/api/2/myself":
			w.WriteHeader(200)
		case "/rest/api/2/issue":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.IsType(t, "", body["fields"].(map[string]any)["description"])
			_ = json.NewEncoder(w).Encode(map[string]string{"key": "OPS-1"})
		default:
			t.Errorf("wrong Server/DC path: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	p := New(Config{BaseURL: server.URL, APIToken: "pat", ProjectKey: "OPS"})
	require.NoError(t, p.TestConnection(t.Context()))
	_, err := p.CreateTicket(t.Context(), integrations.IncidentRef{ID: uuid.New(), Title: "test"})
	require.NoError(t, err)
}

func TestJiraDoesNotFollowLoginRedirect(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; http.Redirect(w, r, "/login.jsp", 302) }))
	defer server.Close()
	err := New(Config{BaseURL: server.URL, APIToken: "secret"}).TestConnection(t.Context())
	require.ErrorContains(t, err, "302")
	require.Equal(t, 1, calls)
	require.NotContains(t, err.Error(), "secret")
}
