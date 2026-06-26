package sessiontoken

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAndHash(t *testing.T) {
	token, hash, err := New()
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, Hash(token), hash)
	require.NotEqual(t, token, hash)
}

func TestNewRandFailure(t *testing.T) {
	orig := randRead
	t.Cleanup(func() { randRead = orig })
	randRead = func([]byte) (int, error) { return 0, errors.New("rand down") }
	_, _, err := New()
	require.Error(t, err)
}
