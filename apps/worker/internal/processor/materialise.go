package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type MaterialiseStore interface {
	MaterialiseOnCallForTeam(ctx context.Context, teamID uuid.UUID) error
	ListTeamIDsWithSchedules(ctx context.Context) ([]uuid.UUID, error)
}

type MaterialiseProcessor struct {
	log   *slog.Logger
	store MaterialiseStore
}

func NewMaterialiseProcessor(log *slog.Logger, store MaterialiseStore) *MaterialiseProcessor {
	if log == nil {
		log = slog.Default()
	}
	return &MaterialiseProcessor{log: log, store: store}
}

func (p *MaterialiseProcessor) Handle(ctx context.Context, job Job) error {
	var payload struct {
		TeamID string `json:"team_id"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	if payload.TeamID == "" {
		teamIDs, err := p.store.ListTeamIDsWithSchedules(ctx)
		if err != nil {
			return err
		}
		for _, teamID := range teamIDs {
			if err := p.store.MaterialiseOnCallForTeam(ctx, teamID); err != nil {
				return err
			}
		}
		p.log.Info("materialise_oncall nightly", "teams", len(teamIDs))
		return nil
	}
	teamID, err := uuid.Parse(payload.TeamID)
	if err != nil {
		return fmt.Errorf("invalid team_id: %w", err)
	}
	if err := p.store.MaterialiseOnCallForTeam(ctx, teamID); err != nil {
		return err
	}
	p.log.Info("materialise_oncall", "team_id", teamID.String())
	return nil
}
