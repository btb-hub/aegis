package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/aegis/aegis/pkg/integrations/jira"
	"github.com/aegis/aegis/pkg/integrations/resolve"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type JiraSyncStore interface {
	RunJiraSync(context.Context, uuid.UUID, func(db.JiraSyncRepository) error) error
}
type JiraSyncProcessor struct {
	store     JiraSyncStore
	publicURL string
}

func NewJiraSyncProcessor(store JiraSyncStore, publicURL string) *JiraSyncProcessor {
	return &JiraSyncProcessor{store: store, publicURL: publicURL}
}
func (p *JiraSyncProcessor) Handle(ctx context.Context, job Job) error {
	var payload struct {
		ID string `json:"incident_id"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return err
	}
	id, err := uuid.Parse(payload.ID)
	if err != nil {
		return err
	}
	var syncErr error
	err = p.store.RunJiraSync(ctx, id, func(repo db.JiraSyncRepository) error {
		syncErr = p.sync(ctx, repo, id)
		// Commit attempt/reconciliation metadata even on an external failure.
		if syncErr != nil {
			raw, _ := json.Marshal(map[string]string{"provider": "jira", "message": syncErr.Error()})
			if err := repo.AppendTimelineEvent(ctx, id, "integration_failed", nil, raw); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return syncErr
}
func (p *JiraSyncProcessor) sync(ctx context.Context, repo db.JiraSyncRepository, id uuid.UUID) error {
	incident, err := repo.GetIncidentByID(ctx, id)
	if err != nil {
		return err
	}
	state, err := repo.GetJiraSyncState(ctx, id)
	if err != nil {
		return err
	}
	var connector db.Integration
	var config []byte
	if state.IntegrationID != nil {
		connector, err = repo.GetIntegration(ctx, *state.IntegrationID)
		if err != nil {
			return err
		}
		if !connector.Enabled {
			return fmt.Errorf("Jira connector is disabled; enable the originating connector")
		}
		config, err = resolve.MergeConfig(connector.Config, state.Config)
		if err != nil {
			return err
		}
	} else {
		if incident.JiraIssueKey != nil {
			return &integrations.HTTPError{Provider: "jira", Operation: "originating connector", Status: 422, Message: "existing Jira issue has no recorded connector; associate its original connector before synchronizing"}
		}
		workspaceID, err := repo.GetTeamWorkspaceID(ctx, incident.TeamID)
		if err != nil {
			return err
		}
		slot, err := repo.GetWorkspaceIntegration(ctx, workspaceID, "jira")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		global, err := repo.GetIntegrationByKind(ctx, "jira")
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var g *resolve.Slot
		if err == nil {
			g = &resolve.Slot{Enabled: global.Enabled, Config: global.Config}
		}
		mode := "inherit"
		if slot.Mode != nil {
			mode = *slot.Mode
		}
		result := resolve.Resolve(resolve.Input{Kind: "jira", Slot: &resolve.Slot{Mode: mode, Enabled: slot.Enabled, Config: slot.Config}, Global: g})
		if !result.OK {
			return nil
		}
		connector = global
		if mode == "custom" {
			connector = slot
		}
		config = result.Config
		state.IntegrationID = &connector.ID
		// Pin routing/identity metadata, while reading credentials from the original connector on each job.
		var metadata map[string]any
		if err := json.Unmarshal(config, &metadata); err != nil {
			return err
		}
		if metadata["deployment"] == nil || metadata["deployment"] == "" {
			deployment := "server_dc"
			if metadata["auth_type"] == "basic" {
				deployment = "cloud"
			}
			metadata["deployment"] = deployment
		}
		delete(metadata, "api_token")
		delete(metadata, "email")
		delete(metadata, "auth_type")
		state.Config, _ = json.Marshal(metadata)
		state.IssueKey = incident.JiraIssueKey
		if err := repo.SaveJiraSyncState(ctx, state); err != nil {
			return err
		}
	}
	provider, err := jira.NewFromJSON(config)
	if err != nil {
		return err
	}
	if state.IssueKey == nil {
		// Always reconcile first, including after a process crash before local commit.
		key, err := provider.FindTicket(ctx, id.String())
		if err != nil {
			return err
		}
		if key == "" {
			if state.CreateAttempted {
				return &integrations.HTTPError{Provider: "jira", Operation: "reconcile issue creation", Status: 503, Message: "previous create outcome is uncertain; waiting for label search to find the issue; do not recreate until Jira is checked"}
			}
			state.CreateAttempted = true
			if err := repo.SaveJiraSyncState(ctx, state); err != nil {
				return err
			}
			ref := toIncidentRef(incident)
			ref.URL = strings.TrimRight(p.publicURL, "/") + "/incidents?incident=" + id.String()
			key, err = provider.CreateTicket(ctx, ref)
			if err != nil {
				if definiteJiraRejection(err) {
					state.CreateAttempted = false
					if saveErr := repo.SaveJiraSyncState(ctx, state); saveErr != nil {
						return saveErr
					}
				}
				return err
			}
		}
		state.IssueKey = &key
		if err := repo.SaveJiraSyncState(ctx, state); err != nil {
			return err
		}
		raw, _ := json.Marshal(map[string]string{"jira_issue_key": key, "jira_issue_url": provider.IssueURL(key)})
		if err := repo.AppendTimelineEvent(ctx, id, "jira_linked", nil, raw); err != nil {
			return err
		}
	}
	if err := repo.SaveJiraIssue(ctx, id, *state.IntegrationID, *state.IssueKey, provider.IssueURL(*state.IssueKey)); err != nil {
		return err
	}
	if err := repo.SaveJiraSyncState(ctx, state); err != nil {
		return err
	}
	if incident.AssigneeID != nil && (state.AssigneeID == nil || *state.AssigneeID != *incident.AssigneeID) {
		user, err := repo.GetUserByID(ctx, *incident.AssigneeID)
		if err != nil {
			return err
		}
		if err := provider.UpdateAssignee(ctx, *state.IssueKey, user.Email); err != nil {
			return err
		}
		state.AssigneeID = incident.AssigneeID
		if err := repo.SaveJiraSyncState(ctx, state); err != nil {
			return err
		}
	}
	comments, err := repo.PendingJiraComments(ctx, id)
	if err != nil {
		return err
	}
	for _, comment := range comments {
		marker := "Aegis event: " + comment.EventID.String()
		external, err := provider.FindComment(ctx, *state.IssueKey, marker)
		if err != nil {
			return err
		}
		if external == "" {
			if comment.Attempted {
				return &integrations.HTTPError{Provider: "jira", Operation: "reconcile comment", Status: 503, Message: "previous comment outcome is uncertain; waiting for marker lookup; do not resend until Jira is checked"}
			}
			if err := repo.SaveJiraComment(ctx, comment.EventID, nil, true); err != nil {
				return err
			}
			body := fmt.Sprintf("%s (%s)\n%s\n[%s]", comment.Author, comment.CreatedAt.UTC().Format("2006-01-02 15:04 UTC"), comment.Body, marker)
			external, err = provider.AddComment(ctx, *state.IssueKey, body)
			if err != nil {
				if definiteJiraRejection(err) {
					if saveErr := repo.SaveJiraComment(ctx, comment.EventID, nil, false); saveErr != nil {
						return saveErr
					}
				}
				return err
			}
		}
		if err := repo.SaveJiraComment(ctx, comment.EventID, &external, true); err != nil {
			return err
		}
	}
	return nil
}

func definiteJiraRejection(err error) bool {
	var e *integrations.HTTPError
	return errors.As(err, &e) && e.Status < 500 && e.Status != 408
}
