package db

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDevSeedUsersIncludesProviders(t *testing.T) {
	seeds := DevSeedUsers()
	require.Len(t, seeds, 4)

	providers := map[string]SeedDevUser{}
	for _, seed := range seeds {
		providers[seed.Provider] = seed
	}
	require.NotEmpty(t, providers["google"].AvatarURL)
	require.Equal(t, "U0SEEDBOB", providers["slack"].SlackUserID)
	require.NotEmpty(t, providers["slack"].AvatarURL)
	require.Equal(t, uuid.MustParse("00000000-0000-4000-8000-000000000003"), providers["express"].ExpressUserHuid)
	require.NotEmpty(t, providers["express"].AvatarURL)
	require.Equal(t, "admin", providers["dev"].Role)
}

func TestOptionalString(t *testing.T) {
	require.Nil(t, optionalString(""))
	require.Equal(t, "x", *optionalString("x"))
}
