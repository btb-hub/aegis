package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestListAlertsRequiresSession(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListAlertsWithSearch(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?q=cpu", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	items, ok := body["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
}

func TestListAlertsRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		SessionTTL: time.Hour,
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
	users := &authMockUsers{users: map[uuid.UUID]db.User{}}
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &failListAlertRepo{})

	r := gin.New()
	NewAlertHandler(alerts, auth).Register(r)

	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

type failListAlertRepo struct{}

func (f *failListAlertRepo) CreateAlertAndJob(context.Context, db.CreateAlertJobInput) (db.CreateAlertJobResult, error) {
	return db.CreateAlertJobResult{}, nil
}

func (f *failListAlertRepo) ListAlerts(context.Context, db.ListAlertsParams) ([]db.Alert, error) {
	return nil, errListAlerts
}

var errListAlerts = &listAlertsError{}

type listAlertsError struct{}

func (e *listAlertsError) Error() string { return "list failed" }
