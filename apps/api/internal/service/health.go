package service

import (
	"context"

	"github.com/aegis/aegis/pkg/db"
)

type HealthService struct {
	store *db.Store
}

func NewHealthService(store *db.Store) *HealthService {
	return &HealthService{store: store}
}

func (s *HealthService) Live() bool {
	return true
}

func (s *HealthService) Ready(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	return s.store.Ping(ctx)
}
