package handler

import (
	"context"
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BackupStatusService interface {
	Status(ctx context.Context) (service.BackupStatus, error)
	Review(ctx context.Context, actorID uuid.UUID, checkID, decision, comment string) (*service.BackupReview, error)
}

// BackupHandler serves the Backup status page API (AEG-110, AEG-111).
type BackupHandler struct {
	backups   BackupStatusService
	auth      middleware.SessionUserResolver
	publicURL string
}

func NewBackupHandler(backups BackupStatusService, auth middleware.SessionUserResolver, publicURL string) *BackupHandler {
	return &BackupHandler{backups: backups, auth: auth, publicURL: publicURL}
}

func (h *BackupHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1/backups", middleware.RequireSession(h.auth))
	api.GET("", h.status)
	review := api.Group("/checks/:check_id", middleware.RequireMutate(), func(c *gin.Context) {
		middleware.RequireSameOrigin(config.PublicURL(c.Request.Context(), h.publicURL))(c)
	})
	review.POST("/confirm", h.review(service.BackupDecisionConfirmed))
	review.POST("/reject", h.review(service.BackupDecisionRejected))
}

func (h *BackupHandler) status(c *gin.Context) {
	status, err := h.backups.Status(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	WriteJSON(c, http.StatusOK, status)
}

func (h *BackupHandler) review(decision string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.UserFromContext(c)
		if !ok {
			WriteError(c, apperrors.Unauthorized("missing session"))
			return
		}
		var body struct {
			Comment string `json:"comment"`
		}
		if c.Request.ContentLength != 0 {
			if err := c.ShouldBindJSON(&body); err != nil {
				WriteError(c, service.ErrInvalidBody())
				return
			}
		}
		review, err := h.backups.Review(c.Request.Context(), user.ID, c.Param("check_id"), decision, body.Comment)
		if err != nil {
			WriteError(c, err)
			return
		}
		WriteJSON(c, http.StatusCreated, review)
	}
}
