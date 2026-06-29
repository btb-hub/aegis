package loader

import (
	"encoding/json"
	"testing"

	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegisterFromRowsLoadsExpress(t *testing.T) {
	reg := integrations.NewRegistry()
	cfg, err := json.Marshal(intexpress.Config{
		BotID: "bot", Host: "https://cts.example.com", SecretKey: "secret",
	})
	require.NoError(t, err)
	RegisterFromRows(reg, []integrations.IntegrationRow{
		{ID: uuid.New(), Kind: "express", Config: cfg, Enabled: true},
	}, "http://localhost:8080")
	_, ok := reg.Chat("express")
	require.True(t, ok)
}
