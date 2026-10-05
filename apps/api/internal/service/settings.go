package service

import (
	"context"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/oncall"
	"time"
)

type SettingsRepository interface {
	GetPublicationSettings(context.Context) (db.PublicationSettings, error)
	UpdatePublicationSettings(context.Context, string, string, time.Time) (db.PublicationSettings, error)
}
type SettingsService struct{ repo SettingsRepository }

func NewSettingsService(repo SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}
func (s *SettingsService) Get(ctx context.Context) (db.PublicationSettings, error) {
	out, err := s.repo.GetPublicationSettings(ctx)
	if err != nil {
		return out, err
	}
	out.NextRunAt, err = oncall.NextPublication(time.Now(), out.Time, out.Timezone)
	return out, err
}
func (s *SettingsService) Update(ctx context.Context, clock, zone string) (db.PublicationSettings, error) {
	now := time.Now()
	if _, err := oncall.NextPublication(now, clock, zone); err != nil {
		return db.PublicationSettings{}, apperrors.Validation(err.Error(), nil)
	}
	return s.repo.UpdatePublicationSettings(ctx, clock, zone, now)
}
