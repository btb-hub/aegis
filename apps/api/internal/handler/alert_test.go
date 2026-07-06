package handler

import (
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
	require.EqualValues(t, 1, body["total"])
	require.EqualValues(t, 1, body["page"])
	require.EqualValues(t, 100, body["page_size"])
}

func TestListAlertsWithPagination(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?page=2&page_size=25", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.EqualValues(t, 2, body["page"])
	require.EqualValues(t, 25, body["page_size"])
}

func TestListAlertsInvalidPageSize(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?page_size=0", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListAlertsWithTeamFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	teamID := uuid.New()
	alertRepo := &teamFilterAlertRepo{}
	teamRepo := &teamLookupRepo{team: db.Team{ID: teamID, Name: "Platform"}}
	cfg := &config.Config{SessionTTL: time.Hour}
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	teams := service.NewTeamService(teamRepo, nil)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, alertRepo)

	r := gin.New()
	NewAlertHandler(alerts, teams, auth).Register(r)

	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?team_id="+teamID.String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, map[string]string{"team": "Platform"}, alertRepo.last.LabelFilters)
}

func TestListAlertsUnknownTeam(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?team_id="+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListAlertsRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		SessionTTL: time.Hour,
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	teams := service.NewTeamService(&emptyTeamRepo{}, nil)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &failListAlertRepo{})

	r := gin.New()
	NewAlertHandler(alerts, teams, auth).Register(r)

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

func (f *failListAlertRepo) CountAlerts(context.Context, db.ListAlertsParams) (int, error) {
	return 0, errListAlerts
}

func (f *failListAlertRepo) GroupAlerts(context.Context, db.ListAlertsParams, db.AlertGroupBy) ([]db.AlertGroupBucket, error) {
	return nil, errListAlerts
}

func (f *failListAlertRepo) AlertAnalytics(context.Context, db.ListAlertsParams, string) (db.AlertAnalytics, error) {
	return db.AlertAnalytics{}, errListAlerts
}

func (f *failListAlertRepo) StreamAlertsCSV(context.Context, db.ListAlertsParams, io.Writer) error {
	return errListAlerts
}

type teamFilterAlertRepo struct {
	last db.ListAlertsParams
}

func (r *teamFilterAlertRepo) CreateAlertAndJob(context.Context, db.CreateAlertJobInput) (db.CreateAlertJobResult, error) {
	return db.CreateAlertJobResult{}, nil
}
func (r *teamFilterAlertRepo) ListAlerts(_ context.Context, params db.ListAlertsParams) ([]db.Alert, error) {
	r.last = params
	return []db.Alert{{ID: uuid.New(), Title: "CPU", Status: "firing"}}, nil
}
func (r *teamFilterAlertRepo) CountAlerts(context.Context, db.ListAlertsParams) (int, error) {
	return 1, nil
}

func (r *teamFilterAlertRepo) GroupAlerts(context.Context, db.ListAlertsParams, db.AlertGroupBy) ([]db.AlertGroupBucket, error) {
	return nil, nil
}

func (r *teamFilterAlertRepo) AlertAnalytics(context.Context, db.ListAlertsParams, string) (db.AlertAnalytics, error) {
	return db.AlertAnalytics{}, nil
}

func (r *teamFilterAlertRepo) StreamAlertsCSV(_ context.Context, _ db.ListAlertsParams, w io.Writer) error {
	_, err := w.Write([]byte("id,fingerprint,status,severity,title,body,labels,received_at\n"))
	return err
}

type teamLookupRepo struct {
	team db.Team
}

func (r *teamLookupRepo) ListTeamsFiltered(context.Context, uuid.UUID) ([]db.Team, error) { return nil, nil }
func (r *teamLookupRepo) GetTeam(context.Context, uuid.UUID) (db.Team, error) { return r.team, nil }
func (r *teamLookupRepo) CreateTeam(_ context.Context, workspaceID uuid.UUID, name, description string, supportTier *string) (db.Team, error) {
	return db.Team{WorkspaceID: workspaceID, Name: name, Description: description, SupportTier: supportTier}, nil
}
func (r *teamLookupRepo) UpdateTeam(context.Context, uuid.UUID, string, string, *string) (db.Team, error) {
	return db.Team{}, nil
}
func (r *teamLookupRepo) MoveTeamsToWorkspace(context.Context, uuid.UUID, []uuid.UUID) error { return nil }
func (r *teamLookupRepo) DeleteTeam(context.Context, uuid.UUID) error { return nil }
func (r *teamLookupRepo) ListTeamMembers(context.Context, uuid.UUID) ([]db.TeamMember, error) {
	return nil, nil
}
func (r *teamLookupRepo) AddTeamMember(context.Context, uuid.UUID, uuid.UUID, string) (db.TeamMembership, error) {
	return db.TeamMembership{}, nil
}
func (r *teamLookupRepo) UpdateTeamMemberRole(context.Context, uuid.UUID, uuid.UUID, string) (db.TeamMembership, error) {
	return db.TeamMembership{}, nil
}
func (r *teamLookupRepo) RemoveTeamMember(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *teamLookupRepo) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return db.User{}, nil
}
func (r *teamLookupRepo) GetWorkspace(_ context.Context, id uuid.UUID) (db.Workspace, error) {
	return db.Workspace{ID: id, Name: "Default", Slug: "default"}, nil
}

var errListAlerts = &listAlertsError{}

type listAlertsError struct{}

func (e *listAlertsError) Error() string { return "list failed" }

func TestListAlertsGroupBySeverity(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?group_by=severity", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "severity", body["group_by"])
	require.EqualValues(t, 1, body["total"])
	groups, ok := body["groups"].([]any)
	require.True(t, ok)
	require.Len(t, groups, 1)
	group := groups[0].(map[string]any)
	require.Equal(t, "critical", group["key"])
	require.EqualValues(t, 1, group["count"])
	_, hasSample := group["sample"]
	require.True(t, hasSample)
}

func TestListAlertsGroupByLabel(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?group_by=label:team", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "label:team", body["group_by"])
	groups := body["groups"].([]any)
	group := groups[0].(map[string]any)
	require.Equal(t, "platform", group["key"])
}

func TestListAlertsInvalidGroupBy(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?group_by=label:", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListAlertsGroupRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{SessionTTL: time.Hour}
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	teams := service.NewTeamService(&emptyTeamRepo{}, nil)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &failGroupAlertRepo{})

	r := gin.New()
	NewAlertHandler(alerts, teams, auth).Register(r)

	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?group_by=severity", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

type failGroupAlertRepo struct {
	authMockAlertRepo
}

func (f *failGroupAlertRepo) GroupAlerts(context.Context, db.ListAlertsParams, db.AlertGroupBy) ([]db.AlertGroupBucket, error) {
	return nil, errListAlerts
}

func (f *failGroupAlertRepo) AlertAnalytics(context.Context, db.ListAlertsParams, string) (db.AlertAnalytics, error) {
	return db.AlertAnalytics{}, errListAlerts
}

func (f *failGroupAlertRepo) StreamAlertsCSV(context.Context, db.ListAlertsParams, io.Writer) error {
	return errListAlerts
}

func TestListAlertsWithAnalytics(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?include_analytics=true", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	analytics, ok := body["analytics"].(map[string]any)
	require.True(t, ok)
	bySeverity, ok := analytics["by_severity"].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, bySeverity["critical"])
}

func TestExportAlertsCSV(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/export", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	require.Contains(t, w.Body.String(), "id,fingerprint")
}

func TestExportAlertsRequiresSession(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/export", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestExportAlertsWithTeamFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	teamID := uuid.New()
	alertRepo := &teamFilterAlertRepo{}
	teamRepo := &teamLookupRepo{team: db.Team{ID: teamID, Name: "Platform"}}
	cfg := &config.Config{SessionTTL: time.Hour}
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	teams := service.NewTeamService(teamRepo, nil)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, alertRepo)

	r := gin.New()
	NewAlertHandler(alerts, teams, auth).Register(r)

	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/export?team_id="+teamID.String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestListAlertsAnalyticsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{SessionTTL: time.Hour}
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	teams := service.NewTeamService(&emptyTeamRepo{}, nil)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &failAnalyticsAlertRepo{})

	r := gin.New()
	NewAlertHandler(alerts, teams, auth).Register(r)

	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?include_analytics=true", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

type failAnalyticsAlertRepo struct {
	authMockAlertRepo
}

func (f *failAnalyticsAlertRepo) AlertAnalytics(context.Context, db.ListAlertsParams, string) (db.AlertAnalytics, error) {
	return db.AlertAnalytics{}, errListAlerts
}
