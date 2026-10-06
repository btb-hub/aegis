package handler

import (
	"context"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"time"
)

type expressCommandTestStore struct {
	repo     *phase2HandlerRepo
	receipts map[string]bool
	replies  []string
}

func (s *expressCommandTestStore) RunExpressCommand(ctx context.Context, id uuid.UUID, command string, fn func(db.ExpressCommandRepository) (string, error)) error {
	if s.receipts == nil {
		s.receipts = map[string]bool{}
	}
	key := id.String() + command
	if s.receipts[key] {
		return nil
	}
	_, err := fn(s)
	if err == nil {
		s.receipts[key] = true
	}
	return err
}
func (s *expressCommandTestStore) HandleExpressNotificationResult(context.Context, uuid.UUID, string, string, string) error {
	return nil
}
func (s *expressCommandTestStore) RedeemExpressLinkCode(ctx context.Context, code string, id uuid.UUID) (db.User, error) {
	return s.repo.RedeemExpressLinkCode(ctx, code, id)
}
func (s *expressCommandTestStore) GetUserByExpressHuid(ctx context.Context, id uuid.UUID) (db.User, error) {
	return s.repo.GetUserByExpressHuid(ctx, id)
}
func (s *expressCommandTestStore) GetIncidentByID(ctx context.Context, id uuid.UUID) (db.Incident, error) {
	return s.repo.GetIncidentByID(ctx, id)
}
func (s *expressCommandTestStore) AcknowledgeIncident(ctx context.Context, id, actor uuid.UUID) (db.Incident, error) {
	return s.repo.AcknowledgeIncident(ctx, id, actor)
}
func (s *expressCommandTestStore) CancelEscalationJobs(ctx context.Context, id uuid.UUID) error {
	return s.repo.CancelEscalationJobs(ctx, id)
}
func (s *expressCommandTestStore) ListTeams(context.Context) ([]db.Team, error) {
	var teams []db.Team
	for _, team := range s.repo.teams {
		teams = append(teams, team)
	}
	return teams, nil
}
func (s *expressCommandTestStore) CurrentOnCallUsers(ctx context.Context, id uuid.UUID, at time.Time) ([]db.OnCallUser, error) {
	return s.repo.CurrentOnCallUsers(ctx, id, at)
}
func (s *expressCommandTestStore) EnqueueExpressReply(_ context.Context, _ uuid.UUID, _, _, body, _ string) error {
	s.replies = append(s.replies, body)
	return nil
}
