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
	teams  *service.TeamService
	auth   *service.AuthService
}

func NewAlertHandler(alerts *service.AlertService, teams *service.TeamService, auth *service.AuthService) *AlertHandler {
	return &AlertHandler{alerts: alerts, teams: teams, auth: auth}
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
	parsed, err := parseListAlertsQuery(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	if parsed.TeamID != nil {
		team, err := h.teams.GetTeam(c.Request.Context(), *parsed.TeamID)
		if err != nil {
			WriteError(c, err)
			return
		}
		parsed.Params.LabelFilters["team"] = team.Name
	}

	result, err := h.alerts.List(c.Request.Context(), parsed.Params)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(result.Items))
	for _, alert := range result.Items {
		items = append(items, service.AlertJSON(alert))
	}
	WriteJSON(c, http.StatusOK, gin.H{
		"items":     items,
		"total":     result.Total,
		"page":      parsed.Page,
		"page_size": parsed.PageSize,
	})
}
