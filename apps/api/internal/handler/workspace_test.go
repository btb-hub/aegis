package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

type workspaceTestEnv struct {
	router *gin.Engine
	repo   *workspaceEscalationRepoMock
	auth   *service.AuthService
}

type workspaceEscalationRepoMock struct {
	authMockUsers
	authMockSessions
	workspaces            map[uuid.UUID]db.Workspace
	teams                 map[uuid.UUID]db.Team
	paths                 []db.EscalationPath
	listWorkspacesErr     error
	listFromTeamErr       error
	listToTeamErr         error
	listWorkspacePathsErr error
}

func newWorkspaceEscalationRepoMock() *workspaceEscalationRepoMock {
	return &workspaceEscalationRepoMock{
		authMockUsers:    *newAuthMockUsers(),
		authMockSessions: authMockSessions{byHash: map[string]db.Session{}},
		workspaces:       map[uuid.UUID]db.Workspace{},
		teams:            map[uuid.UUID]db.Team{},
	}
}

func (m *workspaceEscalationRepoMock) ListWorkspaces(context.Context) ([]db.Workspace, error) {
	if m.listWorkspacesErr != nil {
		return nil, m.listWorkspacesErr
	}
	items := make([]db.Workspace, 0, len(m.workspaces))
	for _, item := range m.workspaces {
		items = append(items, item)
	}
	return items, nil
}

func (m *workspaceEscalationRepoMock) GetWorkspace(_ context.Context, id uuid.UUID) (db.Workspace, error) {
	item, ok := m.workspaces[id]
	if !ok {
		return db.Workspace{}, pgx.ErrNoRows
	}
	return item, nil
}

func (m *workspaceEscalationRepoMock) CreateWorkspace(_ context.Context, name, slug, description string) (db.Workspace, error) {
	item := db.Workspace{ID: uuid.New(), Name: name, Slug: slug, Description: description, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.workspaces[item.ID] = item
	return item, nil
}

func (m *workspaceEscalationRepoMock) CreateWorkspaceWithSlots(ctx context.Context, name, slug, description string) (db.Workspace, error) {
	return m.CreateWorkspace(ctx, name, slug, description)
}

func (m *workspaceEscalationRepoMock) EnsureWorkspaceSlots(context.Context, uuid.UUID) error {
	return nil
}

func (m *workspaceEscalationRepoMock) UpdateWorkspace(_ context.Context, id uuid.UUID, name, slug, description string) (db.Workspace, error) {
	item, ok := m.workspaces[id]
	if !ok {
		return db.Workspace{}, pgx.ErrNoRows
	}
	item.Name = name
	item.Slug = slug
	item.Description = description
	m.workspaces[id] = item
	return item, nil
}

func (m *workspaceEscalationRepoMock) ListWorkspacesWithCounts(_ context.Context) ([]db.WorkspaceSummary, error) {
	if m.listWorkspacesErr != nil {
		return nil, m.listWorkspacesErr
	}
	items := make([]db.WorkspaceSummary, 0, len(m.workspaces))
	for _, item := range m.workspaces {
		summary := db.WorkspaceSummary{Workspace: item}
		for _, team := range m.teams {
			if team.WorkspaceID == item.ID {
				summary.TeamCount++
			}
		}
		for _, path := range m.paths {
			if path.WorkspaceID == item.ID {
				summary.RoutingRuleCount++
			}
		}
		items = append(items, summary)
	}
	return items, nil
}

func (m *workspaceEscalationRepoMock) GetWorkspaceUsage(_ context.Context, id uuid.UUID) (db.WorkspaceUsage, error) {
	var usage db.WorkspaceUsage
	for _, team := range m.teams {
		if team.WorkspaceID == id {
			usage.TeamCount++
		}
	}
	for _, path := range m.paths {
		if path.WorkspaceID == id {
			usage.EscalationPathCount++
		}
	}
	return usage, nil
}

func (m *workspaceEscalationRepoMock) ListTeamsFiltered(_ context.Context, workspaceID uuid.UUID) ([]db.Team, error) {
	var items []db.Team
	for _, team := range m.teams {
		if workspaceID == uuid.Nil || team.WorkspaceID == workspaceID {
			items = append(items, team)
		}
	}
	return items, nil
}

func (m *workspaceEscalationRepoMock) MoveTeamsToWorkspace(_ context.Context, workspaceID uuid.UUID, teamIDs []uuid.UUID) error {
	for _, teamID := range teamIDs {
		team, ok := m.teams[teamID]
		if !ok {
			return pgx.ErrNoRows
		}
		team.WorkspaceID = workspaceID
		m.teams[teamID] = team
	}
	return nil
}

func (m *workspaceEscalationRepoMock) DeleteWorkspace(_ context.Context, id uuid.UUID) error {
	if _, ok := m.workspaces[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.workspaces, id)
	return nil
}

func (m *workspaceEscalationRepoMock) GetTeam(_ context.Context, id uuid.UUID) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return team, nil
}

func (m *workspaceEscalationRepoMock) CreateTeam(_ context.Context, workspaceID uuid.UUID, name, description string, supportTier *string) (db.Team, error) {
	team := db.Team{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        name,
		Description: description,
		SupportTier: supportTier,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.teams[team.ID] = team
	return team, nil
}

func (m *workspaceEscalationRepoMock) UpdateTeam(_ context.Context, id uuid.UUID, name, description string, supportTier *string) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	team.Name = name
	team.Description = description
	team.SupportTier = supportTier
	m.teams[id] = team
	return team, nil
}

func (m *workspaceEscalationRepoMock) DeleteTeam(_ context.Context, id uuid.UUID) error {
	if _, ok := m.teams[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.teams, id)
	return nil
}

func (m *workspaceEscalationRepoMock) ListTeamMembers(context.Context, uuid.UUID) ([]db.TeamMember, error) {
	return []db.TeamMember{}, nil
}

func (m *workspaceEscalationRepoMock) AddTeamMember(context.Context, uuid.UUID, uuid.UUID, string) (db.TeamMembership, error) {
	return db.TeamMembership{}, nil
}

func (m *workspaceEscalationRepoMock) UpdateTeamMemberRole(context.Context, uuid.UUID, uuid.UUID, string) (db.TeamMembership, error) {
	return db.TeamMembership{}, nil
}

func (m *workspaceEscalationRepoMock) RemoveTeamMember(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (m *workspaceEscalationRepoMock) ListEscalationPathsByWorkspace(_ context.Context, workspaceID uuid.UUID) ([]db.EscalationPath, error) {
	if m.listWorkspacePathsErr != nil {
		return nil, m.listWorkspacePathsErr
	}
	var out []db.EscalationPath
	for _, path := range m.paths {
		if path.WorkspaceID == workspaceID {
			out = append(out, path)
		}
	}
	return out, nil
}

func (m *workspaceEscalationRepoMock) ListEscalationPathsFromTeam(_ context.Context, fromTeamID uuid.UUID) ([]db.EscalationPath, error) {
	if m.listFromTeamErr != nil {
		return nil, m.listFromTeamErr
	}
	var out []db.EscalationPath
	for _, path := range m.paths {
		if path.FromTeamID == fromTeamID {
			out = append(out, path)
		}
	}
	return out, nil
}

func (m *workspaceEscalationRepoMock) ListEscalationPathsToTeam(_ context.Context, toTeamID uuid.UUID) ([]db.EscalationPath, error) {
	if m.listToTeamErr != nil {
		return nil, m.listToTeamErr
	}
	var out []db.EscalationPath
	for _, path := range m.paths {
		if path.ToTeamID == toTeamID {
			out = append(out, path)
		}
	}
	return out, nil
}

func (m *workspaceEscalationRepoMock) ReplaceEscalationPaths(_ context.Context, workspaceID uuid.UUID, paths []db.EscalationPath) error {
	m.paths = paths
	return nil
}

func (m *workspaceEscalationRepoMock) AddEscalationPath(_ context.Context, path db.EscalationPath) (db.EscalationPath, error) {
	path.ID = uuid.New()
	m.paths = append(m.paths, path)
	return path, nil
}

func (m *workspaceEscalationRepoMock) DeleteEscalationPath(_ context.Context, id uuid.UUID) error {
	for i, path := range m.paths {
		if path.ID == id {
			m.paths = append(m.paths[:i], m.paths[i+1:]...)
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (m *workspaceEscalationRepoMock) HasEscalationPath(_ context.Context, fromTeamID, toTeamID uuid.UUID) (bool, error) {
	for _, path := range m.paths {
		if path.FromTeamID == fromTeamID && path.ToTeamID == toTeamID {
			return true, nil
		}
	}
	return false, nil
}

func (m *workspaceEscalationRepoMock) ListHandoffTargetTeams(_ context.Context, fromTeamID uuid.UUID) ([]db.Team, error) {
	var out []db.Team
	for _, path := range m.paths {
		if path.FromTeamID == fromTeamID {
			if team, ok := m.teams[path.ToTeamID]; ok {
				out = append(out, team)
			}
		}
	}
	return out, nil
}

func setupWorkspaceRouter(t *testing.T) *workspaceTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newWorkspaceEscalationRepoMock()
	cfg := &config.Config{SessionTTL: time.Hour}
	auth := service.NewAuthService(cfg, repo, repo, &authMockOIDC{})
	workspaces := service.NewWorkspaceService(repo)
	escalation := service.NewEscalationService(repo)
	teams := service.NewTeamService(repo, escalation)
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth, "http://localhost:3000").Register(r)
	NewWorkspaceHandler(workspaces, teams, auth).Register(r)
	NewEscalationHandler(escalation, auth).Register(r)
	return &workspaceTestEnv{router: r, repo: repo, auth: auth}
}

func (env *workspaceTestEnv) sessionForRole(t *testing.T, role string) string {
	t.Helper()
	userID := uuid.New()
	env.repo.users[userID] = db.User{ID: userID, Role: role, Email: "u@example.com", DisplayName: "User"}
	token, hash, err := sessionTokenPair()
	require.NoError(t, err)
	_, err = env.repo.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))
	require.NoError(t, err)
	return token
}

func tierPtr(tier string) *string { return &tier }

func TestWorkspacesCRUDAndEscalationPaths(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	memberToken := env.sessionForRole(t, "member")

	createBody, _ := json.Marshal(map[string]string{"name": "Platform", "slug": "platform", "description": "Core"})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wCreate, reqCreate)
	require.Equal(t, http.StatusCreated, wCreate.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &created))
	workspaceID := created["id"].(string)

	wList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	reqList.AddCookie(&http.Cookie{Name: sessionCookie, Value: memberToken})
	env.router.ServeHTTP(wList, reqList)
	require.Equal(t, http.StatusOK, wList.Code)

	wGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+workspaceID, nil)
	reqGet.AddCookie(&http.Cookie{Name: sessionCookie, Value: memberToken})
	env.router.ServeHTTP(wGet, reqGet)
	require.Equal(t, http.StatusOK, wGet.Code)

	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	require.NoError(t, err)
	l2ID := uuid.New()
	l3ID := uuid.New()
	env.repo.teams[l2ID] = db.Team{ID: l2ID, WorkspaceID: parsedWorkspaceID, Name: "Platform L2", SupportTier: tierPtr("l2")}
	env.repo.teams[l3ID] = db.Team{ID: l3ID, WorkspaceID: parsedWorkspaceID, Name: "Platform L3", SupportTier: tierPtr("l3")}

	addBody, _ := json.Marshal(map[string]string{
		"from_team_id": l2ID.String(),
		"to_team_id":   l3ID.String(),
	})
	wAdd := httptest.NewRecorder()
	reqAdd := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/escalation-paths", bytes.NewReader(addBody))
	reqAdd.Header.Set("Content-Type", "application/json")
	reqAdd.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wAdd, reqAdd)
	require.Equal(t, http.StatusCreated, wAdd.Code)

	wWorkspacePaths := httptest.NewRecorder()
	reqWorkspacePaths := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+workspaceID+"/escalation-paths", nil)
	reqWorkspacePaths.AddCookie(&http.Cookie{Name: sessionCookie, Value: memberToken})
	env.router.ServeHTTP(wWorkspacePaths, reqWorkspacePaths)
	require.Equal(t, http.StatusOK, wWorkspacePaths.Code)

	wTargets := httptest.NewRecorder()
	reqTargets := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+l2ID.String()+"/handoff-targets", nil)
	reqTargets.AddCookie(&http.Cookie{Name: sessionCookie, Value: memberToken})
	env.router.ServeHTTP(wTargets, reqTargets)
	require.Equal(t, http.StatusOK, wTargets.Code)

	wOutgoing := httptest.NewRecorder()
	reqOutgoing := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+l2ID.String()+"/escalation-paths/outgoing", nil)
	reqOutgoing.AddCookie(&http.Cookie{Name: sessionCookie, Value: memberToken})
	env.router.ServeHTTP(wOutgoing, reqOutgoing)
	require.Equal(t, http.StatusOK, wOutgoing.Code)

	wIncoming := httptest.NewRecorder()
	reqIncoming := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+l3ID.String()+"/escalation-paths/incoming", nil)
	reqIncoming.AddCookie(&http.Cookie{Name: sessionCookie, Value: memberToken})
	env.router.ServeHTTP(wIncoming, reqIncoming)
	require.Equal(t, http.StatusOK, wIncoming.Code)

	var path map[string]any
	require.NoError(t, json.Unmarshal(wAdd.Body.Bytes(), &path))
	pathID := path["id"].(string)

	wDeletePath := httptest.NewRecorder()
	reqDeletePath := httptest.NewRequest(http.MethodDelete, "/api/v1/escalation-paths/"+pathID, nil)
	reqDeletePath.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wDeletePath, reqDeletePath)
	require.Equal(t, http.StatusNoContent, wDeletePath.Code)

	patchBody, _ := json.Marshal(map[string]string{"name": "Platform Ops", "description": "Updated"})
	wPatch := httptest.NewRecorder()
	reqPatch := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+workspaceID, bytes.NewReader(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wPatch, reqPatch)
	require.Equal(t, http.StatusOK, wPatch.Code)

	delete(env.repo.teams, l2ID)
	delete(env.repo.teams, l3ID)

	wDelete := httptest.NewRecorder()
	reqDelete := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/"+workspaceID, nil)
	reqDelete.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, http.StatusNoContent, wDelete.Code)
}

func TestWorkspacesCreateForbiddenForMember(t *testing.T) {
	env := setupWorkspaceRouter(t)
	token := env.sessionForRole(t, "member")
	body, _ := json.Marshal(map[string]string{"name": "Platform"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestEscalationReplaceWorkspacePaths(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	workspaceID := uuid.New()
	l2ID := uuid.New()
	l3ID := uuid.New()
	env.repo.workspaces[workspaceID] = db.Workspace{ID: workspaceID, Name: "Platform", Slug: "platform"}
	env.repo.teams[l2ID] = db.Team{ID: l2ID, WorkspaceID: workspaceID, Name: "Platform L2", SupportTier: tierPtr("l2")}
	env.repo.teams[l3ID] = db.Team{ID: l3ID, WorkspaceID: workspaceID, Name: "Platform L3", SupportTier: tierPtr("l3")}

	body, _ := json.Marshal(map[string]any{
		"paths": []map[string]any{{
			"from_team_id": l2ID.String(),
			"to_team_id":   l3ID.String(),
		}},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/"+workspaceID.String()+"/escalation-paths", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestEscalationAddPathInvalidBody(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+uuid.New().String()+"/escalation-paths", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkspacesCreateInvalidBody(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkspacesGetInvalidID(t *testing.T) {
	env := setupWorkspaceRouter(t)
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/not-a-uuid", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkspacesGetNotFound(t *testing.T) {
	env := setupWorkspaceRouter(t)
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestWorkspacesUpdateInvalidBody(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+uuid.New().String(), bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkspacesUpdateNotFound(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{"name": "Missing"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestWorkspacesDeleteNotFound(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestWorkspacesDeleteInvalidID(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/not-a-uuid", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEscalationHandoffTargetsInvalidTeam(t *testing.T) {
	env := setupWorkspaceRouter(t)
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/not-a-uuid/handoff-targets", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEscalationHandoffTargetsNotFound(t *testing.T) {
	env := setupWorkspaceRouter(t)
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String()+"/handoff-targets", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestEscalationReplaceInvalidTeamUUID(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	workspaceID := uuid.New()
	env.repo.workspaces[workspaceID] = db.Workspace{ID: workspaceID, Name: "Platform", Slug: "platform"}
	body, _ := json.Marshal(map[string]any{
		"paths": []map[string]any{{
			"from_team_id": "bad",
			"to_team_id":   uuid.New().String(),
		}},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/"+workspaceID.String()+"/escalation-paths", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEscalationAddPathInvalidTeamUUID(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{
		"from_team_id": "bad",
		"to_team_id":   uuid.New().String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+uuid.New().String()+"/escalation-paths", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEscalationDeletePathNotFound(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/escalation-paths/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestEscalationDeletePathInvalidID(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/escalation-paths/not-a-uuid", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEscalationOutgoingPathsNotFound(t *testing.T) {
	env := setupWorkspaceRouter(t)
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String()+"/escalation-paths/outgoing", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestEscalationOutgoingPathsRepoError(t *testing.T) {
	env := setupWorkspaceRouter(t)
	env.repo.listFromTeamErr = errors.New("db down")
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String()+"/escalation-paths/outgoing", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEscalationIncomingPathsRepoError(t *testing.T) {
	env := setupWorkspaceRouter(t)
	env.repo.listToTeamErr = errors.New("db down")
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String()+"/escalation-paths/incoming", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEscalationListWorkspacePathsRepoError(t *testing.T) {
	env := setupWorkspaceRouter(t)
	env.repo.listWorkspacePathsErr = errors.New("db down")
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+uuid.New().String()+"/escalation-paths", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWorkspacesListRepoError(t *testing.T) {
	env := setupWorkspaceRouter(t)
	env.repo.listWorkspacesErr = errors.New("db down")
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWorkspacesCreateValidationError(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{"name": "   "})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkspacesAssignTeams(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")

	defaultWS := db.DefaultWorkspaceID
	targetWS := uuid.New()
	env.repo.workspaces[targetWS] = db.Workspace{ID: targetWS, Name: "Platform", Slug: "platform"}
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, WorkspaceID: defaultWS, Name: "Core L2", SupportTier: tierPtr("l2")}

	body, _ := json.Marshal(map[string]any{"team_ids": []string{teamID.String()}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+targetWS.String()+"/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, targetWS, env.repo.teams[teamID].WorkspaceID)
}

func TestWorkspacesAssignTeamsBlockedByEscalationPath(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")

	defaultWS := db.DefaultWorkspaceID
	targetWS := uuid.New()
	otherWS := uuid.New()
	env.repo.workspaces[targetWS] = db.Workspace{ID: targetWS, Name: "Platform", Slug: "platform"}
	env.repo.workspaces[otherWS] = db.Workspace{ID: otherWS, Name: "Other", Slug: "other"}

	l2ID := uuid.New()
	l3ID := uuid.New()
	env.repo.teams[l2ID] = db.Team{ID: l2ID, WorkspaceID: defaultWS, Name: "L2", SupportTier: tierPtr("l2")}
	env.repo.teams[l3ID] = db.Team{ID: l3ID, WorkspaceID: otherWS, Name: "L3", SupportTier: tierPtr("l3")}
	env.repo.paths = []db.EscalationPath{{
		ID: uuid.New(), WorkspaceID: defaultWS, FromTeamID: l2ID, ToTeamID: l3ID, CrossWorkspace: false,
	}}

	body, _ := json.Marshal(map[string]any{"team_ids": []string{l2ID.String()}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+targetWS.String()+"/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	details, ok := resp["details"].(map[string]any)
	require.True(t, ok)
	require.NotNil(t, details["blocked_teams"])
}

func TestWorkspacesDeleteNotEmpty(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	workspaceID := uuid.New()
	env.repo.workspaces[workspaceID] = db.Workspace{ID: workspaceID, Name: "Platform", Slug: "platform"}
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, WorkspaceID: workspaceID, Name: "Team"}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/"+workspaceID.String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestWorkspacesDeleteDefaultForbidden(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/"+db.DefaultWorkspaceID.String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestWorkspacesAssignTeamsValidation(t *testing.T) {
	env := setupWorkspaceRouter(t)
	adminToken := env.sessionForRole(t, "admin")
	workspaceID := uuid.New()
	env.repo.workspaces[workspaceID] = db.Workspace{ID: workspaceID, Name: "Platform", Slug: "platform"}

	wInvalid := httptest.NewRecorder()
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID.String()+"/teams", bytes.NewBufferString(`{`))
	reqInvalid.Header.Set("Content-Type", "application/json")
	reqInvalid.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wInvalid, reqInvalid)
	require.Equal(t, http.StatusBadRequest, wInvalid.Code)

	wEmpty := httptest.NewRecorder()
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID.String()+"/teams", bytes.NewReader([]byte(`{"team_ids":[]}`)))
	reqEmpty.Header.Set("Content-Type", "application/json")
	reqEmpty.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wEmpty, reqEmpty)
	require.Equal(t, http.StatusBadRequest, wEmpty.Code)

	wBad := httptest.NewRecorder()
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID.String()+"/teams", bytes.NewReader([]byte(`{"team_ids":["not-a-uuid"]}`)))
	reqBad.Header.Set("Content-Type", "application/json")
	reqBad.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	env.router.ServeHTTP(wBad, reqBad)
	require.Equal(t, http.StatusBadRequest, wBad.Code)
}

func TestWorkspacesAssignTeamsForbiddenForMember(t *testing.T) {
	env := setupWorkspaceRouter(t)
	token := env.sessionForRole(t, "member")
	body, _ := json.Marshal(map[string]any{"team_ids": []string{uuid.New().String()}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+uuid.New().String()+"/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestWorkspacesListIncludesCounts(t *testing.T) {
	env := setupWorkspaceRouter(t)
	token := env.sessionForRole(t, "member")
	workspaceID := uuid.New()
	env.repo.workspaces[workspaceID] = db.Workspace{ID: workspaceID, Name: "Platform", Slug: "platform"}
	env.repo.teams[uuid.New()] = db.Team{ID: uuid.New(), WorkspaceID: workspaceID, Name: "L2"}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	items := body["items"].([]any)
	require.NotEmpty(t, items)
	first := items[0].(map[string]any)
	require.Contains(t, first, "team_count")
	require.Contains(t, first, "routing_rule_count")
}
