package service

import (
	"context"
	"net/http"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/rbac"
	"github.com/google/uuid"
)

type UserListRepository interface {
	ListUsers(ctx context.Context, params db.ListUsersParams) ([]db.User, error)
	CountUsers(ctx context.Context, params db.ListUsersParams) (int, error)
	ListUserIdentitiesByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]db.UserIdentity, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	UpdateUserRole(ctx context.Context, id uuid.UUID, role string) (db.User, error)
	CountUsersByRole(ctx context.Context, role string) (int, error)
	WriteAuditLog(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error
}

type UserListResult struct {
	Items    []UserProfile
	Total    int
	Page     int
	PageSize int
}

type UserService struct {
	repo UserListRepository
	cfg  *config.Config
}

func NewUserService(repo UserListRepository, cfg *config.Config) *UserService {
	return &UserService{repo: repo, cfg: cfg}
}

func (s *UserService) ListUsers(ctx context.Context, params db.ListUsersParams, page, pageSize int) (UserListResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = db.DefaultUserListLimit
	}
	if pageSize > db.DefaultUserListLimit {
		pageSize = db.DefaultUserListLimit
	}

	listParams := db.ListUsersParams{
		Query:  params.Query,
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}

	total, err := s.repo.CountUsers(ctx, listParams)
	if err != nil {
		return UserListResult{}, err
	}

	users, err := s.repo.ListUsers(ctx, listParams)
	if err != nil {
		return UserListResult{}, err
	}

	userIDs := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	identityMap, err := s.repo.ListUserIdentitiesByUserIDs(ctx, userIDs)
	if err != nil {
		return UserListResult{}, err
	}

	items := make([]UserProfile, 0, len(users))
	for _, user := range users {
		items = append(items, UserProfile{
			User:       user,
			Identities: identityMap[user.ID],
		})
	}

	return UserListResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// UpdateUserRole changes targetID's role, guarding against demoting the last
// remaining admin and against demoting a user pinned to admin by ADMIN_EMAILS.
func (s *UserService) UpdateUserRole(ctx context.Context, actorID, targetID uuid.UUID, role string) (db.User, error) {
	parsed, err := rbac.Parse(role)
	if err != nil {
		return db.User{}, apperrors.Validation("invalid role", map[string]any{"role": role})
	}

	target, err := s.repo.GetUserByID(ctx, targetID)
	if err != nil {
		return db.User{}, apperrors.NotFound("user")
	}

	if target.Role == string(parsed) {
		return target, nil
	}

	if s.cfg.IsAdminEmail(target.Email) && parsed != rbac.RoleAdmin {
		return db.User{}, apperrors.New("admin_emails_pinned",
			"This user is pinned to admin by ADMIN_EMAILS. Remove the email from ADMIN_EMAILS and restart the API, then demote.",
			http.StatusConflict)
	}

	if target.Role == string(rbac.RoleAdmin) && parsed != rbac.RoleAdmin {
		n, err := s.repo.CountUsersByRole(ctx, string(rbac.RoleAdmin))
		if err != nil {
			return db.User{}, err
		}
		if n <= 1 {
			return db.User{}, apperrors.New("last_admin", "Cannot demote the last admin", http.StatusConflict)
		}
	}

	oldRole := target.Role
	updated, err := s.repo.UpdateUserRole(ctx, targetID, string(parsed))
	if err != nil {
		return db.User{}, err
	}

	if err := s.repo.WriteAuditLog(ctx, &actorID, "user.role_changed", "user", targetID, map[string]any{
		"old_role": oldRole,
		"new_role": string(parsed),
		"reason":   "admin_api",
	}); err != nil {
		return db.User{}, err
	}

	return updated, nil
}

// IsRolePinned reports whether email is pinned to the admin role by ADMIN_EMAILS.
func (s *UserService) IsRolePinned(email string) bool {
	return s.cfg.IsAdminEmail(email)
}
