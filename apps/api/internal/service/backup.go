package service

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/backupchecker"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
)

const (
	BackupDecisionConfirmed = "confirmed"
	BackupDecisionRejected  = "rejected"
	maxBackupCommentLength  = 500
)

type BackupReviewRepository interface {
	ListBackupCheckReviews(ctx context.Context, checkIDs []string) (map[string]db.BackupCheckReview, error)
	CreateBackupCheckReview(ctx context.Context, review db.BackupCheckReview) (db.BackupCheckReview, error)
}

// BackupChecker is the part of the backup-checker API the status page uses.
type BackupChecker interface {
	ListChecks(ctx context.Context) ([]backupchecker.Check, error)
	GetCheck(ctx context.Context, id string) (backupchecker.Check, error)
}

type BackupService struct {
	cfg        *config.Config
	repo       BackupReviewRepository
	newChecker func(config.BackupConfig) BackupChecker
	now        func() time.Time
}

func NewBackupService(cfg *config.Config, repo BackupReviewRepository) *BackupService {
	return &BackupService{cfg: cfg, repo: repo, now: time.Now,
		newChecker: func(c config.BackupConfig) BackupChecker { return backupchecker.New(c, nil) }}
}

type BackupReviewer struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type BackupReview struct {
	CheckID   string         `json:"check_id"`
	Decision  string         `json:"decision"`
	Comment   string         `json:"comment"`
	Reviewer  BackupReviewer `json:"reviewer"`
	CreatedAt time.Time      `json:"created_at"`
}

type BackupItem struct {
	Backup string              `json:"backup"`
	Check  backupchecker.Check `json:"check"`
	Stale  bool                `json:"stale"`
	Review *BackupReview       `json:"review"`
}

type BackupStatus struct {
	StaleAfterHours float64      `json:"stale_after_hours"`
	Items           []BackupItem `json:"items"`
}

func (s *BackupService) settings(ctx context.Context) (config.BackupConfig, BackupChecker) {
	c := config.Runtime(ctx, s.cfg).Backups
	if c.StaleAfter <= 0 {
		c.StaleAfter = config.DefaultBackupStaleAfter
	}
	return c, s.newChecker(c)
}

// Status returns the latest check per backup with the review of that exact run.
func (s *BackupService) Status(ctx context.Context) (BackupStatus, error) {
	cfg, checker := s.settings(ctx)
	checks, err := checker.ListChecks(ctx)
	if err != nil {
		return BackupStatus{}, mapCheckerError(err)
	}
	latest := latestChecks(checks)
	ids := make([]string, 0, len(latest))
	for _, check := range latest {
		ids = append(ids, check.ID)
	}
	reviews, err := s.repo.ListBackupCheckReviews(ctx, ids)
	if err != nil {
		return BackupStatus{}, err
	}
	now := s.now()
	items := make([]BackupItem, 0, len(latest))
	for _, check := range latest {
		item := BackupItem{Backup: check.Backup, Check: check, Stale: now.Sub(checkedAt(check)) > cfg.StaleAfter}
		if review, ok := reviews[check.ID]; ok {
			item.Review = backupReviewJSON(review)
		}
		items = append(items, item)
	}
	return BackupStatus{StaleAfterHours: cfg.StaleAfter.Hours(), Items: items}, nil
}

// Review records an on-call decision for the latest run of a backup.
func (s *BackupService) Review(ctx context.Context, actorID uuid.UUID, checkID, decision, comment string) (*BackupReview, error) {
	comment = strings.TrimSpace(comment)
	switch decision {
	case BackupDecisionConfirmed:
		comment = ""
	case BackupDecisionRejected:
		if comment == "" {
			return nil, apperrors.Validation("comment is required to reject a check", map[string]any{"field": "comment"})
		}
		if utf8.RuneCountInString(comment) > maxBackupCommentLength {
			return nil, apperrors.Validation("comment must be 500 characters or fewer", map[string]any{"field": "comment"})
		}
	default:
		return nil, apperrors.Validation("unknown decision", nil)
	}
	_, checker := s.settings(ctx)
	check, err := checker.GetCheck(ctx, checkID)
	if err != nil {
		return nil, mapCheckerError(err)
	}
	if decision == BackupDecisionConfirmed && check.Status != backupchecker.StatusPassed {
		return nil, apperrors.Conflict("only a passed check can be confirmed")
	}
	if !check.Finished() {
		return nil, apperrors.Conflict("wait for the check to finish before rejecting it")
	}
	checks, err := checker.ListChecks(ctx)
	if err != nil {
		return nil, mapCheckerError(err)
	}
	for _, latest := range latestChecks(checks) {
		if latest.Backup == check.Backup && latest.ID != check.ID {
			return nil, apperrors.Conflict("a newer check exists for this backup; reload the page")
		}
	}
	stored, err := s.repo.CreateBackupCheckReview(ctx, db.BackupCheckReview{CheckID: check.ID, Backup: check.Backup, Decision: decision, Comment: comment, ReviewerID: actorID})
	if errors.Is(err, db.ErrBackupReviewExists) {
		return nil, apperrors.Conflict("this check already has a review")
	}
	if err != nil {
		return nil, err
	}
	return backupReviewJSON(stored), nil
}

func latestChecks(checks []backupchecker.Check) []backupchecker.Check {
	byBackup := map[string]backupchecker.Check{}
	for _, check := range checks {
		current, ok := byBackup[check.Backup]
		if !ok || check.StartedAt.After(current.StartedAt) {
			byBackup[check.Backup] = check
		}
	}
	out := make([]backupchecker.Check, 0, len(byBackup))
	for _, check := range byBackup {
		out = append(out, check)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Backup < out[j].Backup })
	return out
}

func checkedAt(check backupchecker.Check) time.Time {
	if check.FinishedAt != nil {
		return *check.FinishedAt
	}
	return check.StartedAt
}

func backupReviewJSON(r db.BackupCheckReview) *BackupReview {
	return &BackupReview{CheckID: r.CheckID, Decision: r.Decision, Comment: r.Comment, CreatedAt: r.CreatedAt,
		Reviewer: BackupReviewer{ID: r.ReviewerID.String(), DisplayName: r.ReviewerName, Email: r.ReviewerEmail}}
}

func mapCheckerError(err error) error {
	switch {
	case errors.Is(err, backupchecker.ErrNotConfigured):
		return apperrors.New("BACKUP_CHECKER_NOT_CONFIGURED", "Backup checker is not connected. An administrator can connect it in Settings under Backups.", http.StatusServiceUnavailable)
	case errors.Is(err, backupchecker.ErrUnavailable):
		return apperrors.New("BACKUP_CHECKER_UNAVAILABLE", "Backup checker is not responding. Try again in a few minutes or ask an administrator to check the connection.", http.StatusServiceUnavailable)
	case errors.Is(err, backupchecker.ErrInvalidResponse):
		return apperrors.New("BACKUP_CHECKER_INVALID_RESPONSE", "Backup checker sent a response Aegis cannot read. Check that a supported checker version is installed.", http.StatusBadGateway)
	case errors.Is(err, backupchecker.ErrNotFound):
		return apperrors.NotFound("backup check")
	case errors.Is(err, backupchecker.ErrRunning):
		return apperrors.Conflict("a check for this backup is already running")
	}
	return err
}
