package service

import (
	"context"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type overrideRepoMock struct {
	teams     map[uuid.UUID]db.Team
	overrides map[uuid.UUID]db.Override
	members   map[uuid.UUID]map[uuid.UUID]struct{}
	enqueued  []uuid.UUID
}

func newOverrideRepoMock() *overrideRepoMock {
	return &overrideRepoMock{
		teams:     map[uuid.UUID]db.Team{},
		overrides: map[uuid.UUID]db.Override{},
		members:   map[uuid.UUID]map[uuid.UUID]struct{}{},
	}
}

func (m *overrideRepoMock) GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return team, nil
}

func (m *overrideRepoMock) TeamMemberUserIDs(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	ids := m.members[teamID]
	if ids == nil {
		return map[uuid.UUID]struct{}{}, nil
	}
	return ids, nil
}

func (m *overrideRepoMock) ListOverridesByTeam(ctx context.Context, teamID uuid.UUID) ([]db.Override, error) {
	items := make([]db.Override, 0)
	for _, override := range m.overrides {
		if override.TeamID == teamID {
			items = append(items, override)
		}
	}
	return items, nil
}

func (m *overrideRepoMock) GetOverrideForTeam(ctx context.Context, teamID, overrideID uuid.UUID) (db.Override, error) {
	override, ok := m.overrides[overrideID]
	if !ok || override.TeamID != teamID {
		return db.Override{}, pgx.ErrNoRows
	}
	return override, nil
}

func (m *overrideRepoMock) CreateOverride(ctx context.Context, teamID, userID uuid.UUID, startAt, endAt time.Time) (db.Override, error) {
	id := uuid.New()
	override := db.Override{ID: id, TeamID: teamID, UserID: userID, StartAt: startAt, EndAt: endAt, CreatedAt: time.Now()}
	m.overrides[id] = override
	return override, nil
}

func (m *overrideRepoMock) DeleteOverrideForTeam(ctx context.Context, teamID, overrideID uuid.UUID) error {
	override, ok := m.overrides[overrideID]
	if !ok || override.TeamID != teamID {
		return pgx.ErrNoRows
	}
	delete(m.overrides, overrideID)
	return nil
}

func (m *overrideRepoMock) EnqueueMaterialiseOnCall(ctx context.Context, teamID uuid.UUID) error {
	m.enqueued = append(m.enqueued, teamID)
	return nil
}

func TestOverrideServiceCreateAndDelete(t *testing.T) {
	repo := newOverrideRepoMock()
	teamID := uuid.New()
	userID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.members[teamID] = map[uuid.UUID]struct{}{userID: {}}

	svc := NewOverrideService(repo)
	start := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	override, err := svc.CreateOverride(context.Background(), teamID, CreateOverrideInput{UserID: userID, StartAt: start, EndAt: end})
	require.NoError(t, err)
	require.Equal(t, userID, override.UserID)
	require.Len(t, repo.enqueued, 1)

	err = svc.DeleteOverride(context.Background(), teamID, override.ID)
	require.NoError(t, err)
	require.Len(t, repo.enqueued, 2)
}

func TestOverrideServiceValidation(t *testing.T) {
	repo := newOverrideRepoMock()
	teamID := uuid.New()
	userID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	svc := NewOverrideService(repo)

	_, err := svc.CreateOverride(context.Background(), teamID, CreateOverrideInput{
		UserID: userID, StartAt: time.Now(), EndAt: time.Now().Add(-time.Hour),
	})
	require.Error(t, err)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestOverrideServiceTeamNotFound(t *testing.T) {
	svc := NewOverrideService(newOverrideRepoMock())
	_, err := svc.ListOverrides(context.Background(), uuid.New())
	require.Error(t, err)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}
