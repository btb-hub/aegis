package processor

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type failStore struct {
	mockStore
}

func (f *failStore) FailJob(ctx context.Context, id, message string) error {
	return errors.New("stored failure")
}

func TestWorkerUnknownKind(t *testing.T) {
	store := &mockStore{
		claim: true,
		job:   Job{ID: "j1", Kind: "unknown", Payload: json.RawMessage(`{}`)},
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil))
	err := w.RunOnce(context.Background())
	require.Error(t, err)
}

func TestWorkerHandlerFailure(t *testing.T) {
	store := &failStore{
		mockStore: mockStore{
			claim: true,
			job:   Job{ID: "j1", Kind: "process_alert", Payload: json.RawMessage(`{`)},
		},
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil))
	err := w.RunOnce(context.Background())
	require.Error(t, err)
}
