package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIncidentHandoff(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	l2Team := uuid.New()
	l3Team := uuid.New()
	l3User := uuid.New()
	repo.teams[l2Team] = db.Team{ID: l2Team, Name: "L2"}
	repo.teams[l3Team] = db.Team{ID: l3Team, Name: "L3"}
	repo.users[l3User] = db.User{ID: l3User, Email: "l3@example.com", DisplayName: "L3 User"}
	repo.memberships[l3Team] = map[uuid.UUID]db.TeamMembership{
		l3User: {TeamID: l3Team, UserID: l3User},
	}
	repo.incidents[incidentID] = db.Incident{
		ID: incidentID, TeamID: l2Team, Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp",
	}

	body, _ := json.Marshal(map[string]string{"to_team_id": l3Team.String(), "note": "escalate"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/handoff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, l3User.String(), resp["assignee_id"])
}

func TestIncidentBounceRequiresNote(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{
		ID: incidentID, TeamID: uuid.New(), Status: "acknowledged", Severity: "critical", Title: "CPU", Fingerprint: "fp",
	}

	body, _ := json.Marshal(map[string]string{"note": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/bounce", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnalyticsHandoffs(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.handoffStats = db.HandoffStats{Count: 3, MedianResponseSeconds: 90}

	from := "2026-06-01T00:00:00Z"
	to := "2026-06-30T00:00:00Z"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/handoffs?from="+from+"&to="+to, nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, float64(3), resp["count"])
}
