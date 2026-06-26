package oidc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeToken struct {
	extras map[string]any
}

func (f fakeToken) Extra(key string) any {
	return f.extras[key]
}

func TestUserInfoFromTokenFallback(t *testing.T) {
	info := UserInfoFromToken("google", fakeToken{})
	require.Equal(t, "unknown", info.Sub)
}

func TestUserInfoFromTokenWithIDToken(t *testing.T) {
	info := UserInfoFromToken("slack", fakeToken{extras: map[string]any{"id_token": "jwt"}})
	require.Equal(t, "sub-from-token", info.Sub)
}
