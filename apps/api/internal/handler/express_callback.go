package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/gin-gonic/gin"
)

type ExpressCallbackHandler struct {
	commands     *service.ExpressCommandService
	integrations *service.IntegrationService
}

func NewExpressCallbackHandler(commands *service.ExpressCommandService, integrations *service.IntegrationService) *ExpressCallbackHandler {
	return &ExpressCallbackHandler{commands: commands, integrations: integrations}
}

func (h *ExpressCallbackHandler) Register(r gin.IRouter) {
	r.GET("/api/v1/callbacks/express/status", h.status)
	r.GET("/api/v1/callbacks/express/bot/status", h.status)
	r.POST("/api/v1/callbacks/express/command", h.command)
	r.POST("/api/v1/callbacks/express/bot/command", h.command)
	r.POST("/api/v1/callbacks/express/bot", h.command)
	r.POST("/api/v1/callbacks/express/notification/callback", h.notificationResult)
	r.POST("/api/v1/callbacks/express/bot/notification/callback", h.notificationResult)
}

func (h *ExpressCallbackHandler) status(c *gin.Context) {
	_, err := h.integrations.ExpressSecretKey(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"reason":     "bot_disabled",
			"error_data": gin.H{"status_message": "eXpress integration is not configured"},
			"errors":     []any{},
		})
		return
	}
	if auth := c.GetHeader("Authorization"); strings.TrimSpace(auth) != "" {
		if _, err := h.integrations.ExpressConnector(c.Request.Context(), "", auth); err != nil {
			WriteError(c, apperrors.Unauthorized("invalid express signature"))
			return
		}
	}
	WriteJSON(c, http.StatusOK, gin.H{
		"status": "ok",
		"result": gin.H{
			"enabled":        true,
			"status_message": "Bot is working",
			"commands": []gin.H{
				{"body": "/oncall", "name": "oncall", "description": "Current on-call engineers", "command_type": "user"},
				{
					"body":         "/link",
					"name":         "link",
					"description":  "Bind Aegis paging identity",
					"command_type": "user",
				},
				{
					"body":         "/ack_incident",
					"name":         "ack_incident",
					"description":  "Acknowledge incident",
					"command_type": "user",
				},
			},
		},
	})
}

func (h *ExpressCallbackHandler) command(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		WriteError(c, apperrors.Validation("invalid body", nil))
		return
	}
	event, err := intexpress.ParseCommandEvent(body)
	if err != nil {
		WriteError(c, apperrors.Validation("invalid command JSON", nil))
		return
	}
	connector, err := h.integrations.ExpressConnector(c.Request.Context(), event.BotID, c.GetHeader("Authorization"))
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.commands.Process(c.Request.Context(), connector, event); err != nil {
		WriteJSON(c, http.StatusServiceUnavailable, gin.H{"reason": "bot_disabled", "error_data": gin.H{"status_message": "Command could not be persisted; retry"}, "errors": []any{}})
		return
	}
	writeCommandAccepted(c)
}
func (h *ExpressCallbackHandler) notificationResult(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		WriteError(c, apperrors.Validation("invalid body", nil))
		return
	}
	connectors, err := h.integrations.ExpressConnectors(c.Request.Context(), "", c.GetHeader("Authorization"))
	if err != nil {
		WriteError(c, err)
		return
	}
	for _, connector := range connectors {
		if err := h.commands.NotificationResult(c.Request.Context(), connector, body); err != nil {
			WriteError(c, err)
			return
		}
	}
	writeCommandAccepted(c)
}

func writeCommandAccepted(c *gin.Context) {
	WriteJSON(c, http.StatusAccepted, gin.H{"result": "accepted"})
}
