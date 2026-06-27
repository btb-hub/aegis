package routing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchTeamByPriority(t *testing.T) {
	rules := []Rule{
		{TeamID: "team-low", MatchLabels: map[string]string{"team": "platform"}, Priority: 1},
		{TeamID: "team-high", MatchLabels: map[string]string{"team": "platform"}, Priority: 10},
	}
	teamID, ok := MatchTeam(rules, map[string]string{"team": "platform", "alertname": "HighCPU"})
	require.True(t, ok)
	require.Equal(t, "team-high", teamID)
}

func TestMatchTeamRequiresAllLabels(t *testing.T) {
	rules := []Rule{
		{TeamID: "team-a", MatchLabels: map[string]string{"team": "platform", "env": "prod"}, Priority: 5},
	}
	_, ok := MatchTeam(rules, map[string]string{"team": "platform"})
	require.False(t, ok)
	teamID, ok := MatchTeam(rules, map[string]string{"team": "platform", "env": "prod"})
	require.True(t, ok)
	require.Equal(t, "team-a", teamID)
}

func TestMatchTeamEmptyRules(t *testing.T) {
	_, ok := MatchTeam(nil, map[string]string{"team": "platform"})
	require.False(t, ok)
}

func TestLabelsMatchEmptyRequired(t *testing.T) {
	require.False(t, labelsMatch(map[string]string{}, map[string]string{"a": "b"}))
}
