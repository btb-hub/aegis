package oncall

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNextPublicationDST(t *testing.T) {
	for _, tc := range []struct{ now, zone, clock, want string }{
		{"2026-03-29T00:00:00Z", "Europe/Berlin", "02:30", "2026-03-29T01:00:00Z"},
		{"2026-10-25T00:00:00Z", "Europe/Berlin", "02:30", "2026-10-25T00:30:00Z"},
		{"2026-10-25T00:45:00Z", "Europe/Berlin", "02:30", "2026-10-26T01:30:00Z"},
		{"2026-10-05T00:00:00Z", "Europe/Moscow", "08:00", "2026-10-05T05:00:00Z"},
	} {
		now, _ := time.Parse(time.RFC3339, tc.now)
		got, err := NextPublication(now, tc.clock, tc.zone)
		require.NoError(t, err)
		require.Equal(t, tc.want, got.Format(time.RFC3339))
	}
}
func TestPublicationValidation(t *testing.T) {
	_, err := NextPublication(time.Now(), "25:00", "UTC")
	require.Error(t, err)
	_, err = NextPublication(time.Now(), "03:00", "Bad/Zone")
	require.Error(t, err)
}

func TestPublicationInvalidSettings(t *testing.T) {
	_, err := NextPublication(time.Now(), "25:00", "UTC")
	require.Error(t, err)
	_, err = NextPublication(time.Now(), "03:00", "Invalid/Zone")
	require.Error(t, err)
}
