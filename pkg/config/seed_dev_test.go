package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeedDevAllowedLocalhost(t *testing.T) {
	require.NoError(t, SeedDevAllowed("http://localhost:3000"))
	require.NoError(t, SeedDevAllowed("http://127.0.0.1:8080"))
}

func TestSeedDevAllowedWithForce(t *testing.T) {
	t.Setenv("SEED_DEV", "1")
	require.NoError(t, SeedDevAllowed("https://aegis.example.com"))
}

func TestSeedDevAllowedRejectsProduction(t *testing.T) {
	t.Setenv("SEED_DEV", "")
	require.Error(t, SeedDevAllowed("https://aegis.example.com"))
}

func TestSeedDevAllowedRequiresPublicURL(t *testing.T) {
	require.Error(t, SeedDevAllowed(""))
}
