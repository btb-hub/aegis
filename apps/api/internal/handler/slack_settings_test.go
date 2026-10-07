package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func seedSlackWorkspace(repo *phase2HandlerRepo, incidentID uuid.UUID, secret string) (uuid.UUID, uuid.UUID) {
	workspaceID := uuid.New()
	incident := repo.incidents[incidentID]
	repo.teams[incident.TeamID] = db.Team{ID: incident.TeamID, WorkspaceID: workspaceID}
	globalID, slotID := uuid.New(), uuid.New()
	cfg, _ := json.Marshal(map[string]string{"bot_token": "test-bot-token", "signing_secret": secret})
	repo.integrations[globalID] = db.Integration{ID: globalID, Kind: "slack", Enabled: true, Config: cfg}
	mode := "inherit"
	repo.integrations[slotID] = db.Integration{ID: slotID, Kind: "slack", Enabled: true, WorkspaceID: &workspaceID, Mode: &mode, Config: []byte(`{}`)}
	return globalID, slotID
}

func slackAckBody(t *testing.T, incidentID, slackID string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"type": "block_actions", "user": map[string]string{"id": slackID},
		"actions": []map[string]string{{"action_id": "ack_incident", "value": incidentID}},
	})
	require.NoError(t, err)
	return url.Values{"payload": {string(raw)}}.Encode()
}

func sendSlackCallback(r *gin.Engine, body, secret string, at time.Time) *httptest.ResponseRecorder {
	ts := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("v0:" + ts + ":" + body))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/slack/interactive", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", "v0="+hex.EncodeToString(mac.Sum(nil)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSlackCallbackUsesUISavedCredentials(t *testing.T) {
	t.Setenv("SLACK_SIGNING_SECRET", "")
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID, userID := uuid.New(), uuid.New()
	slackID := "U-SAVED"
	repo.users[userID] = db.User{ID: userID, Role: "member", SlackUserID: &slackID}
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open"}
	globalID, _ := seedSlackWorkspace(repo, incidentID, "unused")
	delete(repo.integrations, globalID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations", bytes.NewBufferString(`{"kind":"slack","enabled":true,"config":{"bot_token":"test-bot-token","signing_secret":"saved-secret"}}`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	saved := httptest.NewRecorder()
	r.ServeHTTP(saved, req)
	require.Equal(t, http.StatusCreated, saved.Code)
	require.NotContains(t, saved.Body.String(), "saved-secret")
	w := sendSlackCallback(r, slackAckBody(t, incidentID.String(), slackID), "saved-secret", time.Now())
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "open", repo.incidents[incidentID].Status)
	require.Len(t, repo.chatAckRequests, 1)
	require.Empty(t, repo.cancelledEscalations)
}

func TestSlackCallbackWorkspaceCredentials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change string
		want   int
	}{
		{"inherit", "", http.StatusOK},
		{"custom without global", "custom", http.StatusOK},
		{"wrong secret", "wrong", http.StatusUnauthorized},
		{"another workspace secret", "cross_workspace", http.StatusUnauthorized},
		{"custom rejects global secret", "custom_global", http.StatusUnauthorized},
		{"disabled slot", "slot_disabled", http.StatusBadRequest},
		{"missing slot", "slot_missing", http.StatusBadRequest},
		{"missing global", "global_missing", http.StatusBadRequest},
		{"disabled global", "global_disabled", http.StatusBadRequest},
		{"incomplete global", "global_incomplete", http.StatusBadRequest},
		{"incomplete custom", "custom_incomplete", http.StatusBadRequest},
		{"expired timestamp", "expired", http.StatusUnauthorized},
		{"malformed payload", "malformed", http.StatusBadRequest},
		{"invalid incident id", "invalid_id", http.StatusBadRequest},
		{"unknown incident", "unknown", http.StatusNotFound},
		{"missing incident team", "missing_team", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SLACK_SIGNING_SECRET", "secret")
			r, repo := setupPhase2Router(t)
			incidentID, userID := uuid.New(), uuid.New()
			slackID := "U-CALLBACK"
			repo.users[userID] = db.User{ID: userID, Role: "member", SlackUserID: &slackID}
			repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open"}
			globalID, slotID := seedSlackWorkspace(repo, incidentID, "secret")
			global, slot := repo.integrations[globalID], repo.integrations[slotID]
			body := slackAckBody(t, incidentID.String(), slackID)
			signingSecret, at := "secret", time.Now()
			custom := "custom"
			switch tc.change {
			case "custom":
				slot.Mode, slot.Config = &custom, []byte(`{"bot_token":"custom-token","signing_secret":"custom-secret"}`)
				signingSecret = "custom-secret"
				delete(repo.integrations, globalID)
			case "wrong":
				signingSecret = "wrong-secret"
			case "cross_workspace":
				otherID, otherWorkspace := uuid.New(), uuid.New()
				repo.integrations[otherID] = db.Integration{ID: otherID, Kind: "slack", Enabled: true, WorkspaceID: &otherWorkspace, Mode: &custom, Config: []byte(`{"bot_token":"other-token","signing_secret":"other-secret"}`)}
				signingSecret = "other-secret"
			case "custom_global":
				slot.Mode, slot.Config = &custom, []byte(`{"bot_token":"custom-token","signing_secret":"custom-secret"}`)
			case "slot_disabled":
				slot.Enabled = false
			case "slot_missing":
				delete(repo.integrations, slotID)
			case "global_missing":
				delete(repo.integrations, globalID)
			case "global_disabled":
				global.Enabled = false
			case "global_incomplete":
				global.Config = []byte(`{"signing_secret":"secret"}`)
			case "custom_incomplete":
				slot.Mode, slot.Config = &custom, []byte(`{"signing_secret":"secret"}`)
			case "expired":
				at = at.Add(-10 * time.Minute)
			case "malformed":
				body = "payload=not-json"
			case "invalid_id":
				body = slackAckBody(t, "invalid", slackID)
			case "unknown":
				body = slackAckBody(t, uuid.New().String(), slackID)
			case "missing_team":
				delete(repo.teams, repo.incidents[incidentID].TeamID)
			}
			if _, ok := repo.integrations[globalID]; ok {
				repo.integrations[globalID] = global
			}
			if _, ok := repo.integrations[slotID]; ok {
				repo.integrations[slotID] = slot
			}
			w := sendSlackCallback(r, body, signingSecret, at)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			require.NotContains(t, w.Body.String(), signingSecret)
			if tc.want == http.StatusOK {
				require.Equal(t, "open", repo.incidents[incidentID].Status)
				require.Len(t, repo.chatAckRequests, 1)
				require.Empty(t, repo.cancelledEscalations)
			} else {
				require.Equal(t, "open", repo.incidents[incidentID].Status)
				require.Nil(t, repo.incidents[incidentID].AcknowledgedAt)
				require.Empty(t, repo.cancelledEscalations)
				require.Zero(t, repo.slackUserLookups)
			}
		})
	}
}

func TestSlackCallbackSecretRotationWithoutRestart(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID, userID := uuid.New(), uuid.New()
	slackID := "U-ROTATE"
	repo.users[userID] = db.User{ID: userID, Role: "member", SlackUserID: &slackID}
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open"}
	globalID, _ := seedSlackWorkspace(repo, incidentID, "old-secret")
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/integrations/"+globalID.String(), bytes.NewBufferString(`{"config":{"signing_secret":"new-secret"}}`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "new-secret")
	body := slackAckBody(t, incidentID.String(), slackID)
	rejected := sendSlackCallback(r, body, "old-secret", time.Now())
	require.Equal(t, http.StatusUnauthorized, rejected.Code)
	require.Equal(t, "open", repo.incidents[incidentID].Status)
	require.Empty(t, repo.cancelledEscalations)
	accepted := sendSlackCallback(r, body, "new-secret", time.Now())
	require.Equal(t, http.StatusOK, accepted.Code, accepted.Body.String())
	require.Equal(t, "open", repo.incidents[incidentID].Status)
	require.Len(t, repo.chatAckRequests, 1)
}

type slackCallbackFailureRepo struct {
	*phase2HandlerRepo
	fail string
}

func (r *slackCallbackFailureRepo) GetIncidentByID(ctx context.Context, id uuid.UUID) (db.Incident, error) {
	if r.fail == "incident" {
		return db.Incident{}, errors.New("repository unavailable")
	}
	return r.phase2HandlerRepo.GetIncidentByID(ctx, id)
}
func (r *slackCallbackFailureRepo) GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error) {
	if r.fail == "team" {
		return db.Team{}, errors.New("repository unavailable")
	}
	return r.phase2HandlerRepo.GetTeam(ctx, id)
}
func (r *slackCallbackFailureRepo) GetWorkspaceIntegration(ctx context.Context, id uuid.UUID, kind string) (db.Integration, error) {
	if r.fail == "slot" {
		return db.Integration{}, errors.New("repository unavailable")
	}
	return r.phase2HandlerRepo.GetWorkspaceIntegration(ctx, id, kind)
}
func (r *slackCallbackFailureRepo) GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error) {
	if r.fail == "global" {
		return db.Integration{}, errors.New("repository unavailable")
	}
	return r.phase2HandlerRepo.GetIntegrationByKind(ctx, kind)
}
func TestSlackCallbackRepositoryFailuresDoNotMutate(t *testing.T) {
	for _, fail := range []string{"incident", "team", "slot", "global"} {
		t.Run(fail, func(t *testing.T) {
			repo := &slackCallbackFailureRepo{phase2HandlerRepo: newPhase2HandlerRepo(), fail: fail}
			incidentID := uuid.New()
			repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open"}
			seedSlackWorkspace(repo.phase2HandlerRepo, incidentID, "secret")
			r := gin.New()
			incidents := service.NewIncidentService(repo, time.Hour, time.Minute)
			integrations := service.NewIntegrationService(repo, "https://aegis.example")
			NewSlackCallbackHandler(incidents, integrations).Register(r)
			w := sendSlackCallback(r, slackAckBody(t, incidentID.String(), "U123"), "secret", time.Now())
			require.Equal(t, http.StatusInternalServerError, w.Code)
			require.Equal(t, "open", repo.incidents[incidentID].Status)
			require.Empty(t, repo.cancelledEscalations)
			require.Zero(t, repo.slackUserLookups)
		})
	}
}

func TestSlackSettingsAreReadOnlyForMembersAndViewers(t *testing.T) {
	for _, role := range []string{"member", "viewer"} {
		t.Run(role, func(t *testing.T) {
			r, repo := setupPhase2Router(t)
			cookie := seedAdmin(t, r, repo)
			for id, user := range repo.users {
				user.Role = role
				repo.users[id] = user
			}
			id := uuid.New()
			original := db.Integration{ID: id, Kind: "slack", Enabled: true, Config: []byte(`{"bot_token":"stored-test-token","signing_secret":"stored-test-secret"}`)}
			repo.integrations[id] = original
			req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
			req.AddCookie(cookie)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
			require.NotContains(t, w.Body.String(), "stored-test-token")
			require.NotContains(t, w.Body.String(), "stored-test-secret")
			for _, action := range []struct{ method, path string }{
				{http.MethodPost, "/api/v1/integrations"},
				{http.MethodPatch, "/api/v1/integrations/" + id.String()},
				{http.MethodDelete, "/api/v1/integrations/" + id.String()},
				{http.MethodPost, "/api/v1/integrations/" + id.String() + "/test"},
			} {
				req := httptest.NewRequest(action.method, action.path, bytes.NewBufferString(`{"enabled":false,"config":{"signing_secret":"replacement"}}`))
				req.AddCookie(cookie)
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				require.Equal(t, http.StatusForbidden, w.Code, action.path)
				require.Equal(t, original, repo.integrations[id])
				require.Len(t, repo.integrations, 1)
			}
		})
	}
}
