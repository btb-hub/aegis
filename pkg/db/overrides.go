package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListOverridesByTeam(ctx context.Context, teamID uuid.UUID) ([]Override, error) {
	const q = `
SELECT id, team_id, user_id, start_at, end_at, created_at
FROM overrides
WHERE team_id = $1
ORDER BY start_at`
	rows, err := s.pool.Query(ctx, q, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var overrides []Override
	for rows.Next() {
		var override Override
		if err := rows.Scan(&override.ID, &override.TeamID, &override.UserID, &override.StartAt, &override.EndAt, &override.CreatedAt); err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}
	return overrides, rows.Err()
}

func (s *Store) GetOverrideForTeam(ctx context.Context, teamID, overrideID uuid.UUID) (Override, error) {
	const q = `
SELECT id, team_id, user_id, start_at, end_at, created_at
FROM overrides
WHERE id = $1 AND team_id = $2`
	var override Override
	err := s.pool.QueryRow(ctx, q, overrideID, teamID).Scan(
		&override.ID, &override.TeamID, &override.UserID, &override.StartAt, &override.EndAt, &override.CreatedAt,
	)
	return override, err
}

func (s *Store) CreateOverride(ctx context.Context, teamID, userID uuid.UUID, startAt, endAt time.Time) (Override, error) {
	const q = `
INSERT INTO overrides (team_id, user_id, start_at, end_at)
VALUES ($1, $2, $3, $4)
RETURNING id, team_id, user_id, start_at, end_at, created_at`
	var override Override
	err := s.pool.QueryRow(ctx, q, teamID, userID, startAt, endAt).Scan(
		&override.ID, &override.TeamID, &override.UserID, &override.StartAt, &override.EndAt, &override.CreatedAt,
	)
	return override, err
}

func (s *Store) DeleteOverrideForTeam(ctx context.Context, teamID, overrideID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM overrides WHERE id = $1 AND team_id = $2`, overrideID, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) ReplaceOnCallSlots(ctx context.Context, teamID uuid.UUID, slots []OnCallSlot) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM on_call_slots WHERE team_id = $1`, teamID); err != nil {
		return err
	}
	const insertQ = `
INSERT INTO on_call_slots (team_id, user_id, start_at, end_at, source)
VALUES ($1, $2, $3, $4, $5)`
	for _, slot := range slots {
		if _, err := tx.Exec(ctx, insertQ, teamID, slot.UserID, slot.StartAt, slot.EndAt, slot.Source); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ListOnCallSlotsInRange(ctx context.Context, teamID uuid.UUID, from, to time.Time) ([]OnCallSlot, error) {
	const q = `
SELECT id, team_id, user_id, start_at, end_at, source, created_at
FROM on_call_slots
WHERE team_id = $1 AND start_at < $3 AND end_at > $2
ORDER BY start_at`
	rows, err := s.pool.Query(ctx, q, teamID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []OnCallSlot
	for rows.Next() {
		var slot OnCallSlot
		if err := rows.Scan(&slot.ID, &slot.TeamID, &slot.UserID, &slot.StartAt, &slot.EndAt, &slot.Source, &slot.CreatedAt); err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}

func (s *Store) CurrentOnCallUsers(ctx context.Context, teamID uuid.UUID, at time.Time) ([]OnCallUser, error) {
	const q = `
SELECT s.user_id, u.email, u.display_name, s.source
FROM on_call_slots s
JOIN users u ON u.id = s.user_id
WHERE s.team_id = $1 AND s.start_at <= $2 AND s.end_at > $2
ORDER BY s.source DESC, u.display_name`
	rows, err := s.pool.Query(ctx, q, teamID, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []OnCallUser
	for rows.Next() {
		var user OnCallUser
		if err := rows.Scan(&user.UserID, &user.Email, &user.DisplayName, &user.Source); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) EnqueueJob(ctx context.Context, kind string, payload []byte, runAt time.Time) (uuid.UUID, error) {
	const q = `INSERT INTO jobs (kind, payload, status, run_at) VALUES ($1, $2, 'pending', $3) RETURNING id`
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, q, kind, payload, runAt).Scan(&id)
	return id, err
}

func (s *Store) EnqueueMaterialiseOnCall(ctx context.Context, teamID uuid.UUID) error {
	payload := []byte(`{"team_id":"` + teamID.String() + `"}`)
	_, err := s.EnqueueJob(ctx, "materialise_oncall", payload, time.Now())
	return err
}
