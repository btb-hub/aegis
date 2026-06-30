package loader

import (
	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/aegis/aegis/pkg/integrations"
	intjira "github.com/aegis/aegis/pkg/integrations/jira"
	intslack "github.com/aegis/aegis/pkg/integrations/slack"
)

func RegisterFromRows(reg *integrations.Registry, rows []integrations.IntegrationRow, publicURL string) {
	for _, row := range rows {
		if !row.Enabled {
			continue
		}
		switch row.Kind {
		case "jira":
			provider, err := intjira.NewFromJSON(row.Config)
			if err != nil {
				continue
			}
			reg.RegisterTicket(provider)
		case "slack":
			provider, err := intslack.NewFromJSON(row.Config, publicURL)
			if err != nil {
				continue
			}
			reg.RegisterChat(provider)
		case "express":
			provider, err := intexpress.NewFromJSON(row.Config)
			if err != nil {
				continue
			}
			reg.RegisterChat(provider)
		}
	}
}
