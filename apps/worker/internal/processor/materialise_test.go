package processor

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type materialiseMockStore struct {
	teamIDs        []uuid.UUID
	calls          []uuid.UUID
	listErr        error
	materialiseErr error
}

func (m *materialiseMockStore) MaterialiseOnCallForTeam(ctx context.Context, teamID uuid.UUID) error {
	if m.materialiseErr != nil {
		return m.materialiseErr
	}
	m.calls = append(m.calls, teamID)
	return nil
}

func (m *materialiseMockStore) ListTeamIDsWithSchedules(ctx context.Context) ([]uuid.UUID, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.teamIDs, nil
}

func TestMaterialiseProcessorSingleTeam(t *testing.T) {
	store := &materialiseMockStore{}
	p := NewMaterialiseProcessor(nil, store)
	teamID := uuid.New()
	err := p.Handle(context.Background(), Job{
		ID:      "j1",
		Kind:    "materialise_oncall",
		Payload: json.RawMessage(`{"team_id":"` + teamID.String() + `"}`),
	})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{teamID}, store.calls)
}

func TestMaterialiseProcessorNightly(t *testing.T) {
	store := &materialiseMockStore{teamIDs: []uuid.UUID{uuid.New(), uuid.New()}}
	p := NewMaterialiseProcessor(nil, store)
	err := p.Handle(context.Background(), Job{ID: "j1", Kind: "materialise_oncall", Payload: json.RawMessage(`{}`)})
	require.NoError(t, err)
	require.Len(t, store.calls, 2)
}

func TestMaterialiseProcessorInvalidPayload(t *testing.T) {
	p := NewMaterialiseProcessor(nil, &materialiseMockStore{})
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{`)})
	require.Error(t, err)
}

func TestMaterialiseProcessorInvalidTeamID(t *testing.T) {
	p := NewMaterialiseProcessor(nil, &materialiseMockStore{})
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"bad"}`)})
	require.Error(t, err)
}

func TestMaterialiseProcessorListTeamsError(t *testing.T) {
	store := &materialiseMockStore{listErr: errors.New("list failed")}
	p := NewMaterialiseProcessor(nil, store)
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{}`)})
	require.Error(t, err)
}

func TestMaterialiseProcessorMaterialiseError(t *testing.T) {
	teamID := uuid.New()
	store := &materialiseMockStore{materialiseErr: errors.New("materialise failed")}
	p := NewMaterialiseProcessor(nil, store)
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"` + teamID.String() + `"}`)})
	require.Error(t, err)
}

func TestMaterialiseProcessorNightlyMaterialiseError(t *testing.T) {
	store := &materialiseMockStore{
		teamIDs:        []uuid.UUID{uuid.New()},
		materialiseErr: errors.New("materialise failed"),
	}
	p := NewMaterialiseProcessor(nil, store)
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{}`)})
	require.Error(t, err)
}
