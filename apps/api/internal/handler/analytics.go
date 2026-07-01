package handler

import (
	"net/http"
	"time"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	handoffs *service.HandoffService
	auth     *service.AuthService
}

func NewAnalyticsHandler(handoffs *service.HandoffService, auth *service.AuthService) *AnalyticsHandler {
	return &AnalyticsHandler{handoffs: handoffs, auth: auth}
}

func (h *AnalyticsHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))
	api.GET("/analytics/handoffs", h.handoffStats)
}

func (h *AnalyticsHandler) handoffStats(c *gin.Context) {
	from, to, err := parseTimeRange(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	stats, err := h.handoffs.Stats(c.Request.Context(), from, to)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, gin.H{
		"count":                   stats.Count,
		"median_response_seconds": stats.MedianResponseSeconds,
		"from":                    from,
		"to":                      to,
	})
}

func parseTimeRange(c *gin.Context) (time.Time, time.Time, error) {
	fromRaw := c.Query("from")
	toRaw := c.Query("to")
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, apperrors.Validation("from and to query params are required", nil)
	}
	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, apperrors.Validation("from must be RFC3339", nil)
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, apperrors.Validation("to must be RFC3339", nil)
	}
	return from, to, nil
}
