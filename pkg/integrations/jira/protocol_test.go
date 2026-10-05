package jira

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssigneeNameEscapedQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			require.Equal(t, "a+b@example.com", r.URL.Query().Get("username"))
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "alex", "emailAddress": "a+b@example.com", "active": true}})
			return
		}
		var b map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
		require.Equal(t, map[string]any{"name": "alex"}, b["fields"].(map[string]any)["assignee"])
		w.WriteHeader(204)
	}))
	defer srv.Close()
	p := New(Config{BaseURL: srv.URL})
	require.NoError(t, p.UpdateAssignee(t.Context(), "OPS-1", "a+b@example.com"))
}
func TestCloudCommentADFAndServerText(t *testing.T) {
	for _, deployment := range []string{"cloud", "server_dc"} {
		t.Run(deployment, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var b map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
				if deployment == "cloud" {
					require.IsType(t, map[string]any{}, b["body"])
				} else {
					require.Equal(t, "line one\nline two", b["body"])
				}
				_ = json.NewEncoder(w).Encode(map[string]string{"id": "42"})
			}))
			defer srv.Close()
			p := New(Config{BaseURL: srv.URL, Deployment: deployment})
			id, err := p.AddComment(t.Context(), "OPS-1", "line one\nline two")
			require.NoError(t, err)
			require.Equal(t, "42", id)
		})
	}
}
