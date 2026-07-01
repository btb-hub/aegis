package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestIncidentBounceSuccess(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	l2User := uuid.New()
	repo.incidents[incidentID] = db.Incident{
		ID: incidentID, TeamID: uuid.New(), AssigneeID: &l2User, Status: "acknowledged", Severity: "critical", Title: "CPU", Fingerprint: "fp",
	}

	body, _ := json.Marshal(map[string]string{"note": "wrong team"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/bounce", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestIncidentHandoffUnknownTeam(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{
		ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp",
	}

	body, _ := json.Marshal(map[string]string{"to_team_id": uuid.New().String(), "note": "escalate"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/handoff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestIncidentHandoffInvalidTeamID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{
		ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp",
	}

	body, _ := json.Marshal(map[string]string{"to_team_id": "not-a-uuid", "note": "escalate"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/handoff", bytes.NewReader(body))
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

func TestIncidentHandoffNoOnCall(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	emptyTeam := uuid.New()
	repo.teams[emptyTeam] = db.Team{ID: emptyTeam, Name: "Empty"}
	repo.incidents[incidentID] = db.Incident{
		ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp",
	}

	body, _ := json.Marshal(map[string]string{"to_team_id": emptyTeam.String(), "note": "escalate"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/handoff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncidentHandoffUnauthorized(t *testing.T) {
	r, repo := setupPhase2Router(t)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{
		ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp",
	}

	body, _ := json.Marshal(map[string]string{"to_team_id": uuid.New().String(), "note": "escalate"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/handoff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIncidentHandoffInvalidBody(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/handoff", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncidentBounceUnauthorized(t *testing.T) {
	r, repo := setupPhase2Router(t)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, Status: "acknowledged", Severity: "critical", Title: "CPU", Fingerprint: "fp"}

	body, _ := json.Marshal(map[string]string{"note": "reason"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/bounce", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAnalyticsHandoffsInvalidFrom(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/handoffs?from=bad&to=2026-06-30T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncidentHandoffResolvedConflict(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	l3Team := uuid.New()
	l3User := uuid.New()
	repo.teams[l3Team] = db.Team{ID: l3Team, Name: "L3"}
	repo.users[l3User] = db.User{ID: l3User, Email: "l3@example.com"}
	repo.memberships[l3Team] = map[uuid.UUID]db.TeamMembership{l3User: {TeamID: l3Team, UserID: l3User}}
	repo.incidents[incidentID] = db.Incident{
		ID: incidentID, TeamID: uuid.New(), Status: "resolved", Severity: "critical", Title: "CPU", Fingerprint: "fp",
	}

	body, _ := json.Marshal(map[string]string{"to_team_id": l3Team.String(), "note": "escalate"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/handoff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestIncidentBounceInvalidBody(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, Status: "acknowledged", Severity: "critical", Title: "CPU", Fingerprint: "fp"}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/bounce", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncidentBounceConflict(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.bounceFails = true
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, Status: "acknowledged", Severity: "critical", Title: "CPU", Fingerprint: "fp"}

	body, _ := json.Marshal(map[string]string{"note": "no prior handoff"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/bounce", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestIncidentHandoffNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	l3Team := uuid.New()
	repo.teams[l3Team] = db.Team{ID: l3Team, Name: "L3"}

	body, _ := json.Marshal(map[string]string{"to_team_id": l3Team.String(), "note": "escalate"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+uuid.New().String()+"/handoff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAnalyticsHandoffsInvalidTo(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/handoffs?from=2026-06-01T00:00:00Z&to=bad", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnalyticsHandoffsMissingRange(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/handoffs", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnalyticsHandoffsStatsError(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	repo.handoffStatsErr = fmt.Errorf("db down")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/handoffs?from=2026-06-01T00:00:00Z&to=2026-06-30T00:00:00Z", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
