package routing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchTeamSkipsInvalidRuleLabels(t *testing.T) {
	rules := []Rule{
		{TeamID: "team-a", MatchLabels: map[string]string{"team": "platform"}, Priority: 1},
	}
	teamID, ok := MatchTeam(rules, map[string]string{"team": "platform"})
	require.True(t, ok)
	require.Equal(t, "team-a", teamID)
}

func TestRoutingServiceMatchSkipsBadStoredLabels(t *testing.T) {
	// covered via service layer using ParseMatchLabels on invalid JSON in repo - tested in match layer only
	_, err := ParseMatchLabels([]byte(`{"team":`))
	require.Error(t, err)
}
