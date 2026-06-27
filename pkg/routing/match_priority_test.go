package routing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchTeamEqualPriorityUsesTeamID(t *testing.T) {
	rules := []Rule{
		{TeamID: "team-b", MatchLabels: map[string]string{"team": "platform"}, Priority: 5},
		{TeamID: "team-a", MatchLabels: map[string]string{"team": "platform"}, Priority: 5},
	}
	teamID, ok := MatchTeam(rules, map[string]string{"team": "platform"})
	require.True(t, ok)
	require.Equal(t, "team-a", teamID)
}

func TestParseMatchLabelsObject(t *testing.T) {
	labels, err := ParseMatchLabels([]byte(`{"team":"platform","env":"prod"}`))
	require.NoError(t, err)
	require.Equal(t, "platform", labels["team"])
}

func TestParseMatchLabelsNullJSON(t *testing.T) {
	labels, err := ParseMatchLabels([]byte(`null`))
	require.NoError(t, err)
	require.Empty(t, labels)
}
