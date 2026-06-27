package handler

import (
	"net/http"
	"time"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OverrideHandler struct {
	overrides *service.OverrideService
	auth      *service.AuthService
}

func NewOverrideHandler(overrides *service.OverrideService, auth *service.AuthService) *OverrideHandler {
	return &OverrideHandler{overrides: overrides, auth: auth}
}

func (h *OverrideHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))

	api.GET("/teams/:id/overrides", h.listOverrides)

	admin := api.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.POST("/teams/:id/overrides", h.createOverride)
	admin.DELETE("/teams/:id/overrides/:oid", h.deleteOverride)
}

func (h *OverrideHandler) listOverrides(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	overrides, err := h.overrides.ListOverrides(c.Request.Context(), teamID)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(overrides))
	for _, override := range overrides {
		items = append(items, service.OverrideJSON(override))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *OverrideHandler) createOverride(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		UserID  string `json:"user_id"`
		StartAt string `json:"start_at"`
		EndAt   string `json:"end_at"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		WriteError(c, apperrors.Validation("user_id must be a valid uuid", nil))
		return
	}
	startAt, err := time.Parse(time.RFC3339, body.StartAt)
	if err != nil {
		WriteError(c, apperrors.Validation("start_at must be RFC3339", nil))
		return
	}
	endAt, err := time.Parse(time.RFC3339, body.EndAt)
	if err != nil {
		WriteError(c, apperrors.Validation("end_at must be RFC3339", nil))
		return
	}
	override, err := h.overrides.CreateOverride(c.Request.Context(), teamID, service.CreateOverrideInput{
		UserID:  userID,
		StartAt: startAt,
		EndAt:   endAt,
	})
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusCreated, service.OverrideJSON(override))
}

func (h *OverrideHandler) deleteOverride(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	overrideID, err := parseUUIDParam(c, "oid")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.overrides.DeleteOverride(c.Request.Context(), teamID, overrideID); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
