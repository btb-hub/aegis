package express

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVerifyAuthorizationMissingHeader(t *testing.T) {
	require.Error(t, VerifyAuthorization("", "secret"))
}

func TestVerifyJWTExpired(t *testing.T) {
	token := signTestJWT(t, "secret", map[string]any{"exp": float64(time.Now().Add(-time.Hour).Unix())})
	_, err := verifyJWT(token, "secret")
	require.Error(t, err)
}

func TestParseCommandEventMissingHuid(t *testing.T) {
	_, err := ParseCommandEvent([]byte(`{"command":{"body":"/link x"}}`))
	require.Error(t, err)
}

func TestParseAckCommandMissingIncident(t *testing.T) {
	event, err := ParseCommandEvent([]byte(`{"command":{"body":"/help"},"from":{"user_huid":"6fafda2c-6505-57a5-a088-25ea5d1d0364"}}`))
	require.NoError(t, err)
	_, _, err = ParseAckCommand(event)
	require.Error(t, err)
}

func TestParseLinkCommandMissingCode(t *testing.T) {
	event, err := ParseCommandEvent([]byte(`{"command":{"body":"/link"},"from":{"user_huid":"6fafda2c-6505-57a5-a088-25ea5d1d0364"}}`))
	require.NoError(t, err)
	_, _, err = ParseLinkCommand(event)
	require.Error(t, err)
}
