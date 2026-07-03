package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserListSearchPattern(t *testing.T) {
	require.Equal(t, "", userListSearchPattern(""))
	require.Equal(t, "%alice%", userListSearchPattern("alice"))
	require.Equal(t, `%100\%%`, userListSearchPattern("100%"))
}

func TestNormalizeListUsersParams(t *testing.T) {
	params := normalizeListUsersParams(ListUsersParams{Limit: 0, Offset: -1})
	require.Equal(t, DefaultUserListLimit, params.Limit)
	require.Equal(t, 0, params.Offset)

	params = normalizeListUsersParams(ListUsersParams{Limit: 500})
	require.Equal(t, DefaultUserListLimit, params.Limit)
}
