package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func incidentSessionForRole(t *testing.T, repo *phase2HandlerRepo, role string) *http.Cookie {
	t.Helper()
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID, Role: role, Email: "u@example.com", DisplayName: "User", Locale: "en"}
	token, hash, err := sessionTokenPair()
	require.NoError(t, err)
	_, err = repo.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))
	require.NoError(t, err)
	return &http.Cookie{Name: sessionCookie, Value: token}
}

func TestCreateIncidentFromAlertHappyPath(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, teamID.String(), resp["team_id"])
	require.Len(t, repo.enqueuedJobs, 2)
}

func TestCreateIncidentFromAlertForbiddenForViewer(t *testing.T) {
	r, repo := setupPhase2Router(t)
	viewer := incidentSessionForRole(t, repo, "viewer")
	alertID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(viewer)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreateIncidentFromAlertNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}

	body := bytes.NewBufferString(`{"alert_id":"` + uuid.New().String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateIncidentFromAlertConflictOpenLink(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}
	repo.alertOpenLinks = map[uuid.UUID]db.Incident{
		alertID: {ID: uuid.New(), Status: "open"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateIncidentFromAlertFingerprintTeamMismatch(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	otherTeam := uuid.New()
	existingID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-dup"},
	}
	repo.fingerprintOpen = map[string]db.Incident{
		"fp-dup": {ID: existingID, TeamID: otherTeam, Status: "open"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	details, ok := resp["details"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, existingID.String(), details["incident_id"])
}

func TestCreateIncidentFromAlertMissingTeamID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateIncidentFromAlertWithAssignee(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	assigneeID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.memberships[teamID] = map[uuid.UUID]db.TeamMembership{assigneeID: {TeamID: teamID, UserID: assigneeID}}
	repo.users[assigneeID] = db.User{ID: assigneeID, Email: "oncall@example.com"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `","assignee_id":"` + assigneeID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestCreateIncidentFromAlertInvalidAssignee(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	assigneeID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `","assignee_id":"` + assigneeID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateIncidentFromAlertAdminAllowed(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := incidentSessionForRole(t, repo, "admin")
	alertID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestCreateIncidentFromAlertInvalidAlertID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	body := bytes.NewBufferString(`{"alert_id":"not-a-uuid","team_id":"` + uuid.New().String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateIncidentFromAlertInvalidBody(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewBufferString(`{`))
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateIncidentFromAlertFingerprintLink(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	existingID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-dup"},
	}
	repo.fingerprintOpen = map[string]db.Incident{
		"fp-dup": {ID: existingID, TeamID: teamID, Status: "open"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, repo.enqueuedJobs)
}

func TestCreateIncidentFromAlertInvalidTeamID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	body := bytes.NewBufferString(`{"alert_id":"` + uuid.New().String() + `","team_id":"bad"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateIncidentFromAlertInvalidAssigneeUUID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	body := bytes.NewBufferString(`{"alert_id":"` + uuid.New().String() + `","team_id":"` + uuid.New().String() + `","assignee_id":"bad"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateIncidentFromAlertUnknownTeam(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateIncidentFromAlertConflictAcknowledgedLink(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}
	repo.alertOpenLinks = map[uuid.UUID]db.Incident{
		alertID: {ID: uuid.New(), Status: "acknowledged"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateIncidentFromAlertResolvedOnlyLinkSucceeds(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "firing", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}
	// Resolved-only links are not in alertOpenLinks (open/acked semantics).

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestCreateIncidentFromAlertNonFiring(t *testing.T) {
	r, repo := setupPhase2Router(t)
	member := incidentSessionForRole(t, repo, "member")
	alertID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	repo.firingAlerts = map[uuid.UUID]db.Alert{
		alertID: {ID: alertID, Status: "resolved", Severity: "critical", Title: "CPU", Fingerprint: "fp-1"},
	}

	body := bytes.NewBufferString(`{"alert_id":"` + alertID.String() + `","team_id":"` + teamID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", body)
	req.AddCookie(member)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
