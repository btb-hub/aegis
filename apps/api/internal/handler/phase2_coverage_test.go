package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type failingPhase2Repo struct {
	phase2HandlerRepo
	listRulesErr error
}

func (m *failingPhase2Repo) ListRoutingRules(context.Context) ([]db.RoutingRule, error) {
	if m.listRulesErr != nil {
		return nil, m.listRulesErr
	}
	return m.phase2HandlerRepo.ListRoutingRules(context.Background())
}

func TestIncidentsListInternalError(t *testing.T) {
	repo := &failingPhase2Repo{}
	repo.phase2HandlerRepo = *newPhase2HandlerRepo()
	repo.listIncidentsErr = pgx.ErrTxClosed
	r := setupPhase2RouterWithRepo(t, repo)
	admin := seedAdmin(t, r, &repo.phase2HandlerRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoutingRulesListInternalError(t *testing.T) {
	repo := &failingPhase2Repo{listRulesErr: pgx.ErrTxClosed}
	repo.phase2HandlerRepo = *newPhase2HandlerRepo()
	r := setupPhase2RouterWithRepo(t, repo)
	admin := seedAdmin(t, r, &repo.phase2HandlerRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routing-rules", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoutingRulesUpdateInvalidBody(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/routing-rules/"+uuid.New().String(), bytes.NewBufferString(`{`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSlackCallbackUserNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}

	payload := map[string]any{
		"type": "block_actions",
		"user": map[string]string{"id": "U404"},
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
	require.Equal(t, http.StatusNotFound, w.Code)
}

func setupPhase2RouterWithRepo(t *testing.T, repo *failingPhase2Repo) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{SessionTTL: time.Hour, PublicURL: "http://localhost:8080"}
	auth := service.NewAuthService(cfg, repo, repo, &authMockOIDC{})
	incidents := service.NewIncidentService(repo)
	routingRules := service.NewRoutingService(repo)
	integrationsSvc := service.NewIntegrationService(repo, cfg.PublicURL)
	expressLinks := service.NewExpressLinkService(repo)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &authMockAlertRepo{id: uuid.New()})
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth, cfg.PublicURL).Register(r)
	NewAlertHandler(alerts).Register(r)
	NewIncidentHandler(incidents, auth).Register(r)
	NewRoutingHandler(routingRules, auth).Register(r)
	NewIntegrationHandler(integrationsSvc, auth).Register(r)
	NewSlackCallbackHandler(incidents, "secret").Register(r)
	NewExpressCallbackHandler(incidents, expressLinks, integrationsSvc).Register(r)
	NewExpressLinkHandler(expressLinks, auth).Register(r)
	return r
}
