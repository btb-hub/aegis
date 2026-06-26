package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanMutate(t *testing.T) {
	require.True(t, CanMutate(RoleAdmin))
	require.True(t, CanMutate(RoleMember))
	require.False(t, CanMutate(RoleViewer))
}

func TestCanAdminister(t *testing.T) {
	require.True(t, CanAdminister(RoleAdmin))
	require.False(t, CanAdminister(RoleMember))
}

func TestParse(t *testing.T) {
	role, err := Parse("member")
	require.NoError(t, err)
	require.Equal(t, RoleMember, role)

	_, err = Parse("superuser")
	require.Error(t, err)
}
