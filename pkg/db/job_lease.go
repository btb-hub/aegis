package db

import (
	"context"
	"github.com/google/uuid"
)

func (s *Store) RescheduleJob(ctx context.Context, id uuid.UUID, message string) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status=CASE WHEN attempts<4 THEN 'pending' ELSE 'failed' END,run_at=now()+CASE attempts WHEN 1 THEN interval '1 minute' WHEN 2 THEN interval '5 minutes' ELSE interval '15 minutes' END,last_error=$2,updated_at=now() WHERE id=$1`, id, message)
	return err
}
func (s *Store) RenewClaimedJob(ctx context.Context, id uuid.UUID, attempt int32) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE jobs SET updated_at=now() WHERE id=$1 AND attempts=$2 AND status='running'`, id, attempt)
	return tag.RowsAffected() > 0, err
}
func (s *Store) FinishClaimedJob(ctx context.Context, id uuid.UUID, attempt int32, message string, success, retry bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status=CASE WHEN $4 THEN 'done' WHEN $5 AND attempts<4 THEN 'pending' ELSE 'failed' END,last_error=NULLIF($3,''),run_at=CASE WHEN $5 THEN now()+CASE attempts WHEN 1 THEN interval '1 minute' WHEN 2 THEN interval '5 minutes' ELSE interval '15 minutes' END ELSE run_at END,updated_at=now() WHERE id=$1 AND attempts=$2 AND status='running'`, id, attempt, message, success, retry)
	return err
}
