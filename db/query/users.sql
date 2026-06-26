-- name: UpsertUser :one
INSERT INTO users (provider, provider_sub, email, display_name, role, locale)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (provider, provider_sub) DO UPDATE
SET email = EXCLUDED.email,
    display_name = EXCLUDED.display_name
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUserLocale :one
UPDATE users SET locale = $2 WHERE id = $1 RETURNING *;
