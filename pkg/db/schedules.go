package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CreateScheduleLayerInput struct {
	Priority           int32
	RotationType       string
	HandoffWeekday     int32
	HandoffTime        time.Time
	ParticipantUserIDs []uuid.UUID
}

func (s *Store) ListSchedulesByTeam(ctx context.Context, teamID uuid.UUID) ([]Schedule, error) {
	const q = `
SELECT id, team_id, name, timezone, created_at, updated_at
FROM schedules
WHERE team_id = $1
ORDER BY name`
	rows, err := s.pool.Query(ctx, q, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var schedule Schedule
		if err := rows.Scan(&schedule.ID, &schedule.TeamID, &schedule.Name, &schedule.Timezone, &schedule.CreatedAt, &schedule.UpdatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}
	return schedules, rows.Err()
}

func (s *Store) ListScheduleLayers(ctx context.Context, scheduleID uuid.UUID) ([]ScheduleLayer, error) {
	const q = `
SELECT id, schedule_id, priority, rotation_type, handoff_weekday, handoff_time, participant_user_ids, created_at, updated_at
FROM schedule_layers
WHERE schedule_id = $1
ORDER BY priority, created_at`
	rows, err := s.pool.Query(ctx, q, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var layers []ScheduleLayer
	for rows.Next() {
		layer, err := scanScheduleLayer(rows)
		if err != nil {
			return nil, err
		}
		layers = append(layers, layer)
	}
	return layers, rows.Err()
}

func (s *Store) GetScheduleForTeam(ctx context.Context, teamID, scheduleID uuid.UUID) (Schedule, error) {
	const q = `
SELECT id, team_id, name, timezone, created_at, updated_at
FROM schedules
WHERE id = $1 AND team_id = $2`
	var schedule Schedule
	err := s.pool.QueryRow(ctx, q, scheduleID, teamID).Scan(
		&schedule.ID, &schedule.TeamID, &schedule.Name, &schedule.Timezone, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	return schedule, err
}

func (s *Store) CreateScheduleWithLayer(
	ctx context.Context,
	teamID uuid.UUID,
	name, timezone string,
	layer CreateScheduleLayerInput,
) (ScheduleWithLayers, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ScheduleWithLayers{}, err
	}
	defer tx.Rollback(ctx)

	var schedule Schedule
	const scheduleQ = `
INSERT INTO schedules (team_id, name, timezone)
VALUES ($1, $2, $3)
RETURNING id, team_id, name, timezone, created_at, updated_at`
	err = tx.QueryRow(ctx, scheduleQ, teamID, name, timezone).Scan(
		&schedule.ID, &schedule.TeamID, &schedule.Name, &schedule.Timezone, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		return ScheduleWithLayers{}, err
	}

	createdLayer, err := insertScheduleLayer(ctx, tx, schedule.ID, layer)
	if err != nil {
		return ScheduleWithLayers{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ScheduleWithLayers{}, err
	}
	return ScheduleWithLayers{Schedule: schedule, Layers: []ScheduleLayer{createdLayer}}, nil
}

func (s *Store) UpdateScheduleWithLayer(
	ctx context.Context,
	teamID, scheduleID uuid.UUID,
	name, timezone string,
	layer CreateScheduleLayerInput,
) (ScheduleWithLayers, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ScheduleWithLayers{}, err
	}
	defer tx.Rollback(ctx)

	var schedule Schedule
	const scheduleQ = `
UPDATE schedules
SET name = $3, timezone = $4, updated_at = now()
WHERE id = $1 AND team_id = $2
RETURNING id, team_id, name, timezone, created_at, updated_at`
	err = tx.QueryRow(ctx, scheduleQ, scheduleID, teamID, name, timezone).Scan(
		&schedule.ID, &schedule.TeamID, &schedule.Name, &schedule.Timezone, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		return ScheduleWithLayers{}, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM schedule_layers WHERE schedule_id = $1`, scheduleID); err != nil {
		return ScheduleWithLayers{}, err
	}

	createdLayer, err := insertScheduleLayer(ctx, tx, schedule.ID, layer)
	if err != nil {
		return ScheduleWithLayers{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ScheduleWithLayers{}, err
	}
	return ScheduleWithLayers{Schedule: schedule, Layers: []ScheduleLayer{createdLayer}}, nil
}

func (s *Store) DeleteScheduleForTeam(ctx context.Context, teamID, scheduleID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1 AND team_id = $2`, scheduleID, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) TeamMemberUserIDs(ctx context.Context, teamID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	const q = `SELECT user_id FROM team_memberships WHERE team_id = $1`
	rows, err := s.pool.Query(ctx, q, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := map[uuid.UUID]struct{}{}
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		ids[userID] = struct{}{}
	}
	return ids, rows.Err()
}

type scheduleLayerScanner interface {
	Scan(dest ...any) error
}

func scanScheduleLayer(row scheduleLayerScanner) (ScheduleLayer, error) {
	var layer ScheduleLayer
	err := row.Scan(
		&layer.ID, &layer.ScheduleID, &layer.Priority, &layer.RotationType, &layer.HandoffWeekday,
		&layer.HandoffTime, &layer.ParticipantUserIDs, &layer.CreatedAt, &layer.UpdatedAt,
	)
	return layer, err
}

func insertScheduleLayer(ctx context.Context, tx pgx.Tx, scheduleID uuid.UUID, layer CreateScheduleLayerInput) (ScheduleLayer, error) {
	const q = `
INSERT INTO schedule_layers (schedule_id, priority, rotation_type, handoff_weekday, handoff_time, participant_user_ids)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, schedule_id, priority, rotation_type, handoff_weekday, handoff_time, participant_user_ids, created_at, updated_at`
	row := tx.QueryRow(ctx, q, scheduleID, layer.Priority, layer.RotationType, layer.HandoffWeekday, layer.HandoffTime, layer.ParticipantUserIDs)
	return scanScheduleLayer(row)
}

func (s *Store) GetScheduleWithLayersForTeam(ctx context.Context, teamID, scheduleID uuid.UUID) (ScheduleWithLayers, error) {
	schedule, err := s.GetScheduleForTeam(ctx, teamID, scheduleID)
	if err != nil {
		return ScheduleWithLayers{}, err
	}
	layers, err := s.ListScheduleLayers(ctx, scheduleID)
	if err != nil {
		return ScheduleWithLayers{}, err
	}
	if layers == nil {
		layers = []ScheduleLayer{}
	}
	return ScheduleWithLayers{Schedule: schedule, Layers: layers}, nil
}

func (s *Store) ListSchedulesWithLayersByTeam(ctx context.Context, teamID uuid.UUID) ([]ScheduleWithLayers, error) {
	schedules, err := s.ListSchedulesByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	result := make([]ScheduleWithLayers, 0, len(schedules))
	for _, schedule := range schedules {
		layers, err := s.ListScheduleLayers(ctx, schedule.ID)
		if err != nil {
			return nil, err
		}
		if layers == nil {
			layers = []ScheduleLayer{}
		}
		result = append(result, ScheduleWithLayers{Schedule: schedule, Layers: layers})
	}
	return result, nil
}
