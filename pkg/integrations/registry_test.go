package integrations

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubTicket struct {
	kind string
	err  error
	key  string
}

func (s stubTicket) Kind() string { return s.kind }

func (s stubTicket) CreateTicket(_ context.Context, _ IncidentRef) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.key, nil
}

func (s stubTicket) TestConnection(context.Context) error { return s.err }

type stubChat struct {
	kind string
	err  error
	ref  string
}

func (s stubChat) Kind() string { return s.kind }

func (s stubChat) SendPage(_ context.Context, _ IncidentRef, _ PageRecipient) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.ref, nil
}

func (s stubChat) TestConnection(context.Context) error { return s.err }

type mockLoader struct {
	rows []IntegrationRow
	err  error
}

func (m mockLoader) ListEnabledIntegrations(context.Context) ([]IntegrationRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rows, nil
}

func TestLoadRegistry(t *testing.T) {
	loader := mockLoader{rows: []IntegrationRow{
		{ID: uuid.New(), Kind: "jira", Enabled: true},
		{ID: uuid.New(), Kind: "slack", Enabled: true},
		{ID: uuid.New(), Kind: "express", Enabled: true},
	}}

	reg, err := Load(context.Background(), loader,
		map[string]Factory{
			"jira": func(IntegrationRow) (any, error) {
				return stubTicket{kind: "jira", key: "PROJ-1"}, nil
			},
		},
		map[string]Factory{
			"slack": func(IntegrationRow) (any, error) {
				return stubChat{kind: "slack", ref: "msg-1"}, nil
			},
		},
	)
	require.NoError(t, err)

	jira, ok := reg.Ticket("jira")
	require.True(t, ok)
	key, err := jira.CreateTicket(context.Background(), IncidentRef{})
	require.NoError(t, err)
	require.Equal(t, "PROJ-1", key)

	slack, ok := reg.Chat("slack")
	require.True(t, ok)
	ref, err := slack.SendPage(context.Background(), IncidentRef{}, PageRecipient{})
	require.NoError(t, err)
	require.Equal(t, "msg-1", ref)
}

func TestForEachTicketDoesNotPanicOnError(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterTicket(stubTicket{kind: "jira", err: errors.New("down")})

	called := 0
	ForEachTicket(reg, func(p TicketProvider) error {
		called++
		_, err := p.CreateTicket(context.Background(), IncidentRef{})
		return err
	})
	require.Equal(t, 1, called)
}

func TestForEachChatDoesNotPanicOnError(t *testing.T) {
	reg := NewRegistry()
	reg.chats["slack"] = stubChat{kind: "slack", err: errors.New("down")}
	reg.chats["express"] = stubChat{kind: "express", ref: "ok"}

	called := 0
	ForEachChat(reg, func(p ChatProvider) error {
		called++
		_, err := p.SendPage(context.Background(), IncidentRef{}, PageRecipient{})
		return err
	})
	require.Equal(t, 2, called)
}

func TestLoadSkipsBrokenFactory(t *testing.T) {
	loader := mockLoader{rows: []IntegrationRow{{Kind: "jira"}}}
	reg, err := Load(context.Background(), loader,
		map[string]Factory{"jira": func(IntegrationRow) (any, error) {
			return nil, errors.New("bad config")
		}},
		nil,
	)
	require.NoError(t, err)
	_, ok := reg.Ticket("jira")
	require.False(t, ok)
}

func TestRegisterChat(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterChat(stubChat{kind: "slack", ref: "msg-1"})
	chat, ok := reg.Chat("slack")
	require.True(t, ok)
	ref, err := chat.SendPage(context.Background(), IncidentRef{}, PageRecipient{})
	require.NoError(t, err)
	require.Equal(t, "msg-1", ref)
}
