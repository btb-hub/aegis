package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFromJSONSuccess(t *testing.T) {
	provider, err := NewFromJSON([]byte(`{"bot_token":"xoxb-test","signing_secret":"secret"}`), "http://localhost:8080")
	require.NoError(t, err)
	require.Equal(t, "slack", provider.Kind())
}

func TestParseInteractiveAckUnsupportedType(t *testing.T) {
	form := url.Values{"payload": {`{"type":"view_submission","actions":[]}`}}
	_, _, err := ParseInteractiveAck(form)
	require.Error(t, err)
}

func TestVerifySignatureMissingHeaders(t *testing.T) {
	require.Error(t, VerifySignature("", "", "", nil))
}

func TestNewFromJSONInvalidJSON(t *testing.T) {
	_, err := NewFromJSON([]byte(`{`), "http://localhost:8080")
	require.Error(t, err)
}

func TestNewFromJSONIncompleteConfig(t *testing.T) {
	_, err := NewFromJSON([]byte(`{"bot_token":"xoxb-test"}`), "http://localhost:8080")
	require.Error(t, err)
}

func TestVerifySignatureTooOld(t *testing.T) {
	secret := "secret"
	body := []byte(`payload={}`)
	ts := "1"
	base := "v0:" + ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(base))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))
	require.Error(t, VerifySignature(secret, ts, sig, body))
}

func TestParseInteractiveAckMissingAction(t *testing.T) {
	form := url.Values{"payload": {`{"type":"block_actions","user":{"id":"U1"},"actions":[]}`}}
	_, _, err := ParseInteractiveAck(form)
	require.Error(t, err)
}
