package service

import (
	"context"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
)

type UserListRepository interface {
	ListUsers(ctx context.Context, params db.ListUsersParams) ([]db.User, error)
	CountUsers(ctx context.Context, params db.ListUsersParams) (int, error)
	ListUserIdentitiesByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]db.UserIdentity, error)
}

type UserListResult struct {
	Items    []UserProfile
	Total    int
	Page     int
	PageSize int
}

type UserService struct {
	repo UserListRepository
}

func NewUserService(repo UserListRepository) *UserService {
	return &UserService{repo: repo}
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
