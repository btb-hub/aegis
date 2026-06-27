package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSendPageMissingSlackUserID(t *testing.T) {
	provider := New(Config{BotToken: "xoxb-test", SigningSecret: "secret"})
	_, err := provider.SendPage(t.Context(), integrations.IncidentRef{ID: uuid.New()}, integrations.PageRecipient{})
	require.Error(t, err)
}

func TestSendPageFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
	}))
	defer server.Close()

	slackID := "U123"
	provider := New(Config{BotToken: "xoxb-test", SigningSecret: "secret", APIBaseURL: server.URL})
	_, err := provider.SendPage(t.Context(), integrations.IncidentRef{ID: uuid.New(), Title: "CPU", Severity: "critical"}, integrations.PageRecipient{Locale: "en", SlackUserID: &slackID})
	require.Error(t, err)
}

func TestParseInteractiveAckMissingPayload(t *testing.T) {
	_, _, err := ParseInteractiveAck(nil)
	require.Error(t, err)
}
