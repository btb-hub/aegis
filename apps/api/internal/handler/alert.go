package handler

import (
	"io"
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	alerts *service.AlertService
	auth   *service.AuthService
}

func NewAlertHandler(alerts *service.AlertService, auth *service.AuthService) *AlertHandler {
	return &AlertHandler{alerts: alerts, auth: auth}
}

func (h *AlertHandler) Register(r gin.IRouter) {
	r.POST("/api/v1/alerts/webhook", h.webhook)

	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))
	api.GET("/alerts", h.listAlerts)
}

func (h *AlertHandler) webhook(c *gin.Context) {
	secret := c.GetHeader("X-Aegis-Webhook-Secret")
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	id, err := h.alerts.Ingest(c.Request.Context(), secret, raw)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusAccepted, gin.H{"id": id.String(), "status": "accepted"})
}

func (h *AlertHandler) listAlerts(c *gin.Context) {
	alerts, err := h.alerts.List(c.Request.Context(), c.Query("q"))
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(alerts))
	for _, alert := range alerts {
		items = append(items, service.AlertJSON(alert))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}
