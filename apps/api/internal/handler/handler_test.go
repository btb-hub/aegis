package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func setupRouter(t *testing.T) (*gin.Engine, *service.AuthService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		SessionTTL: time.Hour,
		PublicURL:  "http://localhost:3000",
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
	users := newAuthMockUsers()
	sessions := &authMockSessions{byHash: map[string]db.Session{}}
	auth := service.NewAuthService(cfg, users, sessions, &authMockOIDC{})
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &authMockAlertRepo{id: uuid.New()})
	teams := service.NewTeamService(&emptyTeamRepo{}, nil)
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth, cfg.PublicURL).Register(r)
	NewAlertHandler(alerts, teams, auth).Register(r)
	return r, auth
}

type authMockUsers struct {
	users             map[uuid.UUID]db.User
	identities        map[string]db.UserIdentity
	listIdentitiesErr error
}

func newAuthMockUsers() *authMockUsers {
	return &authMockUsers{
		users:      map[uuid.UUID]db.User{},
		identities: map[string]db.UserIdentity{},
	}
}

func (m *authMockUsers) ResolveOIDCLogin(ctx context.Context, input db.OIDCLoginInput) (db.OIDCLoginResult, error) {
	user := db.User{
		ID:          uuid.New(),
		Provider:    input.Provider,
		ProviderSub: input.ProviderSub,
		Email:       input.Email,
		DisplayName: input.DisplayName,
		Role:        "member",
		Locale:      "en",
	}
	m.users[user.ID] = user
	identity := db.UserIdentity{
		ID: uuid.New(), UserID: user.ID, Provider: input.Provider, ProviderSub: input.ProviderSub, LinkedAt: time.Now(),
	}
	m.identities[input.Provider+":"+input.ProviderSub] = identity
	return db.OIDCLoginResult{User: user, Identities: []db.UserIdentity{identity}, NewIdentityLinked: true}, nil
}

func (m *authMockUsers) UpsertDevUser(ctx context.Context, email, displayName, role, locale string) (db.User, error) {
	user := db.User{ID: uuid.New(), Provider: "dev", Email: email, DisplayName: displayName, Role: role, Locale: locale}
	m.users[user.ID] = user
	return user, nil
}

func (m *authMockUsers) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	user, ok := m.users[id]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return user, nil
}

func (m *authMockUsers) ListUserIdentities(ctx context.Context, userID uuid.UUID) ([]db.UserIdentity, error) {
	if m.listIdentitiesErr != nil {
		return nil, m.listIdentitiesErr
	}
	var items []db.UserIdentity
	for _, identity := range m.identities {
		if identity.UserID == userID {
			items = append(items, identity)
		}
	}
	return items, nil
}

func (m *authMockUsers) UpdateUserLocale(ctx context.Context, id uuid.UUID, locale string) (db.User, error) {
	user := m.users[id]
	user.Locale = locale
	m.users[id] = user
	return user, nil
}

func (m *authMockUsers) UpdateUserProfile(ctx context.Context, id uuid.UUID, displayName, locale string) (db.User, error) {
	user := m.users[id]
	user.DisplayName = displayName
	user.Locale = locale
	m.users[id] = user
	return user, nil
}

func (m *authMockUsers) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) (db.User, error) {
	user := m.users[id]
	user.Role = role
	m.users[id] = user
	return user, nil
}

func (m *authMockUsers) WriteAuditLog(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error {
	return nil
}

type authMockSessions struct{ byHash map[string]db.Session }

func (m *authMockSessions) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.Session, error) {
	s := db.Session{UserID: userID, TokenHash: tokenHash}
	m.byHash[tokenHash] = s
	return s, nil
}
func (m *authMockSessions) GetSessionByTokenHash(ctx context.Context, tokenHash string) (db.Session, error) {
	s, ok := m.byHash[tokenHash]
	if !ok {
		return db.Session{}, errors.New("not found")
	}
	return s, nil
}
func (m *authMockSessions) DeleteSession(ctx context.Context, tokenHash string) error {
	delete(m.byHash, tokenHash)
	return nil
}

type authMockOIDC struct{}

func (m *authMockOIDC) AuthCodeURL(provider, state string) (string, error) {
	return "https://idp/authorize", nil
}
func (m *authMockOIDC) Exchange(ctx context.Context, provider, code string) (*service.OIDCUserInfo, error) {
	return &service.OIDCUserInfo{Sub: "sub", Email: "u@example.com", DisplayName: "User"}, nil
}

type authMockAlertRepo struct {
	id         uuid.UUID
	incidentID *uuid.UUID
	ingestErr  error
}

func (m *authMockAlertRepo) sampleAlert() db.Alert {
	alert := db.Alert{ID: m.id, Status: "firing", Severity: "critical", Title: "CPU", Labels: []byte(`{}`)}
	if m.incidentID != nil {
		alert.IncidentID = m.incidentID
	}
	return alert
}

func (m *authMockAlertRepo) CreateAlertAndJob(ctx context.Context, input db.CreateAlertJobInput) (db.CreateAlertJobResult, error) {
	if m.ingestErr != nil {
		return db.CreateAlertJobResult{}, m.ingestErr
	}
	return db.CreateAlertJobResult{AlertID: m.id, JobID: uuid.New()}, nil
}

func (m *authMockAlertRepo) ListAlerts(ctx context.Context, params db.ListAlertsParams) ([]db.Alert, error) {
	return []db.Alert{m.sampleAlert()}, nil
}

func (m *authMockAlertRepo) CountAlerts(ctx context.Context, params db.ListAlertsParams) (int, error) {
	return 1, nil
}

func (m *authMockAlertRepo) GroupAlerts(ctx context.Context, filters db.ListAlertsParams, groupBy db.AlertGroupBy) ([]db.AlertGroupBucket, error) {
	key := "critical"
	if groupBy.LabelKey != "" {
		key = "platform"
	}
	sample := m.sampleAlert()
	return []db.AlertGroupBucket{{Key: key, Count: 1, Sample: &sample}}, nil
}

func (m *authMockAlertRepo) AlertAnalytics(ctx context.Context, params db.ListAlertsParams, labelKey string) (db.AlertAnalytics, error) {
	return db.AlertAnalytics{
		BySeverity: map[string]int{"critical": 1},
		ByStatus:   map[string]int{"firing": 1},
	}, nil
}

func (m *authMockAlertRepo) StreamAlertsCSV(ctx context.Context, params db.ListAlertsParams, w io.Writer) error {
	_, err := w.Write([]byte("id,fingerprint,status,severity,title,body,labels,received_at\n"))
	return err
}

type emptyTeamRepo struct{}

func (e *emptyTeamRepo) ListTeamsFiltered(ctx context.Context, workspaceID uuid.UUID) ([]db.Team, error) {
	return nil, nil
}
func (e *emptyTeamRepo) GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error) {
	return db.Team{}, pgx.ErrNoRows
}
func (e *emptyTeamRepo) CreateTeam(ctx context.Context, workspaceID uuid.UUID, name, description string, supportTier *string) (db.Team, error) {
	return db.Team{WorkspaceID: workspaceID, Name: name, Description: description, SupportTier: supportTier}, nil
}
func (e *emptyTeamRepo) UpdateTeam(ctx context.Context, id uuid.UUID, name, description string, supportTier *string) (db.Team, error) {
	return db.Team{}, nil
}
func (e *emptyTeamRepo) MoveTeamsToWorkspace(ctx context.Context, workspaceID uuid.UUID, teamIDs []uuid.UUID) error {
	return nil
}
func (e *emptyTeamRepo) DeleteTeam(ctx context.Context, id uuid.UUID) error { return nil }
func (e *emptyTeamRepo) ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]db.TeamMember, error) {
	return nil, nil
}
func (e *emptyTeamRepo) AddTeamMember(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMembership, error) {
	return db.TeamMembership{}, nil
}
func (e *emptyTeamRepo) UpdateTeamMemberRole(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMembership, error) {
	return db.TeamMembership{}, nil
}
func (e *emptyTeamRepo) RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error { return nil }
func (e *emptyTeamRepo) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}
func (e *emptyTeamRepo) GetWorkspace(ctx context.Context, id uuid.UUID) (db.Workspace, error) {
	return db.Workspace{ID: id, Name: "Default", Slug: "default"}, nil
}

func TestHealthz(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestWebhookAccepted(t *testing.T) {
	r, _ := setupRouter(t)
	body := bytes.NewBufferString(`{"status":"firing","labels":{"alertname":"X"},"annotations":{"summary":"Y"}}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/webhook", body)
	req.Header.Set("X-Aegis-Webhook-Secret", "secret")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)
}

func TestWebhookInvalidSecret(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/webhook", bytes.NewBufferString(`{}`))
	req.Header.Set("X-Aegis-Webhook-Secret", "bad")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMeRequiresSession(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCallbackRedirectSetsCookie(t *testing.T) {
	r, _ := setupRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: "aegis_oauth_state", Value: "abc"})
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000", w.Header().Get("Location"))

	var session string
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			session = c.Value
		}
	}
	require.NotEmpty(t, session)
}

func TestAuthCallbackAndMe(t *testing.T) {
	r, _ := setupRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: "aegis_oauth_state", Value: "abc"})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "http://localhost:3000", w.Header().Get("Location"))

	cookies := w.Result().Cookies()
	var session string
	for _, c := range cookies {
		if c.Name == sessionCookie {
			session = c.Value
		}
	}
	require.NotEmpty(t, session)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req2.AddCookie(&http.Cookie{Name: sessionCookie, Value: session})
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body))
	require.NotNil(t, body["identities"])
}

func TestPatchMeSuccessReturnsIdentities(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	payload, _ := json.Marshal(map[string]string{"locale": "ru"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/auth/me", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "ru", body["locale"])
	require.NotNil(t, body["identities"])
}

func TestPatchMeInvalidLocale(t *testing.T) {
	r, auth := setupRouter(t)
	token, _, err := auth.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)

	payload, _ := json.Marshal(map[string]string{"locale": "de"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/auth/me", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "INVALID_LOCALE", body["code"])
}

func TestWriteErrorAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	WriteError(c, apperrors.Unauthorized("nope"))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWriteErrorInternal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	WriteError(c, errors.New("boom"))
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWriteJSONHelper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	WriteJSON(c, http.StatusOK, gin.H{"ok": true})
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAuthLoginUnknownProvider(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/unknown/login", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCallbackJSONFormat(t *testing.T) {
	r, _ := setupRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz&format=json", nil)
	req.AddCookie(&http.Cookie{Name: "aegis_oauth_state", Value: "abc"})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "u@example.com", body["email"])
	require.NotNil(t, body["identities"])
}

func TestCallbackJSONWhenProfileLoadFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		SessionTTL: time.Hour,
		PublicURL:  "http://localhost:3000",
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
	users := newAuthMockUsers()
	users.listIdentitiesErr = errors.New("identities unavailable")
	auth := service.NewAuthService(cfg, users, &authMockSessions{byHash: map[string]db.Session{}}, &authMockOIDC{})
	r := gin.New()
	NewAuthHandler(auth, cfg.PublicURL).Register(r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=xyz&format=json", nil)
	req.AddCookie(&http.Cookie{Name: "aegis_oauth_state", Value: "abc"})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "u@example.com", body["email"])
	require.Nil(t, body["identities"])
}

func TestAuthCallbackBadState(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=bad&code=xyz", nil)
	req.AddCookie(&http.Cookie{Name: "aegis_oauth_state", Value: "good"})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHealthReadyz(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestLogoutMissingSession(t *testing.T) {
	r, _ := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
