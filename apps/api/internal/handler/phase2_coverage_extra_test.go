package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestRoutingRulesUpdateInvalidTeamID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	body := bytes.NewBufferString(`{"workspace_id":"` + uuid.New().String() + `","team_id":"bad","match_labels":{"team":"platform"},"priority":1}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/routing-rules/"+uuid.New().String(), body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSlackCallbackInvalidIncidentID(t *testing.T) {
	r, _ := setupPhase2Router(t)
	payload := map[string]any{
		"type": "block_actions",
		"user": map[string]string{"id": "U123"},
		"actions": []map[string]string{{"action_id": "ack_incident", "value": "not-a-uuid"}},
	}
	raw, _ := json.Marshal(payload)
	form := "payload=" + string(raw)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	base := "v0:" + ts + ":" + form
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(base))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/slack/interactive", bytes.NewBufferString(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIntegrationsListEmpty(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	_ = repo
}

func TestIncidentsAcknowledgeUnauthorized(t *testing.T) {
	r, repo := setupPhase2Router(t)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/acknowledge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIntegrationsTestSlack(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	cfg, _ := json.Marshal(map[string]string{
		"bot_token": "xoxb-test", "signing_secret": "secret", "api_base_url": server.URL,
	})
	id := uuid.New()
	repo.integrations[id] = db.Integration{ID: id, Kind: "slack", Name: "Slack", Config: cfg, Enabled: true}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+id.String()+"/test", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRoutingRulesCreateInvalidTeamID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	body := bytes.NewBufferString(`{"workspace_id":"` + uuid.New().String() + `","team_id":"bad","match_labels":{"team":"platform"},"priority":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routing-rules", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	_ = repo
}

func TestIncidentsResolveUnauthorized(t *testing.T) {
	r, repo := setupPhase2Router(t)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/resolve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSlackCallbackAcknowledgeConflict(t *testing.T) {
	r, repo := setupPhase2Router(t)
	incidentID := uuid.New()
	userID := uuid.New()
	slackID := "U777"
	repo.users[userID] = db.User{ID: userID, Role: "member", SlackUserID: &slackID}
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "acknowledged", Severity: "critical", Title: "CPU", Fingerprint: "fp"}

	payload := map[string]any{
		"type": "block_actions",
		"user": map[string]string{"id": slackID},
		"actions": []map[string]string{{"action_id": "ack_incident", "value": incidentID.String()}},
	}
	raw, _ := json.Marshal(payload)
	form := "payload=" + string(raw)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	base := "v0:" + ts + ":" + form
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(base))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/slack/interactive", bytes.NewBufferString(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestIncidentsGetAlertsError(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	repo.alertListErr = pgx.ErrTxClosed
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/"+incidentID.String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoutingRulesUpdateNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	workspaceID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, WorkspaceID: workspaceID, Name: "Platform"}
	body := bytes.NewBufferString(`{"workspace_id":"` + workspaceID.String() + `","team_id":"` + teamID.String() + `","match_labels":{"team":"platform"},"priority":1}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/routing-rules/"+uuid.New().String(), body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestIntegrationsListInternalError(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.listIntegrationsErr = pgx.ErrTxClosed
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIncidentsTimelineNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/"+uuid.New().String()+"/timeline", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
	_ = repo
}

func TestIntegrationsUpsertWithWorkspaceReturnsConflict(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	workspaceID := uuid.New()
	body, _ := json.Marshal(map[string]any{
		"kind":         "jira",
		"name":         "Platform Jira",
		"enabled":      true,
		"workspace_id": workspaceID.String(),
		"config":       map[string]string{"project_key": "OPS"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations", bytes.NewReader(body))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestIntegrationsUpsertInvalidWorkspaceID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	body, _ := json.Marshal(map[string]any{
		"kind":         "jira",
		"name":         "Platform Jira",
		"enabled":      true,
		"workspace_id": "bad",
		"config":       map[string]string{"project_key": "OPS"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations", bytes.NewReader(body))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	_ = repo
}

func TestIntegrationsDeleteIntegration(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	id := uuid.New()
	repo.integrations[id] = db.Integration{ID: id, Kind: "jira", Name: "Jira", Config: []byte(`{"project_key":"OPS"}`), Enabled: true}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/"+id.String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestIntegrationsDeleteNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/"+uuid.New().String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
	_ = repo
}

func TestIntegrationsTestInvalidID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/not-a-uuid/test", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	_ = repo
}
