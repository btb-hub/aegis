package processor

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAlertProcessorHandle(t *testing.T) {
	p := NewAlertProcessor(nil)
	err := p.Handle(context.Background(), Job{
		ID:      "job-1",
		Kind:    "process_alert",
		Payload: json.RawMessage(`{"alert_id":"abc"}`),
	})
	require.NoError(t, err)
}

func TestAlertProcessorInvalidPayload(t *testing.T) {
	p := NewAlertProcessor(nil)
	err := p.Handle(context.Background(), Job{ID: "1", Payload: json.RawMessage(`{`)})
	require.Error(t, err)
}

func TestKindFor(t *testing.T) {
	require.Equal(t, "process_alert", KindFor(Job{Kind: "process_alert"}))
}

func noopMaterialise() *MaterialiseProcessor {
	return NewMaterialiseProcessor(nil, materialiseStoreStub{})
}

type materialiseStoreStub struct{}

func (materialiseStoreStub) MaterialiseOnCallForTeam(ctx context.Context, teamID uuid.UUID) error {
	return nil
}
func (materialiseStoreStub) ListTeamIDsWithSchedules(ctx context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

func TestWorkerNoJob(t *testing.T) {
	w := NewWorker(nil, &mockStore{claim: false}, NewAlertProcessor(nil), noopMaterialise())
	err := w.RunOnce(context.Background())
	require.NoError(t, err)
}

type mockStore struct {
	claim bool
	job   Job
}

func (m *mockStore) ClaimNextJob(ctx context.Context) (bool, Job, error) {
	if !m.claim {
		return false, Job{}, nil
	}
	return true, m.job, nil
}
func (m *mockStore) CompleteJob(ctx context.Context, id string) error { return nil }
func (m *mockStore) FailJob(ctx context.Context, id, message string) error {
	return errors.New("fail")
}

func TestWorkerProcessesJob(t *testing.T) {
	store := &mockStore{
		claim: true,
		job: Job{ID: "j1", Kind: "process_alert", Payload: json.RawMessage(`{"alert_id":"x"}`)},
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil), noopMaterialise())
	require.NoError(t, w.RunOnce(context.Background()))
}

func TestWorkerProcessesMaterialiseJob(t *testing.T) {
	teamID := uuid.New()
	store := &mockStore{
		claim: true,
		job: Job{
			ID:      "j1",
			Kind:    "materialise_oncall",
			Payload: json.RawMessage(`{"team_id":"` + teamID.String() + `"}`),
		},
	}
	w := NewWorker(nil, store, NewAlertProcessor(nil), NewMaterialiseProcessor(nil, &materialiseMockStore{}))
	require.NoError(t, w.RunOnce(context.Background()))
}

type claimErrorStore struct {
	mockStore
}

func (m *claimErrorStore) ClaimNextJob(ctx context.Context) (bool, Job, error) {
	return false, Job{}, errors.New("claim failed")
}

func TestWorkerClaimError(t *testing.T) {
	w := NewWorker(nil, &claimErrorStore{}, NewAlertProcessor(nil), noopMaterialise())
	err := w.RunOnce(context.Background())
	require.Error(t, err)
}
