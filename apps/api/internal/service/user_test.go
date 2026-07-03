package service

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type userListRepoMock struct {
	users      []db.User
	identities map[uuid.UUID][]db.UserIdentity
}

func (m *userListRepoMock) ListUsers(ctx context.Context, params db.ListUsersParams) ([]db.User, error) {
	query := strings.ToLower(strings.TrimSpace(params.Query))
	var filtered []db.User
	for _, user := range m.users {
		if query == "" ||
			strings.Contains(strings.ToLower(user.Email), query) ||
			strings.Contains(strings.ToLower(user.DisplayName), query) {
			filtered = append(filtered, user)
		}
	}
	sortUsersByDisplayName(filtered)
	start := params.Offset
	if start > len(filtered) {
		return []db.User{}, nil
	}
	end := start + params.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	slice := filtered[start:end]
	return slice, nil
}

func (m *userListRepoMock) CountUsers(ctx context.Context, params db.ListUsersParams) (int, error) {
	items, err := m.ListUsers(ctx, db.ListUsersParams{Query: params.Query, Limit: len(m.users), Offset: 0})
	return len(items), err
}

func (m *userListRepoMock) ListUserIdentitiesByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]db.UserIdentity, error) {
	out := make(map[uuid.UUID][]db.UserIdentity, len(userIDs))
	for _, userID := range userIDs {
		if identities, ok := m.identities[userID]; ok {
			out[userID] = identities
		}
	}
	return out, nil
}

func sortUsersByDisplayName(users []db.User) {
	sort.Slice(users, func(i, j int) bool {
		if users[i].DisplayName == users[j].DisplayName {
			return users[i].ID.String() < users[j].ID.String()
		}
		return users[i].DisplayName < users[j].DisplayName
	})
}

func TestUserServiceListUsers(t *testing.T) {
	aliceID := uuid.New()
	bobID := uuid.New()
	repo := &userListRepoMock{
		users: []db.User{
			{ID: bobID, Email: "bob@example.com", DisplayName: "Bob"},
			{ID: aliceID, Email: "alice@example.com", DisplayName: "Alice"},
		},
		identities: map[uuid.UUID][]db.UserIdentity{
			aliceID: {{Provider: "google", ProviderSub: "g-1"}},
		},
	}
	svc := NewUserService(repo)

	result, err := svc.ListUsers(context.Background(), db.ListUsersParams{}, 1, 10)
	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
	require.Len(t, result.Items, 2)
	require.Equal(t, "Alice", result.Items[0].User.DisplayName)
	require.Equal(t, "google", result.Items[0].Identities[0].Provider)

	filtered, err := svc.ListUsers(context.Background(), db.ListUsersParams{Query: "bob"}, 1, 10)
	require.NoError(t, err)
	require.Equal(t, 1, filtered.Total)
	require.Equal(t, "Bob", filtered.Items[0].User.DisplayName)
}

func TestUserServiceListUsersNormalizesPagination(t *testing.T) {
	repo := &userListRepoMock{
		users:      []db.User{{ID: uuid.New(), DisplayName: "Only"}},
		identities: map[uuid.UUID][]db.UserIdentity{},
	}
	svc := NewUserService(repo)

	result, err := svc.ListUsers(context.Background(), db.ListUsersParams{}, 0, 500)
	require.NoError(t, err)
	require.Equal(t, 1, result.Page)
	require.Equal(t, db.DefaultUserListLimit, result.PageSize)
}

func TestUserServiceListUsersPagination(t *testing.T) {
	repo := &userListRepoMock{
		users: []db.User{
			{ID: uuid.New(), DisplayName: "A"},
			{ID: uuid.New(), DisplayName: "B"},
			{ID: uuid.New(), DisplayName: "C"},
		},
		identities: map[uuid.UUID][]db.UserIdentity{},
	}
	svc := NewUserService(repo)

	page1, err := svc.ListUsers(context.Background(), db.ListUsersParams{}, 1, 2)
	require.NoError(t, err)
	require.Equal(t, 3, page1.Total)
	require.Len(t, page1.Items, 2)

	page2, err := svc.ListUsers(context.Background(), db.ListUsersParams{}, 2, 2)
	require.NoError(t, err)
	require.Len(t, page2.Items, 1)
}
