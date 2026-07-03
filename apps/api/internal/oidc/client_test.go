package oidc

import (
	"encoding/base64"
	"encoding/json"
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
	claims := map[string]any{"sub": "sub-from-token", "email": "user@example.com", "name": "Example User"}
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	token := "hdr." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	info := UserInfoFromToken("slack", fakeToken{extras: map[string]any{"id_token": token}})
	require.Equal(t, "sub-from-token", info.Sub)
	require.Equal(t, "user@example.com", info.Email)
}
