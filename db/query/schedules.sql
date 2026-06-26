-- name: ListSchedulesByTeam :many
SELECT id, team_id, name, timezone, created_at, updated_at
FROM schedules
WHERE team_id = $1
ORDER BY name;

-- name: GetSchedule :one
SELECT id, team_id, name, timezone, created_at, updated_at
FROM schedules
WHERE id = $1;

-- name: GetScheduleForTeam :one
SELECT id, team_id, name, timezone, created_at, updated_at
FROM schedules
WHERE id = $1 AND team_id = $2;

-- name: CreateSchedule :one
INSERT INTO schedules (team_id, name, timezone)
VALUES ($1, $2, $3)
RETURNING id, team_id, name, timezone, created_at, updated_at;

-- name: UpdateSchedule :one
UPDATE schedules
SET name = $2, timezone = $3, updated_at = now()
WHERE id = $1
RETURNING id, team_id, name, timezone, created_at, updated_at;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = $1;

-- name: ListScheduleLayers :many
SELECT id, schedule_id, priority, rotation_type, handoff_weekday, handoff_time, participant_user_ids, created_at, updated_at
FROM schedule_layers
WHERE schedule_id = $1
ORDER BY priority, created_at;

-- name: CreateScheduleLayer :one
INSERT INTO schedule_layers (schedule_id, priority, rotation_type, handoff_weekday, handoff_time, participant_user_ids)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, schedule_id, priority, rotation_type, handoff_weekday, handoff_time, participant_user_ids, created_at, updated_at;

-- name: UpdateScheduleLayer :one
UPDATE schedule_layers
SET priority = $2, rotation_type = $3, handoff_weekday = $4, handoff_time = $5, participant_user_ids = $6, updated_at = now()
WHERE id = $7
RETURNING id, schedule_id, priority, rotation_type, handoff_weekday, handoff_time, participant_user_ids, created_at, updated_at;

-- name: DeleteScheduleLayers :exec
DELETE FROM schedule_layers WHERE schedule_id = $1;
