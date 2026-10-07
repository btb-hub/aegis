package integrations

import (
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/aegis/aegis/pkg/i18n"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIncidentBodyLimitsUnicodeAndIncludesLifecycleDetails(t *testing.T) {
	require.NoError(t, i18n.LoadMessages("../i18n/messages"))
	id := uuid.New()
	for _, kind := range []string{"created", "acknowledged", "resolved", "escalated"} {
		body := IncidentEventBody(IncidentChannelPost{Incident: IncidentRef{ID: id, Title: strings.Repeat("ж", 5000), Severity: "critical", URL: IncidentURL("https://aegis.local/", id.String())}, Kind: kind, TeamName: "Platform", ActorName: "Alice", Locale: "en", OccurredAt: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)})
		require.True(t, utf8.ValidString(body))
		require.Less(t, len([]rune(body)), 3000)
		require.Contains(t, body, id.String())
		require.Contains(t, body, "Platform")
		require.Contains(t, body, "critical")
		require.Contains(t, body, "2026-10-07T12:00:00Z")
		require.Contains(t, body, "https://aegis.local/incidents?incident_id=")
		if kind == "acknowledged" || kind == "resolved" {
			require.Contains(t, body, "Alice")
		}
	}
}

func TestRetryAfterUsesSecondsOrFutureDate(t *testing.T) {
	require.Equal(t, 30*time.Second, ParseRetryAfter("30"))
	require.Zero(t, ParseRetryAfter("-1"))
	require.Zero(t, ParseRetryAfter("garbage"))
	require.Zero(t, ParseRetryAfter(time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)))
	delay := ParseRetryAfter(time.Now().Add(time.Minute).UTC().Format(http.TimeFormat))
	require.InDelta(t, 60, delay.Seconds(), 2)
}
