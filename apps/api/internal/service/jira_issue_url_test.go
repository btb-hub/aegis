package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type jiraURLRepo struct {
	incidentMockRepo
	team      db.Team
	teamErr   error
	slot      db.Integration
	slotErr   error
	global    db.Integration
	globalErr error
}

func (r *jiraURLRepo) GetTeam(context.Context, uuid.UUID) (db.Team, error) { return r.team, r.teamErr }
func (r *jiraURLRepo) GetWorkspaceIntegration(context.Context, uuid.UUID, string) (db.Integration, error) {
	return r.slot, r.slotErr
}
func (r *jiraURLRepo) GetIntegrationByKind(context.Context, string) (db.Integration, error) {
	return r.global, r.globalErr
}

// JIRA-LINK-AC1, incident.go::JiraIssueURL: these tests use a real service and
// resolver with an in-memory repository. They do not contact Jira or a database.
func TestJiraIssueURLBoundary(t *testing.T) {
	ctx := context.Background()
	key := "OPS-42"
	incident := db.Incident{TeamID: uuid.New(), JiraIssueKey: &key}
	custom, inherit, invalid := "custom", "inherit", "invalid"
	valid := []byte(`{"base_url":"https://jira.customer.test/context/","api_token":"secret","project_key":"OPS"}`)
	failure := errors.New("repository unavailable")
	cases := []struct {
		name     string
		repo     *jiraURLRepo
		incident db.Incident
		want     string
		wantErr  error
	}{
		{"custom host and path", &jiraURLRepo{slot: db.Integration{Enabled: true, Mode: &custom, Config: valid}}, incident, "https://jira.customer.test/context/browse/OPS-42", nil},
		{"missing team", &jiraURLRepo{teamErr: pgx.ErrNoRows}, incident, "", nil},
		{"team read fails", &jiraURLRepo{teamErr: failure}, incident, "", failure},
		{"missing slot", &jiraURLRepo{slotErr: pgx.ErrNoRows}, incident, "", nil},
		{"slot read fails", &jiraURLRepo{slotErr: failure}, incident, "", failure},
		{"invalid mode", &jiraURLRepo{slot: db.Integration{Enabled: true, Mode: &invalid}}, incident, "", nil},
		{"missing global", &jiraURLRepo{slot: db.Integration{Enabled: true, Mode: &inherit, Config: []byte(`{}`)}, globalErr: pgx.ErrNoRows}, incident, "", nil},
		{"global read fails", &jiraURLRepo{slot: db.Integration{Enabled: true, Mode: &inherit, Config: []byte(`{}`)}, globalErr: failure}, incident, "", failure},
		{"inherited host", &jiraURLRepo{slot: db.Integration{Enabled: true, Mode: &inherit, Config: []byte(`{}`)}, global: db.Integration{Enabled: true, Config: valid}}, incident, "https://jira.customer.test/context/browse/OPS-42", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewIncidentService(tc.repo, time.Hour, time.Minute)
			got, err := svc.JiraIssueURL(ctx, tc.incident)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.want, got)
		})
	}
	// A repository without integration lookup support cannot invent a destination.
	svc := NewIncidentService(&incidentMockRepo{}, time.Hour, time.Minute)
	got, err := svc.JiraIssueURL(ctx, incident)
	require.NoError(t, err)
	require.Empty(t, got)
}

// JIRA-LINK-AC1: reject untrusted URL components and malformed keys. This
// characterizes output only; it does not establish reachability of any host.
func TestJiraIssueURLRejectsUnsafeValues(t *testing.T) {
	mode := "custom"
	key := "OPS-42"
	for _, base := range []string{"javascript:alert(1)", "https://user:password@jira.test", "https://jira.test/?next=x", "https://jira.test/#fragment", "http://:80", "http://[invalid"} {
		t.Run(base, func(t *testing.T) {
			config := []byte(`{"base_url":` + jsonString(base) + `,"api_token":"secret","project_key":"OPS"}`)
			svc := NewIncidentService(&jiraURLRepo{slot: db.Integration{Enabled: true, Mode: &mode, Config: config}}, time.Hour, time.Minute)
			got, err := svc.JiraIssueURL(context.Background(), db.Incident{JiraIssueKey: &key})
			require.NoError(t, err)
			require.Empty(t, got)
		})
	}
	for _, key := range []string{"", "../OPS-1", "OPS/1", "OPS-1?foo"} {
		t.Run("key="+key, func(t *testing.T) {
			svc := NewIncidentService(&jiraURLRepo{}, time.Hour, time.Minute)
			got, err := svc.JiraIssueURL(context.Background(), db.Incident{JiraIssueKey: &key})
			require.NoError(t, err)
			require.Empty(t, got)
		})
	}
}

func jsonString(s string) string { b, _ := json.Marshal(s); return string(b) }
