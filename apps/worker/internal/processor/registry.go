package processor

import (
	"context"
	"fmt"

	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/integrations"
	"github.com/aegis/aegis/pkg/integrations/loader"
	"github.com/aegis/aegis/pkg/integrations/resolve"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type workspaceRegistryStore interface {
	GetTeamWorkspaceID(ctx context.Context, teamID uuid.UUID) (uuid.UUID, error)
	GetWorkspaceIntegration(ctx context.Context, workspaceID uuid.UUID, kind string) (db.Integration, error)
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
}

type skipNotice struct {
	Kind   string
	Reason string
}

func loadWorkspaceRegistry(
	ctx context.Context,
	store workspaceRegistryStore,
	teamID uuid.UUID,
	publicURL string,
) (*integrations.Registry, []skipNotice, error) {
	workspaceID, err := store.GetTeamWorkspaceID(ctx, teamID)
	if err != nil {
		return nil, nil, err
	}

	rows := make([]integrations.IntegrationRow, 0, 3)
	notices := make([]skipNotice, 0, 3)
	for _, kind := range []string{"jira", "slack", "express"} {
		var slot *resolve.Slot
		workspaceIntegration, slotErr := store.GetWorkspaceIntegration(ctx, workspaceID, kind)
		if slotErr == nil {
			mode := "inherit"
			if workspaceIntegration.Mode != nil {
				mode = *workspaceIntegration.Mode
			}
			slot = &resolve.Slot{
				Mode:    mode,
				Enabled: workspaceIntegration.Enabled,
				Config:  workspaceIntegration.Config,
			}
		} else if slotErr != pgx.ErrNoRows {
			return nil, nil, slotErr
		}

		var global *resolve.Slot
		globalIntegration, globalErr := store.GetIntegrationByKind(ctx, kind)
		if globalErr == nil {
			global = &resolve.Slot{
				Enabled: globalIntegration.Enabled,
				Config:  globalIntegration.Config,
			}
		} else if globalErr != pgx.ErrNoRows {
			return nil, nil, globalErr
		}

		result := resolve.Resolve(resolve.Input{Kind: kind, Slot: slot, Global: global})
		if !result.OK {
			notices = append(notices, skipNotice{Kind: kind, Reason: result.Reason})
			continue
		}
		rows = append(rows, integrations.IntegrationRow{
			ID:      workspaceIntegration.ID,
			Kind:    kind,
			Config:  result.Config,
			Enabled: true,
		})
	}

	reg := integrations.NewRegistry()
	loader.RegisterFromRows(reg, rows, publicURL)
	return reg, notices, nil
}

func skipMessage(notice skipNotice) string {
	name := map[string]string{
		"jira":    "Jira",
		"slack":   "Slack",
		"express": "eXpress",
	}[notice.Kind]
	if name == "" {
		name = notice.Kind
	}

	switch notice.Reason {
	case resolve.ReasonNoGlobal:
		return fmt.Sprintf("%s skipped: no global connector. Configure global %s or set the workspace slot to Custom.", name, name)
	case resolve.ReasonGlobalDisabled:
		return fmt.Sprintf("%s skipped: the global connector is disabled. Enable global %s or set the workspace slot to Custom.", name, name)
	case resolve.ReasonSlotDisabled:
		return fmt.Sprintf("%s skipped: the workspace slot is disabled. Enable it in workspace Integrations.", name)
	case resolve.ReasonSlotMissing:
		return fmt.Sprintf("%s skipped: the workspace slot is missing. Configure it in workspace Integrations.", name)
	case resolve.ReasonCustomIncomplete:
		return fmt.Sprintf("%s skipped: the workspace connector is incomplete. Complete its settings in workspace Integrations.", name)
	default:
		return fmt.Sprintf("%s skipped: the connector is unavailable. Check workspace Integrations.", name)
	}
}
