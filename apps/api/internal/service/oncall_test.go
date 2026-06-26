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

type onCallRepoMock struct {
	teams map[uuid.UUID]db.Team
	users []db.OnCallUser
	slots []db.OnCallSlot
}

func newOnCallRepoMock() *onCallRepoMock {
	return &onCallRepoMock{teams: map[uuid.UUID]db.Team{}}
}

func (m *onCallRepoMock) GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return team, nil
}

func (m *onCallRepoMock) CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]db.OnCallUser, error) {
	return m.users, nil
}

func (m *onCallRepoMock) ListOnCallSlotsInRange(ctx context.Context, teamID uuid.UUID, from, to time.Time) ([]db.OnCallSlot, error) {
	return m.slots, nil
}

func TestOnCallServiceCurrent(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	userID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.users = []db.OnCallUser{{UserID: userID, Email: "a@example.com", DisplayName: "Alice", Source: "rotation"}}

	svc := NewOnCallService(repo)
	users, err := svc.CurrentOnCall(context.Background(), teamID)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, userID, users[0].UserID)
}

func TestOnCallServiceCalendarValidation(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	svc := NewOnCallService(repo)

	_, err := svc.Calendar(context.Background(), teamID, time.Now(), time.Now().Add(-time.Hour))
	require.Error(t, err)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestOnCallServiceTeamNotFound(t *testing.T) {
	svc := NewOnCallService(newOnCallRepoMock())
	_, err := svc.CurrentOnCall(context.Background(), uuid.New())
	require.Error(t, err)
}
