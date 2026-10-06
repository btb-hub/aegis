package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type deliveryFixture struct {
	*flowAlertStore
	express         db.Integration
	user            db.User
	sent            map[string]bool
	fail            string
	records, events []string
	syncs           int
}

func (f *deliveryFixture) HasDelivery(_ context.Context, _, _ uuid.UUID, key string) (bool, error) {
	if f.fail == "has" {
		return false, errors.New("db")
	}
	return f.sent[key], nil
}
func (f *deliveryFixture) RecordDelivery(_ context.Context, _, _ uuid.UUID, key, status, _, _ string) error {
	if f.fail == "record" {
		return errors.New("db")
	}
	f.records = append(f.records, key+":"+status)
	if status == "sent" || status == "failed_permanent" {
		f.sent[key] = true
	}
	return nil
}
func (f *deliveryFixture) AppendTimelineEvent(_ context.Context, _ uuid.UUID, kind string, _ *uuid.UUID, _ []byte) error {
	f.events = append(f.events, kind)
	if f.fail == "timeline" {
		return errors.New("db")
	}
	return nil
}
func (f *deliveryFixture) GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error) {
	if kind == "express" {
		if f.fail == "global" {
			return db.Integration{}, errors.New("db")
		}
		return f.express, nil
	}
	return f.flowAlertStore.GetIntegrationByKind(ctx, kind)
}
func (f *deliveryFixture) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	if f.fail == "user" {
		return db.User{}, errors.New("db")
	}
	return f.user, nil
}
func (f *deliveryFixture) EnqueueJiraSync(context.Context, uuid.UUID) error {
	f.syncs++
	if f.fail == "sync" {
		return errors.New("db")
	}
	return nil
}
func TestIndependentChannelPrivateHandoffAndEscalation(t *testing.T) {
	channel, personal := 0, 0
	privateFails := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/token"):
			_, _ = w.Write([]byte(`{"status":"ok","result":"token"}`))
		case r.Method == "GET":
			require.Equal(t, "/api/v1/botx/chats/personal", r.URL.Path)
			_, _ = w.Write([]byte(`{"status":"ok","result":{"group_chat_id":"personal-chat"}}`))
		default:
			require.Equal(t, "/api/v4/botx/notifications/direct", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if body["group_chat_id"] == "oncall" {
				channel++
			} else {
				personal++
				if privateFails {
					w.WriteHeader(503)
					return
				}
			}
			_, _ = fmt.Fprintf(w, `{"status":"ok","result":{"sync_id":"msg-%d-%d"}}`, channel, personal)
		}
	}))
	defer server.Close()
	base := newFlowAlertStore(t)
	huid := uuid.NewString()
	user := db.User{ID: base.assigneeID, ExpressUserHuid: pgtype.UUID{Bytes: uuid.MustParse(huid), Valid: true}, Locale: "en"}
	cfg, _ := json.Marshal(map[string]string{"host": server.URL, "bot_id": "bot", "secret_key": "secret", "oncall_group_chat_id": "oncall"})
	f := &deliveryFixture{flowAlertStore: base, express: db.Integration{ID: uuid.New(), Kind: "express", Config: cfg, Enabled: true}, user: user, sent: map[string]bool{}}
	f.workspaceIntegrations = map[string]db.Integration{"express": {Enabled: true, Mode: strPtr("inherit"), Config: []byte(`{}`)}}
	incident := db.Incident{ID: uuid.New(), TeamID: base.teamID, AssigneeID: &base.assigneeID, Status: "open", Title: "CPU", Severity: "P1"}
	f.incidentForNotify = incident
	job := Job{ID: "event-open", Payload: json.RawMessage(`{"incident_id":"` + incident.ID.String() + `"}`)}
	p := NewNotifyIncidentProcessor(nil, f, "https://aegis.example.test")
	require.Error(t, p.Handle(t.Context(), job))
	require.Equal(t, 1, channel)
	privateFails = false
	require.NoError(t, p.Handle(t.Context(), job))
	require.Equal(t, 1, channel)
	privateCount := personal
	require.NoError(t, p.Handle(t.Context(), job))
	require.Equal(t, privateCount, personal)
	handoff := NewHandoffNotifyProcessor(nil, f, "")
	job.ID = "handoff-event"
	require.NoError(t, handoff.Handle(t.Context(), job))
	personalAfterHandoff := personal
	require.NoError(t, handoff.Handle(t.Context(), job))
	require.Equal(t, personalAfterHandoff, personal)
	require.Equal(t, 2, f.syncs)
	escalate := NewEscalateProcessor(nil, f, "")
	job.ID = "escalation-event"
	require.NoError(t, escalate.Handle(t.Context(), job))
	require.Greater(t, personal, personalAfterHandoff)
	f.fail = "sync"
	require.Error(t, handoff.Handle(t.Context(), job))
	f.fail = "user"
	require.Error(t, p.Handle(t.Context(), job))
	f.fail = "global"
	require.Error(t, p.Handle(t.Context(), job))
}
func mustUUID(raw string) *uuid.UUID { id := uuid.MustParse(raw); return &id }
func TestDeliverOnceFailurePersistence(t *testing.T) {
	f := &deliveryFixture{sent: map[string]bool{}}
	id, connector := uuid.New(), uuid.New()
	calls := 0
	send := func() (string, error) { calls++; return "ref", nil }
	for _, failure := range []string{"has", "record", "timeline"} {
		f.fail = failure
		require.Error(t, deliverOnce(t.Context(), f, id, connector, failure, send))
	}
	f.fail = ""
	require.Error(t, deliverOnce(t.Context(), f, id, connector, "permanent", func() (string, error) { return "", &integrations.HTTPError{Status: 403} }))
	require.NoError(t, deliverOnce(t.Context(), f, id, connector, "permanent", send))
	require.True(t, f.sent["permanent"])
}

type replyFixture struct {
	connector db.Integration
	sent      map[string]bool
	fail      string
}

func (f *replyFixture) GetIntegration(context.Context, uuid.UUID) (db.Integration, error) {
	if f.fail == "get" {
		return db.Integration{}, errors.New("db")
	}
	return f.connector, nil
}
func (f *replyFixture) HasExpressOutbound(_ context.Context, _ uuid.UUID, key string) (bool, error) {
	if f.fail == "has" {
		return false, errors.New("db")
	}
	return f.sent[key], nil
}
func (f *replyFixture) RecordExpressOutbound(_ context.Context, _ uuid.UUID, key, _ string) error {
	if f.fail == "record" {
		return errors.New("db")
	}
	f.sent[key] = true
	return nil
}
func TestRepliesSplitAndResume(t *testing.T) {
	sends := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"status":"ok","result":"token"}`))
			return
		}
		sends++
		_, _ = fmt.Fprintf(w, `{"status":"ok","result":{"sync_id":"msg-%d"}}`, sends)
	}))
	defer server.Close()
	cfg, _ := json.Marshal(map[string]string{"host": server.URL, "bot_id": "bot", "secret_key": "secret"})
	f := &replyFixture{connector: db.Integration{ID: uuid.New(), Enabled: true, Config: cfg}, sent: map[string]bool{}}
	p := NewExpressReplyProcessor(f)
	raw, _ := json.Marshal(map[string]string{"integration_id": f.connector.ID.String(), "chat_id": "incoming", "body": strings.Repeat("a", 2500) + "\n" + strings.Repeat("б", 2500)})
	job := Job{ID: "command-1", Payload: raw}
	require.NoError(t, p.Handle(t.Context(), job))
	require.Equal(t, 2, sends)
	require.NoError(t, p.Handle(t.Context(), job))
	require.Equal(t, 2, sends)
	require.Len(t, splitChatText(strings.Repeat("a", 9000), 4000), 3)
	for _, failure := range []string{"get", "has", "record"} {
		f.fail = failure
		f.sent = map[string]bool{}
		require.Error(t, p.Handle(t.Context(), job))
	}
	f.fail = ""
	f.connector.Enabled = false
	require.Error(t, p.Handle(t.Context(), job))
	f.connector.Enabled = true
	f.connector.Config = []byte(`{}`)
	require.Error(t, p.Handle(t.Context(), job))
	require.Error(t, p.Handle(t.Context(), Job{Payload: []byte("bad")}))
	require.Error(t, p.Handle(t.Context(), Job{Payload: []byte(`{"integration_id":"bad"}`)}))
	f.connector.Config = cfg
	raw, _ = json.Marshal(map[string]string{"integration_id": f.connector.ID.String(), "body": "hello"})
	require.Error(t, p.Handle(t.Context(), Job{Payload: raw}))
}
func TestMixedDestinationRetryClassification(t *testing.T) {
	permanent := &integrations.HTTPError{Status: 403}
	transient := &integrations.HTTPError{Status: 503}
	require.True(t, integrations.RetryableError(errors.Join(permanent, transient)))
	require.False(t, integrations.RetryableError(errors.Join(permanent, permanent)))
	require.True(t, integrations.RetryableError(errors.New("network")))
	require.False(t, integrations.RetryableError(nil))
}
