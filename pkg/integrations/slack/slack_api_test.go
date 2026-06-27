package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTestConnectionInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	provider := New(Config{BotToken: "xoxb-test", SigningSecret: "secret", APIBaseURL: server.URL})
	require.Error(t, provider.TestConnection(t.Context()))
}
