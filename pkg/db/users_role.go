package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *Store) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) (User, error) {
	q := `UPDATE users SET role = $2 WHERE id = $1 RETURNING ` + userSelectColumns
	return scanUser(s.pool.QueryRow(ctx, q, id, role))
}

func (s *Store) CountUsersByRole(ctx context.Context, role string) (int, error) {
	const q = `SELECT COUNT(*) FROM users WHERE role = $1`
	var n int
	err := s.pool.QueryRow(ctx, q, role).Scan(&n)
	return n, err
}

func (s *Store) WriteAuditLog(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO audit_log (actor_id, action, resource_type, resource_id, details)
VALUES ($1, $2, $3, $4, $5)`, actorID, action, resourceType, resourceID, payload)
	return err
}
