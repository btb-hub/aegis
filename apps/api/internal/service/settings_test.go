package service

import (
	"context"
	"errors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type settingsFixture struct {
	value db.PublicationSettings
	err   error
}

func (f *settingsFixture) GetPublicationSettings(context.Context) (db.PublicationSettings, error) {
	return f.value, f.err
}
func (f *settingsFixture) UpdatePublicationSettings(_ context.Context, clock, zone string, _ time.Time) (db.PublicationSettings, error) {
	f.value.Time = clock
	f.value.Timezone = zone
	return f.value, f.err
}
func TestSettingsValidation(t *testing.T) {
	f := &settingsFixture{value: db.PublicationSettings{Time: "03:00", Timezone: "UTC"}}
	s := NewSettingsService(f)
	value, err := s.Get(t.Context())
	require.NoError(t, err)
	require.True(t, value.NextRunAt.After(time.Now()))
	value, err = s.Update(t.Context(), "08:00", "Europe/Moscow")
	require.NoError(t, err)
	require.Equal(t, "Europe/Moscow", value.Timezone)
	_, err = s.Update(t.Context(), "invalid", "UTC")
	require.Error(t, err)
	_, err = s.Update(t.Context(), "08:00", "Invalid/Zone")
	require.Error(t, err)
	f.err = errors.New("db")
	_, err = s.Get(t.Context())
	require.Error(t, err)
	_, err = s.Update(t.Context(), "08:00", "UTC")
	require.Error(t, err)
}
