package handler

import (
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ExpressLinkHandler struct {
	links *service.ExpressLinkService
	auth  *service.AuthService
}

func NewExpressLinkHandler(links *service.ExpressLinkService, auth *service.AuthService) *ExpressLinkHandler {
	return &ExpressLinkHandler{links: links, auth: auth}
}

func (h *ExpressLinkHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))
	api.POST("/users/me/express-link-code", h.createLinkCode)
	api.POST("/users/me/express-link", h.bindExpressHuid)
}

func (h *ExpressLinkHandler) createLinkCode(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	code, err := h.links.CreateLinkCode(c.Request.Context(), user.ID)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, gin.H{"code": code, "command": "/link " + code})
}

func (h *ExpressLinkHandler) bindExpressHuid(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var body struct {
		ExpressUserHuid string `json:"express_user_huid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	updated, err := h.links.BindExpressHuid(c.Request.Context(), user.ID, body.ExpressUserHuid)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, gin.H{
		"user_id":           updated.ID.String(),
		"express_user_huid": huidString(updated.ExpressUserHuid),
	})
}

func huidString(huid pgtype.UUID) any {
	if !huid.Valid {
		return nil
	}
	return uuid.UUID(huid.Bytes).String()
}
