package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIncidentsListWithStatusFilter(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents?status=open", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestIntegrationUpsertInvalidBody(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations", bytes.NewBufferString(`{`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoutingRulesCreateInvalidMatchLabels(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}
	body := bytes.NewBufferString(`{"team_id":"` + teamID.String() + `","match_labels":{},"priority":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routing-rules", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIncidentsInvalidID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	for _, spec := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/incidents/not-a-uuid"},
		{http.MethodPost, "/api/v1/incidents/not-a-uuid/acknowledge"},
		{http.MethodPost, "/api/v1/incidents/not-a-uuid/resolve"},
		{http.MethodGet, "/api/v1/incidents/not-a-uuid/timeline"},
	} {
		req := httptest.NewRequest(spec.method, spec.path, nil)
		req.AddCookie(admin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code, spec.path)
	}
}

func TestIncidentsNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/"+id.String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestRoutingRulesValidationErrors(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/routing-rules", bytes.NewBufferString(`{}`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/routing-rules/"+uuid.New().String(), bytes.NewBufferString(`{"team_id":"bad","match_labels":{},"priority":1}`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/routing-rules/not-a-uuid", nil)
	req.AddCookie(admin)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoutingRulesDeleteNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/routing-rules/"+uuid.New().String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestIntegrationsValidationErrors(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations", bytes.NewBufferString(`{"kind":"bad"}`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/not-a-uuid", nil)
	req.AddCookie(admin)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+uuid.New().String()+"/test", nil)
	req.AddCookie(admin)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSlackCallbackInvalidSignature(t *testing.T) {
	r, _ := setupPhase2Router(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/callbacks/slack/interactive", bytes.NewBufferString("payload={}"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIncidentsAcknowledgeConflict(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "acknowledged", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/acknowledge", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestIncidentsResolveConflict(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "resolved", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/resolve", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestRoutingRulesCreateTeamNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	body := bytes.NewBufferString(`{"team_id":"` + uuid.New().String() + `","match_labels":{"team":"platform"},"priority":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routing-rules", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestIncidentsGetWithAlerts(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	alertID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}
	repo.alerts[incidentID] = []db.Alert{{ID: alertID, Severity: "critical", Title: "CPU", Status: "firing"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/"+incidentID.String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
