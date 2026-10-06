package integrations

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMSKShiftDatesAndNames(t *testing.T) {
	start := time.Date(2026, 10, 5, 20, 0, 0, 0, time.UTC)
	end := start.Add(12 * time.Hour)
	require.Equal(t, "05 Oct 23:00 – 06 Oct 11:00 MSK", ShiftTimeMSK(start, end, "en"))
	require.Equal(t, "05.10 23:00 – 06.10 11:00 МСК", ShiftTimeMSK(start, end, "ru"))
	require.Empty(t, ShiftTimeMSK(time.Time{}, end, "en"))
	require.Contains(t, OnCallName(OnCallPerson{DisplayName: "Engineer", StartAt: start, EndAt: end}, "en"), "Engineer (")
	require.Equal(t, "Engineer", OnCallName(OnCallPerson{DisplayName: "Engineer"}, "en"))
}
func TestRetryableErrors(t *testing.T) {
	require.False(t, RetryableError(nil))
	require.True(t, RetryableError(errors.New("network")))
	require.True(t, RetryableError(errors.Join(&HTTPError{Status: 403}, &HTTPError{Status: 429})))
	require.False(t, RetryableError(errors.Join(&HTTPError{Status: 403}, &HTTPError{Status: 404})))
	require.True(t, RetryableError(fmt.Errorf("providers: %w", errors.Join(&HTTPError{Status: 403}, &HTTPError{Status: 503}))))
}
