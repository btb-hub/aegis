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
	teams       map[uuid.UUID]db.Team
	users       []db.OnCallUser
	slots       []db.OnCallSlot
	published   []uuid.UUID
	usersErr    error
	slotsErr    error
	enqueueErr  error
	integration db.Integration
}

func (m *onCallRepoMock) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	if m.integration.Kind == "" {
		return db.Integration{}, pgx.ErrNoRows
	}
	return m.integration, nil
}

func TestOnCallServiceEnqueuePublishGlobalExpress(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID}
	repo.integration = db.Integration{Kind: "express", Enabled: true, Config: []byte(`{"oncall_group_chat_id":"group-1"}`)}
	require.NoError(t, NewOnCallService(repo).EnqueuePublish(t.Context(), teamID))
	require.Equal(t, []uuid.UUID{teamID}, repo.published)
}

func TestOnCallServiceEnqueuePublishRejectsLegacyExpressOnly(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	legacy := "legacy-chat"
	repo.teams[teamID] = db.Team{ID: teamID, ExpressChatID: &legacy}
	require.Error(t, NewOnCallService(repo).EnqueuePublish(t.Context(), teamID))
	require.Empty(t, repo.published)
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
	if m.usersErr != nil {
		return nil, m.usersErr
	}
	return m.users, nil
}

func (m *onCallRepoMock) ListOnCallSlotsInRange(ctx context.Context, teamID uuid.UUID, from, to time.Time) ([]db.OnCallSlot, error) {
	if m.slotsErr != nil {
		return nil, m.slotsErr
	}
	return m.slots, nil
}

func (m *onCallRepoMock) EnqueuePublishOnCall(_ context.Context, teamID uuid.UUID) error {
	if m.enqueueErr != nil {
		return m.enqueueErr
	}
	m.published = append(m.published, teamID)
	return nil
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

func TestOnCallServiceCalendarSuccess(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	slotID := uuid.New()
	userID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.slots = []db.OnCallSlot{{ID: slotID, TeamID: teamID, UserID: userID, Source: "rotation"}}

	svc := NewOnCallService(repo)
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	slots, err := svc.Calendar(context.Background(), teamID, from, to)
	require.NoError(t, err)
	require.Len(t, slots, 1)
}

func TestOnCallServiceCurrentEmptyUsers(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}

	svc := NewOnCallService(repo)
	users, err := svc.CurrentOnCall(context.Background(), teamID)
	require.NoError(t, err)
	require.Empty(t, users)
}

func TestOnCallUserJSON(t *testing.T) {
	userID := uuid.New()
	slackID := "U123"
	huid := uuid.MustParse("83fbf1c7-f14b-5176-bd32-ca15cf00d4b7")
	payload := OnCallUserJSON(db.OnCallUser{
		UserID:          userID,
		Email:           "a@example.com",
		DisplayName:     "Alice",
		Source:          "rotation",
		SlackUserID:     &slackID,
		ExpressUserHuid: db.ExpressHuidToPg(huid),
	})
	require.Equal(t, userID.String(), payload["user_id"])
	require.Equal(t, "Alice", payload["display_name"])
	require.Equal(t, "rotation", payload["source"])
	contacts, ok := payload["contacts"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, "mailto:a@example.com", contacts["email"])
	require.Equal(t, "https://slack.com/app_redirect?channel=U123", contacts["slack"])
	require.Equal(t, "https://xlnk.ms/open/profile/"+huid.String(), contacts["express"])
}

func TestOnCallUserJSONOmitsUnlinkedChat(t *testing.T) {
	payload := OnCallUserJSON(db.OnCallUser{UserID: uuid.New(), Email: "a@example.com", DisplayName: "Alice", Source: "override"})
	contacts, ok := payload["contacts"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, map[string]string{"email": "mailto:a@example.com"}, contacts)
}

func TestOnCallSlotJSON(t *testing.T) {
	slotID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	payload := OnCallSlotJSON(db.OnCallSlot{ID: slotID, TeamID: teamID, UserID: userID, StartAt: now, EndAt: now.Add(time.Hour), Source: "override", CreatedAt: now})
	require.Equal(t, slotID.String(), payload["id"])
	require.Equal(t, "override", payload["source"])
}

func TestOnCallServiceCalendarTeamNotFound(t *testing.T) {
	svc := NewOnCallService(newOnCallRepoMock())
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.Calendar(context.Background(), uuid.New(), from, to)
	require.Error(t, err)
}

type nilSlotsRepo struct {
	onCallRepoMock
}

func (m *nilSlotsRepo) ListOnCallSlotsInRange(ctx context.Context, teamID uuid.UUID, from, to time.Time) ([]db.OnCallSlot, error) {
	return nil, nil
}

func TestOnCallServiceCalendarNilSlots(t *testing.T) {
	repo := &nilSlotsRepo{onCallRepoMock: *newOnCallRepoMock()}
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	svc := NewOnCallService(repo)

	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	slots, err := svc.Calendar(context.Background(), teamID, from, to)
	require.NoError(t, err)
	require.Empty(t, slots)
}

func TestOnCallServiceEnqueuePublish(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	chat := "chat-1"
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform", SlackChannelID: &chat}
	svc := NewOnCallService(repo)
	require.NoError(t, svc.EnqueuePublish(context.Background(), teamID))
	require.Equal(t, []uuid.UUID{teamID}, repo.published)
}

func TestOnCallServiceEnqueuePublishTeamNotFound(t *testing.T) {
	svc := NewOnCallService(newOnCallRepoMock())
	err := svc.EnqueuePublish(context.Background(), uuid.New())
	require.Error(t, err)
}

func TestOnCallServiceEnqueuePublishRequiresChannel(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	svc := NewOnCallService(repo)
	err := svc.EnqueuePublish(context.Background(), teamID)
	require.Error(t, err)
	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestOnCallServiceCurrentUsersError(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.usersErr = context.Canceled
	svc := NewOnCallService(repo)
	_, err := svc.CurrentOnCall(context.Background(), teamID)
	require.Error(t, err)
}

func TestOnCallServiceCalendarSlotsError(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.slotsErr = context.Canceled
	svc := NewOnCallService(repo)
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.Calendar(context.Background(), teamID, from, to)
	require.Error(t, err)
}

func TestOnCallServiceEnqueuePublishRepoError(t *testing.T) {
	repo := newOnCallRepoMock()
	teamID := uuid.New()
	chat := "chat-1"
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform", SlackChannelID: &chat}
	repo.enqueueErr = context.Canceled
	svc := NewOnCallService(repo)
	require.Error(t, svc.EnqueuePublish(context.Background(), teamID))
}
