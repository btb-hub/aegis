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
	incidents   *service.IncidentService
	links       *service.ExpressLinkService
	integrations *service.IntegrationService
}

func NewExpressCallbackHandler(incidents *service.IncidentService, links *service.ExpressLinkService, integrations *service.IntegrationService) *ExpressCallbackHandler {
	return &ExpressCallbackHandler{incidents: incidents, links: links, integrations: integrations}
}

func (h *ExpressCallbackHandler) Register(r gin.IRouter) {
	r.POST("/api/v1/callbacks/express/bot", h.bot)
}

func (h *ExpressCallbackHandler) bot(c *gin.Context) {
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
		user, err := h.links.RedeemLinkCode(c.Request.Context(), code, userHuid)
		if err != nil {
			WriteError(c, err)
			return
		}
		WriteJSON(c, http.StatusOK, gin.H{"status": "linked", "user_id": user.ID.String()})
		return
	}

	incidentID, userHuid, err := intexpress.ParseAckCommand(event)
	if err != nil {
		WriteError(c, apperrors.Validation(err.Error(), nil))
		return
	}
	id, err := uuid.Parse(incidentID)
	if err != nil {
		WriteError(c, apperrors.Validation("invalid incident id", nil))
		return
	}
	incident, err := h.incidents.AcknowledgeByExpressHuid(c.Request.Context(), id, userHuid)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.IncidentJSON(incident))
}
