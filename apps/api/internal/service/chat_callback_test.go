package service

import (
	"encoding/json"
	"testing"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCallbackIntegrationsUseEnabledSavedCredentials(t *testing.T) {
	t.Run("express", func(t *testing.T) {
		repo := &integrationMockRepo{}
		svc := NewIntegrationService(repo, "")
		_, _, err := svc.ExpressCallbackIntegration(t.Context())
		require.Error(t, err)
		row := db.Integration{ID: uuid.New(), Kind: "express", Config: json.RawMessage(`{"bot_id":"bot","host":"https://cts.example.com","secret_key":"secret"}`)}
		repo.items = []db.Integration{row}
		_, _, err = svc.ExpressCallbackIntegration(t.Context())
		require.Error(t, err)
		repo.items[0].Enabled = true
		got, cfg, err := svc.ExpressCallbackIntegration(t.Context())
		require.NoError(t, err)
		require.Equal(t, row.ID, got.ID)
		require.Equal(t, "secret", cfg.SecretKey)
		for _, invalid := range []string{`{`, `{}`} {
			repo.items[0].Config = json.RawMessage(invalid)
			_, _, err = svc.ExpressCallbackIntegration(t.Context())
			require.Error(t, err)
		}
	})
	t.Run("slack workspace modes", func(t *testing.T) {
		workspace := uuid.New()
		mode := "inherit"
		global := db.Integration{ID: uuid.New(), Kind: "slack", Enabled: true, Config: json.RawMessage(`{"bot_token":"global","signing_secret":"secret"}`)}
		slot := db.Integration{ID: uuid.New(), Kind: "slack", WorkspaceID: &workspace, Enabled: true, Mode: &mode, Config: json.RawMessage(`{}`)}
		repo := &integrationMockRepo{items: []db.Integration{global, slot}}
		svc := NewIntegrationService(repo, "")
		got, err := svc.SlackCallbackIntegration(t.Context(), workspace)
		require.NoError(t, err)
		require.Equal(t, global.ID, got.ID)
		require.JSONEq(t, string(global.Config), string(got.Config))
		mode = "custom"
		slot.Config = json.RawMessage(`{"bot_token":"custom","signing_secret":"custom-secret"}`)
		repo.items[1].Config = slot.Config
		got, err = svc.SlackCallbackIntegration(t.Context(), workspace)
		require.NoError(t, err)
		require.Equal(t, slot.ID, got.ID)
		require.JSONEq(t, string(slot.Config), string(got.Config))
		repo.items[1].Enabled = false
		_, err = svc.SlackCallbackIntegration(t.Context(), workspace)
		require.Error(t, err)
		_, err = svc.SlackCallbackIntegration(t.Context(), uuid.New())
		require.Error(t, err)
	})
}
