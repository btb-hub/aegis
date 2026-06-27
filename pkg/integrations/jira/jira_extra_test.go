package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateTicketFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	defer server.Close()

	provider := New(Config{BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	_, err := provider.CreateTicket(t.Context(), integrations.IncidentRef{ID: uuid.New(), Title: "CPU", Severity: "critical"})
	require.Error(t, err)
}

func TestTestConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	provider := New(Config{BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	require.Error(t, provider.TestConnection(t.Context()))
}
