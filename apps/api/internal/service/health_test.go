package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthLive(t *testing.T) {
	svc := NewHealthService(nil)
	require.True(t, svc.Live())
}

func TestHealthReadyNilStore(t *testing.T) {
	svc := NewHealthService(nil)
	require.NoError(t, svc.Ready(context.Background()))
}
