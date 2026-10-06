package db

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

type ExpressCommandRepository interface {
	RedeemExpressLinkCode(context.Context, string, uuid.UUID) (User, error)
	GetUserByExpressHuid(context.Context, uuid.UUID) (User, error)
	GetIncidentByID(context.Context, uuid.UUID) (Incident, error)
	AcknowledgeIncident(context.Context, uuid.UUID, uuid.UUID) (Incident, error)
	CancelEscalationJobs(context.Context, uuid.UUID) error
	ListTeams(context.Context) ([]Team, error)
	CurrentOnCallUsers(context.Context, uuid.UUID, time.Time) ([]OnCallUser, error)
	EnqueueExpressReply(context.Context, uuid.UUID, string, string, string, string) error
}

func (s *Store) RunExpressCommand(ctx context.Context, integrationID uuid.UUID, commandID string, fn func(ExpressCommandRepository) (string, error)) error {
	return s.InTransaction(ctx, func(tx *Store) error {
		if _, err := tx.pool.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,1))", integrationID.String()+":"+commandID); err != nil {
			return err
		}
		var outcome string
		err := tx.pool.QueryRow(ctx, `SELECT outcome FROM express_command_receipts WHERE integration_id=$1 AND command_id=$2`, integrationID, commandID).Scan(&outcome)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		outcome, err = fn(tx)
		if err != nil {
			return err
		}
		_, err = tx.pool.Exec(ctx, `INSERT INTO express_command_receipts(integration_id,command_id,outcome) VALUES($1,$2,$3)`, integrationID, commandID, outcome)
		return err
	})
}
