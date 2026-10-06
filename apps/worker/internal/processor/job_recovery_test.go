package processor

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

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

type retryFixture struct {
	mockStore
	cause      error
	retryError error
}

func (f *retryFixture) RetryJob(_ context.Context, _ string, cause error) error {
	f.cause = cause
	return f.retryError
}
func TestWorkerDurableJobRetry(t *testing.T) {
	network := errors.New("temporary transport failure")
	persistence := errors.New("retry persistence failed")
	for _, kind := range []string{"sync_jira", "express_reply", "notify_incident", "notify_handoff", "escalate_incident", "publish_oncall"} {
		t.Run(kind, func(t *testing.T) {
			f := &retryFixture{mockStore: mockStore{claim: true, job: Job{ID: "job", Kind: kind}}, retryError: persistence}
			w := NewWorker(nil, f, nil, nil, nil, nil, nil, nil)
			w.Register(kind, errorHandler{err: network})
			require.ErrorIs(t, w.RunOnce(t.Context()), persistence)
			require.ErrorIs(t, f.cause, network)
			f.retryError = nil
			require.NoError(t, w.RunOnce(t.Context()))
		})
	}
}
