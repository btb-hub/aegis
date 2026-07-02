package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analytics *service.AnalyticsService
	alerts    *service.AlertService
	handoffs  *service.HandoffService
	auth      *service.AuthService
}

func NewAnalyticsHandler(
	analytics *service.AnalyticsService,
	alerts *service.AlertService,
	handoffs *service.HandoffService,
	auth *service.AuthService,
) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics, alerts: alerts, handoffs: handoffs, auth: auth}
}

func (h *AnalyticsHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))
	api.GET("/analytics/handoffs", h.handoffStats)
	api.GET("/analytics/mtta", h.mtta)
	api.GET("/analytics/mttr", h.mttr)
	api.GET("/analytics/noise", h.noise)
	api.GET("/analytics/on-call-load", h.onCallLoad)
	api.GET("/analytics/overview", h.overview)

	admin := api.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.POST("/setup/test-alert", h.testAlert)
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

func (h *AnalyticsHandler) mtta(c *gin.Context) {
	h.metricSeries(c, h.analytics.MTTA)
}

func (h *AnalyticsHandler) mttr(c *gin.Context) {
	h.metricSeries(c, h.analytics.MTTR)
}

func (h *AnalyticsHandler) noise(c *gin.Context) {
	from, to, err := parseTimeRange(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	limit := 10
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			WriteError(c, apperrors.Validation("limit must be a positive integer", nil))
			return
		}
		limit = parsed
	}
	stats, err := h.analytics.Noise(c.Request.Context(), from, to, limit)
	if err != nil {
		WriteError(c, err)
		return
	}
	out := service.NoiseJSON(stats)
	out["from"] = from
	out["to"] = to
	WriteJSON(c, http.StatusOK, out)
}

func (h *AnalyticsHandler) onCallLoad(c *gin.Context) {
	from, to, err := parseTimeRange(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	stats, err := h.analytics.OnCallLoad(c.Request.Context(), from, to)
	if err != nil {
		WriteError(c, err)
		return
	}
	out := service.OnCallLoadJSON(stats)
	out["from"] = from
	out["to"] = to
	WriteJSON(c, http.StatusOK, out)
}

func (h *AnalyticsHandler) overview(c *gin.Context) {
	from, to, err := parseTimeRange(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	comparePrevious := parseComparePrevious(c)
	result, err := h.analytics.Overview(c.Request.Context(), from, to, comparePrevious)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.OverviewJSON(result))
}

func (h *AnalyticsHandler) testAlert(c *gin.Context) {
	id, err := h.alerts.SendTestAlert(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusAccepted, gin.H{"id": id.String(), "status": "accepted"})
}

func (h *AnalyticsHandler) metricSeries(
	c *gin.Context,
	fetch func(context.Context, time.Time, time.Time, bool) (service.MetricAnalytics, error),
) {
	from, to, err := parseTimeRange(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	comparePrevious := parseComparePrevious(c)
	result, err := fetch(c.Request.Context(), from, to, comparePrevious)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.MetricAnalyticsJSON(result))
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

func parseComparePrevious(c *gin.Context) bool {
	raw := strings.TrimSpace(c.Query("compare_previous"))
	return raw == "1" || strings.EqualFold(raw, "true")
}
