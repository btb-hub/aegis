package processor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type publishMockStore struct {
	team      db.Team
	teams     []db.Team
	onCall    []db.OnCallUser
	users     map[uuid.UUID]db.User
	announced string
	listErr   error
	getErr    error
	onCallErr error
	userErr   error
	setErr    error
}

func (m *publishMockStore) GetTeam(context.Context, uuid.UUID) (db.Team, error) {
	if m.getErr != nil {
		return db.Team{}, m.getErr
	}
	return m.team, nil
}
func (m *publishMockStore) ListTeamsWithChatChannels(context.Context) ([]db.Team, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.teams, nil
}
func (m *publishMockStore) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	if m.onCallErr != nil {
		return nil, m.onCallErr
	}
	return m.onCall, nil
}
func (m *publishMockStore) GetUserByID(_ context.Context, id uuid.UUID) (db.User, error) {
	if m.userErr != nil {
		return db.User{}, m.userErr
	}
	user, ok := m.users[id]
	if !ok {
		return db.User{ID: id, DisplayName: "Unknown"}, nil
	}
	return user, nil
}
func (m *publishMockStore) SetTeamOnCallAnnounced(_ context.Context, _ uuid.UUID, fingerprint string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.announced = fingerprint
	return nil
}
func (m *publishMockStore) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *publishMockStore) GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error) {
	return db.Integration{}, nil
}
func (m *publishMockStore) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	return db.Integration{}, nil
}

type mockAnnouncer struct {
	calls []string
	err   error
}

func (m *mockAnnouncer) AnnounceOnCall(_ context.Context, channelID, teamName string, people []integrations.OnCallPerson, locale string) error {
	m.calls = append(m.calls, channelID+":"+teamName)
	return m.err
}

func TestPublishOnCallSkipsTeamWithoutChannels(t *testing.T) {
	store := &publishMockStore{team: db.Team{ID: uuid.New(), Name: "Platform"}}
	p := NewPublishOnCallProcessor(nil, store, "")
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)})
	require.NoError(t, err)
	require.Empty(t, store.announced)
}

func TestPublishOnCallPostsAndRecordsFingerprint(t *testing.T) {
	userID := uuid.New()
	chat := "chat-1"
	slack := "C123"
	store := &publishMockStore{
		team: db.Team{
			ID:             uuid.New(),
			Name:           "Platform",
			ExpressChatID:  &chat,
			SlackChannelID: &slack,
		},
		onCall: []db.OnCallUser{{UserID: userID, DisplayName: "Alice"}},
		users: map[uuid.UUID]db.User{
			userID: {ID: userID, DisplayName: "Alice", Locale: "en"},
		},
	}
	slackAnn := &mockAnnouncer{}
	expressAnn := &mockAnnouncer{}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return slackAnn, expressAnn, nil
	}

	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)})
	require.NoError(t, err)
	require.Equal(t, userID.String(), store.announced)
	require.Len(t, slackAnn.calls, 1)
	require.Len(t, expressAnn.calls, 1)
}

func TestPublishOnCallSoftFailsOneProvider(t *testing.T) {
	userID := uuid.New()
	chat := "chat-1"
	slack := "C123"
	store := &publishMockStore{
		team:   db.Team{ID: uuid.New(), Name: "Platform", ExpressChatID: &chat, SlackChannelID: &slack},
		onCall: []db.OnCallUser{{UserID: userID, DisplayName: "Alice"}},
		users:  map[uuid.UUID]db.User{userID: {ID: userID, DisplayName: "Alice"}},
	}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return &mockAnnouncer{err: context.Canceled}, &mockAnnouncer{}, nil
	}
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)})
	require.NoError(t, err)
	require.Equal(t, userID.String(), store.announced)
}

func TestPublishOnCallEmptyPayloadListsTeams(t *testing.T) {
	chat := "chat-1"
	team := db.Team{ID: uuid.New(), Name: "Platform", ExpressChatID: &chat}
	store := &publishMockStore{teams: []db.Team{team}, team: team}
	p := NewPublishOnCallProcessor(nil, store, "")
	called := 0
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		called++
		return nil, &mockAnnouncer{}, nil
	}
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{}`)})
	require.NoError(t, err)
	require.Equal(t, 1, called)
}

func TestPublishOnCallInvalidPayload(t *testing.T) {
	p := NewPublishOnCallProcessor(nil, &publishMockStore{}, "")
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{`)}))
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"bad"}`)}))
}

func TestPublishOnCallListError(t *testing.T) {
	store := &publishMockStore{listErr: context.Canceled}
	p := NewPublishOnCallProcessor(nil, store, "")
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{}`)}))
}

type rotationMockStore struct {
	teams      []db.Team
	onCall     []db.OnCallUser
	pending    bool
	enqueued   []uuid.UUID
	listErr    error
	onCallErr  error
	pendingErr error
	enqueueErr error
}

func (m *rotationMockStore) ListTeamsWithChatChannels(context.Context) ([]db.Team, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.teams, nil
}
func (m *rotationMockStore) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	if m.onCallErr != nil {
		return nil, m.onCallErr
	}
	return m.onCall, nil
}
func (m *rotationMockStore) HasPendingPublishOnCall(context.Context, uuid.UUID) (bool, error) {
	if m.pendingErr != nil {
		return false, m.pendingErr
	}
	return m.pending, nil
}
func (m *rotationMockStore) EnqueuePublishOnCall(_ context.Context, teamID uuid.UUID) error {
	if m.enqueueErr != nil {
		return m.enqueueErr
	}
	m.enqueued = append(m.enqueued, teamID)
	return nil
}

func TestEnqueueOnCallRotationPublishesOnChange(t *testing.T) {
	userID := uuid.New()
	chat := "chat-1"
	teamID := uuid.New()
	store := &rotationMockStore{
		teams:  []db.Team{{ID: teamID, ExpressChatID: &chat}},
		onCall: []db.OnCallUser{{UserID: userID}},
	}
	require.NoError(t, EnqueueOnCallRotationPublishes(context.Background(), store, time.Now()))
	require.Equal(t, []uuid.UUID{teamID}, store.enqueued)
}

func TestEnqueueOnCallRotationPublishesSkipsUnchanged(t *testing.T) {
	userID := uuid.New()
	fp := userID.String()
	chat := "chat-1"
	store := &rotationMockStore{
		teams:  []db.Team{{ID: uuid.New(), ExpressChatID: &chat, OnCallAnnouncedUserIDs: &fp}},
		onCall: []db.OnCallUser{{UserID: userID}},
	}
	require.NoError(t, EnqueueOnCallRotationPublishes(context.Background(), store, time.Now()))
	require.Empty(t, store.enqueued)
}

func TestPublishOnCallBothProvidersFail(t *testing.T) {
	userID := uuid.New()
	chat := "chat-1"
	slack := "C123"
	store := &publishMockStore{
		team:   db.Team{ID: uuid.New(), Name: "Platform", ExpressChatID: &chat, SlackChannelID: &slack},
		onCall: []db.OnCallUser{{UserID: userID, DisplayName: "Alice"}},
		users:  map[uuid.UUID]db.User{userID: {ID: userID, DisplayName: "Alice"}},
	}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return &mockAnnouncer{err: context.Canceled}, &mockAnnouncer{err: context.DeadlineExceeded}, nil
	}
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)})
	require.Error(t, err)
	require.Empty(t, store.announced)
}

func TestPublishOnCallGetTeamError(t *testing.T) {
	store := &publishMockStore{getErr: context.Canceled}
	p := NewPublishOnCallProcessor(nil, store, "")
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + uuid.New().String() + `"}`)}))
}

func TestPublishOnCallUsersError(t *testing.T) {
	chat := "chat-1"
	store := &publishMockStore{team: db.Team{ID: uuid.New(), Name: "Platform", ExpressChatID: &chat}, onCallErr: context.Canceled}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return nil, &mockAnnouncer{}, nil
	}
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)}))
}

func TestPublishOnCallUserLookupErrorStillPublishes(t *testing.T) {
	userID := uuid.New()
	chat := "chat-1"
	store := &publishMockStore{
		team:    db.Team{ID: uuid.New(), Name: "Platform", ExpressChatID: &chat},
		onCall:  []db.OnCallUser{{UserID: userID, DisplayName: "Alice"}},
		userErr: context.Canceled,
	}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return nil, &mockAnnouncer{}, nil
	}
	require.NoError(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)}))
	require.Equal(t, userID.String(), store.announced)
}

func TestPublishOnCallExpressOnlyFailure(t *testing.T) {
	chat := "chat-1"
	store := &publishMockStore{team: db.Team{ID: uuid.New(), Name: "Platform", ExpressChatID: &chat}}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return nil, &mockAnnouncer{err: context.Canceled}, nil
	}
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)}))
}

func TestPublishOnCallMissingAnnouncers(t *testing.T) {
	chat := "chat-1"
	store := &publishMockStore{team: db.Team{ID: uuid.New(), Name: "Platform", ExpressChatID: &chat}}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return nil, nil, nil
	}
	require.NoError(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)}))
	require.Empty(t, store.announced)
}

func TestPublishOnCallSlackOnlyFailure(t *testing.T) {
	slack := "C123"
	store := &publishMockStore{team: db.Team{ID: uuid.New(), Name: "Platform", SlackChannelID: &slack}}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return &mockAnnouncer{err: context.Canceled}, nil, nil
	}
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)}))
}

func TestEnqueueOnCallRotationPublishesSkipsEmpty(t *testing.T) {
	chat := "chat-1"
	store := &rotationMockStore{teams: []db.Team{{ID: uuid.New(), ExpressChatID: &chat}}}
	require.NoError(t, EnqueueOnCallRotationPublishes(context.Background(), store, time.Now()))
	require.Empty(t, store.enqueued)
}

func TestEnqueueOnCallRotationPublishesSkipsPending(t *testing.T) {
	userID := uuid.New()
	chat := "chat-1"
	store := &rotationMockStore{
		teams:   []db.Team{{ID: uuid.New(), ExpressChatID: &chat}},
		onCall:  []db.OnCallUser{{UserID: userID}},
		pending: true,
	}
	require.NoError(t, EnqueueOnCallRotationPublishes(context.Background(), store, time.Now()))
	require.Empty(t, store.enqueued)
}

func TestEnqueueOnCallRotationPublishesListError(t *testing.T) {
	store := &rotationMockStore{listErr: context.Canceled}
	require.Error(t, EnqueueOnCallRotationPublishes(context.Background(), store, time.Now()))
}

func TestEnqueueOnCallRotationPublishesOnCallError(t *testing.T) {
	chat := "chat-1"
	store := &rotationMockStore{teams: []db.Team{{ID: uuid.New(), ExpressChatID: &chat}}, onCallErr: context.Canceled}
	require.Error(t, EnqueueOnCallRotationPublishes(context.Background(), store, time.Now()))
}

func TestEnqueueOnCallRotationPublishesPendingError(t *testing.T) {
	userID := uuid.New()
	chat := "chat-1"
	store := &rotationMockStore{
		teams:      []db.Team{{ID: uuid.New(), ExpressChatID: &chat}},
		onCall:     []db.OnCallUser{{UserID: userID}},
		pendingErr: context.Canceled,
	}
	require.Error(t, EnqueueOnCallRotationPublishes(context.Background(), store, time.Now()))
}

func TestEnqueueOnCallRotationPublishesEnqueueError(t *testing.T) {
	userID := uuid.New()
	chat := "chat-1"
	store := &rotationMockStore{
		teams:      []db.Team{{ID: uuid.New(), ExpressChatID: &chat}},
		onCall:     []db.OnCallUser{{UserID: userID}},
		enqueueErr: context.Canceled,
	}
	require.Error(t, EnqueueOnCallRotationPublishes(context.Background(), store, time.Now()))
}

func TestPublishOnCallAnnouncersError(t *testing.T) {
	chat := "chat-1"
	store := &publishMockStore{team: db.Team{ID: uuid.New(), Name: "Platform", ExpressChatID: &chat}}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return nil, nil, context.Canceled
	}
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)}))
}

func TestPublishOnCallSetAnnouncedError(t *testing.T) {
	userID := uuid.New()
	chat := "chat-1"
	store := &publishMockStore{
		team:   db.Team{ID: uuid.New(), Name: "Platform", ExpressChatID: &chat},
		onCall: []db.OnCallUser{{UserID: userID, DisplayName: "Alice"}},
		setErr: context.Canceled,
	}
	p := NewPublishOnCallProcessor(nil, store, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return nil, &mockAnnouncer{}, nil
	}
	require.Error(t, p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + store.team.ID.String() + `"}`)}))
}
