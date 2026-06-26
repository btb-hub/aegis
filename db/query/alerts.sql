-- name: CreateAlert :one
INSERT INTO alerts (fingerprint, status, severity, title, body, labels, raw_payload, search_tsv)
VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    to_tsvector('english', coalesce($4, '') || ' ' || coalesce($5, ''))
)
RETURNING *;

-- name: GetAlertByID :one
SELECT * FROM alerts WHERE id = $1;
