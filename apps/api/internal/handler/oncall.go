package handler

import (
	"net/http"
	"time"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
)

type OnCallHandler struct {
	oncall *service.OnCallService
	auth   *service.AuthService
}

func NewOnCallHandler(oncall *service.OnCallService, auth *service.AuthService) *OnCallHandler {
	return &OnCallHandler{oncall: oncall, auth: auth}
}

func (h *OnCallHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))
	api.GET("/teams/:id/on-call/current", h.currentOnCall)
	api.GET("/teams/:id/on-call/calendar", h.calendar)
}

func (h *OnCallHandler) currentOnCall(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	users, err := h.oncall.CurrentOnCall(c.Request.Context(), teamID)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(users))
	for _, user := range users {
		items = append(items, service.OnCallUserJSON(user))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *OnCallHandler) calendar(c *gin.Context) {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	fromRaw := c.Query("from")
	toRaw := c.Query("to")
	if fromRaw == "" || toRaw == "" {
		WriteError(c, apperrors.Validation("from and to query params are required (RFC3339)", nil))
		return
	}
	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		WriteError(c, apperrors.Validation("from must be RFC3339", nil))
		return
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		WriteError(c, apperrors.Validation("to must be RFC3339", nil))
		return
	}
	slots, err := h.oncall.Calendar(c.Request.Context(), teamID, from, to)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(slots))
	for _, slot := range slots {
		items = append(items, service.OnCallSlotJSON(slot))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}
