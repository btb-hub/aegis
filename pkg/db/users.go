package db

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

const DefaultUserListLimit = 100

type ListUsersParams struct {
	Query  string
	Limit  int
	Offset int
}

func normalizeListUsersParams(params ListUsersParams) ListUsersParams {
	limit := params.Limit
	if limit < 1 {
		limit = DefaultUserListLimit
	}
	if limit > DefaultUserListLimit {
		limit = DefaultUserListLimit
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}
	return ListUsersParams{
		Query:  strings.TrimSpace(params.Query),
		Limit:  limit,
		Offset: offset,
	}
}

func userListSearchPattern(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	var b strings.Builder
	b.WriteByte('%')
	for _, r := range query {
		switch r {
		case '%', '_', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('%')
	return b.String()
}

func (s *Store) CountUsers(ctx context.Context, params ListUsersParams) (int, error) {
	params = normalizeListUsersParams(params)
	pattern := userListSearchPattern(params.Query)

	var count int
	var err error
	if pattern == "" {
		err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	} else {
		err = s.pool.QueryRow(ctx, `
SELECT COUNT(*) FROM users
WHERE email ILIKE $1 ESCAPE E'\\' OR display_name ILIKE $1 ESCAPE E'\\'`,
			pattern,
		).Scan(&count)
	}
	return count, err
}

func (s *Store) ListUsers(ctx context.Context, params ListUsersParams) ([]User, error) {
	params = normalizeListUsersParams(params)
	pattern := userListSearchPattern(params.Query)

	var rows pgx.Rows
	var err error
	if pattern == "" {
		rows, err = s.pool.Query(ctx, `
SELECT `+userSelectColumns+` FROM users
ORDER BY display_name ASC, id ASC
LIMIT $1 OFFSET $2`, params.Limit, params.Offset)
	} else {
		rows, err = s.pool.Query(ctx, `
SELECT `+userSelectColumns+` FROM users
WHERE email ILIKE $1 ESCAPE E'\\' OR display_name ILIKE $1 ESCAPE E'\\'
ORDER BY display_name ASC, id ASC
LIMIT $2 OFFSET $3`, pattern, params.Limit, params.Offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
