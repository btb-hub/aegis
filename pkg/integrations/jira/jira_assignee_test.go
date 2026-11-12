package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateAssigneeNoOpWhenMissingInputs(t *testing.T) {
	provider := New(Config{Deployment: "cloud", BaseURL: "http://example.com", Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
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

	provider := New(Config{Deployment: "cloud", BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.NoError(t, provider.UpdateAssignee(t.Context(), "OPS-42", "ops@example.com"))
	require.True(t, putCalled)
}

func TestUpdateAssigneeReportsUserNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/rest/api/3/user/search", r.URL.Path)
		_ = json.NewEncoder(w).Encode([]map[string]string{})
	}))
	defer server.Close()

	provider := New(Config{Deployment: "cloud", BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.ErrorContains(t, provider.UpdateAssignee(t.Context(), "OPS-42", "missing@example.com"), "no active exact Jira user")
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

	provider := New(Config{Deployment: "cloud", BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.Error(t, provider.UpdateAssignee(t.Context(), "OPS-42", "ops@example.com"))
}

func TestUpdateAssigneeUserSearchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	provider := New(Config{Deployment: "cloud", BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.Error(t, provider.UpdateAssignee(t.Context(), "OPS-42", "ops@example.com"))
}

// JIRA-ASSIGN-LOOKUP: The HTTP fixture exercises Server/DC lookup and issue update,
// not a live Jira directory. A local-part match must not select a fuzzy or inactive user.
func TestServerAssigneeFallback(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		success        bool
	}{
		{"exact", `[{"name":"vkrasnov","emailAddress":"vkrasnov@example.com","active":true}]`, true},
		{"hidden email", `[{"name":"vkrasnov","active":true}]`, true},
		{"fuzzy", `[{"name":"vkrasnov2","active":true}]`, false},
		{"conflicting email", `[{"name":"vkrasnov","emailAddress":"other@example.com","active":true}]`, false},
		{"inactive", `[{"name":"vkrasnov","active":false}]`, false},
		{"ambiguous", `[{"name":"vkrasnov","active":true},{"name":"VKrasnov","active":true}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assigned := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					assigned = true
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, map[string]any{"name": "vkrasnov"}, body["fields"].(map[string]any)["assignee"])
					w.WriteHeader(204)
					return
				}
				switch r.URL.Path {
				case "/rest/api/2/user/search":
					_, _ = w.Write([]byte(`[]`)) // This instance cannot find the user even by username.
				case "/rest/api/2/user/assignable/search":
					require.Equal(t, "vkrasnov", r.URL.Query().Get("username"))
					require.Equal(t, "OPS-1", r.URL.Query().Get("issueKey"))
					_, _ = w.Write([]byte(tc.response))
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
				}
			}))
			defer srv.Close()
			err := New(Config{BaseURL: srv.URL, Deployment: "server_dc"}).UpdateAssignee(t.Context(), "OPS-1", "vkrasnov@example.com")
			if tc.success {
				require.NoError(t, err)
				require.True(t, assigned)
			} else {
				require.Error(t, err)
				require.False(t, assigned)
			}
		})
	}
}

// JIRA-ASSIGN-LOOKUP: A failed fallback is surfaced, never treated as a match;
// this fixture exercises HTTP status handling without contacting a Jira server.
func TestServerAssigneeFallbackError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("username") == "vkrasnov@example.com" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	err := New(Config{BaseURL: srv.URL, Deployment: "server_dc"}).UpdateAssignee(t.Context(), "OPS-1", "vkrasnov@example.com")
	require.ErrorContains(t, err, "403")
}

// JIRA-ASSIGN-LOOKUP: A single fuzzy email-search result is not evidence of identity.
func TestServerAssigneeRejectsSingleFuzzyEmailResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`[{"name":"other","emailAddress":"other@example.com","active":true}]`))
	}))
	defer srv.Close()
	err := New(Config{BaseURL: srv.URL, Deployment: "server_dc"}).UpdateAssignee(t.Context(), "OPS-1", "vkrasnov@example.com")
	require.Error(t, err)
}
