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

func TestCreateTicketUsesFixture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/rest/api/3/issue", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"key": "OPS-42"})
	}))
	defer server.Close()

	provider := New(Config{
		BaseURL:    server.URL,
		Email:      "ops@example.com",
		APIToken:   "token",
		ProjectKey: "OPS",
	})
	key, err := provider.CreateTicket(t.Context(), integrations.IncidentRef{
		ID:       uuid.New(),
		Title:    "CPU high",
		Severity: "critical",
	})
	require.NoError(t, err)
	require.Equal(t, "OPS-42", key)
}

func TestTestConnectionUsesFixture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/rest/api/3/myself", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := New(Config{BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.NoError(t, provider.TestConnection(t.Context()))
}

func TestNewFromJSONRequiresProjectKey(t *testing.T) {
	_, err := NewFromJSON([]byte(`{"base_url":"https://jira.example.com","email":"a@b.com","api_token":"x"}`))
	require.Error(t, err)
}
