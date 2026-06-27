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

func TestCreateTicketMissingKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer server.Close()

	provider := New(Config{BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	_, err := provider.CreateTicket(t.Context(), integrations.IncidentRef{ID: uuid.New(), Title: "CPU", Severity: "critical"})
	require.Error(t, err)
}

func TestCreateTicketInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	provider := New(Config{BaseURL: server.URL, Email: "ops@example.com", APIToken: "token", ProjectKey: "OPS"})
	_, err := provider.CreateTicket(t.Context(), integrations.IncidentRef{ID: uuid.New(), Title: "CPU", Severity: "critical"})
	require.Error(t, err)
}
