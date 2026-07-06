package db

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	const q = `
SELECT id, name, slug, description, created_at, updated_at
FROM workspaces
ORDER BY name`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Workspace
	for rows.Next() {
		var item Workspace
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetWorkspace(ctx context.Context, id uuid.UUID) (Workspace, error) {
	const q = `
SELECT id, name, slug, description, created_at, updated_at
FROM workspaces
WHERE id = $1`
	var item Workspace
	err := s.pool.QueryRow(ctx, q, id).Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) CreateWorkspace(ctx context.Context, name, slug, description string) (Workspace, error) {
	const q = `
INSERT INTO workspaces (name, slug, description)
VALUES ($1, $2, $3)
RETURNING id, name, slug, description, created_at, updated_at`
	var item Workspace
	err := s.pool.QueryRow(ctx, q, name, slug, description).Scan(
		&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Store) UpdateWorkspace(ctx context.Context, id uuid.UUID, name, slug, description string) (Workspace, error) {
	const q = `
UPDATE workspaces
SET name = $2, slug = $3, description = $4, updated_at = now()
WHERE id = $1
RETURNING id, name, slug, description, created_at, updated_at`
	var item Workspace
	err := s.pool.QueryRow(ctx, q, id, name, slug, description).Scan(
		&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Store) DeleteWorkspace(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM workspaces WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func Slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	var b strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "workspace"
	}
	return out
}
