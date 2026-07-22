package handler

import (
	"net/http"

	"github.com/aegis/aegis/apps/api/internal/middleware"
	"github.com/aegis/aegis/apps/api/internal/service"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RoutingHandler struct {
	routing *service.RoutingService
	auth    *service.AuthService
}

func NewRoutingHandler(routing *service.RoutingService, auth *service.AuthService) *RoutingHandler {
	return &RoutingHandler{routing: routing, auth: auth}
}

func (h *RoutingHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth))

	api.GET("/routing-rules", h.listRules)

	admin := api.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.POST("/routing-rules", h.createRule)
	admin.PATCH("/routing-rules/:id", h.updateRule)
	admin.DELETE("/routing-rules/:id", h.deleteRule)
}

func (h *RoutingHandler) listRules(c *gin.Context) {
	rules, err := h.routing.ListRules(c.Request.Context())
	if err != nil {
		WriteError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		items = append(items, service.RoutingRuleJSON(rule))
	}
	WriteJSON(c, http.StatusOK, gin.H{"items": items})
}

func (h *RoutingHandler) createRule(c *gin.Context) {
	input, err := parseRoutingRuleBody(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	rule, err := h.routing.CreateRule(c.Request.Context(), input)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusCreated, service.RoutingRuleJSON(rule))
}

func (h *RoutingHandler) updateRule(c *gin.Context) {
	ruleID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	input, err := parseRoutingRuleBody(c)
	if err != nil {
		WriteError(c, err)
		return
	}
	rule, err := h.routing.UpdateRule(c.Request.Context(), ruleID, input)
	if err != nil {
		WriteError(c, err)
		return
	}
	WriteJSON(c, http.StatusOK, service.RoutingRuleJSON(rule))
}

func (h *RoutingHandler) deleteRule(c *gin.Context) {
	ruleID, err := parseUUIDParam(c, "id")
	if err != nil {
		WriteError(c, err)
		return
	}
	if err := h.routing.DeleteRule(c.Request.Context(), ruleID); err != nil {
		WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseRoutingRuleBody(c *gin.Context) (service.RoutingRuleInput, error) {
	var body struct {
		WorkspaceID    string            `json:"workspace_id"`
		TeamID         string            `json:"team_id"`
		MatchLabels    map[string]string `json:"match_labels"`
		Priority       int32             `json:"priority"`
		CrossWorkspace bool              `json:"cross_workspace"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return service.RoutingRuleInput{}, service.ErrInvalidBody()
	}
	workspaceID, err := uuid.Parse(body.WorkspaceID)
	if err != nil {
		return service.RoutingRuleInput{}, apperrors.Validation("workspace_id must be a valid uuid", nil)
	}
	teamID, err := uuid.Parse(body.TeamID)
	if err != nil {
		return service.RoutingRuleInput{}, apperrors.Validation("team_id must be a valid uuid", nil)
	}
	return service.RoutingRuleInput{
		WorkspaceID:    workspaceID,
		TeamID:         teamID,
		MatchLabels:    body.MatchLabels,
		Priority:       body.Priority,
		CrossWorkspace: body.CrossWorkspace,
	}, nil
}
