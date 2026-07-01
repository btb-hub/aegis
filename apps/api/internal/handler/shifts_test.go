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
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type shiftsHandlerRepo struct {
	teamRepoMock
	overrides map[uuid.UUID]db.Override
	members   map[uuid.UUID]map[uuid.UUID]struct{}
	enqueued  []uuid.UUID
	onCall    []db.OnCallUser
	slots     []db.OnCallSlot
}

func newShiftsHandlerRepo() *shiftsHandlerRepo {
	return &shiftsHandlerRepo{
		teamRepoMock: teamRepoMock{
			authMockUsers:    authMockUsers{users: map[uuid.UUID]db.User{}},
			authMockSessions: authMockSessions{byHash: map[string]db.Session{}},
			teams:            map[uuid.UUID]db.Team{},
			memberships:      map[uuid.UUID]map[uuid.UUID]db.TeamMembership{},
		},
		overrides: map[uuid.UUID]db.Override{},
		members:   map[uuid.UUID]map[uuid.UUID]struct{}{},
	}
}

func (m *shiftsHandlerRepo) TeamMemberUserIDs(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	ids := m.members[teamID]
	if ids == nil {
		return map[uuid.UUID]struct{}{}, nil
	}
	return ids, nil
}

func (m *shiftsHandlerRepo) ListOverridesByTeam(ctx context.Context, teamID uuid.UUID) ([]db.Override, error) {
	items := make([]db.Override, 0)
	for _, override := range m.overrides {
		if override.TeamID == teamID {
			items = append(items, override)
		}
	}
	return items, nil
}

func (m *shiftsHandlerRepo) GetOverrideForTeam(ctx context.Context, teamID, overrideID uuid.UUID) (db.Override, error) {
	override, ok := m.overrides[overrideID]
	if !ok || override.TeamID != teamID {
		return db.Override{}, pgx.ErrNoRows
	}
	return override, nil
}

func (m *shiftsHandlerRepo) CreateOverride(ctx context.Context, teamID, userID uuid.UUID, startAt, endAt time.Time) (db.Override, error) {
	id := uuid.New()
	override := db.Override{ID: id, TeamID: teamID, UserID: userID, StartAt: startAt, EndAt: endAt, CreatedAt: time.Now()}
	m.overrides[id] = override
	return override, nil
}

func (m *shiftsHandlerRepo) DeleteOverrideForTeam(ctx context.Context, teamID, overrideID uuid.UUID) error {
	override, ok := m.overrides[overrideID]
	if !ok || override.TeamID != teamID {
		return pgx.ErrNoRows
	}
	delete(m.overrides, overrideID)
	return nil
}

func (m *shiftsHandlerRepo) EnqueueMaterialiseOnCall(ctx context.Context, teamID uuid.UUID) error {
	m.enqueued = append(m.enqueued, teamID)
	return nil
}

func (m *shiftsHandlerRepo) CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error) {
	return m.onCall, nil
}

func (m *shiftsHandlerRepo) ListOnCallSlotsInRange(ctx context.Context, teamID uuid.UUID, from, to time.Time) ([]db.OnCallSlot, error) {
	return m.slots, nil
}

type shiftsTestEnv struct {
	router *gin.Engine
	repo   *shiftsHandlerRepo
}

func setupShiftsRouter(t *testing.T) *shiftsTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newShiftsHandlerRepo()
	cfg := &config.Config{SessionTTL: time.Hour}
	auth := service.NewAuthService(cfg, repo, repo, &authMockOIDC{})
	overrides := service.NewOverrideService(repo)
	teams := service.NewTeamService(repo)
	oncall := service.NewOnCallService(repo)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &authMockAlertRepo{id: uuid.New()})
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth, "http://localhost:3000").Register(r)
	NewAlertHandler(alerts, teams, auth).Register(r)
	NewOverrideHandler(overrides, auth).Register(r)
	NewOnCallHandler(oncall, auth).Register(r)
	return &shiftsTestEnv{router: r, repo: repo}
}

func (env *shiftsTestEnv) sessionForRole(t *testing.T, role string) string {
	t.Helper()
	userID := uuid.New()
	env.repo.users[userID] = db.User{ID: userID, Role: role, DisplayName: "User"}
	token, hash, err := sessionTokenPair()
	require.NoError(t, err)
	_, err = env.repo.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))
	require.NoError(t, err)
	return token
}

func TestOverridesCreateRequiresAdmin(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "member")
	body, _ := json.Marshal(map[string]string{
		"user_id":  uuid.New().String(),
		"start_at": "2026-06-30T00:00:00Z",
		"end_at":   "2026-07-01T00:00:00Z",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/overrides", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestOverridesCreateAndDelete(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	userID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	env.repo.members[teamID] = map[uuid.UUID]struct{}{userID: {}}
	token := env.sessionForRole(t, "admin")

	body, _ := json.Marshal(map[string]string{
		"user_id":  userID.String(),
		"start_at": "2026-06-30T00:00:00Z",
		"end_at":   "2026-07-01T00:00:00Z",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/overrides", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	overrideID := created["id"].(string)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+teamID.String()+"/overrides/"+overrideID, nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Len(t, env.repo.enqueued, 2)
}

func TestOnCallCurrent(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	userID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	env.repo.onCall = []db.OnCallUser{{UserID: userID, Email: "a@example.com", DisplayName: "Alice", Source: "rotation"}}
	token := env.sessionForRole(t, "member")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/on-call/current", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Items, 1)
}

func TestOnCallCalendarRequiresRange(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "member")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/on-call/calendar", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverridesList(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	userID := uuid.New()
	overrideID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	env.repo.overrides[overrideID] = db.Override{ID: overrideID, TeamID: teamID, UserID: userID}
	token := env.sessionForRole(t, "member")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/overrides", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Items, 1)
}

func TestOverridesListTeamNotFound(t *testing.T) {
	env := setupShiftsRouter(t)
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String()+"/overrides", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOverridesCreateInvalidBody(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "admin")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/overrides", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverridesCreateInvalidUserID(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{
		"user_id": "bad", "start_at": "2026-06-30T00:00:00Z", "end_at": "2026-07-01T00:00:00Z",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/overrides", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverridesCreateInvalidStartAt(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{
		"user_id": uuid.New().String(), "start_at": "bad", "end_at": "2026-07-01T00:00:00Z",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/overrides", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverridesCreateInvalidEndAt(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{
		"user_id": uuid.New().String(), "start_at": "2026-06-30T00:00:00Z", "end_at": "bad",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/overrides", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverridesCreateNotTeamMember(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "admin")
	body, _ := json.Marshal(map[string]string{
		"user_id": uuid.New().String(), "start_at": "2026-06-30T00:00:00Z", "end_at": "2026-07-01T00:00:00Z",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/overrides", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverridesDeleteNotFound(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "admin")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+teamID.String()+"/overrides/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOverridesDeleteInvalidOverrideID(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "admin")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+teamID.String()+"/overrides/not-a-uuid", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOnCallCalendarSuccess(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	slotID := uuid.New()
	userID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	env.repo.slots = []db.OnCallSlot{{ID: slotID, TeamID: teamID, UserID: userID, Source: "rotation"}}
	token := env.sessionForRole(t, "member")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/on-call/calendar?from=2026-06-01T00:00:00Z&to=2026-07-01T00:00:00Z", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Items, 1)
}

func TestOnCallCalendarInvalidFrom(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "member")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/on-call/calendar?from=bad&to=2026-07-01T00:00:00Z", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOnCallCalendarInvalidTo(t *testing.T) {
	env := setupShiftsRouter(t)
	teamID := uuid.New()
	env.repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	token := env.sessionForRole(t, "member")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/on-call/calendar?from=2026-06-01T00:00:00Z&to=bad", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOnCallCurrentTeamNotFound(t *testing.T) {
	env := setupShiftsRouter(t)
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String()+"/on-call/current", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOnCallCurrentInvalidTeamID(t *testing.T) {
	env := setupShiftsRouter(t)
	token := env.sessionForRole(t, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/not-a-uuid/on-call/current", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
