package handler

import (
	"bytes"
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

type userHandlerEnv struct {
	router *gin.Engine
	repo   *userHandlerRepoMock
	auth   *service.AuthService
}

type userHandlerRepoMock struct {
	authMockUsers
	authMockSessions
	userListRepoMock
}

type userListRepoMock struct {
	directory  []db.User
	identities map[uuid.UUID][]db.UserIdentity
}

func (m *userListRepoMock) ListUsers(ctx context.Context, params db.ListUsersParams) ([]db.User, error) {
	start := params.Offset
	if start >= len(m.directory) {
		return []db.User{}, nil
	}
	end := start + params.Limit
	if end > len(m.directory) {
		end = len(m.directory)
	}
	return m.directory[start:end], nil
}

func (m *userListRepoMock) CountUsers(ctx context.Context, params db.ListUsersParams) (int, error) {
	return len(m.directory), nil
}

func (m *userListRepoMock) ListUserIdentitiesByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]db.UserIdentity, error) {
	out := make(map[uuid.UUID][]db.UserIdentity, len(userIDs))
	for _, userID := range userIDs {
		if identities, ok := m.identities[userID]; ok {
			out[userID] = identities
		}
	}
	return out, nil
}

func (m *userListRepoMock) CountUsersByRole(ctx context.Context, role string) (int, error) {
	n := 0
	for _, user := range m.directory {
		if user.Role == role {
			n++
		}
	}
	return n, nil
}

func setupUserRouter(t *testing.T) *userHandlerEnv {
	t.Helper()

	aliceID := uuid.New()
	cfg := &config.Config{SessionTTL: time.Hour}
	directory := []db.User{
		{ID: aliceID, Email: "alice@example.com", DisplayName: "Alice", Role: "member"},
	}
	identities := map[uuid.UUID][]db.UserIdentity{
		aliceID: {{Provider: "google", ProviderSub: "g-1", LinkedAt: time.Now()}},
	}
	return setupUserRouterWith(t, cfg, directory, identities)
}

// setupUserRouterWith builds a user router with a caller-supplied config and
// directory, so tests can exercise pinned-admin and last-admin guard paths.
func setupUserRouterWith(t *testing.T, cfg *config.Config, directory []db.User, identities map[uuid.UUID][]db.UserIdentity) *userHandlerEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := &userHandlerRepoMock{
		authMockUsers:    *newAuthMockUsers(),
		authMockSessions: authMockSessions{byHash: map[string]db.Session{}},
		userListRepoMock: userListRepoMock{
			directory:  directory,
			identities: identities,
		},
	}
	for _, u := range directory {
		repo.users[u.ID] = u
	}

	auth := service.NewAuthService(cfg, repo, repo, &authMockOIDC{})
	users := service.NewUserService(repo, cfg)
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth, "http://localhost:3000").Register(r)
	NewUserHandler(users, auth).Register(r)
	return &userHandlerEnv{router: r, repo: repo, auth: auth}
}

func (env *userHandlerEnv) sessionForRole(t *testing.T, role string) string {
	t.Helper()
	userID := uuid.New()
	env.repo.users[userID] = db.User{ID: userID, Role: role, Email: "admin@example.com", DisplayName: "Admin"}
	token, hash, err := sessionTokenPair()
	require.NoError(t, err)
	_, err = env.repo.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))
	require.NoError(t, err)
	return token
}

func TestUsersListRequiresSession(t *testing.T) {
	env := setupUserRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUsersListForbiddenForMember(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUsersListSuccessForAdmin(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&page_size=10", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, float64(1), body["total"])
	items, ok := body["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "alice@example.com", first["email"])
	require.NotNil(t, first["identities"])
	require.Equal(t, false, first["role_pinned"])
}

func TestUsersListIncludesRolePinnedTrue(t *testing.T) {
	pinnedID := uuid.New()
	cfg := &config.Config{
		SessionTTL:  time.Hour,
		AdminEmails: map[string]struct{}{"pinned@example.com": {}},
	}
	directory := []db.User{
		{ID: pinnedID, Email: "pinned@example.com", DisplayName: "Pinned", Role: "admin"},
	}
	env := setupUserRouterWith(t, cfg, directory, map[uuid.UUID][]db.UserIdentity{})
	token := env.sessionForRole(t, "admin")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	items, ok := body["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, first["role_pinned"])
}

func TestPatchUserRoleSuccessForAdmin(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "admin")

	var aliceID uuid.UUID
	for id := range env.repo.users {
		if env.repo.users[id].Email == "alice@example.com" {
			aliceID = id
		}
	}
	require.NotEqual(t, uuid.Nil, aliceID)

	payload, _ := json.Marshal(map[string]string{"role": "admin"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+aliceID.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "admin", body["role"])
	require.Equal(t, false, body["role_pinned"])
	require.NotNil(t, body["identities"])
}

func TestPatchUserRoleForbiddenForMember(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "member")

	var aliceID uuid.UUID
	for id := range env.repo.users {
		if env.repo.users[id].Email == "alice@example.com" {
			aliceID = id
		}
	}

	payload, _ := json.Marshal(map[string]string{"role": "admin"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+aliceID.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestPatchUserRoleLastAdminConflict(t *testing.T) {
	adminID := uuid.New()
	cfg := &config.Config{SessionTTL: time.Hour}
	directory := []db.User{
		{ID: adminID, Email: "only-admin@example.com", DisplayName: "Only Admin", Role: "admin"},
	}
	env := setupUserRouterWith(t, cfg, directory, map[uuid.UUID][]db.UserIdentity{})
	token := env.sessionForRole(t, "admin")

	payload, _ := json.Marshal(map[string]string{"role": "member"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+adminID.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "last_admin", body["code"])
}

func TestPatchUserRoleAdminEmailsPinnedConflict(t *testing.T) {
	pinnedID := uuid.New()
	cfg := &config.Config{
		SessionTTL:  time.Hour,
		AdminEmails: map[string]struct{}{"pinned@example.com": {}},
	}
	directory := []db.User{
		{ID: pinnedID, Email: "pinned@example.com", DisplayName: "Pinned", Role: "admin"},
	}
	env := setupUserRouterWith(t, cfg, directory, map[uuid.UUID][]db.UserIdentity{})
	token := env.sessionForRole(t, "admin")

	payload, _ := json.Marshal(map[string]string{"role": "member"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+pinnedID.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "admin_emails_pinned", body["code"])
}

func TestPatchUserRoleInvalidID(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "admin")

	payload, _ := json.Marshal(map[string]string{"role": "admin"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/not-a-uuid", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchUserRoleInvalidBody(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "admin")

	var aliceID uuid.UUID
	for id := range env.repo.users {
		if env.repo.users[id].Email == "alice@example.com" {
			aliceID = id
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+aliceID.String(), bytes.NewReader([]byte("{not-json")))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchUserRoleInvalidRole(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "admin")

	var aliceID uuid.UUID
	for id := range env.repo.users {
		if env.repo.users[id].Email == "alice@example.com" {
			aliceID = id
		}
	}

	payload, _ := json.Marshal(map[string]string{"role": "superuser"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+aliceID.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUsersListInvalidPage(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=0", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUsersListInvalidPageSize(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page_size=0", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUsersListCapsPageSize(t *testing.T) {
	env := setupUserRouter(t)
	token := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page_size=500", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, float64(db.DefaultUserListLimit), body["page_size"])
}
