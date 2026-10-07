// Shared idempotent acknowledgement workflow for API and background processing.
package db

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ChatAckStore interface {
	GetUserBySlackID(context.Context, string) (User, error)
	GetUserByExpressHuid(context.Context, uuid.UUID) (User, error)
	GetIncidentByID(context.Context, uuid.UUID) (Incident, error)
	AcknowledgeIncident(context.Context, uuid.UUID, uuid.UUID) (Incident, error)
	CancelEscalationJobs(context.Context, uuid.UUID) error
}

type ChatAckResult struct {
	Code     string   `json:"code"`
	Locale   string   `json:"locale"`
	Incident Incident `json:"incident"`
}

func AcknowledgeChat(ctx context.Context, s ChatAckStore, id uuid.UUID, provider, identity string) (ChatAckResult, error) {
	var user User
	var err error
	if provider == "slack" {
		user, err = s.GetUserBySlackID(ctx, identity)
	} else {
		var huid uuid.UUID
		huid, err = ParseExpressHuid(identity)
		if err == nil {
			user, err = s.GetUserByExpressHuid(ctx, huid)
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ChatAckResult{Code: "link_required"}, nil
	}
	if err != nil {
		return ChatAckResult{}, err
	}
	incident, err := s.GetIncidentByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChatAckResult{Code: "not_found", Locale: user.Locale}, nil
	}
	if err != nil {
		return ChatAckResult{}, err
	}
	code := "already_" + incident.Status
	if incident.Status == "open" {
		incident, err = s.AcknowledgeIncident(ctx, id, user.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			incident, err = s.GetIncidentByID(ctx, id)
			code = "already_" + incident.Status
		} else {
			code = "acknowledged"
		}
		if err != nil {
			return ChatAckResult{}, err
		}
	}
	if incident.Status == "acknowledged" {
		if err = s.CancelEscalationJobs(ctx, id); err != nil {
			return ChatAckResult{}, err
		}
	}
	return ChatAckResult{Code: code, Locale: user.Locale, Incident: incident}, nil
}
