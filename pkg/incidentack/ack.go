// Package incidentack exposes the shared chat acknowledgement workflow.
package incidentack

import (
	"context"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
)

type Store = db.ChatAckStore
type Result = db.ChatAckResult

func Acknowledge(ctx context.Context, s Store, id uuid.UUID, provider, identity string) (Result, error) {
	return db.AcknowledgeChat(ctx, s, id, provider, identity)
}
