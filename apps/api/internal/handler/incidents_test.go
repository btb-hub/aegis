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
}

func newPhase2HandlerRepo() *phase2HandlerRepo {
	return &phase2HandlerRepo{
		teamRepoMock: teamRepoMock{
			authMockUsers:    authMockUsers{users: map[uuid.UUID]db.User{}},
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
func (m *phase2HandlerRepo) GetRoutingRule(context.Context, uuid.UUID) (db.RoutingRule, error) {
	return db.RoutingRule{}, pgx.ErrNoRows
}
func (m *phase2HandlerRepo) CreateRoutingRule(_ context.Context, teamID uuid.UUID, matchLabels map[string]string, priority int32) (db.RoutingRule, error) {
	raw, _ := json.Marshal(matchLabels)
	rule := db.RoutingRule{ID: uuid.New(), TeamID: teamID, MatchLabels: raw, Priority: priority, CreatedAt: time.Now(), UpdatedAt: time.Now()}
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

func (m *phase2HandlerRepo) UpdateRoutingRule(_ context.Context, id, teamID uuid.UUID, matchLabels map[string]string, priority int32) (db.RoutingRule, error) {
	if _, ok := m.rules[id]; !ok {
		return db.RoutingRule{}, pgx.ErrNoRows
	}
	raw, _ := json.Marshal(matchLabels)
	rule := db.RoutingRule{ID: id, TeamID: teamID, MatchLabels: raw, Priority: priority, CreatedAt: time.Now(), UpdatedAt: time.Now()}
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
func (m *phase2HandlerRepo) UpsertIntegration(_ context.Context, kind, name string, config json.RawMessage, enabled bool) (db.Integration, error) {
	item := db.Integration{ID: uuid.New(), Kind: kind, Name: name, Config: config, Enabled: enabled, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.integrations[item.ID] = item
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

func setupPhase2Router(t *testing.T) (*gin.Engine, *phase2HandlerRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newPhase2HandlerRepo()
	cfg := &config.Config{SessionTTL: time.Hour, PublicURL: "http://localhost:8080"}
	auth := service.NewAuthService(cfg, repo, repo, &authMockOIDC{})
	incidents := service.NewIncidentService(repo)
	routingRules := service.NewRoutingService(repo)
	integrationsSvc := service.NewIntegrationService(repo, cfg.PublicURL)
	expressLinks := service.NewExpressLinkService(repo)
	alerts := service.NewAlertService("secret", []string{"alertname", "team"}, &authMockAlertRepo{id: uuid.New()})
	health := service.NewHealthService(nil)

	r := gin.New()
	NewHealthHandler(health).Register(r)
	NewAuthHandler(auth).Register(r)
	NewAlertHandler(alerts).Register(r)
	NewIncidentHandler(incidents, auth).Register(r)
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
	body := bytes.NewBufferString(`{"kind":"jira","name":"Jira","config":{"project_key":"OPS"},"enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations", body)
	req.AddCookie(admin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
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
	teamID := uuid.New()
	repo.teams[teamID] = db.Team{ID: teamID, Name: "Platform"}

	body := bytes.NewBufferString(`{"team_id":"` + teamID.String() + `","match_labels":{"team":"platform"},"priority":10}`)
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
