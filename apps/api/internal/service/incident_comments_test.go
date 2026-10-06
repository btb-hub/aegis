package service

import (
	"context"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestValidateIncidentComment(t *testing.T) {
	require.NoError(t, validateIncidentComment(" useful note ", false))
	require.Error(t, validateIncidentComment(" \n", false))
	require.NoError(t, validateIncidentComment("", true))
}

type commentFixture struct {
	incidentMockRepo
	err  error
	body string
}

func (f *commentFixture) AddIncidentComment(_ context.Context, _, _ uuid.UUID, body string) (db.TimelineEvent, error) {
	f.body = body
	return db.TimelineEvent{Kind: "comment_added"}, f.err
}
func (f *commentFixture) ResolveIncidentWithComment(_ context.Context, _, _ uuid.UUID, body string) (db.Incident, error) {
	f.body = body
	return db.Incident{Status: "resolved"}, f.err
}
func TestIncidentCommentsPersistenceAndErrors(t *testing.T) {
	f := &commentFixture{}
	svc := NewIncidentService(f, time.Hour, time.Minute)
	id, actor := uuid.New(), uuid.New()
	event, err := svc.AddComment(t.Context(), id, actor, " text\nsecond ")
	require.NoError(t, err)
	require.Equal(t, "comment_added", event.Kind)
	require.Equal(t, "text\nsecond", f.body)
	_, err = svc.ResolveWithComment(t.Context(), id, actor, "resolved")
	require.NoError(t, err)
	require.Equal(t, "resolved", f.body)
	f.err = pgx.ErrNoRows
	_, err = svc.AddComment(t.Context(), id, actor, "note")
	require.Error(t, err)
	_, err = svc.ResolveWithComment(t.Context(), id, actor, "note")
	require.Error(t, err)
	_, err = svc.AddComment(t.Context(), id, actor, "")
	require.Error(t, err)
	_, err = svc.ResolveWithComment(t.Context(), id, actor, strings.Repeat("a", 10001))
	require.Error(t, err)
	legacy := NewIncidentService(&incidentMockRepo{incident: db.Incident{Status: "open"}}, time.Hour, time.Minute)
	_, err = legacy.AddComment(t.Context(), id, actor, "note")
	require.Error(t, err)
	_, err = legacy.ResolveWithComment(t.Context(), id, actor, "note")
	require.Error(t, err)
	_, err = legacy.ResolveWithComment(t.Context(), id, actor, "")
	require.NoError(t, err)
}
