package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type scheduleRepoMock struct {
	teams       map[uuid.UUID]db.Team
	schedules   map[uuid.UUID]db.ScheduleWithLayers
	memberUsers map[uuid.UUID]map[uuid.UUID]struct{}
}

func newScheduleRepoMock() *scheduleRepoMock {
	return &scheduleRepoMock{
		teams:       map[uuid.UUID]db.Team{},
		schedules:   map[uuid.UUID]db.ScheduleWithLayers{},
		memberUsers: map[uuid.UUID]map[uuid.UUID]struct{}{},
	}
}

func (m *scheduleRepoMock) GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return team, nil
}

func (m *scheduleRepoMock) TeamMemberUserIDs(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	ids := m.memberUsers[teamID]
	if ids == nil {
		return map[uuid.UUID]struct{}{}, nil
	}
	return ids, nil
}

func (m *scheduleRepoMock) ListSchedulesWithLayersByTeam(ctx context.Context, teamID uuid.UUID) ([]db.ScheduleWithLayers, error) {
	items := make([]db.ScheduleWithLayers, 0)
	for _, schedule := range m.schedules {
		if schedule.Schedule.TeamID == teamID {
			items = append(items, schedule)
		}
	}
	return items, nil
}

func (m *scheduleRepoMock) GetScheduleWithLayersForTeam(ctx context.Context, teamID, scheduleID uuid.UUID) (db.ScheduleWithLayers, error) {
	schedule, ok := m.schedules[scheduleID]
	if !ok || schedule.Schedule.TeamID != teamID {
		return db.ScheduleWithLayers{}, pgx.ErrNoRows
	}
	return schedule, nil
}

func (m *scheduleRepoMock) CreateScheduleWithLayer(ctx context.Context, teamID uuid.UUID, name, timezone string, layer db.CreateScheduleLayerInput) (db.ScheduleWithLayers, error) {
	for _, existing := range m.schedules {
		if existing.Schedule.TeamID == teamID && existing.Schedule.Name == name {
			return db.ScheduleWithLayers{}, errUniqueViolation("schedules_team_id_name_key")
		}
	}
	scheduleID := uuid.New()
	layerID := uuid.New()
	now := time.Now()
	item := db.ScheduleWithLayers{
		Schedule: db.Schedule{
			ID: scheduleID, TeamID: teamID, Name: name, Timezone: timezone, CreatedAt: now, UpdatedAt: now,
		},
		Layers: []db.ScheduleLayer{{
			ID: layerID, ScheduleID: scheduleID, Priority: layer.Priority, RotationType: layer.RotationType,
			HandoffWeekday: layer.HandoffWeekday, HandoffTime: layer.HandoffTime, ParticipantUserIDs: layer.ParticipantUserIDs,
			CreatedAt: now, UpdatedAt: now,
		}},
	}
	m.schedules[scheduleID] = item
	return item, nil
}

func (m *scheduleRepoMock) UpdateScheduleWithLayer(ctx context.Context, teamID, scheduleID uuid.UUID, name, timezone string, layer db.CreateScheduleLayerInput) (db.ScheduleWithLayers, error) {
	existing, ok := m.schedules[scheduleID]
	if !ok || existing.Schedule.TeamID != teamID {
		return db.ScheduleWithLayers{}, pgx.ErrNoRows
	}
	now := time.Now()
	layerID := uuid.New()
	item := db.ScheduleWithLayers{
		Schedule: db.Schedule{
			ID: scheduleID, TeamID: teamID, Name: name, Timezone: timezone,
			CreatedAt: existing.Schedule.CreatedAt, UpdatedAt: now,
		},
		Layers: []db.ScheduleLayer{{
			ID: layerID, ScheduleID: scheduleID, Priority: layer.Priority, RotationType: layer.RotationType,
			HandoffWeekday: layer.HandoffWeekday, HandoffTime: layer.HandoffTime, ParticipantUserIDs: layer.ParticipantUserIDs,
			CreatedAt: now, UpdatedAt: now,
		}},
	}
	m.schedules[scheduleID] = item
	return item, nil
}

func (m *scheduleRepoMock) DeleteScheduleForTeam(ctx context.Context, teamID, scheduleID uuid.UUID) error {
	schedule, ok := m.schedules[scheduleID]
	if !ok || schedule.Schedule.TeamID != teamID {
		return pgx.ErrNoRows
	}
	delete(m.schedules, scheduleID)
	return nil
}

func (m *scheduleRepoMock) EnqueueMaterialiseOnCall(ctx context.Context, teamID uuid.UUID) error {
	return nil
}

func validScheduleInput(participants ...uuid.UUID) CreateScheduleInput {
	return CreateScheduleInput{
		Name:     "Primary",
		Timezone: "Europe/Moscow",
		Rotation: WeeklyRotationInput{
			HandoffWeekday: 1,
			HandoffTime:    "09:00",
			Participants:   participants,
		},
	}
}

func TestCreateScheduleValidatesTimezone(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	input := validScheduleInput(userID)
	input.Timezone = "Not/AZone"
	_, err := svc.CreateSchedule(context.Background(), teamID, input)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestCreateScheduleRequiresParticipants(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	svc := NewScheduleService(repo)

	input := validScheduleInput()
	_, err := svc.CreateSchedule(context.Background(), teamID, input)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestCreateScheduleParticipantsMustBeMembers(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	svc := NewScheduleService(repo)

	_, err := svc.CreateSchedule(context.Background(), teamID, validScheduleInput(uuid.New()))
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestCreateScheduleSuccess(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	schedule, err := svc.CreateSchedule(context.Background(), teamID, validScheduleInput(userID))
	require.NoError(t, err)
	require.Equal(t, "Primary", schedule.Schedule.Name)
	require.Len(t, schedule.Layers, 1)
	require.Equal(t, "weekly", schedule.Layers[0].RotationType)
}

func TestCreateScheduleDuplicateNameConflict(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	_, err := svc.CreateSchedule(context.Background(), teamID, validScheduleInput(userID))
	require.NoError(t, err)
	_, err = svc.CreateSchedule(context.Background(), teamID, validScheduleInput(userID))
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestUpdateScheduleSuccess(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	created, err := svc.CreateSchedule(context.Background(), teamID, validScheduleInput(userID))
	require.NoError(t, err)

	input := validScheduleInput(userID)
	input.Name = "Backup"
	input.Timezone = "UTC"
	updated, err := svc.UpdateSchedule(context.Background(), teamID, created.Schedule.ID, input)
	require.NoError(t, err)
	require.Equal(t, "Backup", updated.Schedule.Name)
	require.Equal(t, "UTC", updated.Schedule.Timezone)
}

func TestDeleteScheduleNotFound(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	svc := NewScheduleService(repo)
	err := svc.DeleteSchedule(context.Background(), teamID, uuid.New())
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestParseParticipantIDsInvalid(t *testing.T) {
	_, err := ParseParticipantIDs([]string{"bad"})
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestParseParticipantIDsSuccess(t *testing.T) {
	id := uuid.New()
	ids, err := ParseParticipantIDs([]string{id.String()})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{id}, ids)
}

func TestScheduleJSONHelpers(t *testing.T) {
	now := time.Now()
	schedule := db.ScheduleWithLayers{
		Schedule: db.Schedule{ID: uuid.New(), TeamID: uuid.New(), Name: "Primary", Timezone: "UTC", CreatedAt: now, UpdatedAt: now},
		Layers: []db.ScheduleLayer{{
			ID: uuid.New(), ScheduleID: uuid.New(), RotationType: "weekly", HandoffWeekday: 1,
			HandoffTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), ParticipantUserIDs: []uuid.UUID{uuid.New()},
			CreatedAt: now, UpdatedAt: now,
		}},
	}
	body := ScheduleJSON(schedule)
	require.Equal(t, "Primary", body["name"])
	layers := body["layers"].([]map[string]any)
	require.Equal(t, "09:00", layers[0]["handoff_time"])
}

func TestListSchedulesTeamNotFound(t *testing.T) {
	svc := NewScheduleService(newScheduleRepoMock())
	_, err := svc.ListSchedules(context.Background(), uuid.New())
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestGetScheduleSuccess(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	created, err := svc.CreateSchedule(context.Background(), teamID, validScheduleInput(userID))
	require.NoError(t, err)

	got, err := svc.GetSchedule(context.Background(), teamID, created.Schedule.ID)
	require.NoError(t, err)
	require.Equal(t, created.Schedule.ID, got.Schedule.ID)
}

func TestDeleteScheduleSuccess(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	created, err := svc.CreateSchedule(context.Background(), teamID, validScheduleInput(userID))
	require.NoError(t, err)
	require.NoError(t, svc.DeleteSchedule(context.Background(), teamID, created.Schedule.ID))
}

func TestValidateScheduleNameRequired(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	input := validScheduleInput(userID)
	input.Name = "  "
	_, err := svc.CreateSchedule(context.Background(), teamID, input)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestValidateHandoffWeekday(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	input := validScheduleInput(userID)
	input.Rotation.HandoffWeekday = 9
	_, err := svc.CreateSchedule(context.Background(), teamID, input)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestValidateHandoffTime(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	input := validScheduleInput(userID)
	input.Rotation.HandoffTime = "25:99"
	_, err := svc.CreateSchedule(context.Background(), teamID, input)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestValidateDuplicateParticipants(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	input := validScheduleInput(userID, userID)
	_, err := svc.CreateSchedule(context.Background(), teamID, input)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestParseParticipantIDsEmpty(t *testing.T) {
	_, err := ParseParticipantIDs(nil)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestMapScheduleErrorPassthrough(t *testing.T) {
	raw := errors.New("db down")
	require.Equal(t, raw, mapScheduleError(raw))
}

func TestGetScheduleTeamNotFound(t *testing.T) {
	svc := NewScheduleService(newScheduleRepoMock())
	_, err := svc.GetSchedule(context.Background(), uuid.New(), uuid.New())
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestUpdateScheduleNotFound(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	_, err := svc.UpdateSchedule(context.Background(), teamID, uuid.New(), validScheduleInput(userID))
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestListSchedulesSuccess(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	_, err := svc.CreateSchedule(context.Background(), teamID, validScheduleInput(userID))
	require.NoError(t, err)

	schedules, err := svc.ListSchedules(context.Background(), teamID)
	require.NoError(t, err)
	require.Len(t, schedules, 1)
}

func TestCreateScheduleTeamNotFound(t *testing.T) {
	repo := newScheduleRepoMock()
	userID := uuid.New()
	svc := NewScheduleService(repo)
	_, err := svc.CreateSchedule(context.Background(), uuid.New(), validScheduleInput(userID))
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestValidateTimezoneRequired(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	input := validScheduleInput(userID)
	input.Timezone = ""
	_, err := svc.CreateSchedule(context.Background(), teamID, input)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestValidateNilParticipantID(t *testing.T) {
	repo := newScheduleRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	userID := uuid.New()
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	input := validScheduleInput(uuid.Nil)
	_, err := svc.CreateSchedule(context.Background(), teamID, input)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestMapScheduleErrorConflict(t *testing.T) {
	err := mapScheduleError(errUniqueViolation("schedules_team_id_name_key"))
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "CONFLICT", appErr.Code)
}

func TestMapScheduleErrorNotFound(t *testing.T) {
	err := mapScheduleError(pgx.ErrNoRows)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

type failingScheduleEnqueueRepo struct {
	scheduleRepoMock
	failAfter int
	calls     int
}

func (m *failingScheduleEnqueueRepo) EnqueueMaterialiseOnCall(ctx context.Context, teamID uuid.UUID) error {
	m.calls++
	if m.calls > m.failAfter {
		return errors.New("enqueue failed")
	}
	return nil
}

type nilSchedulesRepo struct {
	scheduleRepoMock
}

func (m *nilSchedulesRepo) ListSchedulesWithLayersByTeam(ctx context.Context, teamID uuid.UUID) ([]db.ScheduleWithLayers, error) {
	return nil, nil
}

func TestListSchedulesNilResult(t *testing.T) {
	repo := &nilSchedulesRepo{scheduleRepoMock: *newScheduleRepoMock()}
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	svc := NewScheduleService(repo)

	items, err := svc.ListSchedules(context.Background(), teamID)
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestUpdateScheduleEnqueueFailure(t *testing.T) {
	repo := &failingScheduleEnqueueRepo{scheduleRepoMock: *newScheduleRepoMock(), failAfter: 1}
	teamID := uuid.New()
	userID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	created, err := svc.CreateSchedule(context.Background(), teamID, validScheduleInput(userID))
	require.NoError(t, err)

	_, err = svc.UpdateSchedule(context.Background(), teamID, created.Schedule.ID, validScheduleInput(userID))
	require.Error(t, err)
}

func TestDeleteScheduleEnqueueFailure(t *testing.T) {
	repo := &failingScheduleEnqueueRepo{scheduleRepoMock: *newScheduleRepoMock(), failAfter: 1}
	teamID := uuid.New()
	userID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	repo.memberUsers[teamID] = map[uuid.UUID]struct{}{userID: {}}
	svc := NewScheduleService(repo)

	created, err := svc.CreateSchedule(context.Background(), teamID, validScheduleInput(userID))
	require.NoError(t, err)

	err = svc.DeleteSchedule(context.Background(), teamID, created.Schedule.ID)
	require.Error(t, err)
}
