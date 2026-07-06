package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

func setupSavedViewRouter(t *testing.T) (*gin.Engine, *service.AuthService, *mockSavedViewRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{SessionTTL: time.Hour}
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	repo := &mockSavedViewRepo{views: map[uuid.UUID]db.SavedView{}}
	views := service.NewSavedViewService(repo)

	r := gin.New()
	NewSavedViewHandler(views, auth).Register(r)
	return r, auth, repo
}

type mockSavedViewRepo struct {
	views map[uuid.UUID]db.SavedView
}

func (m *mockSavedViewRepo) ListSavedViewsForUser(ctx context.Context, userID uuid.UUID) ([]db.SavedView, error) {
	items := make([]db.SavedView, 0)
	for _, view := range m.views {
		if view.OwnerID == userID || view.Shared {
			items = append(items, view)
		}
	}
	return items, nil
}

func (m *mockSavedViewRepo) GetSavedView(ctx context.Context, id uuid.UUID) (db.SavedView, error) {
	view, ok := m.views[id]
	if !ok {
		return db.SavedView{}, pgx.ErrNoRows
	}
	return view, nil
}

func (m *mockSavedViewRepo) CreateSavedView(ctx context.Context, ownerID uuid.UUID, name string, filter json.RawMessage, shared bool) (db.SavedView, error) {
	view := db.SavedView{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		Name:      name,
		Filter:    filter,
		Shared:    shared,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.views[view.ID] = view
	return view, nil
}

func (m *mockSavedViewRepo) UpdateSavedView(ctx context.Context, id, ownerID uuid.UUID, name string, filter json.RawMessage, shared bool) (db.SavedView, error) {
	view, ok := m.views[id]
	if !ok || view.OwnerID != ownerID {
		return db.SavedView{}, pgx.ErrNoRows
	}
	view.Name = name
	view.Filter = filter
	view.Shared = shared
	m.views[id] = view
	return view, nil
}

func (m *mockSavedViewRepo) DeleteSavedView(ctx context.Context, id, ownerID uuid.UUID) error {
	view, ok := m.views[id]
	if !ok || view.OwnerID != ownerID {
		return pgx.ErrNoRows
	}
	delete(m.views, id)
	return nil
}

func TestSavedViewsCRUD(t *testing.T) {
	r, auth, _ := setupSavedViewRouter(t)
	token, user, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	payload, _ := json.Marshal(map[string]any{
		"name":   "Critical alerts",
		"filter": map[string]any{"severity": "critical"},
		"shared": true,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-views", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	viewID := created["id"].(string)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/saved-views", nil)
	req2.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/saved-views/"+viewID, nil)
	req3.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code)

	updatePayload, _ := json.Marshal(map[string]any{
		"name":   "Updated",
		"filter": map[string]any{"severity": "warning"},
		"shared": false,
	})
	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodPatch, "/api/v1/saved-views/"+viewID, bytes.NewReader(updatePayload))
	req4.Header.Set("Content-Type", "application/json")
	req4.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w4, req4)
	require.Equal(t, http.StatusOK, w4.Code)

	w5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodDelete, "/api/v1/saved-views/"+viewID, nil)
	req5.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w5, req5)
	require.Equal(t, http.StatusNoContent, w5.Code)

	_ = user
}

func TestSavedViewsRequiresSession(t *testing.T) {
	r, _, _ := setupSavedViewRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/saved-views", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSavedViewGetNotFound(t *testing.T) {
	r, auth, _ := setupSavedViewRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/saved-views/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSavedViewCreateInvalidBody(t *testing.T) {
	r, auth, _ := setupSavedViewRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-views", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSavedViewUpdateInvalidBody(t *testing.T) {
	r, auth, repo := setupSavedViewRouter(t)
	token, user, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)
	viewID := uuid.New()
	repo.views[viewID] = db.SavedView{ID: viewID, OwnerID: user.ID, Name: "Mine", Filter: []byte(`{}`)}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/saved-views/"+viewID.String(), bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSavedViewDeleteInvalidID(t *testing.T) {
	r, auth, _ := setupSavedViewRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/saved-views/not-a-uuid", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

type failExportAlertRepo struct {
	authMockAlertRepo
}

func (f *failExportAlertRepo) StreamAlertsCSV(context.Context, db.ListAlertsParams, io.Writer) error {
	return errListAlerts
}

func TestSavedViewDeleteNotFound(t *testing.T) {
	r, auth, _ := setupSavedViewRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/saved-views/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSavedViewCreateValidationError(t *testing.T) {
	r, auth, _ := setupSavedViewRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	payload, _ := json.Marshal(map[string]any{"name": " ", "filter": map[string]any{}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-views", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSavedViewUpdateNotFound(t *testing.T) {
	r, auth, _ := setupSavedViewRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	payload, _ := json.Marshal(map[string]any{"name": "Updated", "filter": map[string]any{}, "shared": false})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/saved-views/"+uuid.New().String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestExportAlertsUnknownTeam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{SessionTTL: time.Hour}
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	teams := service.NewTeamService(&emptyTeamRepo{}, nil)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &authMockAlertRepo{id: uuid.New()})

	r := gin.New()
	NewAlertHandler(alerts, teams, auth).Register(r)

	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/export?team_id="+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSavedViewGetInvalidID(t *testing.T) {
	r, auth, _ := setupSavedViewRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/saved-views/not-a-uuid", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExportAlertsRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{SessionTTL: time.Hour}
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	teams := service.NewTeamService(&emptyTeamRepo{}, nil)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &failExportAlertRepo{authMockAlertRepo: authMockAlertRepo{id: uuid.New()}})

	r := gin.New()
	NewAlertHandler(alerts, teams, auth).Register(r)

	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/export", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
