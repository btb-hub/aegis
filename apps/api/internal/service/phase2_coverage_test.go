package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestMapRoutingErrorForeignKey(t *testing.T) {
	err := mapRoutingError(&pgconn.PgError{Code: "23503"})
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestRoutingServiceUpdateRuleTeamNotFound(t *testing.T) {
	svc := NewRoutingService(&routingMockRepo{})
	_, err := svc.UpdateRule(context.Background(), uuid.New(), uuid.New(), map[string]string{"team": "platform"}, 1)
	require.Error(t, err)
}

func TestRoutingServiceDeleteRuleNotFound(t *testing.T) {
	repo := &routingMockRepo{deleteErr: pgx.ErrNoRows}
	svc := NewRoutingService(repo)
	err := svc.DeleteRule(context.Background(), uuid.New())
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestIncidentServiceGetNotFound(t *testing.T) {
	svc := NewIncidentService(&incidentMockRepo{})
	_, err := svc.Get(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestIncidentServiceTimelineSuccess(t *testing.T) {
	incidentID := uuid.New()
	svc := NewIncidentService(&incidentMockRepo{
		incident: db.Incident{ID: incidentID, Status: "open"},
		events:   []db.TimelineEvent{{ID: uuid.New(), Kind: "created", Payload: []byte(`{}`), CreatedAt: time.Now()}},
	})
	events, err := svc.Timeline(context.Background(), incidentID)
	require.NoError(t, err)
	require.Len(t, events, 1)
}

func TestIncidentServiceAlertsNotFound(t *testing.T) {
	svc := NewIncidentService(&incidentMockRepo{})
	_, err := svc.Alerts(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestIncidentServiceResolveConflict(t *testing.T) {
	incidentID := uuid.New()
	svc := NewIncidentService(&incidentMockRepo{incident: db.Incident{ID: incidentID, Status: "resolved"}})
	_, err := svc.Resolve(context.Background(), incidentID, uuid.New())
	require.Error(t, err)
}

func TestIntegrationServiceBuildRegistrySkipsBadJira(t *testing.T) {
	repo := &integrationMockRepo{items: []db.Integration{
		{ID: uuid.New(), Kind: "jira", Config: []byte(`{`), Enabled: true},
		{ID: uuid.New(), Kind: "slack", Config: mustJSON(map[string]string{"bot_token": "xoxb-test", "signing_secret": "secret"}), Enabled: true},
	}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	reg, err := svc.LoadRegistry(context.Background())
	require.NoError(t, err)
	_, ok := reg.Ticket("jira")
	require.False(t, ok)
	_, ok = reg.Chat("slack")
	require.True(t, ok)
}

func TestIntegrationServiceTestSlackSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	cfg := mustJSON(map[string]string{
		"bot_token": "xoxb-test", "signing_secret": "secret", "api_base_url": server.URL,
	})
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{ID: id, Kind: "slack", Config: cfg, Enabled: true}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	require.NoError(t, svc.Test(context.Background(), id))
}

func TestIntegrationServiceUpsertDefaultName(t *testing.T) {
	repo := &integrationMockRepo{}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	item, err := svc.Upsert(context.Background(), "slack", "", json.RawMessage(`{}`), true)
	require.NoError(t, err)
	require.Equal(t, "slack", item.Name)
}

func TestIntegrationServiceTestProviderNotConfigured(t *testing.T) {
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{ID: id, Kind: "express", Config: json.RawMessage(`{}`), Enabled: true}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	err := svc.Test(context.Background(), id)
	require.Error(t, err)
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestIntegrationServiceTestExpressSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "bot-token"})
	}))
	defer server.Close()

	cfg := mustJSON(map[string]string{
		"bot_id": "bot", "host": server.URL, "secret_key": "secret",
	})
	id := uuid.New()
	repo := &integrationMockRepo{items: []db.Integration{{ID: id, Kind: "express", Config: cfg, Enabled: true}}}
	svc := NewIntegrationService(repo, "http://localhost:8080")
	require.NoError(t, svc.Test(context.Background(), id))
}

func TestMapIntegrationErrorGeneric(t *testing.T) {
	err := mapIntegrationError(pgx.ErrTxClosed)
	require.ErrorIs(t, err, pgx.ErrTxClosed)
}

func TestMapIncidentErrorGeneric(t *testing.T) {
	err := mapIncidentError(pgx.ErrTxClosed)
	require.ErrorIs(t, err, pgx.ErrTxClosed)
}

func mustJSON(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}
