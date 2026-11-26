package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// BackupCheckReview is an on-call decision about one backup-checker run.
type BackupCheckReview struct {
	ID            uuid.UUID
	CheckID       string
	Backup        string
	Decision      string
	Comment       string
	ReviewerID    uuid.UUID
	ReviewerName  string
	ReviewerEmail string
	CreatedAt     time.Time
}

// ErrBackupReviewExists means the check run already has a decision.
var ErrBackupReviewExists = errors.New("backup check run already reviewed")

const backupReviewColumns = `r.id, r.check_id, r.backup, r.decision, r.comment, r.reviewer_id, u.display_name, u.email, r.created_at`

func scanBackupReview(row pgx.Row) (BackupCheckReview, error) {
	var r BackupCheckReview
	err := row.Scan(&r.ID, &r.CheckID, &r.Backup, &r.Decision, &r.Comment, &r.ReviewerID, &r.ReviewerName, &r.ReviewerEmail, &r.CreatedAt)
	return r, err
}

// ListBackupCheckReviews returns the reviews for the given check runs, keyed by check id.
func (s *Store) ListBackupCheckReviews(ctx context.Context, checkIDs []string) (map[string]BackupCheckReview, error) {
	out := map[string]BackupCheckReview{}
	if len(checkIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT `+backupReviewColumns+`
FROM backup_check_reviews r JOIN users u ON u.id = r.reviewer_id
WHERE r.check_id = ANY($1)`, checkIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		review, err := scanBackupReview(rows)
		if err != nil {
			return nil, err
		}
		out[review.CheckID] = review
	}
	return out, rows.Err()
}

// CreateBackupCheckReview stores the decision and its audit entry in one transaction.
func (s *Store) CreateBackupCheckReview(ctx context.Context, review BackupCheckReview) (BackupCheckReview, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BackupCheckReview{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO backup_check_reviews (check_id, backup, decision, comment, reviewer_id)
VALUES ($1, $2, $3, $4, $5) RETURNING id`, review.CheckID, review.Backup, review.Decision, review.Comment, review.ReviewerID).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return BackupCheckReview{}, ErrBackupReviewExists
	}
	if err != nil {
		return BackupCheckReview{}, err
	}
	details := map[string]any{"check_id": review.CheckID, "backup": review.Backup, "decision": review.Decision}
	if review.Comment != "" {
		details["comment"] = review.Comment
	}
	if err = writeAuditLogTx(ctx, tx, review.ReviewerID, "backup.check_"+review.Decision, "backup_check", id, details); err != nil {
		return BackupCheckReview{}, err
	}
	stored, err := scanBackupReview(tx.QueryRow(ctx, `SELECT `+backupReviewColumns+`
FROM backup_check_reviews r JOIN users u ON u.id = r.reviewer_id WHERE r.id = $1`, id))
	if err != nil {
		return BackupCheckReview{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return BackupCheckReview{}, err
	}
	return stored, nil
}
