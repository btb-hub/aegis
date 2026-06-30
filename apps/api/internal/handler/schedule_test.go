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

type scheduleHandlerRepo struct {
	teamRepoMock
	schedules   map[uuid.UUID]db.ScheduleWithLayers
	memberUsers map[uuid.UUID]map[uuid.UUID]struct{}
}

func newScheduleHandlerRepo() *scheduleHandlerRepo {
	return &scheduleHandlerRepo{
		teamRepoMock: teamRepoMock{
			authMockUsers:    authMockUsers{users: map[uuid.UUID]db.User{}},
			authMockSessions: authMockSessions{byHash: map[string]db.Session{}},
			teams:            map[uuid.UUID]db.Team{},
			memberships:      map[uuid.UUID]map[uuid.UUID]db.TeamMembership{},
		},
		schedules:   map[uuid.UUID]db.ScheduleWithLayers{},
		memberUsers: map[uuid.UUID]map[uuid.UUID]struct{}{},
	}
}

func (m *scheduleHandlerRepo) TeamMemberUserIDs(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	ids := m.memberUsers[teamID]
	if ids == nil {
		return map[uuid.UUID]struct{}{}, nil
	}
	return ids, nil
}

func (m *scheduleHandlerRepo) ListSchedulesWithLayersByTeam(ctx context.Context, teamID uuid.UUID) ([]db.ScheduleWithLayers, error) {
	items := make([]db.ScheduleWithLayers, 0)
	for _, schedule := range m.schedules {
		if schedule.Schedule.TeamID == teamID {
			items = append(items, schedule)
		}
	}
	return items, nil
}

func (m *scheduleHandlerRepo) GetScheduleWithLayersForTeam(ctx context.Context, teamID, scheduleID uuid.UUID) (db.ScheduleWithLayers, error) {
	schedule, ok := m.schedules[scheduleID]
	if !ok || schedule.Schedule.TeamID != teamID {
		return db.ScheduleWithLayers{}, pgx.ErrNoRows
	}
	return schedule, nil
}

func (m *scheduleHandlerRepo) CreateScheduleWithLayer(ctx context.Context, teamID uuid.UUID, name, timezone string, layer db.CreateScheduleLayerInput) (db.ScheduleWithLayers, error) {
	scheduleID := uuid.New()
	now := time.Now()
	item := db.ScheduleWithLayers{
		Schedule: db.Schedule{ID: scheduleID, TeamID: teamID, Name: name, Timezone: timezone, CreatedAt: now, UpdatedAt: now},
		Layers: []db.ScheduleLayer{{
			ID: uuid.New(), ScheduleID: scheduleID, Priority: layer.Priority, RotationType: layer.RotationType,
			HandoffWeekday: layer.HandoffWeekday, HandoffTime: layer.HandoffTime, ParticipantUserIDs: layer.ParticipantUserIDs,
			CreatedAt: now, UpdatedAt: now,
		}},
	}
	m.schedules[scheduleID] = item
	return item, nil
}

func (m *scheduleHandlerRepo) UpdateScheduleWithLayer(ctx context.Context, teamID, scheduleID uuid.UUID, name, timezone string, layer db.CreateScheduleLayerInput) (db.ScheduleWithLayers, error) {
	existing, ok := m.schedules[scheduleID]
	if !ok || existing.Schedule.TeamID != teamID {
		return db.ScheduleWithLayers{}, pgx.ErrNoRows
	}
	now := time.Now()
	item := db.ScheduleWithLayers{
		Schedule: db.Schedule{
			ID: scheduleID, TeamID: teamID, Name: name, Timezone: timezone,
			CreatedAt: existing.Schedule.CreatedAt, UpdatedAt: now,
		},
		Layers: []db.ScheduleLayer{{
			ID: uuid.New(), ScheduleID: scheduleID, Priority: layer.Priority, RotationType: layer.RotationType,
			HandoffWeekday: layer.HandoffWeekday, HandoffTime: layer.HandoffTime, ParticipantUserIDs: layer.ParticipantUserIDs,
			CreatedAt: now, UpdatedAt: now,
		}},
	}
	m.schedules[scheduleID] = item
	return item, nil
}

func (m *scheduleHandlerRepo) DeleteScheduleForTeam(ctx context.Context, teamID, scheduleID uuid.UUID) error {
	schedule, ok := m.schedules[scheduleID]
	if !ok || schedule.Schedule.TeamID != teamID {
		return pgx.ErrNoRows
	}
	delete(m.schedules, scheduleID)
	return nil
}

func (m *scheduleHandlerRepo) EnqueueMaterialiseOnCall(ctx context.Context, teamID uuid.UUID) error {
	return nil
}

func setupScheduleRouter(t *testing.T) (*gin.Engine, *scheduleHandlerRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newScheduleHandlerRepo()
	cfg := &config.Config{SessionTTL: time.Hour}
	auth := service.NewAuthService(cfg, repo, repo, &authMockOIDC{})
	teams := service.NewTeamService(repo)
	schedules := service.NewScheduleService(repo)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &authMockAlertRepo{id: uuid.New()})
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth, "http://localhost:3000").Register(r)
	NewAlertHandler(alerts).Register(r)
	NewTeamHandler(teams, auth).Register(r)
	NewScheduleHandler(schedules, auth).Register(r)
	return r, repo
}

func scheduleBody(participantID uuid.UUID) []byte {
	body, _ := json.Marshal(map[string]any{
		"name":     "Primary",
		"timezone": "Europe/Moscow",
		"rotation": map[string]any{
			"handoff_weekday": 1,
			"handoff_time":    "09:00",
			"participants":    []string{participantID.String()},
		},
	})
	return body
}

func TestSchedulesListRequiresSession(t *testing.T) {
	r, _ := setupScheduleRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String()+"/schedules", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSchedulesCreateForbiddenForMember(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	memberID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{memberID: {}}

	token := sessionForRoleOnRepo(t, repo, "member")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewReader(scheduleBody(memberID)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestSchedulesCreateAllowedForAdmin(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	memberID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{memberID: {}}

	token := sessionForRoleOnRepo(t, repo, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewReader(scheduleBody(memberID)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestSchedulesListAllowedForViewer(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	memberID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{memberID: {}}

	adminToken := sessionForRoleOnRepo(t, repo, "admin")
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewReader(scheduleBody(memberID)))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	r.ServeHTTP(wCreate, reqCreate)
	require.Equal(t, http.StatusCreated, wCreate.Code)

	viewerToken := sessionForRoleOnRepo(t, repo, "viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/schedules", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: viewerToken})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestSchedulesCreateRejectsNonMemberParticipant(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	token := sessionForRoleOnRepo(t, repo, "admin")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewReader(scheduleBody(uuid.New())))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSchedulesUpdateAndDeleteAdmin(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	memberID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{memberID: {}}
	token := sessionForRoleOnRepo(t, repo, "admin")

	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewReader(scheduleBody(memberID)))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(wCreate, reqCreate)
	require.Equal(t, http.StatusCreated, wCreate.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &created))
	scheduleID := created["id"].(string)

	patchBody, _ := json.Marshal(map[string]any{
		"name": "Backup", "timezone": "UTC",
		"rotation": map[string]any{"handoff_weekday": 2, "handoff_time": "10:00", "participants": []string{memberID.String()}},
	})
	wPatch := httptest.NewRecorder()
	reqPatch := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+teamID.String()+"/schedules/"+scheduleID, bytes.NewReader(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(wPatch, reqPatch)
	require.Equal(t, http.StatusOK, wPatch.Code)

	wDelete := httptest.NewRecorder()
	reqDelete := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+teamID.String()+"/schedules/"+scheduleID, nil)
	reqDelete.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, http.StatusNoContent, wDelete.Code)
}

func TestSchedulesGetSuccess(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	memberID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{memberID: {}}
	token := sessionForRoleOnRepo(t, repo, "viewer")

	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewReader(scheduleBody(memberID)))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: sessionForRoleOnRepo(t, repo, "admin")})
	r.ServeHTTP(wCreate, reqCreate)

	var created map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &created))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/schedules/"+created["id"].(string), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestSchedulesGetNotFound(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	token := sessionForRoleOnRepo(t, repo, "viewer")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/schedules/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSchedulesInvalidTeamID(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	token := sessionForRoleOnRepo(t, repo, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/not-a-uuid/schedules", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSchedulesCreateInvalidBody(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	token := sessionForRoleOnRepo(t, repo, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSchedulesDeleteSuccess(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	memberID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{memberID: {}}
	adminToken := sessionForRoleOnRepo(t, repo, "admin")

	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewReader(scheduleBody(memberID)))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	r.ServeHTTP(wCreate, reqCreate)
	require.Equal(t, http.StatusCreated, wCreate.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &created))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+teamID.String()+"/schedules/"+created["id"].(string), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestSchedulesDeleteForbiddenForViewer(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	memberID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{memberID: {}}
	adminToken := sessionForRoleOnRepo(t, repo, "admin")

	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewReader(scheduleBody(memberID)))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	r.ServeHTTP(wCreate, reqCreate)

	var created map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &created))

	viewerToken := sessionForRoleOnRepo(t, repo, "viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+teamID.String()+"/schedules/"+created["id"].(string), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: viewerToken})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestSchedulesCreateTeamNotFound(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	token := sessionForRoleOnRepo(t, repo, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+uuid.New().String()+"/schedules", bytes.NewReader(scheduleBody(uuid.New())))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSchedulesUpdateNotFound(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	memberID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{memberID: {}}
	token := sessionForRoleOnRepo(t, repo, "admin")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+teamID.String()+"/schedules/"+uuid.New().String(), bytes.NewReader(scheduleBody(memberID)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSchedulesListTeamNotFound(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	token := sessionForRoleOnRepo(t, repo, "viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+uuid.New().String()+"/schedules", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSchedulesUpdateInvalidScheduleID(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	token := sessionForRoleOnRepo(t, repo, "admin")
	body := scheduleBody(uuid.New())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+teamID.String()+"/schedules/not-a-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSchedulesDeleteNotFound(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	token := sessionForRoleOnRepo(t, repo, "admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/"+teamID.String()+"/schedules/"+uuid.New().String(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSchedulesUpdateInvalidBody(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	memberID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{memberID: {}}
	adminToken := sessionForRoleOnRepo(t, repo, "admin")

	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID.String()+"/schedules", bytes.NewReader(scheduleBody(memberID)))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	r.ServeHTTP(wCreate, reqCreate)

	var created map[string]any
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &created))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+teamID.String()+"/schedules/"+created["id"].(string), bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminToken})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSchedulesGetInvalidScheduleID(t *testing.T) {
	r, repo := setupScheduleRouter(t)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	token := sessionForRoleOnRepo(t, repo, "viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String()+"/schedules/not-a-uuid", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func sessionForRoleOnRepo(t *testing.T, repo *scheduleHandlerRepo, role string) string {
	t.Helper()
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID, Role: role}
	token, hash, err := sessionTokenPair()
	require.NoError(t, err)
	_, err = repo.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))
	require.NoError(t, err)
	return token
}
