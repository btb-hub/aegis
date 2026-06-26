package locale

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	require.NoError(t, Validate("en"))
	require.NoError(t, Validate("ru"))
	require.Error(t, Validate("de"))
}

func TestNormalize(t *testing.T) {
	require.Equal(t, "en", Normalize(""))
	require.Equal(t, "ru", Normalize("ru"))
	require.Equal(t, "en", Normalize("fr"))
}
