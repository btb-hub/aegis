package incidentchat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/i18n"
	"github.com/aegis/aegis/pkg/incidentack"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type deliveryStore struct {
	Store
	cfg       db.Integration
	slot      db.Integration
	incident  db.Incident
	team      db.Team
	users     []db.OnCallUser
	user      db.User
	lookupErr error
}

func (s *deliveryStore) GetIntegration(context.Context, uuid.UUID) (db.Integration, error) {
	return s.cfg, s.lookupErr
}
func (s *deliveryStore) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	return s.incident, nil
}
func (s *deliveryStore) GetTeam(context.Context, uuid.UUID) (db.Team, error) { return s.team, nil }
func (s *deliveryStore) GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error) {
	return s.slot, nil
}
func (s *deliveryStore) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	return s.users, nil
}
func (s *deliveryStore) GetUserByID(context.Context, uuid.UUID) (db.User, error) { return s.user, nil }
func (s *deliveryStore) ClaimChannelDelivery(context.Context) (db.ChannelDelivery, error) {
	return db.ChannelDelivery{}, pgx.ErrNoRows
}
func (s *deliveryStore) ClaimChatAckAction(context.Context) (db.ChatAckRequest, error) {
	return db.ChatAckRequest{}, pgx.ErrNoRows
}
func (s *deliveryStore) ClaimChatAckFeedback(context.Context) (db.ChatAckRequest, error) {
	return db.ChatAckRequest{}, pgx.ErrNoRows
}

func TestChannelRunnerProvidersAndCurrentState(t *testing.T) {
	require.NoError(t, i18n.LoadMessages("../i18n/messages"))
	for _, provider := range []string{"slack", "express"} {
		t.Run(provider, func(t *testing.T) {
			var payloads []map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					_, _ = w.Write([]byte(`{"status":"ok","result":"token"}`))
					return
				}
				if provider == "slack" {
					require.Equal(t, "Bearer workspace-token", r.Header.Get("Authorization"))
				}
				var payload map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				payloads = append(payloads, payload)
				_, _ = w.Write([]byte(`{"ok":true,"ts":"message","status":"ok","result":{"sync_id":"message"}}`))
			}))
			defer server.Close()
			mode := "inherit"
			huid := uuid.New()
			cfg, _ := json.Marshal(map[string]string{"bot_token": "token", "signing_secret": "secret", "api_base_url": server.URL, "bot_id": "bot", "host": server.URL, "secret_key": "secret"})
			s := &deliveryStore{cfg: db.Integration{ID: uuid.New(), Enabled: true, Config: cfg}, slot: db.Integration{Enabled: true, Mode: &mode, Config: json.RawMessage(`{"bot_token":"workspace-token"}`)}, incident: db.Incident{ID: uuid.New(), Status: "open", Title: "CPU", Severity: "critical"}, team: db.Team{WorkspaceID: uuid.New()}, users: []db.OnCallUser{{UserID: uuid.New()}}, user: db.User{ExpressUserHuid: pgtype.UUID{Bytes: huid, Valid: true}}}
			r := New(s, "https://aegis.local/")
			d := db.ChannelDelivery{IntegrationID: s.cfg.ID, Provider: provider, Destination: "GROUP", Kind: "created", Snapshot: db.ChannelSnapshot{Incident: s.incident, TeamName: "Platform", Locale: "en", SlackUserGroupID: "S123"}}
			ref, err := r.sendChannel(t.Context(), d)
			require.NoError(t, err)
			require.Equal(t, "message", ref)
			raw, _ := json.Marshal(payloads[0])
			require.Contains(t, string(raw), "ack_incident")
			require.Contains(t, string(raw), s.incident.ID.String())
			s.incident.Status = "resolved"
			_, err = r.sendChannel(t.Context(), d)
			require.NoError(t, err)
			raw, _ = json.Marshal(payloads[1])
			require.NotContains(t, string(raw), "ack_incident")
			require.NotContains(t, string(raw), "S123")
			require.NotContains(t, string(raw), huid.String())
			req := db.ChatAckRequest{Provider: provider, IntegrationID: s.cfg.ID, IncidentID: s.incident.ID, WorkspaceID: &s.team.WorkspaceID, ChatID: "GROUP", UserIdentity: huid.String()}
			require.NoError(t, r.sendFeedback(t.Context(), req, incidentack.Result{Code: "already_resolved", Locale: "ru"}))
			raw, _ = json.Marshal(payloads[2])
			require.Contains(t, string(raw), "решён")
			if provider == "slack" {
				s.slot.Enabled = false
				require.Error(t, r.sendFeedback(t.Context(), req, incidentack.Result{Code: "acknowledged"}))
				require.Len(t, payloads, 3)
			}
		})
	}
}

func TestChannelConfigurationFailuresArePermanent(t *testing.T) {
	mode := "inherit"
	for _, tc := range []struct {
		name, provider string
		mutate         func(*deliveryStore)
	}{
		{"removed", "slack", func(s *deliveryStore) { s.lookupErr = pgx.ErrNoRows }},
		{"disabled", "slack", func(s *deliveryStore) { s.cfg.Enabled = false }},
		{"slot disabled", "slack", func(s *deliveryStore) { s.slot.Enabled = false }},
		{"slot missing mode", "slack", func(s *deliveryStore) { s.slot.Mode = nil }},
		{"credentials changed", "slack", func(s *deliveryStore) { custom := "custom"; s.slot.Mode = &custom; s.slot.ID = uuid.New() }},
		{"invalid slack config", "slack", func(s *deliveryStore) { s.cfg.Config = json.RawMessage(`{`) }},
		{"invalid workspace overlay", "slack", func(s *deliveryStore) { s.slot.Config = json.RawMessage(`{`) }},
		{"invalid express config", "express", func(s *deliveryStore) { s.cfg.Config = json.RawMessage(`{`) }},
		{"unknown provider", "other", func(*deliveryStore) {}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &deliveryStore{cfg: db.Integration{Enabled: true, Config: json.RawMessage(`{}`)}, slot: db.Integration{Enabled: true, Mode: &mode}}
			tc.mutate(s)
			_, err := New(s, "").sendChannel(t.Context(), db.ChannelDelivery{Provider: tc.provider})
			require.Error(t, err)
			require.False(t, integrations.RetryableError(err))
			if tc.provider != "other" {
				err = New(s, "").sendFeedback(t.Context(), db.ChatAckRequest{Provider: tc.provider, WorkspaceID: &s.team.WorkspaceID}, incidentack.Result{Code: "acknowledged", Locale: "en"})
				require.Error(t, err)
				require.False(t, integrations.RetryableError(err))
			}
		})
	}
}

func TestRunnerStopsItsIndependentLoops(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 1100*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() { New(&deliveryStore{}, "").Run(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop")
	}
}
