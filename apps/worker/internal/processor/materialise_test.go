package processor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type materialiseMockStore struct {
	teamIDs []uuid.UUID
	calls   []uuid.UUID
}

func (m *materialiseMockStore) MaterialiseOnCallForTeam(ctx context.Context, teamID uuid.UUID) error {
	m.calls = append(m.calls, teamID)
	return nil
}

func (m *materialiseMockStore) ListTeamIDsWithSchedules(ctx context.Context) ([]uuid.UUID, error) {
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

func TestMaterialiseProcessorInvalidTeamID(t *testing.T) {
	p := NewMaterialiseProcessor(nil, &materialiseMockStore{})
	err := p.Handle(context.Background(), Job{Payload: json.RawMessage(`{"team_id":"bad"}`)})
	require.Error(t, err)
}
