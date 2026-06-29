package loader

import (
	"encoding/json"
	"testing"

	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegisterFromRowsLoadsAllProviders(t *testing.T) {
	reg := integrations.NewRegistry()
	expressCfg, err := json.Marshal(intexpress.Config{
		BotID: "bot", Host: "https://cts.example.com", SecretKey: "secret",
	})
	require.NoError(t, err)

	RegisterFromRows(reg, []integrations.IntegrationRow{
		{ID: uuid.New(), Kind: "jira", Config: []byte(`{"base_url":"https://jira.example.com","email":"a@b.com","api_token":"x","project_key":"OPS"}`), Enabled: true},
		{ID: uuid.New(), Kind: "slack", Config: []byte(`{"bot_token":"xoxb-test","signing_secret":"secret"}`), Enabled: true},
		{ID: uuid.New(), Kind: "express", Config: expressCfg, Enabled: true},
	}, "http://localhost:8080")

	_, ok := reg.Ticket("jira")
	require.True(t, ok)
	_, ok = reg.Chat("slack")
	require.True(t, ok)
	_, ok = reg.Chat("express")
	require.True(t, ok)
}

func TestRegisterFromRowsSkipsDisabledAndInvalid(t *testing.T) {
	reg := integrations.NewRegistry()
	RegisterFromRows(reg, []integrations.IntegrationRow{
		{ID: uuid.New(), Kind: "jira", Config: []byte(`{`), Enabled: true},
		{ID: uuid.New(), Kind: "slack", Config: []byte(`{"bot_token":"x"}`), Enabled: true},
		{ID: uuid.New(), Kind: "express", Config: []byte(`{}`), Enabled: true},
		{ID: uuid.New(), Kind: "jira", Config: []byte(`{"base_url":"https://jira.example.com","email":"a@b.com","api_token":"x","project_key":"OPS"}`), Enabled: false},
	}, "http://localhost:8080")

	require.Empty(t, reg.TicketProviders())
	require.Empty(t, reg.ChatProviders())
}
