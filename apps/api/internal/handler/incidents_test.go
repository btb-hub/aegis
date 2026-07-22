package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type phase2HandlerRepo struct {
	teamRepoMock
	incidents         map[uuid.UUID]db.Incident
	events            map[uuid.UUID][]db.TimelineEvent
	alerts            map[uuid.UUID][]db.Alert
	rules             map[uuid.UUID]db.RoutingRule
	integrations      map[uuid.UUID]db.Integration
	listIncidentsErr  error
	listIntegrationsErr error
	alertListErr      error
	handoffStats      db.HandoffStats
	handoffStatsErr   error
	mttaSeries        db.MetricTimeSeries
	mttaSeriesErr     error
	mttrSeries        db.MetricTimeSeries
	mttrSeriesErr     error
	noiseStats        db.NoiseStats
	noiseErr          error
	onCallLoad        db.OnCallLoadStats
	onCallLoadErr     error
	escalationStats   db.EscalationStats
	escalationErr     error
	escalationPaths   []db.EscalationPath
	alertRepo         *authMockAlertRepo
	bounceFails       bool
}

func newPhase2HandlerRepo() *phase2HandlerRepo {
	return &phase2HandlerRepo{
		teamRepoMock: teamRepoMock{
			authMockUsers:    *newAuthMockUsers(),
			authMockSessions: authMockSessions{byHash: map[string]db.Session{}},
			teams:            map[uuid.UUID]db.Team{},
			memberships:      map[uuid.UUID]map[uuid.UUID]db.TeamMembership{},
		},
		incidents: map[uuid.UUID]db.Incident{},
		events:    map[uuid.UUID][]db.TimelineEvent{},
		alerts:    map[uuid.UUID][]db.Alert{},
		rules:        map[uuid.UUID]db.RoutingRule{},
		integrations: map[uuid.UUID]db.Integration{},
	}
}

func (m *phase2HandlerRepo) ListIncidents(_ context.Context, status string) ([]db.Incident, error) {
	if m.listIncidentsErr != nil {
		return nil, m.listIncidentsErr
	}
	items := make([]db.Incident, 0, len(m.incidents))
	for _, incident := range m.incidents {
		if status != "" && incident.Status != status {
			continue
		}
		items = append(items, incident)
	}
	return items, nil
}
func (m *phase2HandlerRepo) GetIncidentByID(_ context.Context, id uuid.UUID) (db.Incident, error) {
	incident, ok := m.incidents[id]
	if !ok {
		return db.Incident{}, pgx.ErrNoRows
	}
	return incident, nil
}
func (m *phase2HandlerRepo) AcknowledgeIncident(_ context.Context, incidentID, actorID uuid.UUID) (db.Incident, error) {
	incident, ok := m.incidents[incidentID]
	if !ok || incident.Status != "open" {
		return db.Incident{}, pgx.ErrNoRows
	}
	now := time.Now()
	incident.Status = "acknowledged"
	incident.AcknowledgedAt = &now
	m.incidents[incidentID] = incident
	return incident, nil
}
func (m *phase2HandlerRepo) ResolveIncident(_ context.Context, incidentID, actorID uuid.UUID) (db.Incident, error) {
	incident, ok := m.incidents[incidentID]
	if !ok || incident.Status == "resolved" {
		return db.Incident{}, pgx.ErrNoRows
	}
	now := time.Now()
	incident.Status = "resolved"
	incident.ResolvedAt = &now
	m.incidents[incidentID] = incident
	return incident, nil
}
func (m *phase2HandlerRepo) ListTimelineEvents(_ context.Context, incidentID uuid.UUID) ([]db.TimelineEvent, error) {
	return m.events[incidentID], nil
}
func (m *phase2HandlerRepo) ListAlertsForIncident(_ context.Context, incidentID uuid.UUID) ([]db.Alert, error) {
	if m.alertListErr != nil {
		return nil, m.alertListErr
	}
	return m.alerts[incidentID], nil
}
func (m *phase2HandlerRepo) CancelEscalationJobs(context.Context, uuid.UUID) error { return nil }
func (m *phase2HandlerRepo) GetUserBySlackID(_ context.Context, slackUserID string) (db.User, error) {
	for _, user := range m.users {
		if user.SlackUserID != nil && *user.SlackUserID == slackUserID {
			return user, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *phase2HandlerRepo) GetUserByExpressHuid(_ context.Context, expressHuid uuid.UUID) (db.User, error) {
	for _, user := range m.users {
		if user.ExpressUserHuid.Valid && uuid.UUID(user.ExpressUserHuid.Bytes) == expressHuid {
			return user, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *phase2HandlerRepo) CreateExpressLinkCode(_ context.Context, userID uuid.UUID, _ time.Duration) (string, error) {
	return "ABC123", nil
}

func (m *phase2HandlerRepo) RedeemExpressLinkCode(_ context.Context, code string, expressHuid uuid.UUID) (db.User, error) {
	if code != "ABC123" {
		return db.User{}, fmt.Errorf("link code invalid or expired")
	}
	for id, user := range m.users {
		user.ExpressUserHuid = db.ExpressHuidToPg(expressHuid)
		m.users[id] = user
		return user, nil
	}
	return db.User{}, pgx.ErrNoRows
}

func (m *phase2HandlerRepo) UpdateUserExpressHuid(_ context.Context, userID, expressHuid uuid.UUID) (db.User, error) {
	user, ok := m.users[userID]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	user.ExpressUserHuid = db.ExpressHuidToPg(expressHuid)
	m.users[userID] = user
	return user, nil
}

func (m *phase2HandlerRepo) ListRoutingRules(context.Context) ([]db.RoutingRule, error) {
	items := make([]db.RoutingRule, 0, len(m.rules))
	for _, rule := range m.rules {
		items = append(items, rule)
	}
	return items, nil
}
func (m *phase2HandlerRepo) GetRoutingRule(_ context.Context, id uuid.UUID) (db.RoutingRule, error) {
	rule, ok := m.rules[id]
	if !ok {
		return db.RoutingRule{}, pgx.ErrNoRows
	}
	return rule, nil
}
func (m *phase2HandlerRepo) CreateRoutingRule(
	_ context.Context,
	workspaceID, teamID uuid.UUID,
	matchLabels map[string]string,
	priority int32,
	crossWorkspace bool,
) (db.RoutingRule, error) {
	raw, _ := json.Marshal(matchLabels)
	rule := db.RoutingRule{
		ID: uuid.New(), WorkspaceID: workspaceID, TeamID: teamID, MatchLabels: raw,
		Priority: priority, CrossWorkspace: crossWorkspace, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.rules[rule.ID] = rule
	return rule, nil
}
func (m *phase2HandlerRepo) DeleteRoutingRule(_ context.Context, id uuid.UUID) error {
	if _, ok := m.rules[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.rules, id)
	return nil
}

func (m *phase2HandlerRepo) UpdateRoutingRule(
	_ context.Context,
	id, workspaceID, teamID uuid.UUID,
	matchLabels map[string]string,
	priority int32,
	crossWorkspace bool,
) (db.RoutingRule, error) {
	if _, ok := m.rules[id]; !ok {
		return db.RoutingRule{}, pgx.ErrNoRows
	}
	raw, _ := json.Marshal(matchLabels)
	rule := db.RoutingRule{
		ID: id, WorkspaceID: workspaceID, TeamID: teamID, MatchLabels: raw,
		Priority: priority, CrossWorkspace: crossWorkspace, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.rules[id] = rule
	return rule, nil
}

func (m *phase2HandlerRepo) GetIntegration(_ context.Context, id uuid.UUID) (db.Integration, error) {
	item, ok := m.integrations[id]
	if !ok {
		return db.Integration{}, pgx.ErrNoRows
	}
	return item, nil
}
func (m *phase2HandlerRepo) GetIntegrationByKind(_ context.Context, kind string) (db.Integration, error) {
	for _, item := range m.integrations {
		if item.Kind == kind && item.WorkspaceID == nil {
			return item, nil
		}
	}
	return db.Integration{}, pgx.ErrNoRows
}
func (m *phase2HandlerRepo) GetWorkspaceIntegration(_ context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error) {
	for _, item := range m.integrations {
		if item.Kind == kind && item.WorkspaceID != nil && *item.WorkspaceID == workspaceID {
			return item, nil
		}
	}
	return db.Integration{}, pgx.ErrNoRows
}
func (m *phase2HandlerRepo) UpsertIntegration(_ context.Context, kind, name string, config json.RawMessage, enabled bool, workspaceID *uuid.UUID, mode *string) (db.Integration, error) {
	item := db.Integration{ID: uuid.New(), Kind: kind, Name: name, Config: config, Enabled: enabled, WorkspaceID: workspaceID, Mode: mode, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.integrations[item.ID] = item
	return item, nil
}
func (m *phase2HandlerRepo) UpdateIntegration(_ context.Context, id uuid.UUID, name string, config json.RawMessage, enabled bool, mode *string) (db.Integration, error) {
	item, ok := m.integrations[id]
	if !ok {
		return db.Integration{}, pgx.ErrNoRows
	}
	item.Name = name
	item.Config = config
	item.Enabled = enabled
	item.Mode = mode
	item.UpdatedAt = time.Now()
	m.integrations[id] = item
	return item, nil
}
func (m *phase2HandlerRepo) DeleteIntegration(_ context.Context, id uuid.UUID) error {
	if _, ok := m.integrations[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.integrations, id)
	return nil
}
func (m *phase2HandlerRepo) ListIntegrations(ctx context.Context) ([]db.Integration, error) {
	if m.listIntegrationsErr != nil {
		return nil, m.listIntegrationsErr
	}
	items := make([]db.Integration, 0, len(m.integrations))
	for _, item := range m.integrations {
		items = append(items, item)
	}
	return items, nil
}
func (m *phase2HandlerRepo) ListEnabledIntegrations(context.Context) ([]integrations.IntegrationRow, error) {
	return nil, nil
}

func (m *phase2HandlerRepo) GetWorkspace(_ context.Context, id uuid.UUID) (db.Workspace, error) {
	return db.Workspace{ID: id, Name: "Default", Slug: "default"}, nil
}

func (m *phase2HandlerRepo) GetTeam(_ context.Context, id uuid.UUID) (db.Team, error) {
	team, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return team, nil
}

func (m *phase2HandlerRepo) CurrentOnCallUsers(_ context.Context, teamID uuid.UUID, _ time.Time) ([]db.OnCallUser, error) {
	for userID := range m.memberships[teamID] {
		user, ok := m.users[userID]
		if !ok {
			continue
		}
		return []db.OnCallUser{{UserID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Source: "rotation"}}, nil
	}
	return nil, nil
}

func (m *phase2HandlerRepo) HandoffIncident(_ context.Context, input db.HandoffIncidentInput) (db.Incident, db.Handoff, error) {
	incident, ok := m.incidents[input.IncidentID]
	if !ok || incident.Status == "resolved" {
		return db.Incident{}, db.Handoff{}, pgx.ErrNoRows
	}
	assignee := input.ToUserID
	incident.AssigneeID = &assignee
	m.incidents[input.IncidentID] = incident
	handoff := db.Handoff{
		ID:         uuid.New(),
		IncidentID: input.IncidentID,
		FromTeamID: incident.TeamID,
		ToTeamID:   input.ToTeamID,
		ToUserID:   &input.ToUserID,
		CreatedAt:  time.Now(),
	}
	return incident, handoff, nil
}

func (m *phase2HandlerRepo) BounceIncident(_ context.Context, input db.BounceIncidentInput) (db.Incident, error) {
	if m.bounceFails {
		return db.Incident{}, pgx.ErrNoRows
	}
	incident, ok := m.incidents[input.IncidentID]
	if !ok || incident.Status == "resolved" {
		return db.Incident{}, pgx.ErrNoRows
	}
	l2 := input.ActorID
	incident.AssigneeID = &l2
	m.incidents[input.IncidentID] = incident
	return incident, nil
}

func (m *phase2HandlerRepo) EnqueueHandoffNotify(context.Context, uuid.UUID) error { return nil }

func (m *phase2HandlerRepo) HasEscalationPath(_ context.Context, fromTeamID, toTeamID uuid.UUID) (bool, error) {
	for _, path := range m.escalationPaths {
		if path.FromTeamID == fromTeamID && path.ToTeamID == toTeamID {
			return true, nil
		}
	}
	return len(m.escalationPaths) == 0, nil
}

func (m *phase2HandlerRepo) HandoffStats(context.Context, time.Time, time.Time) (db.HandoffStats, error) {
	if m.handoffStatsErr != nil {
		return db.HandoffStats{}, m.handoffStatsErr
	}
	return m.handoffStats, nil
}

func (m *phase2HandlerRepo) MTTASeries(context.Context, time.Time, time.Time) (db.MetricTimeSeries, error) {
	if m.mttaSeriesErr != nil {
		return db.MetricTimeSeries{}, m.mttaSeriesErr
	}
	return m.mttaSeries, nil
}

func (m *phase2HandlerRepo) MTTRSeries(context.Context, time.Time, time.Time) (db.MetricTimeSeries, error) {
	if m.mttrSeriesErr != nil {
		return db.MetricTimeSeries{}, m.mttrSeriesErr
	}
	return m.mttrSeries, nil
}

func (m *phase2HandlerRepo) TopNoise(context.Context, time.Time, time.Time, int) (db.NoiseStats, error) {
	if m.noiseErr != nil {
		return db.NoiseStats{}, m.noiseErr
	}
	return m.noiseStats, nil
}

func (m *phase2HandlerRepo) OnCallLoad(context.Context, time.Time, time.Time) (db.OnCallLoadStats, error) {
	if m.onCallLoadErr != nil {
		return db.OnCallLoadStats{}, m.onCallLoadErr
	}
	return m.onCallLoad, nil
}

func (m *phase2HandlerRepo) EscalationStats(context.Context, time.Time, time.Time) (db.EscalationStats, error) {
	if m.escalationErr != nil {
		return db.EscalationStats{}, m.escalationErr
	}
	return m.escalationStats, nil
}

func setupPhase2Router(t *testing.T) (*gin.Engine, *phase2HandlerRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newPhase2HandlerRepo()
	cfg := &config.Config{SessionTTL: time.Hour, PublicURL: "http://localhost:8080"}
	auth := service.NewAuthService(cfg, repo, repo, &authMockOIDC{})
	incidents := service.NewIncidentService(repo)
	handoffs := service.NewHandoffService(repo)
	analytics := service.NewAnalyticsService(repo)
	routingRules := service.NewRoutingService(repo)
	integrationsSvc := service.NewIntegrationService(repo, cfg.PublicURL)
	expressLinks := service.NewExpressLinkService(repo)
	teams := service.NewTeamService(&emptyTeamRepo{}, nil)
	alertsRepo := &authMockAlertRepo{id: uuid.New()}
	repo.alertRepo = alertsRepo
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, alertsRepo)
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth, cfg.PublicURL).Register(r)
	NewAlertHandler(alerts, teams, auth).Register(r)
	NewIncidentHandler(incidents, handoffs, auth).Register(r)
	NewAnalyticsHandler(analytics, alerts, handoffs, auth).Register(r)
	NewRoutingHandler(routingRules, auth).Register(r)
	NewIntegrationHandler(integrationsSvc, auth).Register(r)
	NewSlackCallbackHandler(incidents, "secret").Register(r)
	NewExpressCallbackHandler(incidents, expressLinks, integrationsSvc).Register(r)
	NewExpressLinkHandler(expressLinks, auth).Register(r)
	return r, repo
}

func TestIncidentsGetAndResolve(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/"+incidentID.String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/resolve", nil)
	req.AddCookie(admin)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestIntegrationsUpsert(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	body := bytes.NewBufferString(`{"kind":"jira","name":"Jira","config":{"base_url":"https://jira.example.com","email":"ops@example.com","api_token":"token","project_key":"OPS"},"enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	cfg, ok := created["config"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "***", cfg["api_token"])
	require.Equal(t, "https://jira.example.com", cfg["base_url"])
}

func TestIntegrationsPatchKeepsSecrets(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	id := uuid.New()
	repo.integrations[id] = db.Integration{
		ID: id, Kind: "jira", Name: "Jira", Enabled: true,
		Config: []byte(`{"base_url":"https://jira.example.com","email":"ops@example.com","api_token":"keep-me","project_key":"OPS"}`),
	}
	body := bytes.NewBufferString(`{"name":"Jira Prod","config":{"base_url":"https://jira.example.com","email":"ops@example.com","api_token":"","project_key":"OPS"}}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/integrations/"+id.String(), body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	updated := repo.integrations[id]
	var cfg map[string]any
	require.NoError(t, json.Unmarshal(updated.Config, &cfg))
	require.Equal(t, "keep-me", cfg["api_token"])
	require.Equal(t, "Jira Prod", updated.Name)
}

func TestIntegrationsPatchAcceptsWorkspaceMode(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	id := uuid.New()
	workspaceID := uuid.New()
	inherit := "inherit"
	repo.integrations[id] = db.Integration{
		ID: id, Kind: "slack", Name: "Slack", Enabled: true, WorkspaceID: &workspaceID, Mode: &inherit,
		Config: []byte(`{}`),
	}
	body := bytes.NewBufferString(`{"mode":"custom","config":{"bot_token":"token","signing_secret":"secret"}}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/integrations/"+id.String(), body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, repo.integrations[id].Mode)
	require.Equal(t, "custom", *repo.integrations[id].Mode)
}

func TestIntegrationsGet(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	id := uuid.New()
	repo.integrations[id] = db.Integration{
		ID: id, Kind: "slack", Name: "Slack", Enabled: true,
		Config: []byte(`{"bot_token":"xoxb-live","signing_secret":"shh"}`),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/"+id.String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	cfg := body["config"].(map[string]any)
	require.Equal(t, "***", cfg["bot_token"])
}

func TestIntegrationsGetWorkspaceSlotStatusUsesGlobal(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	workspaceID := uuid.New()
	slotID := uuid.New()
	inherit := "inherit"
	repo.integrations[uuid.New()] = db.Integration{
		ID: uuid.New(), Kind: "jira", Name: "Jira", Enabled: true,
		Config: []byte(`{"base_url":"https://jira.example.com","email":"ops@example.com","api_token":"token","project_key":"GLOBAL"}`),
	}
	repo.integrations[slotID] = db.Integration{
		ID: slotID, Kind: "jira", Name: "Jira", Enabled: true, WorkspaceID: &workspaceID, Mode: &inherit,
		Config: []byte(`{"project_key":"OPS"}`),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/"+slotID.String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "using_global", body["slot_status"])
}

func TestIntegrationsGetNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/"+uuid.New().String(), nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestIntegrationsPatchInvalidBody(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	id := uuid.New()
	repo.integrations[id] = db.Integration{ID: id, Kind: "slack", Name: "Slack", Config: []byte(`{"bot_token":"x","signing_secret":"y"}`), Enabled: true}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/integrations/"+id.String(), bytes.NewBufferString(`{`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIntegrationsPatchEnableOnly(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	id := uuid.New()
	repo.integrations[id] = db.Integration{
		ID: id, Kind: "slack", Name: "Slack", Enabled: true,
		Config: []byte(`{"bot_token":"x","signing_secret":"y"}`),
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/integrations/"+id.String(), bytes.NewBufferString(`{"enabled":false}`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.False(t, repo.integrations[id].Enabled)
}

func TestIntegrationsGetInvalidID(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/not-a-uuid", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIntegrationsPatchNotFound(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/integrations/"+uuid.New().String(), bytes.NewBufferString(`{"enabled":false}`))
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestIncidentsListAndAcknowledge(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	incidentID := uuid.New()
	repo.incidents[incidentID] = db.Incident{ID: incidentID, TeamID: uuid.New(), Status: "open", Severity: "critical", Title: "CPU", Fingerprint: "fp"}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	req.AddCookie(admin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incidentID.String()+"/acknowledge", nil)
	req.AddCookie(admin)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "acknowledged", repo.incidents[incidentID].Status)
}

func TestRoutingRulesCRUD(t *testing.T) {
	r, repo := setupPhase2Router(t)
	admin := seedAdmin(t, r, repo)
	workspaceID := uuid.New()
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, WorkspaceID: workspaceID, Name: "Platform"}

	body := bytes.NewBufferString(`{"workspace_id":"` + workspaceID.String() + `","team_id":"` + teamID.String() + `","match_labels":{"team":"platform"},"priority":10}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routing-rules", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	require.Len(t, repo.rules, 1)
}

func seedAdmin(t *testing.T, r *gin.Engine, repo *phase2HandlerRepo) *http.Cookie {
	t.Helper()
	userID := uuid.New()
	repo.users[userID] = db.User{ID: userID, Email: "admin@example.com", Role: "admin", Locale: "en"}
	token, hash, err := sessionTokenPair()
	require.NoError(t, err)
	_, err = repo.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))
	require.NoError(t, err)
	return &http.Cookie{Name: sessionCookie, Value: token}
}
