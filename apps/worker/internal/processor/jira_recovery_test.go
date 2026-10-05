package processor

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJiraSyncPinsInferredDeploymentWithoutCredentials(t *testing.T) {
	for _, auth := range []string{"bearer", "basic"} {
		t.Run(auth, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				if auth == "basic" {
					require.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
				} else {
					require.Equal(t, "/rest/api/2/search", r.URL.Path)
				}
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer server.Close()
			f := newJiraSyncFixture(server.URL)
			var cfg map[string]any
			require.NoError(t, json.Unmarshal(f.connector.Config, &cfg))
			delete(cfg, "deployment")
			cfg["auth_type"] = auth
			cfg["email"] = "ops@example.test"
			f.connector.Config, _ = json.Marshal(cfg)
			require.Error(t, NewJiraSyncProcessor(f, "").Handle(t.Context(), jiraSyncJob(f)))
			var pinned map[string]any
			require.NoError(t, json.Unmarshal(f.state.Config, &pinned))
			expected := "server_dc"
			if auth == "basic" {
				expected = "cloud"
			}
			require.Equal(t, expected, pinned["deployment"])
			for _, key := range []string{"api_token", "email", "auth_type"} {
				require.NotContains(t, pinned, key)
			}
			require.False(t, f.state.CreateAttempted)
		})
	}
}
func TestJiraSyncDefiniteRejectionClearsAttempt(t *testing.T) {
	for _, operation := range []string{"create", "comment"} {
		t.Run(operation, func(t *testing.T) {
			posts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					posts++
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if operation == "create" {
					_, _ = w.Write([]byte(`{"issues":[]}`))
				} else {
					_, _ = w.Write([]byte(`{"comments":[],"total":0}`))
				}
			}))
			defer server.Close()
			f := newJiraSyncFixture(server.URL)
			if operation == "comment" {
				key := "OPS-1"
				f.state.IssueKey = &key
				f.state.IntegrationID = &f.connector.ID
				f.state.AssigneeID = f.incident.AssigneeID
			}
			require.Error(t, NewJiraSyncProcessor(f, "").Handle(t.Context(), jiraSyncJob(f)))
			require.Equal(t, 1, posts)
			require.False(t, f.state.CreateAttempted)
			require.False(t, f.comments[0].Attempted)
		})
	}
}
