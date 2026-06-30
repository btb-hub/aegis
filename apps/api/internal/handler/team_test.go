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
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type teamTestEnv struct {
	router *gin.Engine
	repo   *teamRepoMock
	auth   *service.AuthService
}

func setupTeamRouter(t *testing.T) *teamTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newTeamRepoMock()
	cfg := &config.Config{SessionTTL: time.Hour}
	auth := service.NewAuthService(cfg, repo, repo, &authMockOIDC{})
	teams := service.NewTeamService(repo)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &authMockAlertRepo{id: uuid.New()})
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth, "http://localhost:3000").Register(r)
	NewAlertHandler(alerts).Register(r)
	NewTeamHandler(teams, auth).Register(r)
	return &teamTestEnv{router: r, repo: repo, auth: auth}
}

type teamRepoMock struct {
	authMockUsers
	authMockSessions
	teams       map[uuid.UUID]db.Team
	memberships map[uuid.UUID]map[uuid.UUID]db.TeamMembership
}

func newTeamRepoMock() *teamRepoMock {
	return &teamRepoMock{
		authMockUsers:    authMockUsers{users: map[uuid.UUID]db.User{}},
		authMockSessions: authMockSessions{byHash: map[string]db.Session{}},
		teams:            map[uuid.UUID]db.Team{},
		memberships:      map[uuid.UUID]map[uuid.UUID]db.TeamMembership{},
	}
}

func (m *teamRepoMock) ListTeams(ctx context.Context) ([]db.Team, error) {
	items := make([]db.Team, 0, len(m.teams))
	for _, team := range m.teams {
		items = append(items, team)
	}
	return items, nil
}

func (m *teamRepoMock) GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return team, nil
}

func (m *teamRepoMock) CreateTeam(ctx context.Context, name, description string) (db.Team, error) {
	team := db.Team{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.teams[team.ID] = team
	return team, nil
}

func (m *teamRepoMock) UpdateTeam(ctx context.Context, id uuid.UUID, name, description string) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	team.Name = name
	team.Description = description
	m.teams[id] = team
	return team, nil
}

func (m *teamRepoMock) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.teams[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.teams, id)
	return nil
}

func (m *teamRepoMock) ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]db.TeamMember, error) {
	byUser := m.memberships[teamID]
	items := make([]db.TeamMember, 0, len(byUser))
	for userID, membership := range byUser {
		user := m.users[userID]
		items = append(items, db.TeamMember{
			ID:          membership.ID,
			TeamID:      membership.TeamID,
			UserID:      membership.UserID,
			TeamRole:    membership.TeamRole,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			CreatedAt:   membership.CreatedAt,
		})
	}
	return items, nil
}

func (m *teamRepoMock) AddTeamMember(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMembership, error) {
	if m.memberships[teamID] == nil {
		m.memberships[teamID] = map[uuid.UUID]db.TeamMembership{}
	}
	membership := db.TeamMembership{
		ID:        uuid.New(),
		TeamID:    teamID,
		UserID:    userID,
		TeamRole:  teamRole,
		CreatedAt: time.Now(),
	}
	m.memberships[teamID][userID] = membership
	return membership, nil
}

func (m *teamRepoMock) UpdateTeamMemberRole(ctx context.Context, teamID, userID uuid.UUID, teamRole string) (db.TeamMembership, error) {
	byUser, ok := m.memberships[teamID]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	membership, ok := byUser[userID]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	membership.TeamRole = teamRole
	byUser[userID] = membership
	return membership, nil
}

func (m *teamRepoMock) RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error {
	byUser, ok := m.memberships[teamID]
	if !ok {
		return pgx.ErrNoRows
	}
	if _, ok := byUser[userID]; !ok {
		return pgx.ErrNoRows
	}
	delete(byUser, userID)
	return nil
}

func (env *teamTestEnv) sessionForRole(t *testing.T, role string) string {
	t.Helper()
	userID := uuid.New()
	env.repo.users[userID] = db.User{ID: userID, Role: role, Email: "u@example.com", DisplayName: "User"}
	token, hash, err := sessionTokenPair()
	require.NoError(t, err)
	_, err = env.repo.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))
	require.NoError(t, err)
	return token
}

func TestTeamsListRequiresSession(t *testing.T) {
	env := setupTeamRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTeamsCreateForbiddenForMember(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "member")
	body, _ := json.Marshal(map[string]string{"name": "Platform"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestTeamsCreateForbiddenForViewer(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "viewer")
	body, _ := json.Marshal(map[string]string{"name": "Platform"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestTeamsCreateAllowedForAdmin(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{"name": "Platform", "description": "Core"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestTeamsListAllowedForViewer(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(body))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)
	require.Equal(t, http.StatusCreated, wCreate.Code)

	viewerToken := env.sessionForRole(t, "viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: viewerToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	items, ok := resp["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
}

func TestTeamMembersAdminFlow(t *testing.T) {
	env := setupTeamRouter(t)
	adminID := uuid.New()
	env.repo.users[adminID] = db.User{ID: adminID, Role: "admin", DisplayName: "Admin"}
	memberID := uuid.New()
	env.repo.users[memberID] = db.User{ID: memberID, Role: "member", Email: "m@example.com", DisplayName: "Member"}

	adminToken, hash, err := sessionTokenPair()
	require.NoError(t, err)
	_, err = env.repo.CreateSession(context.Background(), adminID, hash, time.Now().Add(time.Hour))
	require.NoError(t, err)

	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)
	require.Equal(t, http.StatusCreated, wCreate.Code)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))
	teamID := team["id"].(string)

	addBody, _ := json.Marshal(map[string]string{"user_id": memberID.String(), "team_role": "lead"})
	wAdd := httptest.NewRecorder()
	reqAdd := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID+"/members", bytes.NewReader(addBody))
	reqAdd.Header.Set("Content-Type", "application/json")
	reqAdd.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wAdd, reqAdd)
	require.Equal(t, http.StatusCreated, wAdd.Code)

	wList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID+"/members", nil)
	reqList.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wList, reqList)
	require.Equal(t, http.StatusOK, wList.Code)

	wDelete := httptest.NewRecorder()
	reqDelete := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+teamID+"/members/"+memberID.String(), nil)
	reqDelete.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, http.StatusNoContent, wDelete.Code)
}

func TestTeamMembersAddForbiddenForMember(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)
	require.Equal(t, http.StatusCreated, wCreate.Code)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))
	teamID := team["id"].(string)

	memberToken := env.sessionForRole(t, "member")
	addBody, _ := json.Marshal(map[string]string{"user_id": uuid.New().String()})
	wAdd := httptest.NewRecorder()
	reqAdd := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID+"/members", bytes.NewReader(addBody))
	reqAdd.Header.Set("Content-Type", "application/json")
	reqAdd.AddCookie(&http.Cookie{Name: sessionCookie, Value: memberToken})
	env.router.ServeHTTP(wAdd, reqAdd)
	require.Equal(t, http.StatusForbidden, wAdd.Code)
}

func TestGetTeamInvalidID(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/not-a-uuid", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTeamSuccess(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+team["id"].(string), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetTeamNotFound(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListMembersNotFoundTeam(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String()+"/members", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddMemberInvalidUserID(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	addBody, _ := json.Marshal(map[string]string{"user_id": "bad"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+team["id"].(string)+"/members", bytes.NewReader(addBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSessionTokenBearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer token-value")
	require.Equal(t, "token-value", SessionToken(c))
}

func TestUpdateTeamAdmin(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	patchBody, _ := json.Marshal(map[string]string{"name": "Core", "description": "updated"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+team["id"].(string), bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteTeamAdmin(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+team["id"].(string), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestUpdateMemberAdmin(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	memberID := uuid.New()
	env.repo.users[memberID] = db.User{ID: memberID, DisplayName: "Member"}

	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))
	teamID := team["id"].(string)

	addBody, _ := json.Marshal(map[string]string{"user_id": memberID.String(), "team_role": "member"})
	wAdd := httptest.NewRecorder()
	reqAdd := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID+"/members", bytes.NewReader(addBody))
	reqAdd.Header.Set("Content-Type", "application/json")
	reqAdd.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wAdd, reqAdd)
	require.Equal(t, http.StatusCreated, wAdd.Code)

	patchBody, _ := json.Marshal(map[string]string{"team_role": "lead"})
	wPatch := httptest.NewRecorder()
	reqPatch := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+teamID+"/members/"+memberID.String(), bytes.NewReader(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wPatch, reqPatch)
	require.Equal(t, http.StatusOK, wPatch.Code)
}

func TestTeamsCreateInvalidBody(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTeamsCreateValidationError(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{"name": "   "})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateTeamValidationError(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	patchBody, _ := json.Marshal(map[string]string{"name": "  "})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+team["id"].(string), bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteTeamNotFound(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveMemberNotFound(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+team["id"].(string)+"/members/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateMemberInvalidBody(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+team["id"].(string)+"/members/"+uuid.New().String(), bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddMemberUnknownUser(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	addBody, _ := json.Marshal(map[string]string{"user_id": uuid.New().String()})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+team["id"].(string)+"/members", bytes.NewReader(addBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateMemberNotFoundHandler(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	patchBody, _ := json.Marshal(map[string]string{"team_role": "lead"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+team["id"].(string)+"/members/"+uuid.New().String(), bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateMemberInvalidUserIDParam(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "admin")
	patchBody, _ := json.Marshal(map[string]string{"team_role": "lead"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+uuid.New().String()+"/members/not-a-uuid", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddMemberInvalidBody(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+team["id"].(string)+"/members", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMemberInvalidRoleHandler(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	memberID := uuid.New()
	env.repo.users[memberID] = db.User{ID: memberID, DisplayName: "Member"}

	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))
	teamID := team["id"].(string)

	addBody, _ := json.Marshal(map[string]string{"user_id": memberID.String()})
	wAdd := httptest.NewRecorder()
	reqAdd := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID+"/members", bytes.NewReader(addBody))
	reqAdd.Header.Set("Content-Type", "application/json")
	reqAdd.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wAdd, reqAdd)

	patchBody, _ := json.Marshal(map[string]string{"team_role": "owner"})
	wPatch := httptest.NewRecorder()
	reqPatch := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+teamID+"/members/"+memberID.String(), bytes.NewReader(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wPatch, reqPatch)
	require.Equal(t, http.StatusBadRequest, wPatch.Code)
}

func TestRemoveMemberInvalidUserIDParam(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+uuid.New().String()+"/members/not-a-uuid", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateTeamInvalidBody(t *testing.T) {
	env := setupTeamRouter(t)
	token := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+uuid.New().String(), bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteTeamForbiddenForViewer(t *testing.T) {
	env := setupTeamRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	createBody, _ := json.Marshal(map[string]string{"name": "Platform"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)

	var team map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &team))

	viewerToken := env.sessionForRole(t, "viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+team["id"].(string), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: viewerToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func sessionTokenPair() (string, string, error) {
	return sessiontoken.New()
}
