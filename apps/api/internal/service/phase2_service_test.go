package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestIntegrationServiceTestJira(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(map[string]string{
		"base_url": server.URL, "email": "ops@example.com", "api_token": "token", "project_key": "OPS",
	})
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{ID: id, Kind: "jira", Config: cfg, Enabled: true}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	require.NoError(t, svc.Test(context.Background(), id))
}

func TestIntegrationServiceBuildRegistry(t *testing.T) {
	cfg, _ := json.Marshal(map[string]string{
		"base_url": "https://jira.example.com", "email": "ops@example.com", "api_token": "token", "project_key": "OPS",
	})
	repo := &integrationMockRepo{items: []db.Integration{{ID: uuid.New(), Kind: "jira", Config: cfg, Enabled: true}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	reg, err := svc.LoadRegistry(context.Background())
	require.NoError(t, err)
	_, ok := reg.Ticket("jira")
	require.True(t, ok)
}

func TestIntegrationServiceTestNotFound(t *testing.T) {
	svc := NewIntegrationService(&integrationMockRepo{}, "http://localhost:8080")
	err := svc.Test(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestIntegrationServiceDeleteNotFound(t *testing.T) {
	repo := &integrationMockRepo{deleteErr: pgx.ErrNoRows}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	err := svc.Delete(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestIncidentServiceAlerts(t *testing.T) {
	id := uuid.New()
	svc := NewIncidentService(&incidentMockRepo{
		incident: db.Incident{ID: id, Status: "open"},
		alerts:   []db.Alert{{ID: uuid.New(), Title: "CPU"}},
	})
	alerts, err := svc.Alerts(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
}

func TestRoutingServiceCreateTeamNotFound(t *testing.T) {
	svc := NewRoutingService(&routingMockRepo{})
	_, err := svc.CreateRule(context.Background(), uuid.New(), map[string]string{"team": "platform"}, 1)
	require.Error(t, err)
}

func TestRoutingServiceValidateEmptyLabelValue(t *testing.T) {
	teamID := uuid.New()
	svc := NewRoutingService(&routingMockRepo{team: db.Team{ID: teamID}})
	_, err := svc.CreateRule(context.Background(), teamID, map[string]string{"team": ""}, 1)
	require.Error(t, err)
}
