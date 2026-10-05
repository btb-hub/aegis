package processor

import (
	"context"
	"errors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type publicationFixture struct {
	publishMockStore
	fail           string
	attempts       []db.PublicationAttempt
	needs, pending bool
	enqueued       int
	refs           []string
}

func (f *publicationFixture) RunOnCallDelivery(_ context.Context, a db.PublicationAttempt, send func() error) error {
	f.attempts = append(f.attempts, a)
	return send()
}
func (f *publicationFixture) NeedsOnCallPublication(context.Context, db.PublicationAttempt, time.Time) (bool, error) {
	if f.fail == "needs" {
		return false, errors.New("db")
	}
	return f.needs, nil
}
func (f *publicationFixture) GetIntegrationByKind(_ context.Context, _ string) (db.Integration, error) {
	if f.fail == "global" {
		return db.Integration{}, errors.New("db")
	}
	if f.fail == "missing-global" {
		return db.Integration{}, pgx.ErrNoRows
	}
	return f.integration, nil
}
func (f *publicationFixture) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	if f.fail == "workspace" {
		return uuid.Nil, errors.New("db")
	}
	return uuid.New(), nil
}
func (f *publicationFixture) GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error) {
	if f.fail == "slot" {
		return db.Integration{}, errors.New("db")
	}
	return db.Integration{}, pgx.ErrNoRows
}
func (f *publicationFixture) HasPendingPublishOnCall(context.Context, uuid.UUID) (bool, error) {
	if f.fail == "pending" {
		return false, errors.New("db")
	}
	return f.pending, nil
}
func (f *publicationFixture) EnqueuePublishOnCall(context.Context, uuid.UUID) error {
	f.enqueued++
	if f.fail == "enqueue" {
		return errors.New("db")
	}
	return nil
}
func (f *publicationFixture) RecordOnCallExpressRef(_ context.Context, _, _ uuid.UUID, _, key, _ string) error {
	f.refs = append(f.refs, key)
	if f.fail == "ref" {
		return errors.New("db")
	}
	return nil
}

type trackedAnnouncer struct{ mockAnnouncer }

func (a *trackedAnnouncer) AnnounceOnCallWithRef(context.Context, string, string, string, []integrations.OnCallPerson, string) (string, error) {
	return "message-ref", a.err
}
func TestPublicationPerDestinationStateAndConfiguration(t *testing.T) {
	team := db.Team{ID: uuid.New(), Name: "Ops", SlackChannelID: strPtr("slack-channel")}
	f := &publicationFixture{publishMockStore: publishMockStore{team: team, teams: []db.Team{team}, onCall: []db.OnCallUser{{UserID: uuid.New(), StartAt: time.Now(), EndAt: time.Now().Add(time.Hour)}}, integration: db.Integration{ID: uuid.New(), Kind: "express", Enabled: true, Config: []byte(`{"host":"https://express.example.test","bot_id":"bot","secret_key":"secret","oncall_group_chat_id":"oncall"}`)}}, needs: true}
	p := NewPublishOnCallProcessor(nil, f, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return &mockAnnouncer{}, &trackedAnnouncer{}, nil
	}
	require.NoError(t, p.Handle(t.Context(), Job{ID: "manual", Payload: []byte(`{}`)}))
	require.Len(t, f.attempts, 2)
	require.Len(t, f.refs, 1)
	require.Equal(t, f.attempts[1].PublicationKey, f.refs[0])
	first := f.attempts[1]
	f.integration.Config = []byte(`{"host":"https://express.example.test","bot_id":"bot","secret_key":"changed","oncall_group_chat_id":"oncall"}`)
	require.NoError(t, p.publishTeam(t.Context(), team, "rotation"))
	require.NotEqual(t, first.ConfigVersion, f.attempts[3].ConfigVersion)
	for _, op := range []string{"global", "workspace", "slot"} {
		f.fail = op
		_, err := publicationConfigVersion(t.Context(), f, team.ID, "slack")
		require.Error(t, err)
	}
	f.fail = "missing-global"
	_, err := publicationConfigVersion(t.Context(), f, team.ID, "express")
	require.NoError(t, err)
	f.fail = "ref"
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return nil, &trackedAnnouncer{}, nil
	}
	require.Error(t, p.publishTeam(t.Context(), team, "manual:ref"))
	f.fail = ""
	f.needs = false
	require.NoError(t, EnqueueOnCallRotationPublishes(t.Context(), f, time.Now()))
	require.Zero(t, f.enqueued)
	f.needs = true
	require.NoError(t, EnqueueOnCallRotationPublishes(t.Context(), f, time.Now()))
	require.Equal(t, 1, f.enqueued)
	f.pending = true
	require.NoError(t, EnqueueOnCallRotationPublishes(t.Context(), f, time.Now()))
	require.Equal(t, 1, f.enqueued)
	f.pending = false
	for _, op := range []string{"needs", "pending", "enqueue", "workspace", "slot"} {
		f.fail = op
		require.Error(t, EnqueueOnCallRotationPublishes(t.Context(), f, time.Now()), op)
	}
}

type leasedFixture struct {
	mockStore
	started, stopped, finished bool
	result                     error
}

func (f *leasedFixture) BeginJob(ctx context.Context, _ Job) (context.Context, func()) {
	f.started = true
	return ctx, func() { f.stopped = true }
}
func (f *leasedFixture) FinishClaimedJob(_ context.Context, _ Job, result error) error {
	f.finished = true
	f.result = result
	return nil
}

type errorHandler struct{ err error }

func (h errorHandler) Handle(context.Context, Job) error { return h.err }
func TestWorkerLeaseLifecycle(t *testing.T) {
	f := &leasedFixture{mockStore: mockStore{claim: true, job: Job{ID: "job", Kind: "sync_jira"}}}
	w := NewWorker(nil, f, nil, nil, nil, nil, nil, nil)
	w.Register("sync_jira", errorHandler{err: errors.New("network")})
	require.NoError(t, w.RunOnce(t.Context()))
	require.True(t, f.started && f.stopped && f.finished)
	require.Error(t, f.result)
}

type atomicPublicationFixture struct{ publicationFixture }

func (f *atomicPublicationFixture) RunOnCallDeliveryWithRef(_ context.Context, a db.PublicationAttempt, _ uuid.UUID, send func() (string, error)) error {
	ref, err := send()
	f.attempts = append(f.attempts, a)
	if ref != "" {
		f.refs = append(f.refs, a.PublicationKey)
	}
	return err
}
func TestPublicationUsesAtomicReferenceStore(t *testing.T) {
	team := db.Team{ID: uuid.New(), Name: "Ops"}
	f := &atomicPublicationFixture{publicationFixture: publicationFixture{publishMockStore: publishMockStore{team: team, integration: db.Integration{ID: uuid.New(), Enabled: true, Config: []byte(`{"host":"https://express.example.test","bot_id":"bot","secret_key":"secret","oncall_group_chat_id":"oncall"}`)}}}}
	p := NewPublishOnCallProcessor(nil, f, "")
	p.announcers = func(context.Context, uuid.UUID) (onCallAnnouncer, onCallAnnouncer, error) {
		return nil, &trackedAnnouncer{}, nil
	}
	require.NoError(t, p.publishTeam(t.Context(), team, "rotation"))
	require.Len(t, f.refs, 1)
	require.Equal(t, f.refs[0], f.attempts[0].PublicationKey)
}
