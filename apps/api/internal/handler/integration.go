package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IntegrationHandler struct {
	integrations *service.IntegrationService
	auth         *service.AuthService
}

func NewIntegrationHandler(integrations *service.IntegrationService, auth *service.AuthService) *IntegrationHandler {
	return &IntegrationHandler{integrations: integrations, auth: auth}
}

func (h *IntegrationHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))

	api.GET("/integrations", h.listIntegrations)
	api.GET("/integrations/:id", h.getIntegration)

	admin := api.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.POST("/integrations", h.upsertIntegration)
	admin.PATCH("/integrations/:id", h.patchIntegration)
	admin.DELETE("/integrations/:id", h.deleteIntegration)
	admin.POST("/integrations/:id/test", h.testIntegration)
}

func (h *IntegrationHandler) listIntegrations(c *gin.Context) {
	items, err := h.integrations.List(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	globalEnabled := make(map[string]bool)
	for _, item := range items {
		if item.WorkspaceID == nil && item.Enabled {
			globalEnabled[item.Kind] = true
		}
	}
	for _, item := range items {
		out = append(out, service.IntegrationJSON(item, globalEnabled[item.Kind]))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": out})
}

func (h *IntegrationHandler) getIntegration(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	item, err := h.integrations.Get(c.Request.Context(), id)
	if err != nil {
		WriteError(c, err)
		return
	}
	out, err := h.integrations.JSON(c.Request.Context(), item)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, out)
}

func (h *IntegrationHandler) upsertIntegration(c *gin.Context) {
	var body struct {
		Kind        string          `json:"kind"`
		Name        string          `json:"name"`
		Config      json.RawMessage `json:"config"`
		Enabled     bool            `json:"enabled"`
		WorkspaceID *string         `json:"workspace_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	var workspaceID *uuid.UUID
	if body.WorkspaceID != nil && *body.WorkspaceID != "" {
		id, err := uuid.Parse(*body.WorkspaceID)
		if err != nil {
			WriteError(c, apperrors.Validation("workspace_id must be a valid uuid", nil))
			return
		}
		workspaceID = &id
	}
	item, err := h.integrations.Upsert(c.Request.Context(), body.Kind, body.Name, body.Config, body.Enabled, workspaceID)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusCreated, service.IntegrationJSON(item))
}

func (h *IntegrationHandler) patchIntegration(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		Name    *string          `json:"name"`
		Enabled *bool            `json:"enabled"`
		Config  *json.RawMessage `json:"config"`
		Mode    *string          `json:"mode"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	var config json.RawMessage
	if body.Config != nil {
		config = *body.Config
	}
	item, err := h.integrations.Update(c.Request.Context(), id, body.Name, body.Enabled, config, body.Mode)
	if err != nil {
		WriteError(c, err)
		return
	}
	out, err := h.integrations.JSON(c.Request.Context(), item)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, out)
}

func (h *IntegrationHandler) deleteIntegration(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.integrations.Delete(c.Request.Context(), id); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *IntegrationHandler) testIntegration(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.integrations.Test(c.Request.Context(), id); err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, gin.H{"status": "ok"})
}

type SlackCallbackHandler struct {
	incidents *service.IncidentService
	secret    string
}

func NewSlackCallbackHandler(incidents *service.IncidentService, signingSecret string) *SlackCallbackHandler {
	return &SlackCallbackHandler{incidents: incidents, secret: signingSecret}
}

func (h *SlackCallbackHandler) Register(r gin.IRouter) {
	r.POST("/api/v1/callbacks/slack/interactive", h.interactive)
}

func (h *SlackCallbackHandler) interactive(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, apperrors.Validation("invalid body", nil))
		return
	}
	if err := verifySlackSignature(h.secret, c.GetHeader("X-Slack-Request-Timestamp"), c.GetHeader("X-Slack-Signature"), body); err != nil {
		WriteError(c, apperrors.Unauthorized("invalid slack signature"))
		return
	}
	incidentID, slackUserID, err := parseSlackAck(body)
	if err != nil {
		WriteError(c, apperrors.Validation(err.Error(), nil))
		return
	}
	id, err := uuid.Parse(incidentID)
	if err != nil {
		WriteError(c, apperrors.Validation("invalid incident id", nil))
		return
	}
	incident, err := h.incidents.AcknowledgeBySlackUser(c.Request.Context(), id, slackUserID)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.IncidentJSON(incident))
}
