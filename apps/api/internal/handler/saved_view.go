package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

type SavedViewHandler struct {
	views *service.SavedViewService
	auth  *service.AuthService
}

func NewSavedViewHandler(views *service.SavedViewService, auth *service.AuthService) *SavedViewHandler {
	return &SavedViewHandler{views: views, auth: auth}
}

func (h *SavedViewHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))

	api.GET("/saved-views", h.listViews)
	api.POST("/saved-views", h.createView)
	api.GET("/saved-views/:id", h.getView)
	api.PATCH("/saved-views/:id", h.updateView)
	api.DELETE("/saved-views/:id", h.deleteView)
}

func (h *SavedViewHandler) listViews(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	views, err := h.views.List(c.Request.Context(), user.ID)
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(views))
	for _, view := range views {
		items = append(items, service.SavedViewJSON(view))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *SavedViewHandler) getView(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	view, err := h.views.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.SavedViewJSON(view))
}

func (h *SavedViewHandler) createView(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var body struct {
		Name   string         `json:"name"`
		Filter map[string]any `json:"filter"`
		Shared bool           `json:"shared"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	filter, err := json.Marshal(body.Filter)
	if err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	view, err := h.views.Create(c.Request.Context(), user.ID, body.Name, filter, body.Shared)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusCreated, service.SavedViewJSON(view))
}

func (h *SavedViewHandler) updateView(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	var body struct {
		Name   string         `json:"name"`
		Filter map[string]any `json:"filter"`
		Shared bool           `json:"shared"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	filter, err := json.Marshal(body.Filter)
	if err != nil {
		WriteError(c, service.ErrInvalidBody())
		return
	}
	view, err := h.views.Update(c.Request.Context(), user.ID, id, body.Name, filter, body.Shared)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.SavedViewJSON(view))
}

func (h *SavedViewHandler) deleteView(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.views.Delete(c.Request.Context(), user.ID, id); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
