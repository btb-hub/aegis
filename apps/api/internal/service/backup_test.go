package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/backupchecker"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeBackupChecker struct {
	checks  []backupchecker.Check
	listErr error
	getErr  error
}

// ListChecks mirrors the real list: restore and manifest results are only in the details.
func (f *fakeBackupChecker) ListChecks(context.Context) ([]backupchecker.Check, error) {
	out := make([]backupchecker.Check, len(f.checks))
	for i, c := range f.checks {
		c.RestoreOK, c.ManifestCountsMatch = nil, nil
		out[i] = c
	}
	return out, f.listErr
}

func (f *fakeBackupChecker) GetCheck(_ context.Context, id string) (backupchecker.Check, error) {
	if f.getErr != nil {
		return backupchecker.Check{}, f.getErr
	}
	for _, c := range f.checks {
		if c.ID == id {
			return c, nil
		}
	}
	return backupchecker.Check{}, backupchecker.ErrNotFound
}

type fakeBackupRepo struct {
	reviews   map[string]db.BackupCheckReview
	listErr   error
	createErr error
}

func (f *fakeBackupRepo) ListBackupCheckReviews(_ context.Context, ids []string) (map[string]db.BackupCheckReview, error) {
	out := map[string]db.BackupCheckReview{}
	for _, id := range ids {
		if r, ok := f.reviews[id]; ok {
			out[id] = r
		}
	}
	return out, f.listErr
}

func (f *fakeBackupRepo) CreateBackupCheckReview(_ context.Context, r db.BackupCheckReview) (db.BackupCheckReview, error) {
	if f.createErr != nil {
		return db.BackupCheckReview{}, f.createErr
	}
	if _, ok := f.reviews[r.CheckID]; ok {
		return db.BackupCheckReview{}, db.ErrBackupReviewExists
	}
	r.ReviewerName, r.ReviewerEmail, r.CreatedAt = "On Call", "oncall@example.test", time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	f.reviews[r.CheckID] = r
	return r, nil
}

var backupNow = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func ptrTo[T any](v T) *T { return &v }

func backupCheck(id, backup, status string, started time.Time) backupchecker.Check {
	c := backupchecker.Check{ID: id, Backup: backup, Engine: "postgres", Status: status, StartedAt: started}
	if status != backupchecker.StatusRunning {
		c.FinishedAt = ptrTo(started.Add(30 * time.Minute))
		c.ReplicasMatch, c.RestoreOK, c.ManifestCountsMatch = ptrTo(true), ptrTo(true), ptrTo(status == backupchecker.StatusPassed)
	}
	return c
}

func backupFixture() (*BackupService, *fakeBackupChecker, *fakeBackupRepo) {
	checker := &fakeBackupChecker{checks: []backupchecker.Check{
		backupCheck("gp-old", "gamepass", backupchecker.StatusPassed, backupNow.Add(-48*time.Hour)),
		backupCheck("gp-new", "gamepass", backupchecker.StatusPassed, backupNow.Add(-9*time.Hour)),
		backupCheck("nish-1", "nish", backupchecker.StatusFailed, backupNow.Add(-30*time.Hour)),
		backupCheck("bill-1", "billing", backupchecker.StatusRunning, backupNow.Add(-time.Hour)),
	}}
	repo := &fakeBackupRepo{reviews: map[string]db.BackupCheckReview{
		"gp-old": {CheckID: "gp-old", Backup: "gamepass", Decision: "confirmed", ReviewerID: uuid.New()},
	}}
	svc := NewBackupService(&config.Config{}, repo)
	svc.now = func() time.Time { return backupNow }
	svc.newChecker = func(config.BackupConfig) BackupChecker { return checker }
	return svc, checker, repo
}

func backupAppErr(t *testing.T, err error) *apperrors.Error {
	t.Helper()
	var e *apperrors.Error
	require.True(t, errors.As(err, &e), "got %v", err)
	return e
}

func TestBackupStatusLatestRunAndStale(t *testing.T) {
	svc, _, _ := backupFixture()
	status, err := svc.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, 26.0, status.StaleAfterHours)
	require.Len(t, status.Items, 3)
	require.Equal(t, []string{"billing", "gamepass", "nish"}, []string{status.Items[0].Backup, status.Items[1].Backup, status.Items[2].Backup})
	gamepass := status.Items[1]
	require.Equal(t, "gp-new", gamepass.Check.ID)
	require.Nil(t, gamepass.Review, "a new automatic run waits for a new review")
	require.False(t, gamepass.Stale)
	require.True(t, status.Items[2].Stale)
	require.False(t, status.Items[0].Stale)
}

func TestBackupStatusShowsReviewOfLatestRun(t *testing.T) {
	svc, _, repo := backupFixture()
	repo.reviews["nish-1"] = db.BackupCheckReview{CheckID: "nish-1", Backup: "nish", Decision: "rejected", Comment: "counts differ", ReviewerID: uuid.New(), ReviewerName: "On Call"}
	status, err := svc.Status(context.Background())
	require.NoError(t, err)
	require.NotNil(t, status.Items[2].Review)
	require.Equal(t, "counts differ", status.Items[2].Review.Comment)
	require.Equal(t, "On Call", status.Items[2].Review.Reviewer.DisplayName)
}

func TestBackupStatusUsesRuntimeStaleAfter(t *testing.T) {
	svc, _, _ := backupFixture()
	ctx := config.WithRuntime(context.Background(), &config.Config{Backups: config.BackupConfig{StaleAfter: 4 * time.Hour}})
	status, err := svc.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, 4.0, status.StaleAfterHours)
	require.True(t, status.Items[1].Stale)
}

func TestBackupStatusErrors(t *testing.T) {
	svc, checker, repo := backupFixture()
	for err, code := range map[error]string{
		backupchecker.ErrNotConfigured:   "BACKUP_CHECKER_NOT_CONFIGURED",
		backupchecker.ErrUnavailable:     "BACKUP_CHECKER_UNAVAILABLE",
		backupchecker.ErrInvalidResponse: "BACKUP_CHECKER_INVALID_RESPONSE",
	} {
		checker.listErr = err
		_, got := svc.Status(context.Background())
		require.Equal(t, code, backupAppErr(t, got).Code)
	}
	require.Equal(t, http.StatusServiceUnavailable, backupAppErr(t, mapCheckerError(backupchecker.ErrUnavailable)).StatusCode)
	require.Equal(t, http.StatusBadGateway, backupAppErr(t, mapCheckerError(backupchecker.ErrInvalidResponse)).StatusCode)
	require.Equal(t, "CONFLICT", backupAppErr(t, mapCheckerError(backupchecker.ErrRunning)).Code)
	plain := errors.New("boom")
	require.Equal(t, plain, mapCheckerError(plain))

	checker.listErr = nil
	repo.listErr = plain
	_, got := svc.Status(context.Background())
	require.ErrorIs(t, got, plain)
}

func TestBackupReviewConfirmAndReject(t *testing.T) {
	svc, _, repo := backupFixture()
	ctx := context.Background()
	actor := uuid.New()

	review, err := svc.Review(ctx, actor, "gp-new", BackupDecisionConfirmed, "ignored")
	require.NoError(t, err)
	require.Equal(t, "confirmed", review.Decision)
	require.Empty(t, review.Comment)
	require.Equal(t, actor.String(), review.Reviewer.ID)
	require.Equal(t, "On Call", review.Reviewer.DisplayName)

	_, err = svc.Review(ctx, actor, "gp-new", BackupDecisionRejected, "late")
	require.Equal(t, "this check already has a review", backupAppErr(t, err).Message)

	review, err = svc.Review(ctx, actor, "nish-1", BackupDecisionRejected, "  counts differ  ")
	require.NoError(t, err)
	require.Equal(t, "counts differ", review.Comment)
	require.Equal(t, "nish", repo.reviews["nish-1"].Backup)
}

func TestBackupReviewValidation(t *testing.T) {
	svc, checker, repo := backupFixture()
	ctx := context.Background()
	actor := uuid.New()
	cases := []struct {
		name, id, decision, comment, code, message string
	}{
		{"empty comment", "nish-1", BackupDecisionRejected, "   ", "VALIDATION_ERROR", "comment is required to reject a check"},
		{"long comment", "nish-1", BackupDecisionRejected, strings.Repeat("я", 501), "VALIDATION_ERROR", "comment must be 500 characters or fewer"},
		{"decision", "nish-1", "maybe", "", "VALIDATION_ERROR", "unknown decision"},
		{"confirm failed", "nish-1", BackupDecisionConfirmed, "", "CONFLICT", "only a passed check can be confirmed"},
		{"reject running", "bill-1", BackupDecisionRejected, "stuck", "CONFLICT", "wait for the check to finish before rejecting it"},
		{"older run", "gp-old", BackupDecisionRejected, "old", "CONFLICT", "a newer check exists for this backup; reload the page"},
		{"missing", "nope", BackupDecisionConfirmed, "", "NOT_FOUND", "backup check not found"},
	}
	for _, tc := range cases {
		_, err := svc.Review(ctx, actor, tc.id, tc.decision, tc.comment)
		e := backupAppErr(t, err)
		require.Equal(t, tc.code, e.Code, tc.name)
		require.Equal(t, tc.message, e.Message, tc.name)
	}
	_, err := svc.Review(ctx, actor, "nish-1", BackupDecisionRejected, strings.Repeat("я", 500))
	require.NoError(t, err)

	checker.listErr = backupchecker.ErrUnavailable
	_, err = svc.Review(ctx, actor, "gp-new", BackupDecisionConfirmed, "")
	require.Equal(t, "BACKUP_CHECKER_UNAVAILABLE", backupAppErr(t, err).Code)
	checker.listErr = nil
	plain := errors.New("db down")
	repo.createErr = plain
	_, err = svc.Review(ctx, actor, "gp-new", BackupDecisionConfirmed, "")
	require.ErrorIs(t, err, plain)
}

func TestBackupStatusReadsCheckDetails(t *testing.T) {
	svc, checker, _ := backupFixture()
	status, err := svc.Status(context.Background())
	require.NoError(t, err)
	for _, item := range status.Items {
		if item.Check.Finished() {
			require.NotNil(t, item.Check.RestoreOK, item.Backup)
			require.NotNil(t, item.Check.ManifestCountsMatch, item.Backup)
		}
	}

	checker.getErr = backupchecker.ErrInvalidResponse
	_, err = svc.Status(context.Background())
	require.Equal(t, "BACKUP_CHECKER_INVALID_RESPONSE", backupAppErr(t, err).Code)

	checker.getErr = nil
	other := &fakeBackupChecker{checks: checker.checks}
	svc.newChecker = func(config.BackupConfig) BackupChecker { return &swappedTarget{other} }
	_, err = svc.Status(context.Background())
	require.Equal(t, "BACKUP_CHECKER_INVALID_RESPONSE", backupAppErr(t, err).Code, "details for another target")
}

type swappedTarget struct{ *fakeBackupChecker }

func (s *swappedTarget) GetCheck(ctx context.Context, id string) (backupchecker.Check, error) {
	c, err := s.fakeBackupChecker.GetCheck(ctx, id)
	c.Backup = "other"
	return c, err
}

func TestNewBackupServiceUsesHTTPChecker(t *testing.T) {
	svc := NewBackupService(&config.Config{}, &fakeBackupRepo{})
	_, err := svc.Status(context.Background())
	require.Equal(t, "BACKUP_CHECKER_NOT_CONFIGURED", backupAppErr(t, err).Code)
}
