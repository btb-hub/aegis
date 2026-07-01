package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListSavedViewsForUser(ctx context.Context, userID uuid.UUID) ([]SavedView, error) {
	const q = `
SELECT id, owner_id, name, filter, shared, created_at, updated_at
FROM saved_views
WHERE owner_id = $1 OR shared = true
ORDER BY name ASC`
	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []SavedView
	for rows.Next() {
		var view SavedView
		if err := rows.Scan(&view.ID, &view.OwnerID, &view.Name, &view.Filter, &view.Shared, &view.CreatedAt, &view.UpdatedAt); err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, rows.Err()
}

func (s *Store) GetSavedView(ctx context.Context, id uuid.UUID) (SavedView, error) {
	const q = `
SELECT id, owner_id, name, filter, shared, created_at, updated_at
FROM saved_views
WHERE id = $1`
	var view SavedView
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&view.ID, &view.OwnerID, &view.Name, &view.Filter, &view.Shared, &view.CreatedAt, &view.UpdatedAt,
	)
	return view, err
}

func (s *Store) CreateSavedView(ctx context.Context, ownerID uuid.UUID, name string, filter json.RawMessage, shared bool) (SavedView, error) {
	const q = `
INSERT INTO saved_views (owner_id, name, filter, shared)
VALUES ($1, $2, $3, $4)
RETURNING id, owner_id, name, filter, shared, created_at, updated_at`
	var view SavedView
	err := s.pool.QueryRow(ctx, q, ownerID, name, filter, shared).Scan(
		&view.ID, &view.OwnerID, &view.Name, &view.Filter, &view.Shared, &view.CreatedAt, &view.UpdatedAt,
	)
	return view, err
}

func (s *Store) UpdateSavedView(ctx context.Context, id, ownerID uuid.UUID, name string, filter json.RawMessage, shared bool) (SavedView, error) {
	const q = `
UPDATE saved_views
SET name = $3, filter = $4, shared = $5, updated_at = now()
WHERE id = $1 AND owner_id = $2
RETURNING id, owner_id, name, filter, shared, created_at, updated_at`
	var view SavedView
	err := s.pool.QueryRow(ctx, q, id, ownerID, name, filter, shared).Scan(
		&view.ID, &view.OwnerID, &view.Name, &view.Filter, &view.Shared, &view.CreatedAt, &view.UpdatedAt,
	)
	return view, err
}

func (s *Store) DeleteSavedView(ctx context.Context, id, ownerID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM saved_views WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
