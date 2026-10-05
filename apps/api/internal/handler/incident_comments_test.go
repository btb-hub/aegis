package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func (r *phase2HandlerRepo) AddIncidentComment(_ context.Context, id, actor uuid.UUID, body string) (db.TimelineEvent, error) {
	if _, ok := r.incidents[id]; !ok {
		return db.TimelineEvent{}, pgx.ErrNoRows
	}
	raw, _ := json.Marshal(map[string]string{"body": body})
	event := db.TimelineEvent{ID: uuid.New(), IncidentID: id, ActorID: &actor, Kind: "comment_added", Payload: raw}
	r.events[id] = append(r.events[id], event)
	return event, nil
}
func (r *phase2HandlerRepo) ResolveIncidentWithComment(ctx context.Context, id, actor uuid.UUID, comment string) (db.Incident, error) {
	out, err := r.ResolveIncident(ctx, id, actor)
	if err != nil {
		return out, err
	}
	if comment != "" {
		_, err = r.AddIncidentComment(ctx, id, actor, comment)
	}
	return out, err
}
func TestCommentAPIResolvedAndPermissions(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	id := uuid.New()
	repo.incidents[id] = db.Incident{ID: id, Status: "resolved"}
	for _, spec := range []struct {
		id, body string
		want     int
	}{{id.String(), `{"body":"postmortem\nline"}`, 201}, {id.String(), `{`, 400}, {id.String(), `{"body":" "}`, 400}, {"bad", `{"body":"note"}`, 400}, {uuid.NewString(), `{"body":"note"}`, 404}} {
		req := httptest.NewRequest("POST", "/api/v1/incidents/"+spec.id+"/comments", bytes.NewBufferString(spec.body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(admin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, spec.want, w.Code, w.Body.String())
	}
	require.Len(t, repo.events[id], 1)
	for uid, user := range repo.users {
		user.Role = "viewer"
		repo.users[uid] = user
	}
	req := httptest.NewRequest("POST", "/api/v1/incidents/"+id.String()+"/comments", bytes.NewBufferString(`{"body":"no"}`))
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}
func TestResolutionCommentAndEmptyBody(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	for _, body := range []string{`{"comment":"restored"}`, ""} {
		id := uuid.New()
		repo.incidents[id] = db.Incident{ID: id, Status: "open"}
		req := httptest.NewRequest("POST", "/api/v1/incidents/"+id.String()+"/resolve", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(admin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Equal(t, "resolved", repo.incidents[id].Status)
	}
}
