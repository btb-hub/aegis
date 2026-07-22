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
	"github.com/stretchr/testify/require"
)

func TestIncidentsTimelineAndRoutingList(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	repo.events[incidentID] = []db.TimelineEvent{{ID: uuid.New(), IncidentID: incidentID, Kind: "created", Payload: []byte(`{}`), CreatedAt: time.Now()}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/"+incidentID.String()+"/timeline", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/routing-rules", nil)
	req.AddCookie(admin)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRoutingRulesUpdateAndDelete(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	workspaceID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, WorkspaceID: workspaceID, Name: "Platform"}

	body := bytes.NewBufferString(`{"workspace_id":"` + workspaceID.String() + `","team_id":"` + teamID.String() + `","match_labels":{"team":"platform"},"priority":10}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routing-rules", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	ruleID := created["id"].(string)

	patch := bytes.NewBufferString(`{"workspace_id":"` + workspaceID.String() + `","team_id":"` + teamID.String() + `","match_labels":{"team":"platform"},"priority":20}`)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/routing-rules/"+ruleID, patch)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/routing-rules/"+ruleID, nil)
	req.AddCookie(admin)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestIntegrationsTestConnection(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(map[string]string{
		"base_url": server.URL, "email": "ops@example.com", "api_token": "token", "project_key": "OPS",
	})
	id := uuid.New()
	repo.integrations[id] = db.Integration{ID: id, Kind: "jira", Name: "Jira", Config: cfg, Enabled: true}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+id.String()+"/test", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestIntegrationsDelete(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	id := uuid.New()
	repo.integrations[id] = db.Integration{ID: id, Kind: "slack", Name: "Slack", Config: []byte(`{}`), Enabled: true}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/"+id.String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestSlackCallbackInvalidPayload(t *testing.T) {
	r, repo := setupPhase2Router(t)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	body := "payload=not-json"
	base := "v0:" + ts + ":" + body
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(base))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/slack/interactive", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	_ = repo
}

func TestIntegrationsListAndDelete(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	body := bytes.NewBufferString(`{"kind":"slack","name":"Slack","config":{"bot_token":"x","signing_secret":"s"},"enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	req.AddCookie(admin)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var created map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &created)
}

func TestSlackCallbackAcknowledge(t *testing.T) {
	r, repo := setupPhase2Router(t)
	incidentID := uuid.New()
	userID := uuid.New()
	slackID := "U555"
	repo.users[userID] = db.User{ID: userID, Role: "member", SlackUserID: &slackID}
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}

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
	require.Equal(t, http.StatusOK, w.Code)
}
