package service

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type auditLogEntry struct {
	ActorID      *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   uuid.UUID
	Details      map[string]any
}

type userListRepoMock struct {
	users      []db.User
	identities map[uuid.UUID][]db.UserIdentity
	auditLogs  []auditLogEntry

	countUsersByRoleErr error
	updateUserRoleErr   error
	writeAuditLogErr    error
}

func (m *userListRepoMock) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return db.User{}, errors.New("user not found")
}

func (m *userListRepoMock) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) (db.User, error) {
	if m.updateUserRoleErr != nil {
		return db.User{}, m.updateUserRoleErr
	}
	for i, user := range m.users {
		if user.ID == id {
			m.users[i].Role = role
			return m.users[i], nil
		}
	}
	return db.User{}, errors.New("user not found")
}

func (m *userListRepoMock) CountUsersByRole(ctx context.Context, role string) (int, error) {
	if m.countUsersByRoleErr != nil {
		return 0, m.countUsersByRoleErr
	}
	n := 0
	for _, user := range m.users {
		if user.Role == role {
			n++
		}
	}
	return n, nil
}

func (m *userListRepoMock) WriteAuditLog(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error {
	if m.writeAuditLogErr != nil {
		return m.writeAuditLogErr
	}
	m.auditLogs = append(m.auditLogs, auditLogEntry{
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
	})
	return nil
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
	svc := NewUserService(repo, &config.Config{})

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
	svc := NewUserService(repo, &config.Config{})

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
	svc := NewUserService(repo, &config.Config{})

	page1, err := svc.ListUsers(context.Background(), db.ListUsersParams{}, 1, 2)
	require.NoError(t, err)
	require.Equal(t, 3, page1.Total)
	require.Len(t, page1.Items, 2)

	page2, err := svc.ListUsers(context.Background(), db.ListUsersParams{}, 2, 2)
	require.NoError(t, err)
	require.Len(t, page2.Items, 1)
}

func TestUpdateUserRolePromotes(t *testing.T) {
	actorID := uuid.New()
	targetID := uuid.New()
	repo := &userListRepoMock{
		users: []db.User{{ID: targetID, Email: "member@example.com", Role: "member"}},
	}
	svc := NewUserService(repo, &config.Config{})

	updated, err := svc.UpdateUserRole(context.Background(), actorID, targetID, "admin")
	require.NoError(t, err)
	require.Equal(t, "admin", updated.Role)

	require.Len(t, repo.auditLogs, 1)
	entry := repo.auditLogs[0]
	require.Equal(t, "user.role_changed", entry.Action)
	require.Equal(t, "user", entry.ResourceType)
	require.Equal(t, targetID, entry.ResourceID)
	require.Equal(t, &actorID, entry.ActorID)
	require.Equal(t, "member", entry.Details["old_role"])
	require.Equal(t, "admin", entry.Details["new_role"])
	require.Equal(t, "admin_api", entry.Details["reason"])
}

func TestUpdateUserRoleLastAdmin(t *testing.T) {
	actorID := uuid.New()
	targetID := uuid.New()
	repo := &userListRepoMock{
		users: []db.User{{ID: targetID, Email: "admin@example.com", Role: "admin"}},
	}
	svc := NewUserService(repo, &config.Config{})

	_, err := svc.UpdateUserRole(context.Background(), actorID, targetID, "member")
	require.Error(t, err)

	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "last_admin", appErr.Code)
	require.Equal(t, http.StatusConflict, appErr.StatusCode)
	require.Empty(t, repo.auditLogs)
}

func TestUpdateUserRolePinnedByEnv(t *testing.T) {
	actorID := uuid.New()
	targetID := uuid.New()
	repo := &userListRepoMock{
		users: []db.User{{ID: targetID, Email: "pinned@example.com", Role: "admin"}},
	}
	cfg := &config.Config{AdminEmails: map[string]struct{}{"pinned@example.com": {}}}
	svc := NewUserService(repo, cfg)

	_, err := svc.UpdateUserRole(context.Background(), actorID, targetID, "member")
	require.Error(t, err)

	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "admin_emails_pinned", appErr.Code)
	require.Equal(t, http.StatusConflict, appErr.StatusCode)
	require.Empty(t, repo.auditLogs)
}

func TestUpdateUserRoleIdempotent(t *testing.T) {
	actorID := uuid.New()
	targetID := uuid.New()
	repo := &userListRepoMock{
		users: []db.User{{ID: targetID, Email: "member@example.com", Role: "member"}},
	}
	svc := NewUserService(repo, &config.Config{})

	updated, err := svc.UpdateUserRole(context.Background(), actorID, targetID, "member")
	require.NoError(t, err)
	require.Equal(t, "member", updated.Role)
	require.Empty(t, repo.auditLogs)
}

func TestUpdateUserRoleInvalid(t *testing.T) {
	actorID := uuid.New()
	targetID := uuid.New()
	repo := &userListRepoMock{
		users: []db.User{{ID: targetID, Email: "member@example.com", Role: "member"}},
	}
	svc := NewUserService(repo, &config.Config{})

	_, err := svc.UpdateUserRole(context.Background(), actorID, targetID, "superuser")
	require.Error(t, err)

	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "VALIDATION_ERROR", appErr.Code)
	require.Empty(t, repo.auditLogs)
}

func TestUpdateUserRoleTargetNotFound(t *testing.T) {
	repo := &userListRepoMock{}
	svc := NewUserService(repo, &config.Config{})

	_, err := svc.UpdateUserRole(context.Background(), uuid.New(), uuid.New(), "admin")
	require.Error(t, err)

	var appErr *apperrors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "NOT_FOUND", appErr.Code)
}

func TestUpdateUserRoleCountUsersByRoleError(t *testing.T) {
	targetID := uuid.New()
	repo := &userListRepoMock{
		users:               []db.User{{ID: targetID, Email: "admin@example.com", Role: "admin"}},
		countUsersByRoleErr: errors.New("db down"),
	}
	svc := NewUserService(repo, &config.Config{})

	_, err := svc.UpdateUserRole(context.Background(), uuid.New(), targetID, "member")
	require.EqualError(t, err, "db down")
}

func TestUpdateUserRoleUpdateError(t *testing.T) {
	targetID := uuid.New()
	repo := &userListRepoMock{
		users:             []db.User{{ID: targetID, Email: "member@example.com", Role: "member"}},
		updateUserRoleErr: errors.New("update failed"),
	}
	svc := NewUserService(repo, &config.Config{})

	_, err := svc.UpdateUserRole(context.Background(), uuid.New(), targetID, "admin")
	require.EqualError(t, err, "update failed")
}

func TestUpdateUserRoleWriteAuditLogError(t *testing.T) {
	targetID := uuid.New()
	repo := &userListRepoMock{
		users:            []db.User{{ID: targetID, Email: "member@example.com", Role: "member"}},
		writeAuditLogErr: errors.New("audit failed"),
	}
	svc := NewUserService(repo, &config.Config{})

	_, err := svc.UpdateUserRole(context.Background(), uuid.New(), targetID, "admin")
	require.EqualError(t, err, "audit failed")
}

func TestIsRolePinned(t *testing.T) {
	cfg := &config.Config{AdminEmails: map[string]struct{}{"pinned@example.com": {}}}
	svc := NewUserService(&userListRepoMock{}, cfg)

	require.True(t, svc.IsRolePinned("pinned@example.com"))
	require.False(t, svc.IsRolePinned("other@example.com"))
}
