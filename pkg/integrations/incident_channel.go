package integrations

import (
	"context"
	"fmt"
	"github.com/aegis/aegis/pkg/i18n"
	"strings"
	"time"
)

type IncidentChannelPost struct {
	Incident                                            IncidentRef
	Kind, TeamName, ActorName, Locale, SlackUserGroupID string
	OccurredAt                                          time.Time
	OnCall                                              []OnCallPerson
	Actionable                                          bool
}

type IncidentChannelProvider interface {
	SendIncidentEvent(context.Context, IncidentChannelPost, string) (string, error)
}

func IncidentEventBody(p IncidentChannelPost) string {
	body := i18n.T(p.Locale, "incident_chat."+p.Kind, map[string]string{
		"id": p.Incident.ID.String(), "severity": ClipText(p.Incident.Severity, 100), "title": ClipText(p.Incident.Title, 1000),
		"team": ClipText(p.TeamName, 250), "actor": ClipText(p.ActorName, 250), "time": p.OccurredAt.UTC().Format(time.RFC3339), "url": p.Incident.URL})
	return body
}

func ClipText(text string, limit int) string {
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit-3]) + "..."
	}
	return text
}

func IncidentURL(publicURL string, id string) string {
	return fmt.Sprintf("%s/incidents?incident_id=%s", strings.TrimRight(publicURL, "/"), id)
}
