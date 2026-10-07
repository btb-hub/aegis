package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
)

type PagingHandler struct {
	paging    *service.PagingService
	auth      *service.AuthService
	publicURL string
}

func NewPagingHandler(paging *service.PagingService, auth *service.AuthService, publicURL string) *PagingHandler {
	return &PagingHandler{paging: paging, auth: auth, publicURL: publicURL}
}

func (h *PagingHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1/users/me/paging-connections", middleware.RequireSession(h.auth))
	api.GET("", h.connections)
	api.POST("/:provider/authorize", middleware.RequireSameOrigin(h.publicURL), h.authorize)
	api.DELETE("/:provider", middleware.RequireSameOrigin(h.publicURL), h.disconnect)
}

func (h *PagingHandler) connections(c *gin.Context) {
	user, _ := middleware.UserFromContext(c)
	connections, err := h.paging.Connections(c.Request.Context(), user)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, gin.H{"connections": connections})
}

func (h *PagingHandler) authorize(c *gin.Context) {
	user, _ := middleware.UserFromContext(c)
	authorizationURL, err := h.paging.Authorize(c.Request.Context(), user.ID, SessionToken(c), c.Param("provider"))
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, gin.H{"authorization_url": authorizationURL})
}

func (h *PagingHandler) disconnect(c *gin.Context) {
	user, _ := middleware.UserFromContext(c)
	if err := h.paging.Disconnect(c.Request.Context(), user.ID, c.Param("provider")); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Callback is reached only for paging-prefixed state, including invalid or expired attempts.
func (h *PagingHandler) Callback(c *gin.Context) {
	provider := c.Param("provider")
	err := h.paging.Complete(c.Request.Context(), SessionToken(c), provider, c.Query("state"), c.Query("code"), c.Query("error"))
	query := url.Values{}
	if provider == "slack" || provider == "express" {
		query.Set("paging_provider", provider)
	}
	if err == nil {
		query.Set("paging_result", "connected")
	} else {
		code := "provider_unavailable"
		var appErr *apperrors.Error
		if errors.As(err, &appErr) && strings.HasPrefix(appErr.Code, "PAGING_") {
			code = appErr.Message
		}
		query.Set("paging_error", code)
	}
	c.Redirect(http.StatusFound, strings.TrimRight(h.publicURL, "/")+"/account?"+query.Encode())
}
