package slack

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIncidentChannelPostAndStaleActions(t *testing.T) {
	require.NoError(t, i18n.LoadMessages("../../i18n/messages"))
	var sent []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		sent = append(sent, payload)
		_, _ = w.Write([]byte(`{"ok":true,"ts":"message"}`))
	}))
	defer server.Close()
	p := New(Config{BotToken: "token", APIBaseURL: server.URL})
	id := uuid.New()
	post := integrations.IncidentChannelPost{Incident: integrations.IncidentRef{ID: id, Severity: "critical", Title: "CPU high", URL: "https://aegis.local/incidents?incident_id=" + id.String()}, Kind: "created", TeamName: "Platform", Locale: "en", OccurredAt: time.Now(), Actionable: true, SlackUserGroupID: "S123"}
	ref, err := p.SendIncidentEvent(context.Background(), post, "C123")
	require.NoError(t, err)
	require.Equal(t, "message", ref)
	require.Equal(t, "C123", sent[0]["channel"])
	raw, _ := json.Marshal(sent[0])
	require.Contains(t, string(raw), "ack_incident")
	require.Contains(t, string(raw), "S123")
	require.Contains(t, string(raw), id.String())
	post.Actionable = false
	_, err = p.SendIncidentEvent(context.Background(), post, "C123")
	require.NoError(t, err)
	raw, _ = json.Marshal(sent[1])
	require.NotContains(t, string(raw), "ack_incident")
	require.NotContains(t, string(raw), "S123")
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestExpiredResponseURLFallsBackToPrivateAPI(t *testing.T) {
	calls := 0
	p := New(Config{BotToken: "token"})
	p.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			require.Equal(t, "hooks.slack.com", r.URL.Host)
			return &http.Response{StatusCode: 410, Body: io.NopCloser(strings.NewReader("expired")), Header: make(http.Header)}, nil
		}
		require.Equal(t, "/api/chat.postEphemeral", r.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "U123", payload["user"])
		require.Equal(t, "C123", payload["channel"])
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})}
	require.NoError(t, p.SendAckFeedback(t.Context(), "https://hooks.slack.com/actions/test", "C123", "U123", "Acknowledged"))
	require.Equal(t, 2, calls)
}

func TestSlackDeliveryFailuresStayClassified(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		network    bool
	}{
		{"rate limit", "", 429, false}, {"invalid JSON", "bad", 200, false}, {"network", "", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New(Config{})
			p.client = &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
				if tc.network {
					return nil, errors.New("connection lost")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: http.Header{"Retry-After": []string{"30"}}}, nil
			})}
			_, err := p.SendIncidentEvent(t.Context(), integrations.IncidentChannelPost{}, "C123")
			require.Error(t, err)
			require.True(t, integrations.RetryableError(err))
			if tc.status == 429 {
				var httpErr *integrations.HTTPError
				require.ErrorAs(t, err, &httpErr)
				require.Equal(t, 30*time.Second, httpErr.RetryAfter)
			}
			err = p.SendAckFeedback(t.Context(), "https://hooks.slack.com/actions/test", "C123", "U123", "Acknowledged")
			if tc.status == 200 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.True(t, integrations.RetryableError(err))
			}
		})
	}
}

func TestIncidentTitlesCannotInjectSlackMentions(t *testing.T) {
	var payload map[string]any
	p := New(Config{})
	p.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})}
	require.NoError(t, i18n.LoadMessages("../../i18n/messages"))
	_, err := p.SendIncidentEvent(t.Context(), integrations.IncidentChannelPost{Kind: "created", Locale: "en", Incident: integrations.IncidentRef{Title: "<!channel> <@U123>", URL: "https://aegis.local/incidents"}}, "C123")
	require.NoError(t, err)
	require.NotContains(t, payload["text"], "<!channel>")
	require.Contains(t, payload["text"], "&lt;!channel&gt;")
	blocks := payload["blocks"].([]any)
	require.Equal(t, "plain_text", blocks[0].(map[string]any)["text"].(map[string]any)["type"])
}

type feedbackTransport struct{ payload map[string]any }

func (t *feedbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	_ = json.NewDecoder(req.Body).Decode(&t.payload)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
}

func TestAckResponseURLIsPrivateAndDoesNotReplacePost(t *testing.T) {
	transport := &feedbackTransport{}
	p := New(Config{})
	p.client = &http.Client{Transport: transport}
	require.NoError(t, p.SendAckFeedback(context.Background(), "https://hooks.slack.com/actions/test", "C123", "U123", "Acknowledged"))
	require.Equal(t, "ephemeral", transport.payload["response_type"])
	require.Equal(t, false, transport.payload["replace_original"])
	err := p.SendAckFeedback(context.Background(), "http://127.0.0.1/internal", "C123", "U123", "Acknowledged")
	require.Error(t, err)
	require.False(t, integrations.RetryableError(err))
}

func TestIncidentChannelPermanentAndTransientErrors(t *testing.T) {
	for _, tc := range []struct {
		code  string
		retry bool
	}{{"channel_not_found", false}, {"not_in_channel", false}, {"ratelimited", true}, {"internal_error", true}} {
		t.Run(tc.code, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": tc.code})
			}))
			defer server.Close()
			p := New(Config{APIBaseURL: server.URL})
			_, err := p.SendIncidentEvent(context.Background(), integrations.IncidentChannelPost{}, "C123")
			require.Error(t, err)
			require.Equal(t, tc.retry, integrations.RetryableError(err))
		})
	}
}
