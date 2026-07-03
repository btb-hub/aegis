package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	users *service.UserService
	auth  *service.AuthService
}

func NewUserHandler(users *service.UserService, auth *service.AuthService) *UserHandler {
	return &UserHandler{users: users, auth: auth}
}

func (h *UserHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth), middleware.RequireAdmin())
	api.GET("/users", h.listUsers)
}

func (h *UserHandler) listUsers(c *gin.Context) {
	page, pageSize, query, err := parseListUsersQuery(c)
	if err != nil {
		WriteError(c, err)
		return
	}

	result, err := h.users.ListUsers(c.Request.Context(), db.ListUsersParams{Query: query}, page, pageSize)
	if err != nil {
		WriteError(c, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, profile := range result.Items {
		items = append(items, service.UserJSON(profile.User, profile.Identities))
	}
	WriteJSON(c, http.StatusOK, gin.H{
		"items":     items,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

func parseListUsersQuery(c *gin.Context) (page int, pageSize int, query string, err error) {
	page = 1
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 1 {
			return 0, 0, "", apperrors.Validation("page must be a positive integer", nil)
		}
		page = value
	}

	pageSize = db.DefaultUserListLimit
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 1 {
			return 0, 0, "", apperrors.Validation("page_size must be a positive integer", nil)
		}
		if value > db.DefaultUserListLimit {
			value = db.DefaultUserListLimit
		}
		pageSize = value
	}

	return page, pageSize, strings.TrimSpace(c.Query("q")), nil
}
