package handler

import (
	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type SettingsHandler struct {
	settings *service.SettingsService
	auth     *service.AuthService
}

func NewSettingsHandler(settings *service.SettingsService, auth *service.AuthService) *SettingsHandler {
	return &SettingsHandler{settings: settings, auth: auth}
}
func (h *SettingsHandler) Register(r gin.IRouter) {
	g := r.Group("/api/v1/settings")
	g.Use(middleware.RequireSession(h.auth), middleware.RequireAdmin())
	g.GET("/oncall-publication", h.get)
	g.PATCH("/oncall-publication", h.update)
}
func (h *SettingsHandler) get(c *gin.Context) {
	out, err := h.settings.Get(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, out)
}
func (h *SettingsHandler) update(c *gin.Context) {
	var body struct {
		Time     string `json:"time"`
		Timezone string `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	out, err := h.settings.Update(c.Request.Context(), body.Time, body.Timezone)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, out)
}
