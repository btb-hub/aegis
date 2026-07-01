package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateAssigneeNoOpWhenMissingInputs(t *testing.T) {
	provider := New(Config{BaseURL: "http://example.com", Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.NoError(t, provider.UpdateAssignee(t.Context(), "", "ops@example.com"))
	require.NoError(t, provider.UpdateAssignee(t.Context(), "OPS-1", ""))
}

func TestUpdateAssigneeSuccess(t *testing.T) {
	var putCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/user/search":
			require.Equal(t, http.MethodGet, r.Method)
			_ = json.NewEncoder(w).Encode([]map[string]string{{"accountId": "acct-1"}})
		case "/rest/api/3/issue/OPS-42":
			require.Equal(t, http.MethodPut, r.Method)
			putCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := New(Config{BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.NoError(t, provider.UpdateAssignee(t.Context(), "OPS-42", "ops@example.com"))
	require.True(t, putCalled)
}

func TestUpdateAssigneeSkipsWhenUserNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/rest/api/3/user/search", r.URL.Path)
		_ = json.NewEncoder(w).Encode([]map[string]string{})
	}))
	defer server.Close()

	provider := New(Config{BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.NoError(t, provider.UpdateAssignee(t.Context(), "OPS-42", "missing@example.com"))
}

func TestUpdateAssigneePutFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/user/search":
			_ = json.NewEncoder(w).Encode([]map[string]string{{"accountId": "acct-1"}})
		case "/rest/api/3/issue/OPS-42":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad"}`))
		}
	}))
	defer server.Close()

	provider := New(Config{BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.Error(t, provider.UpdateAssignee(t.Context(), "OPS-42", "ops@example.com"))
}

func TestUpdateAssigneeUserSearchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	provider := New(Config{BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.Error(t, provider.UpdateAssignee(t.Context(), "OPS-42", "ops@example.com"))
}
