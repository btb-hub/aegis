package processor

import (
	"context"
	"encoding/json"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAlertProcessorRecoveryBetweenReadAndLink(t *testing.T) {
	id := uuid.New()
	store := &alertMockStore{alert: db.Alert{ID: id, Status: "firing", Labels: []byte(`{"team":"platform"}`)}, manualErr: db.ErrAlertNotFiring}
	payload, _ := json.Marshal(map[string]string{"alert_id": id.String()})
	require.NoError(t, NewAlertProcessor(nil, store, time.Hour, time.Minute).Handle(context.Background(), Job{Payload: payload}))
}

type anomalyRoutingStore struct {
	alertMockStore
	rules []db.RoutingRule
}

func (s *anomalyRoutingStore) ListRoutingRules(context.Context) ([]db.RoutingRule, error) {
	return s.rules, nil
}
func TestAnomalyRoutingStaysInSourceWorkspace(t *testing.T) {
	workspace := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	other := uuid.New()
	target := uuid.New()
	labels := map[string]string{"team": "platform", "aegis_module": "anomaly-detector", "aegis_workspace_id": workspace.String()}
	store := &anomalyRoutingStore{rules: []db.RoutingRule{{WorkspaceID: other, TeamID: uuid.New(), Priority: 0, MatchLabels: []byte(`{"team":"platform"}`)}, {WorkspaceID: workspace, TeamID: target, Priority: 1, MatchLabels: []byte(`{"team":"platform"}`), CrossWorkspace: true}}}
	processor := NewAlertProcessor(nil, store, time.Hour, time.Minute)
	got, err := processor.matchTeam(context.Background(), labels)
	require.NoError(t, err)
	require.Equal(t, target, got)
	store.rules = store.rules[:1]
	_, err = processor.matchTeam(context.Background(), labels)
	require.ErrorContains(t, err, "no routing rule")
	delete(labels, "aegis_module")
	_, err = processor.matchTeam(context.Background(), labels)
	require.NoError(t, err)
}
