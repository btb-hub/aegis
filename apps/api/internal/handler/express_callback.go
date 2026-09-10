package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ExpressCallbackHandler struct {
	incidents    *service.IncidentService
	links        *service.ExpressLinkService
	integrations *service.IntegrationService
}

func NewExpressCallbackHandler(incidents *service.IncidentService, links *service.ExpressLinkService, integrations *service.IntegrationService) *ExpressCallbackHandler {
	return &ExpressCallbackHandler{incidents: incidents, links: links, integrations: integrations}
}

func (h *ExpressCallbackHandler) Register(r gin.IRouter) {
	r.GET("/api/v1/callbacks/express/status", h.status)
	r.POST("/api/v1/callbacks/express/command", h.command)
	r.POST("/api/v1/callbacks/express/bot", h.command)
}

func (h *ExpressCallbackHandler) status(c *gin.Context) {
	secret, err := h.integrations.ExpressSecretKey(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"reason":     "bot_disabled",
			"error_data": gin.H{"status_message": "eXpress integration is not configured"},
			"errors":     []any{},
		})
		return
	}
	if err := intexpress.VerifyAuthorization(c.GetHeader("Authorization"), secret); err != nil {
		WriteError(c, apperrors.Unauthorized("invalid express signature"))
		return
	}
	WriteJSON(c, http.StatusOK, gin.H{
		"status": "ok",
		"result": gin.H{
			"enabled":        true,
			"status_message": "Bot is working",
			"commands": []gin.H{
				{
					"name":        "link",
					"body":        "/link",
					"description": "Bind your Aegis account",
				},
			},
		},
	})
}

func (h *ExpressCallbackHandler) command(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, apperrors.Validation("invalid body", nil))
		return
	}
	secret, err := h.integrations.ExpressSecretKey(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := intexpress.VerifyAuthorization(c.GetHeader("Authorization"), secret); err != nil {
		WriteError(c, apperrors.Unauthorized("invalid express signature"))
		return
	}

	event, err := intexpress.ParseCommandEvent(body)
	if err != nil {
		WriteError(c, apperrors.Validation(err.Error(), nil))
		return
	}

	if strings.HasPrefix(strings.TrimSpace(event.Command.Body), "/link") {
		code, userHuid, err := intexpress.ParseLinkCommand(event)
		if err != nil {
			WriteError(c, apperrors.Validation(err.Error(), nil))
			return
		}
		if _, err := h.links.RedeemLinkCode(c.Request.Context(), code, userHuid); err != nil {
			WriteError(c, err)
			return
		}
		writeCommandAccepted(c)
		return
	}

	incidentID, userHuid, err := intexpress.ParseAckCommand(event)
	if err != nil {
		writeCommandAccepted(c)
		return
	}
	id, err := uuid.Parse(incidentID)
	if err != nil {
		WriteError(c, apperrors.Validation("invalid incident id", nil))
		return
	}
	if _, err := h.incidents.AcknowledgeByExpressHuid(c.Request.Context(), id, userHuid); err != nil {
		WriteError(c, err)
		return
	}
	writeCommandAccepted(c)
}

func writeCommandAccepted(c *gin.Context) {
	WriteJSON(c, http.StatusAccepted, gin.H{"result": "accepted"})
}
