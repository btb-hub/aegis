-- name: CreateJob :one
INSERT INTO jobs (kind, payload, status, run_at)
VALUES ($1, $2, 'pending', $3)
RETURNING *;

-- name: ClaimNextJob :one
UPDATE jobs
SET status = 'running', updated_at = now(), attempts = attempts + 1
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'pending' AND run_at <= now()
    ORDER BY run_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: CompleteJob :exec
UPDATE jobs SET status = 'done', updated_at = now(), last_error = NULL WHERE id = $1;

-- name: FailJob :exec
UPDATE jobs SET status = 'failed', updated_at = now(), last_error = $2 WHERE id = $1;

-- name: RequeueJob :exec
UPDATE jobs
SET status = 'pending', updated_at = now(), run_at = $2, last_error = $3
WHERE id = $1;
