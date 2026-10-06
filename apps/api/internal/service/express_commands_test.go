package service

import (
	"context"
	"errors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/i18n"
	express "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

type commandFixture struct {
	incidentMockRepo
	fail              string
	outcomes, replies []string
	teams             []db.Team
	users             []db.OnCallUser
	seen              map[string]bool
}

func (f *commandFixture) err(op string) error {
	if f.fail == op {
		return errors.New("unavailable")
	}
	return nil
}
func (f *commandFixture) RunExpressCommand(_ context.Context, _ uuid.UUID, id string, fn func(db.ExpressCommandRepository) (string, error)) error {
	if f.fail == "run" {
		return errors.New("unavailable")
	}
	if f.seen == nil {
		f.seen = map[string]bool{}
	}
	if f.seen[id] {
		return nil
	}
	outcome, err := fn(f)
	if err == nil {
		f.seen[id] = true
		f.outcomes = append(f.outcomes, outcome)
	}
	return err
}
func (f *commandFixture) HandleExpressNotificationResult(context.Context, uuid.UUID, string, string, string) error {
	return f.err("result")
}
func (f *commandFixture) RedeemExpressLinkCode(context.Context, string, uuid.UUID) (db.User, error) {
	if f.fail == "expired" {
		return db.User{}, db.ErrLinkCodeInvalid
	}
	return f.user, f.err("link")
}
func (f *commandFixture) GetUserByExpressHuid(context.Context, uuid.UUID) (db.User, error) {
	if f.fail == "unlinked" {
		return db.User{}, pgx.ErrNoRows
	}
	return f.user, f.err("user")
}
func (f *commandFixture) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	if f.fail == "missing" {
		return db.Incident{}, pgx.ErrNoRows
	}
	return f.incident, f.err("incident")
}
func (f *commandFixture) AcknowledgeIncident(ctx context.Context, id, actor uuid.UUID) (db.Incident, error) {
	if f.fail == "raced" {
		return db.Incident{}, pgx.ErrNoRows
	}
	if err := f.err("ack"); err != nil {
		return db.Incident{}, err
	}
	return f.incidentMockRepo.AcknowledgeIncident(ctx, id, actor)
}
func (f *commandFixture) CancelEscalationJobs(context.Context, uuid.UUID) error {
	return f.err("cancel")
}
func (f *commandFixture) ListTeams(context.Context) ([]db.Team, error) {
	return f.teams, f.err("teams")
}
func (f *commandFixture) CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]db.OnCallUser, error) {
	return f.users, f.err("oncall")
}
func (f *commandFixture) EnqueueExpressReply(_ context.Context, _ uuid.UUID, _, _, body, _ string) error {
	f.replies = append(f.replies, body)
	return f.err("reply")
}
func TestExpressCommandVisibleOutcomes(t *testing.T) {
	for _, spec := range []struct{ body, huid, fail, status, role, outcome string }{
		{"/link code", "valid", "", "open", "member", "linked"}, {"/link", "valid", "", "open", "member", "link_invalid"}, {"/link code", "bad", "", "open", "member", "link_invalid"}, {"/link code", "valid", "expired", "open", "member", "link_invalid"},
		{"/ack_incident bad", "valid", "", "open", "member", "incident_missing"}, {"/ack_incident ID", "bad", "", "open", "member", "unlinked"}, {"/ack_incident ID", "valid", "unlinked", "open", "member", "unlinked"},
		{"/ack_incident ID", "valid", "missing", "open", "member", "incident_missing"}, {"/ack_incident ID", "valid", "", "open", "viewer", "not_allowed"}, {"/ack_incident ID", "valid", "", "acknowledged", "member", "already_acknowledged"}, {"/ack_incident ID", "valid", "", "resolved", "member", "incident_resolved"}, {"/ack_incident ID", "valid", "raced", "open", "member", "already_acknowledged"}, {"/ack_incident ID", "valid", "", "open", "member", "acknowledged"}, {"unknown", "valid", "", "open", "member", "unsupported"}, {"/ack_incident", "", "", "open", "member", "incident_missing"},
	} {
		t.Run(spec.outcome+spec.fail+spec.body, func(t *testing.T) {
			id := uuid.New()
			f := &commandFixture{incidentMockRepo: incidentMockRepo{incident: db.Incident{ID: id, Status: spec.status}, user: db.User{ID: uuid.New(), Role: spec.role}}, fail: spec.fail}
			var event express.CommandEvent
			event.SyncID = "cmd"
			event.From.ChatID = "incoming-chat"
			event.From.Locale = "ru"
			event.Command.Body = strings.ReplaceAll(spec.body, "ID", id.String())
			event.From.UserHuid = spec.huid
			if spec.huid == "valid" {
				event.From.UserHuid = uuid.NewString()
			}
			svc := NewExpressCommandService(f)
			require.NoError(t, svc.Process(t.Context(), db.Integration{}, event))
			require.Equal(t, []string{spec.outcome}, f.outcomes)
			require.Len(t, f.replies, 1)
			require.NoError(t, svc.Process(t.Context(), db.Integration{}, event))
			require.Len(t, f.replies, 1)
		})
	}
}
func TestExpressOncallSystemAndInfrastructureFailures(t *testing.T) {
	require.NoError(t, i18n.LoadMessages("../../../../pkg/i18n/messages"))
	f := &commandFixture{incidentMockRepo: incidentMockRepo{incident: db.Incident{ID: uuid.New(), Status: "open"}, user: db.User{ID: uuid.New(), Role: "member"}}, teams: []db.Team{{ID: uuid.New(), Name: "Ops"}}, users: []db.OnCallUser{{DisplayName: "Engineer", StartAt: time.Date(2026, 10, 5, 20, 0, 0, 0, time.UTC), EndAt: time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)}}}
	svc := NewExpressCommandService(f)
	var event express.CommandEvent
	event.Command.Body = "/oncall"
	event.SyncID = "oncall"
	event.From.ChatID = "group"
	require.NoError(t, svc.Process(t.Context(), db.Integration{}, event))
	require.Contains(t, f.replies[0], "MSK")
	require.Contains(t, f.replies[0], "06 Oct")
	f.users = nil
	event.SyncID = "empty-team"
	require.NoError(t, svc.Process(t.Context(), db.Integration{}, event))
	f.teams = nil
	event.SyncID = "empty"
	require.NoError(t, svc.Process(t.Context(), db.Integration{}, event))
	event.Command.CommandType = "system"
	event.SyncID = "system"
	require.NoError(t, svc.Process(t.Context(), db.Integration{}, event))
	require.Equal(t, "system", f.outcomes[len(f.outcomes)-1])
	event.Command.CommandType = ""
	event.Command.Body = "system:added"
	event.SyncID = "system2"
	require.NoError(t, svc.Process(t.Context(), db.Integration{}, event))
	event.SyncID = ""
	require.NoError(t, svc.Process(t.Context(), db.Integration{}, event))
	for _, op := range []string{"run", "teams", "oncall", "reply", "link", "user", "incident", "ack", "cancel"} {
		f.fail = op
		f.teams = []db.Team{{ID: uuid.New()}}
		event.SyncID = "failure-" + op
		event.Command.Body = "/oncall"
		event.From.UserHuid = uuid.NewString()
		if op == "link" {
			event.Command.Body = "/link code"
		}
		if op == "user" || op == "incident" || op == "ack" || op == "cancel" {
			event.Command.Body = "/ack_incident " + f.incident.ID.String()
			f.incident.Status = "open"
		}
		require.Error(t, svc.Process(t.Context(), db.Integration{}, event), op)
	}
	f.fail = ""
	event.SyncID = "button"
	event.Command.Body = ""
	event.Command.Data = map[string]any{"incident_id": f.incident.ID.String()}
	require.NoError(t, svc.Process(t.Context(), db.Integration{}, event))
	require.Error(t, svc.NotificationResult(t.Context(), db.Integration{}, []byte("bad")))
	require.Error(t, svc.NotificationResult(t.Context(), db.Integration{}, []byte(`{}`)))
	require.NoError(t, svc.NotificationResult(t.Context(), db.Integration{}, []byte(`{"sync_id":"result","status":"ok"}`)))
	f.fail = "result"
	require.Error(t, svc.NotificationResult(t.Context(), db.Integration{}, []byte(`{"sync_id":"result","status":"error"}`)))
}
