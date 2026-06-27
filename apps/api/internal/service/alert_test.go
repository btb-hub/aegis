package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockAlertRepo struct {
	last db.CreateAlertJobInput
	id   uuid.UUID
}

func (m *mockAlertRepo) CreateAlertAndJob(ctx context.Context, input db.CreateAlertJobInput) (db.CreateAlertJobResult, error) {
	m.last = input
	if m.id == uuid.Nil {
		m.id = uuid.New()
	}
	return db.CreateAlertJobResult{AlertID: m.id, JobID: uuid.New()}, nil
}

func TestAlertIngestSuccess(t *testing.T) {
	repo := &mockAlertRepo{}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	raw := json.RawMessage(`{"status":"firing","labels":{"alertname":"HighCPU"},"annotations":{"summary":"CPU"}}`)

	id, err := svc.Ingest(context.Background(), "secret", raw)
	require.NoError(t, err)
	require.Equal(t, repo.id, id)
	require.Equal(t, "process_alert", repo.last.JobKind)
	require.Equal(t, "CPU", repo.last.Title)
}

func TestAlertIngestBadSecret(t *testing.T) {
	svc := NewAlertService("secret", []string{"alertname", "team"}, &mockAlertRepo{})
	_, err := svc.Ingest(context.Background(), "wrong", json.RawMessage(`{}`))
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "INVALID_WEBHOOK_SECRET", appErr.Code)
}

func TestAlertIngestInvalidPayload(t *testing.T) {
	svc := NewAlertService("secret", []string{"alertname", "team"}, &mockAlertRepo{})
	_, err := svc.Ingest(context.Background(), "secret", json.RawMessage(`{"status":"nope"}`))
	appErr, ok := err.(*apperrors.Error)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
}

func TestAlertIngestRepoFailure(t *testing.T) {
	repo := &failAlertRepo{}
	svc := NewAlertService("secret", []string{"alertname", "team"}, repo)
	_, err := svc.Ingest(context.Background(), "secret", json.RawMessage(`{"status":"firing","labels":{"alertname":"X"}}`))
	require.Error(t, err)
}

type failAlertRepo struct{}

func (f *failAlertRepo) CreateAlertAndJob(ctx context.Context, input db.CreateAlertJobInput) (db.CreateAlertJobResult, error) {
	return db.CreateAlertJobResult{}, errors.New("db down")
}
