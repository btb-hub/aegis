package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeBackupService struct {
	status    service.BackupStatus
	statusErr error
	reviewErr error
	got       struct {
		actor                      uuid.UUID
		checkID, decision, comment string
	}
}

func (f *fakeBackupService) Status(context.Context) (service.BackupStatus, error) {
	return f.status, f.statusErr
}

func (f *fakeBackupService) Review(_ context.Context, actor uuid.UUID, checkID, decision, comment string) (*service.BackupReview, error) {
	f.got.actor, f.got.checkID, f.got.decision, f.got.comment = actor, checkID, decision, comment
	if f.reviewErr != nil {
		return nil, f.reviewErr
	}
	return &service.BackupReview{CheckID: checkID, Decision: decision, Comment: comment}, nil
}

func backupRouter(role string) (*gin.Engine, *fakeBackupService, db.User) {
	gin.SetMode(gin.TestMode)
	user := db.User{ID: uuid.New(), Role: role, Locale: "en"}
	svc := &fakeBackupService{status: service.BackupStatus{StaleAfterHours: 26, Items: []service.BackupItem{}}}
	r := gin.New()
	NewBackupHandler(svc, anomalyAuth{user}, "https://aegis.test").Register(r)
	return r, svc, user
}

func TestBackupStatusRoute(t *testing.T) {
	r, svc, _ := backupRouter("viewer")
	w := anomalyRequest(r, http.MethodGet, "/api/v1/backups", "", "session", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 26.0, body["stale_after_hours"])

	w = anomalyRequest(r, http.MethodGet, "/api/v1/backups", "", "nope", "")
	require.Equal(t, http.StatusUnauthorized, w.Code)

	svc.statusErr = apperrors.New("BACKUP_CHECKER_UNAVAILABLE", "Backup checker is not responding.", http.StatusServiceUnavailable)
	w = anomalyRequest(r, http.MethodGet, "/api/v1/backups", "", "session", "")
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Contains(t, w.Body.String(), "BACKUP_CHECKER_UNAVAILABLE")
}

func TestBackupReviewRoutes(t *testing.T) {
	r, svc, user := backupRouter("member")
	w := anomalyRequest(r, http.MethodPost, "/api/v1/backups/checks/chk-1/confirm", "", "session", "https://aegis.test")
	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, user.ID, svc.got.actor)
	require.Equal(t, "chk-1", svc.got.checkID)
	require.Equal(t, "confirmed", svc.got.decision)

	w = anomalyRequest(r, http.MethodPost, "/api/v1/backups/checks/chk-2/reject", `{"comment":"counts differ"}`, "session", "https://aegis.test")
	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "rejected", svc.got.decision)
	require.Equal(t, "counts differ", svc.got.comment)

	w = anomalyRequest(r, http.MethodPost, "/api/v1/backups/checks/chk-2/reject", `{"comment":`, "session", "https://aegis.test")
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = anomalyRequest(r, http.MethodPost, "/api/v1/backups/checks/chk-2/reject", `{}`, "session", "https://evil.test")
	require.Equal(t, http.StatusForbidden, w.Code)

	svc.reviewErr = apperrors.Validation("comment is required to reject a check", nil)
	w = anomalyRequest(r, http.MethodPost, "/api/v1/backups/checks/chk-2/reject", `{"comment":" "}`, "session", "https://aegis.test")
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.True(t, strings.Contains(w.Body.String(), "comment is required"))
}

func TestBackupReviewForbiddenForViewer(t *testing.T) {
	r, svc, _ := backupRouter("viewer")
	w := anomalyRequest(r, http.MethodPost, "/api/v1/backups/checks/chk-1/confirm", "", "session", "https://aegis.test")
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Empty(t, svc.got.checkID)
}
