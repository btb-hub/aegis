package processor

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type jiraSyncFixture struct {
	incident  db.Incident
	user      db.User
	connector db.Integration
	slot      db.Integration
	state     db.JiraSyncState
	comments  []db.JiraComment
	fail      string
	calls     map[string]int
	events    []string
}

func (f *jiraSyncFixture) failure(op string) error {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls[op]++
	if f.fail == op {
		return errors.New("repository unavailable")
	}
	return nil
}
func (f *jiraSyncFixture) RunJiraSync(ctx context.Context, id uuid.UUID, fn func(db.JiraSyncRepository) error) error {
	if err := f.failure("run"); err != nil {
		return err
	}
	return fn(f)
}
func (f *jiraSyncFixture) GetIncidentByID(context.Context, uuid.UUID) (db.Incident, error) {
	return f.incident, f.failure("incident")
}
func (f *jiraSyncFixture) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return f.user, f.failure("user")
}
func (f *jiraSyncFixture) GetTeamWorkspaceID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), f.failure("workspace")
}
func (f *jiraSyncFixture) GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error) {
	return f.slot, f.failure("slot")
}
func (f *jiraSyncFixture) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	return f.connector, f.failure("global")
}
func (f *jiraSyncFixture) GetIntegration(context.Context, uuid.UUID) (db.Integration, error) {
	return f.connector, f.failure("connector")
}
func (f *jiraSyncFixture) GetJiraSyncState(context.Context, uuid.UUID) (db.JiraSyncState, error) {
	return f.state, f.failure("state")
}
func (f *jiraSyncFixture) SaveJiraSyncState(_ context.Context, s db.JiraSyncState) error {
	if err := f.failure("save_state"); err != nil {
		return err
	}
	f.state = s
	return nil
}
func (f *jiraSyncFixture) SaveJiraIssue(_ context.Context, _, connector uuid.UUID, key, url string) error {
	if err := f.failure("save_issue"); err != nil {
		return err
	}
	f.incident.JiraIssueKey = &key
	f.incident.JiraIssueURL = &url
	f.incident.JiraIntegrationID = &connector
	return nil
}
func (f *jiraSyncFixture) PendingJiraComments(context.Context, uuid.UUID) ([]db.JiraComment, error) {
	var pending []db.JiraComment
	for _, c := range f.comments {
		if c.ExternalID == nil {
			pending = append(pending, c)
		}
	}
	return pending, f.failure("comments")
}
func (f *jiraSyncFixture) SaveJiraComment(_ context.Context, id uuid.UUID, external *string, attempted bool) error {
	if err := f.failure("save_comment"); err != nil {
		return err
	}
	for i := range f.comments {
		if f.comments[i].EventID == id {
			f.comments[i].ExternalID = external
			f.comments[i].Attempted = attempted
		}
	}
	return nil
}
func (f *jiraSyncFixture) AppendTimelineEvent(_ context.Context, _ uuid.UUID, kind string, _ *uuid.UUID, _ []byte) error {
	f.events = append(f.events, kind)
	return f.failure("timeline")
}
func newJiraSyncFixture(host string) *jiraSyncFixture {
	user := db.User{ID: uuid.New(), Email: "engineer@example.test"}
	connector := db.Integration{ID: uuid.New(), Kind: "jira", Enabled: true, Config: []byte(`{"base_url":"` + host + `","api_token":"pat","project_key":"OPS","deployment":"server_dc"}`)}
	return &jiraSyncFixture{incident: db.Incident{ID: uuid.New(), TeamID: uuid.New(), AssigneeID: &user.ID, Status: "resolved", Title: "Resolved before processing", Severity: "P1"}, user: user, connector: connector, slot: db.Integration{Enabled: true, Config: []byte(`{"project_key":"OPS"}`)}, state: db.JiraSyncState{Config: []byte(`{}`)}, comments: []db.JiraComment{{EventID: uuid.New(), Body: "resolution\nrestored", Author: "Engineer", CreatedAt: time.Now()}}}
}
func jiraSyncJob(f *jiraSyncFixture) Job {
	raw, _ := json.Marshal(map[string]string{"incident_id": f.incident.ID.String()})
	return Job{ID: uuid.NewString(), Payload: raw}
}
func TestJiraSyncResolvedCreationHandoffAndUncertainComment(t *testing.T) {
	creates, assignments, comments := 0, 0, 0
	issueVisible, commentVisible := false, false
	failComment := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasPrefix(r.URL.Path, "/rest/api/2/"))
		require.Equal(t, "Bearer pat", r.Header.Get("Authorization"))
		switch {
		case r.URL.Path == "/rest/api/2/search":
			if issueVisible {
				_, _ = w.Write([]byte(`{"issues":[{"key":"OPS-1"}]}`))
			} else {
				_, _ = w.Write([]byte(`{"issues":[]}`))
			}
		case r.URL.Path == "/rest/api/2/issue" && r.Method == "POST":
			creates++
			issueVisible = true
			_, _ = w.Write([]byte(`{"key":"OPS-1"}`))
		case strings.Contains(r.URL.Path, "user/search"):
			_, _ = w.Write([]byte(`[{"name":"engineer"}]`))
		case r.Method == "PUT":
			assignments++
			w.WriteHeader(204)
		case r.Method == "GET":
			if commentVisible {
				_, _ = w.Write([]byte(`{"comments":[{"id":"comment-1","body":"` + r.Header.Get("X-Unused") + `"}],"total":1}`))
			} else {
				_, _ = w.Write([]byte(`{"comments":[],"total":0}`))
			}
		case r.Method == "POST":
			comments++
			if failComment {
				w.WriteHeader(503)
			} else {
				_, _ = w.Write([]byte(`{"id":"comment-1"}`))
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	f := newJiraSyncFixture(server.URL)
	p := NewJiraSyncProcessor(f, "https://aegis.example.test")
	job := jiraSyncJob(f)
	require.Error(t, p.Handle(t.Context(), job))
	require.Equal(t, 1, creates)
	require.Equal(t, 1, assignments)
	require.Equal(t, 1, comments)
	require.True(t, f.comments[0].Attempted)
	require.Equal(t, server.URL+"/browse/OPS-1", *f.incident.JiraIssueURL)
	// Marker missing after an uncertain write never causes a second POST.
	require.ErrorContains(t, p.Handle(t.Context(), job), "uncertain")
	require.Equal(t, 1, comments)
	// Reconcile after Jira becomes visible; preserve connector even after team handoff.
	marker := "Aegis event: " + f.comments[0].EventID.String()
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/comment"):
			_ = json.NewEncoder(w).Encode(map[string]any{"comments": []any{map[string]string{"id": "comment-1", "body": marker}}, "total": 1})
		case strings.Contains(r.URL.Path, "user/search"):
			_, _ = w.Write([]byte(`[{"name":"new-engineer"}]`))
		case r.Method == "PUT":
			assignments++
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected replay %s", r.URL.Path)
		}
	})
	newUser := uuid.New()
	f.incident.TeamID = uuid.New()
	f.incident.AssigneeID = &newUser
	f.user.ID = newUser
	require.NoError(t, p.Handle(t.Context(), job))
	require.Equal(t, 1, creates)
	require.Equal(t, 1, comments)
	require.Equal(t, 2, assignments)
	require.Equal(t, "comment-1", *f.comments[0].ExternalID)
}
func TestJiraSyncRepositoryFailuresAndValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/search"):
			_, _ = w.Write([]byte(`{"issues":[{"key":"OPS-1"}]}`))
		case strings.Contains(r.URL.Path, "user/search"):
			_, _ = w.Write([]byte(`[{"name":"engineer"}]`))
		case r.Method == "PUT":
			w.WriteHeader(204)
		case r.Method == "GET":
			_, _ = w.Write([]byte(`{"comments":[],"total":0}`))
		default:
			_, _ = w.Write([]byte(`{"id":"comment-1"}`))
		}
	}))
	defer server.Close()
	for _, op := range []string{"run", "incident", "state", "workspace", "slot", "global", "save_state", "save_issue", "user", "comments", "save_comment", "timeline"} {
		t.Run(op, func(t *testing.T) {
			f := newJiraSyncFixture(server.URL)
			f.fail = op
			require.Error(t, NewJiraSyncProcessor(f, "").Handle(t.Context(), jiraSyncJob(f)))
		})
	}
	f := newJiraSyncFixture(server.URL)
	p := NewJiraSyncProcessor(f, "")
	require.Error(t, p.Handle(t.Context(), Job{Payload: []byte("bad")}))
	require.Error(t, p.Handle(t.Context(), Job{Payload: []byte(`{"incident_id":"bad"}`)}))
	f.state.IntegrationID = &f.connector.ID
	f.state.Config = []byte(`{}`)
	f.fail = "connector"
	require.Error(t, p.Handle(t.Context(), jiraSyncJob(f)))
	f.fail = ""
	f.connector.Enabled = false
	require.ErrorContains(t, p.Handle(t.Context(), jiraSyncJob(f)), "disabled")
	f.connector.Enabled = true
	f.state.Config = []byte("bad")
	require.Error(t, p.Handle(t.Context(), jiraSyncJob(f)))
	f.state.IntegrationID = nil
	f.incident.JiraIssueKey = new(string)
	require.ErrorContains(t, p.Handle(t.Context(), jiraSyncJob(f)), "no recorded connector")
	f.incident.JiraIssueKey = nil
	f.slot.Enabled = false
	require.NoError(t, p.Handle(t.Context(), jiraSyncJob(f)))
	require.False(t, definiteJiraRejection(errors.New("network")))
	require.False(t, errors.Is(errors.New("x"), pgx.ErrNoRows))
}
func TestJiraSyncUncertainCreateNeverReposts(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"issues":[]}`))
		} else {
			posts++
			w.WriteHeader(503)
		}
	}))
	defer server.Close()
	f := newJiraSyncFixture(server.URL)
	p := NewJiraSyncProcessor(f, "")
	require.Error(t, p.Handle(t.Context(), jiraSyncJob(f)))
	require.True(t, f.state.CreateAttempted)
	require.ErrorContains(t, p.Handle(t.Context(), jiraSyncJob(f)), "uncertain")
	require.Equal(t, 1, posts)
}
