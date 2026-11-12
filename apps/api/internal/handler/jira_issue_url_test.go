package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type jiraURLFailureRepo struct{ *phase2HandlerRepo }

func (r *jiraURLFailureRepo) GetTeam(context.Context, uuid.UUID) (db.Team, error) {
	return db.Team{}, errors.New("team store unavailable")
}

// JIRA-LINK-AC1: if the real detail handler cannot resolve the workspace,
// it returns an error instead of an invented link. The repository is a fake;
// this test does not exercise authentication middleware or a live database.
func TestIncidentDetailJiraLookupFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &jiraURLFailureRepo{newPhase2HandlerRepo()}
	id, key := uuid.New(), "OPS-42"
	repo.incidents[id] = db.Incident{ID: id, TeamID: uuid.New(), JiraIssueKey: &key}
	svc := service.NewIncidentService(repo, time.Hour, time.Minute)
	h := NewIncidentHandler(svc, nil, nil)
	r := gin.New()
	r.GET("/incidents/:id", h.getIncident)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/incidents/"+id.String(), nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NotContains(t, w.Body.String(), "jira_issue_url")
}
