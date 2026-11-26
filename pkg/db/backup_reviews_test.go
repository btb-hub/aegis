package db

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupCheckReviewLifecycle(t *testing.T) {
	s := settingsStore(t)
	ctx := context.Background()
	user, err := s.UpsertUser(ctx, "google", "backup-reviewer", "oncall@example.test", "On Call", "member", "en")
	require.NoError(t, err)

	empty, err := s.ListBackupCheckReviews(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	stored, err := s.CreateBackupCheckReview(ctx, BackupCheckReview{CheckID: "chk-1", Backup: "gamepass", Decision: "rejected", Comment: "counts differ", ReviewerID: user.ID})
	require.NoError(t, err)
	require.Equal(t, "On Call", stored.ReviewerName)
	require.Equal(t, "oncall@example.test", stored.ReviewerEmail)
	require.False(t, stored.CreatedAt.IsZero())

	_, err = s.CreateBackupCheckReview(ctx, BackupCheckReview{CheckID: "chk-1", Backup: "gamepass", Decision: "confirmed", ReviewerID: user.ID})
	require.ErrorIs(t, err, ErrBackupReviewExists)

	_, err = s.CreateBackupCheckReview(ctx, BackupCheckReview{CheckID: "chk-2", Backup: "nish", Decision: "rejected", Comment: "  ", ReviewerID: user.ID})
	require.Error(t, err, "database rejects a rejection without a comment")

	_, err = s.CreateBackupCheckReview(ctx, BackupCheckReview{CheckID: "chk-3", Backup: "nish", Decision: "confirmed", ReviewerID: user.ID})
	require.NoError(t, err)

	reviews, err := s.ListBackupCheckReviews(ctx, []string{"chk-1", "chk-3", "missing"})
	require.NoError(t, err)
	require.Len(t, reviews, 2)
	require.Equal(t, "confirmed", reviews["chk-3"].Decision)

	var action string
	var raw []byte
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT action, details FROM audit_log WHERE resource_id=$1`, stored.ID).Scan(&action, &raw))
	require.Equal(t, "backup.check_rejected", action)
	var details map[string]any
	require.NoError(t, json.Unmarshal(raw, &details))
	require.Equal(t, map[string]any{"check_id": "chk-1", "backup": "gamepass", "decision": "rejected", "comment": "counts differ"}, details)
}
