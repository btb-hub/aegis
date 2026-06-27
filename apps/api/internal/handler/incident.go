package handler

import (
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
)

type IncidentHandler struct {
	incidents *service.IncidentService
	auth      *service.AuthService
}

func NewIncidentHandler(incidents *service.IncidentService, auth *service.AuthService) *IncidentHandler {
	return &IncidentHandler{incidents: incidents, auth: auth}
}

func (h *IncidentHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))

	api.GET("/incidents", h.listIncidents)
	api.GET("/incidents/:id", h.getIncident)
	api.GET("/incidents/:id/timeline", h.listTimeline)
	api.POST("/incidents/:id/acknowledge", h.acknowledge)
	api.POST("/incidents/:id/resolve", h.resolve)
}

func (h *IncidentHandler) listIncidents(c *gin.Context) {
	incidents, err := h.incidents.List(c.Request.Context(), c.Query("status"))
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(incidents))
	for _, incident := range incidents {
		items = append(items, service.IncidentJSON(incident))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *IncidentHandler) getIncident(c *gin.Context) {
	incidentID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	incident, err := h.incidents.Get(c.Request.Context(), incidentID)
	if err != nil {
		WriteError(c, err)
		return
	}
	alerts, err := h.incidents.Alerts(c.Request.Context(), incidentID)
	if err != nil {
		WriteError(c, err)
		return
	}
	alertItems := make([]map[string]any, 0, len(alerts))
	for _, alert := range alerts {
		alertItems = append(alertItems, map[string]any{
			"id":       alert.ID.String(),
			"severity": alert.Severity,
			"title":    alert.Title,
			"status":   alert.Status,
		})
	}
	WriteJSON(c, http.StatusOK, gin.H{
		"incident": service.IncidentJSON(incident),
		"alerts":   alertItems,
	})
}

func (h *IncidentHandler) listTimeline(c *gin.Context) {
	incidentID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	events, err := h.incidents.Timeline(c.Request.Context(), incidentID)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(events))
	for _, event := range events {
		items = append(items, service.TimelineEventJSON(event))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *IncidentHandler) acknowledge(c *gin.Context) {
	incidentID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	user, ok := middleware.UserFromContext(c)
	if !ok {
		WriteError(c, apperrors.Unauthorized("missing session"))
		return
	}
	incident, err := h.incidents.Acknowledge(c.Request.Context(), incidentID, user.ID)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.IncidentJSON(incident))
}

func (h *IncidentHandler) resolve(c *gin.Context) {
	incidentID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	user, ok := middleware.UserFromContext(c)
	if !ok {
		WriteError(c, apperrors.Unauthorized("missing session"))
		return
	}
	incident, err := h.incidents.Resolve(c.Request.Context(), incidentID, user.ID)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.IncidentJSON(incident))
}
