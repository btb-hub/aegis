package oncall

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWhoAtWeeklyRotation(t *testing.T) {
	alice := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	bob := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	handoff, err := time.Parse("15:04", "09:00")
	require.NoError(t, err)

	layers := []WeeklyLayer{{
		Priority:       0,
		Timezone:       "UTC",
		HandoffWeekday: 1,
		HandoffTime:    handoff,
		Participants:   []uuid.UUID{alice, bob},
	}}

	// Monday 2026-06-29 10:00 UTC is in alice's week (handoff Mon 2026-06-29 09:00)
	at := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	user, source, err := WhoAt(at, layers, nil)
	require.NoError(t, err)
	require.Equal(t, alice, user)
	require.Equal(t, SourceRotation, source)

	// Following Monday bob's week
	at = time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	user, _, err = WhoAt(at, layers, nil)
	require.NoError(t, err)
	require.Equal(t, bob, user)
}

func TestOverrideWinsOverRotation(t *testing.T) {
	alice := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	bob := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	cover := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	handoff, err := time.Parse("15:04", "09:00")
	require.NoError(t, err)

	layers := []WeeklyLayer{{
		Priority:       0,
		Timezone:       "UTC",
		HandoffWeekday: 1,
		HandoffTime:    handoff,
		Participants:   []uuid.UUID{alice, bob},
	}}
	overrides := []Override{{
		UserID:  cover,
		StartAt: time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC),
		EndAt:   time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC),
	}}

	at := time.Date(2026, 6, 29, 15, 0, 0, 0, time.UTC)
	user, source, err := WhoAt(at, layers, overrides)
	require.NoError(t, err)
	require.Equal(t, cover, user)
	require.Equal(t, SourceOverride, source)
}

func TestDSTSpringForwardAmericaNewYork(t *testing.T) {
	alice := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	bob := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	handoff, err := time.Parse("15:04", "09:00")
	require.NoError(t, err)

	layers := []WeeklyLayer{{
		Priority:       0,
		Timezone:       "America/New_York",
		HandoffWeekday: 0,
		HandoffTime:    handoff,
		Participants:   []uuid.UUID{alice, bob},
	}}

	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	// Sunday after spring-forward 2026-03-08; 10:00 local is unambiguous
	at := time.Date(2026, 3, 8, 10, 0, 0, 0, loc)
	user, source, err := WhoAt(at.UTC(), layers, nil)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user)
	require.Equal(t, SourceRotation, source)
}

func TestDSTFallBackAmericaNewYork(t *testing.T) {
	alice := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	bob := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	handoff, err := time.Parse("15:04", "09:00")
	require.NoError(t, err)

	layers := []WeeklyLayer{{
		Priority:       0,
		Timezone:       "America/New_York",
		HandoffWeekday: 0,
		HandoffTime:    handoff,
		Participants:   []uuid.UUID{alice, bob},
	}}

	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	// First 01:30 occurrence on fall-back Sunday 2026-11-01
	at := time.Date(2026, 11, 1, 1, 30, 0, 0, loc)
	user, _, err := WhoAt(at.UTC(), layers, nil)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user)
}

func TestMaterialiseAppliesOverride(t *testing.T) {
	alice := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	cover := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	handoff, err := time.Parse("15:04", "09:00")
	require.NoError(t, err)

	layers := []WeeklyLayer{{
		Priority:       0,
		Timezone:       "UTC",
		HandoffWeekday: 1,
		HandoffTime:    handoff,
		Participants:   []uuid.UUID{alice},
	}}
	from := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	overrides := []Override{{
		UserID:  cover,
		StartAt: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		EndAt:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}}

	slots, err := Materialise(layers, overrides, from, to)
	require.NoError(t, err)
	require.NotEmpty(t, slots)

	var overrideFound bool
	for _, slot := range slots {
		if slot.Source == SourceOverride && slot.UserID == cover {
			overrideFound = true
		}
	}
	require.True(t, overrideFound)
}

func TestHigherPriorityLayerWins(t *testing.T) {
	primary := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	backup := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	handoff, err := time.Parse("15:04", "09:00")
	require.NoError(t, err)

	layers := []WeeklyLayer{
		{Priority: 1, Timezone: "UTC", HandoffWeekday: 1, HandoffTime: handoff, Participants: []uuid.UUID{backup}},
		{Priority: 0, Timezone: "UTC", HandoffWeekday: 1, HandoffTime: handoff, Participants: []uuid.UUID{primary}},
	}
	at := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	user, _, err := WhoAt(at, layers, nil)
	require.NoError(t, err)
	require.Equal(t, primary, user)
}
