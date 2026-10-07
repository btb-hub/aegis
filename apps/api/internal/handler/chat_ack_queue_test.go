package handler

import (
	"bytes"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSlackAckReceiptDeduplicatesWithoutChangingIncident(t *testing.T) {
	r, repo := setupPhase2Router(t)
	id := uuid.New()
	repo.incidents[id] = db.Incident{ID: id, TeamID: uuid.New(), Status: "open"}
	global, _ := seedSlackWorkspace(repo, id, "secret")
	body := slackAckBody(t, id.String(), "U-UNLINKED")
	for range 2 {
		started := time.Now()
		response := sendSlackCallback(r, body, "secret", started)
		require.Equal(t, http.StatusOK, response.Code)
		require.Empty(t, response.Body.String())
		require.Less(t, time.Since(started), 3*time.Second)
	}
	require.Len(t, repo.chatAckRequests, 1)
	require.Equal(t, global, repo.chatAckRequests[0].IntegrationID)
	require.Equal(t, "open", repo.incidents[id].Status)
	require.Empty(t, repo.cancelledEscalations)
	require.Zero(t, repo.slackUserLookups)
	bad := sendSlackCallback(r, body, "wrong", time.Now())
	require.Equal(t, http.StatusUnauthorized, bad.Code)
	require.Len(t, repo.chatAckRequests, 1)
}

func TestExpressAckReceiptDeduplicatesAcrossAliases(t *testing.T) {
	r, repo := setupPhase2Router(t)
	seedExpressIntegration(t, repo)
	body := readExpressFixture(t, "command_ack.json")
	token := signExpressJWT(t, "secret", map[string]any{"exp": float64(time.Now().Add(time.Hour).Unix())})
	for _, path := range []string{"/api/v1/callbacks/express/command", "/api/v1/callbacks/express/bot/command", "/api/v1/callbacks/express/bot"} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		r.ServeHTTP(response, req)
		require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	}
	require.Len(t, repo.chatAckRequests, 1)
}
